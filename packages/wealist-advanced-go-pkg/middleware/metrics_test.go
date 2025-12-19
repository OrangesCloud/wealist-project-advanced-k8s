// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 metrics.go의 테스트를 포함합니다.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// newTestMetrics는 테스트용 격리된 메트릭을 생성합니다.
func newTestMetrics(namespace string) *commonmetrics.Metrics {
	return commonmetrics.NewForTest(namespace)
}

// getCounterValue는 CounterVec에서 특정 라벨의 값을 가져옵니다.
func getCounterValue(counter *prometheus.CounterVec, labels ...string) float64 {
	m := &dto.Metric{}
	if err := counter.WithLabelValues(labels...).Write(m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

// getHistogramCount는 HistogramVec에서 특정 라벨의 관측 횟수를 가져옵니다.
func getHistogramCount(histogram *prometheus.HistogramVec, labels ...string) uint64 {
	// Histogram에서 메트릭 가져오기 위해 Collect 사용
	ch := make(chan prometheus.Metric, 1)
	histogram.WithLabelValues(labels...).(prometheus.Histogram).Collect(ch)
	select {
	case m := <-ch:
		metric := &dto.Metric{}
		if err := m.Write(metric); err != nil {
			return 0
		}
		return metric.GetHistogram().GetSampleCount()
	default:
		return 0
	}
}

// TestMetricsMiddleware는 MetricsMiddleware가 올바르게 메트릭을 기록하는지 테스트합니다.
func TestMetricsMiddleware(t *testing.T) {
	m := newTestMetrics("test_service")

	router := gin.New()
	router.Use(MetricsMiddleware(m, "/api"))
	router.GET("/api/test", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // 측정 가능한 지연
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)

	router.ServeHTTP(w, req)

	// 상태 코드 확인
	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}

	// HTTP 요청 카운터 확인
	// FullPath()는 실제 경로가 아닌 라우트 패턴을 반환함
	count := getCounterValue(m.HTTPRequestsTotal, "GET", "/api/test", "2xx")
	if count != 1 {
		t.Errorf("HTTP 요청 카운터가 1이어야 함, 실제: %f", count)
	}

	// HTTP duration 히스토그램 확인
	histCount := getHistogramCount(m.HTTPRequestDuration, "GET", "/api/test")
	if histCount != 1 {
		t.Errorf("HTTP duration 히스토그램 카운트가 1이어야 함, 실제: %d", histCount)
	}
}

// TestMetricsMiddleware_SkipPaths는 특정 경로가 메트릭에서 스킵되는지 테스트합니다.
func TestMetricsMiddleware_SkipPaths(t *testing.T) {
	m := newTestMetrics("test_skip")

	router := gin.New()
	router.Use(MetricsMiddleware(m, "/api"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/health/live", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/health/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 스킵되어야 할 경로들 요청
	skipPaths := []string{"/health", "/metrics", "/health/live", "/health/ready", "/api/health"}
	for _, path := range skipPaths {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		router.ServeHTTP(w, req)
	}

	// 스킵 경로들은 메트릭이 기록되지 않아야 함
	for _, path := range skipPaths {
		count := getCounterValue(m.HTTPRequestsTotal, "GET", path, "2xx")
		if count != 0 {
			t.Errorf("경로 '%s'는 스킵되어야 하는데 메트릭이 기록됨: %f", path, count)
		}
	}
}

// TestMetricsMiddleware_StatusCodes는 다양한 상태 코드가 올바르게 분류되는지 테스트합니다.
func TestMetricsMiddleware_StatusCodes(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		statusCode     int    // HTTP 상태 코드
		expectedStatus string // 예상 메트릭 라벨 (2xx, 4xx, 5xx)
	}{
		{"200 OK", http.StatusOK, "2xx"},
		{"201 Created", http.StatusCreated, "2xx"},
		{"204 No Content", http.StatusNoContent, "2xx"},
		{"400 Bad Request", http.StatusBadRequest, "4xx"},
		{"401 Unauthorized", http.StatusUnauthorized, "4xx"},
		{"404 Not Found", http.StatusNotFound, "4xx"},
		{"500 Internal Server Error", http.StatusInternalServerError, "5xx"},
		{"503 Service Unavailable", http.StatusServiceUnavailable, "5xx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 각 테스트마다 새 메트릭 인스턴스 생성 (격리)
			m := newTestMetrics("test_status_" + tt.expectedStatus)

			router := gin.New()
			router.Use(MetricsMiddleware(m, ""))
			router.GET("/test", func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)

			router.ServeHTTP(w, req)

			// 해당 상태 코드 카테고리의 카운터 확인
			count := getCounterValue(m.HTTPRequestsTotal, "GET", "/test", tt.expectedStatus)
			if count != 1 {
				t.Errorf("상태 코드 %d는 '%s'로 분류되어야 함, 카운터: %f",
					tt.statusCode, tt.expectedStatus, count)
			}
		})
	}
}

// TestMetricsMiddleware_DifferentMethods는 다양한 HTTP 메서드가 기록되는지 테스트합니다.
func TestMetricsMiddleware_DifferentMethods(t *testing.T) {
	m := newTestMetrics("test_methods")

	router := gin.New()
	router.Use(MetricsMiddleware(m, ""))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusCreated) })
	router.PUT("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.DELETE("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/test", nil)
		router.ServeHTTP(w, req)
	}

	// 각 메서드별 카운터 확인
	for _, method := range methods {
		count := getCounterValue(m.HTTPRequestsTotal, method, "/test", "2xx")
		if count != 1 {
			t.Errorf("메서드 '%s' 카운터가 1이어야 함, 실제: %f", method, count)
		}
	}
}

// TestMetricsMiddlewareSimple은 MetricsMiddlewareSimple이 올바르게 동작하는지 테스트합니다.
func TestMetricsMiddlewareSimple(t *testing.T) {
	m := newTestMetrics("test_simple")

	router := gin.New()
	router.Use(MetricsMiddlewareSimple(m))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	count := getCounterValue(m.HTTPRequestsTotal, "GET", "/test", "2xx")
	if count != 1 {
		t.Errorf("HTTP 요청 카운터가 1이어야 함, 실제: %f", count)
	}
}

// TestMetricsMiddleware_SkipWithBasePath는 basePath가 있을 때 스킵 경로가 작동하는지 테스트합니다.
func TestMetricsMiddleware_SkipWithBasePath(t *testing.T) {
	m := newTestMetrics("test_basepath_skip")

	basePath := "/api/boards"

	router := gin.New()
	router.Use(MetricsMiddleware(m, basePath))
	router.GET("/api/boards/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/boards/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/boards/projects", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// basePath + 스킵 경로 요청
	skipPaths := []string{"/api/boards/health", "/api/boards/metrics"}
	for _, path := range skipPaths {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		router.ServeHTTP(w, req)
	}

	// 일반 경로 요청
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/boards/projects", nil)
	router.ServeHTTP(w, req)

	// 스킵 경로는 메트릭이 기록되지 않아야 함
	for _, path := range skipPaths {
		count := getCounterValue(m.HTTPRequestsTotal, "GET", path, "2xx")
		if count != 0 {
			t.Errorf("경로 '%s'는 스킵되어야 함", path)
		}
	}

	// 일반 경로는 메트릭이 기록되어야 함
	count := getCounterValue(m.HTTPRequestsTotal, "GET", "/api/boards/projects", "2xx")
	if count != 1 {
		t.Errorf("/api/boards/projects 경로는 메트릭이 기록되어야 함, 실제: %f", count)
	}
}

// TestMetricsMiddleware_RoutePattern은 FullPath()가 라우트 패턴을 반환하는지 테스트합니다.
func TestMetricsMiddleware_RoutePattern(t *testing.T) {
	m := newTestMetrics("test_pattern")

	router := gin.New()
	router.Use(MetricsMiddleware(m, ""))
	router.GET("/users/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 실제 경로로 요청
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/123", nil)
	router.ServeHTTP(w, req)

	// 라우트 패턴으로 메트릭이 기록되어야 함 (실제 경로가 아닌)
	patternCount := getCounterValue(m.HTTPRequestsTotal, "GET", "/users/:id", "2xx")
	if patternCount != 1 {
		t.Errorf("라우트 패턴 '/users/:id'로 메트릭이 기록되어야 함, 실제: %f", patternCount)
	}

	// 실제 경로로는 기록되지 않아야 함
	actualCount := getCounterValue(m.HTTPRequestsTotal, "GET", "/users/123", "2xx")
	if actualCount != 0 {
		t.Errorf("실제 경로 '/users/123'로 메트릭이 기록되면 안됨, 실제: %f", actualCount)
	}
}

// TestMetricsMiddleware_MultipleRequests는 여러 요청이 누적되는지 테스트합니다.
func TestMetricsMiddleware_MultipleRequests(t *testing.T) {
	m := newTestMetrics("test_multiple")

	router := gin.New()
	router.Use(MetricsMiddleware(m, ""))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 5번 요청
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}

	// 카운터가 5여야 함
	count := getCounterValue(m.HTTPRequestsTotal, "GET", "/test", "2xx")
	if count != 5 {
		t.Errorf("5번 요청 후 카운터가 5여야 함, 실제: %f", count)
	}

	// 히스토그램 카운트도 5여야 함
	histCount := getHistogramCount(m.HTTPRequestDuration, "GET", "/test")
	if histCount != 5 {
		t.Errorf("5번 요청 후 히스토그램 카운트가 5여야 함, 실제: %d", histCount)
	}
}
