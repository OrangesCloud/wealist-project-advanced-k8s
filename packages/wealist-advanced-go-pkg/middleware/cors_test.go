// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 cors.go의 테스트를 포함합니다.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// 테스트 모드 설정
	gin.SetMode(gin.TestMode)
}

// TestDefaultCORSConfig는 DefaultCORSConfig가 올바른 기본값을 반환하는지 테스트합니다.
func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	// AllowedOrigins 확인
	if len(config.AllowedOrigins) != 1 || config.AllowedOrigins[0] != "*" {
		t.Errorf("예상 AllowedOrigins: ['*'], 실제: %v", config.AllowedOrigins)
	}

	// AllowedMethods 확인
	expectedMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	if len(config.AllowedMethods) != len(expectedMethods) {
		t.Errorf("예상 AllowedMethods 개수: %d, 실제: %d", len(expectedMethods), len(config.AllowedMethods))
	}

	// AllowCredentials 확인
	if !config.AllowCredentials {
		t.Error("AllowCredentials는 true여야 함")
	}

	// MaxAge 확인
	if config.MaxAge != 86400 {
		t.Errorf("예상 MaxAge: 86400, 실제: %d", config.MaxAge)
	}
}

// TestCORS는 CORS 미들웨어가 올바른 헤더를 설정하는지 테스트합니다.
func TestCORS(t *testing.T) {
	tests := []struct {
		name           string     // 테스트 케이스 이름
		config         CORSConfig // CORS 설정
		origin         string     // 요청 Origin 헤더
		method         string     // HTTP 메서드
		expectedOrigin string     // 예상 Access-Control-Allow-Origin
		expectedStatus int        // 예상 상태 코드
	}{
		{
			name:           "와일드카드 origin 설정",
			config:         DefaultCORSConfig(),
			origin:         "http://example.com",
			method:         "GET",
			expectedOrigin: "http://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name: "특정 origin만 허용",
			config: CORSConfig{
				AllowedOrigins:   []string{"http://allowed.com"},
				AllowedMethods:   []string{"GET"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
			},
			origin:         "http://allowed.com",
			method:         "GET",
			expectedOrigin: "http://allowed.com",
			expectedStatus: http.StatusOK,
		},
		{
			name: "허용되지 않은 origin",
			config: CORSConfig{
				AllowedOrigins:   []string{"http://allowed.com"},
				AllowedMethods:   []string{"GET"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
			},
			origin:         "http://notallowed.com",
			method:         "GET",
			expectedOrigin: "*", // 허용되지 않으면 * 반환
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS preflight 요청",
			config:         DefaultCORSConfig(),
			origin:         "http://example.com",
			method:         "OPTIONS",
			expectedOrigin: "http://example.com",
			expectedStatus: http.StatusNoContent, // 204
		},
		{
			name:           "origin 없는 요청",
			config:         DefaultCORSConfig(),
			origin:         "",
			method:         "GET",
			expectedOrigin: "*",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORS(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				// OPTIONS는 CORS 미들웨어에서 처리됨
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			router.ServeHTTP(w, req)

			// 상태 코드 확인
			if w.Code != tt.expectedStatus {
				t.Errorf("예상 상태 코드: %d, 실제: %d", tt.expectedStatus, w.Code)
			}

			// Access-Control-Allow-Origin 헤더 확인
			actualOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if actualOrigin != tt.expectedOrigin {
				t.Errorf("예상 Origin: '%s', 실제: '%s'", tt.expectedOrigin, actualOrigin)
			}
		})
	}
}

// TestCORS_Headers는 CORS 헤더가 올바르게 설정되는지 테스트합니다.
func TestCORS_Headers(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
	}

	router := gin.New()
	router.Use(CORS(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	router.ServeHTTP(w, req)

	// Access-Control-Allow-Methods 확인
	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST" {
		t.Errorf("예상 Methods: 'GET, POST', 실제: '%s'", methods)
	}

	// Access-Control-Allow-Headers 확인
	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, Authorization" {
		t.Errorf("예상 Headers: 'Content-Type, Authorization', 실제: '%s'", headers)
	}

	// Access-Control-Expose-Headers 확인
	exposed := w.Header().Get("Access-Control-Expose-Headers")
	if exposed != "X-Request-ID" {
		t.Errorf("예상 Exposed: 'X-Request-ID', 실제: '%s'", exposed)
	}

	// Access-Control-Allow-Credentials 확인
	credentials := w.Header().Get("Access-Control-Allow-Credentials")
	if credentials != "true" {
		t.Errorf("예상 Credentials: 'true', 실제: '%s'", credentials)
	}
}

// TestCORS_NoCredentials는 AllowCredentials가 false일 때 헤더가 없는지 테스트합니다.
func TestCORS_NoCredentials(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowCredentials: false, // false로 설정
	}

	router := gin.New()
	router.Use(CORS(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	router.ServeHTTP(w, req)

	// AllowCredentials가 false면 헤더가 없어야 함
	credentials := w.Header().Get("Access-Control-Allow-Credentials")
	if credentials != "" {
		t.Errorf("Credentials 헤더가 없어야 함, 실제: '%s'", credentials)
	}
}

// TestDefaultCORS는 DefaultCORS 함수가 올바르게 동작하는지 테스트합니다.
func TestDefaultCORS(t *testing.T) {
	router := gin.New()
	router.Use(DefaultCORS())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}

	// 기본 설정이 적용되었는지 확인
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://example.com" {
		t.Errorf("예상 Origin: 'http://example.com', 실제: '%s'", origin)
	}
}

// TestCORSWithOrigins는 CORSWithOrigins 함수가 올바르게 동작하는지 테스트합니다.
func TestCORSWithOrigins(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		origins        string // 설정할 origins
		requestOrigin  string // 요청 origin
		expectedOrigin string // 예상 응답 origin
	}{
		{
			name:           "빈 origins는 기본값 사용",
			origins:        "",
			requestOrigin:  "http://example.com",
			expectedOrigin: "http://example.com",
		},
		{
			name:           "와일드카드 origins",
			origins:        "*",
			requestOrigin:  "http://example.com",
			expectedOrigin: "http://example.com",
		},
		{
			name:           "쉼표로 구분된 origins - 허용됨",
			origins:        "http://allowed1.com, http://allowed2.com",
			requestOrigin:  "http://allowed1.com",
			expectedOrigin: "http://allowed1.com",
		},
		{
			name:           "쉼표로 구분된 origins - 허용 안됨",
			origins:        "http://allowed1.com, http://allowed2.com",
			requestOrigin:  "http://notallowed.com",
			expectedOrigin: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORSWithOrigins(tt.origins))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)

			router.ServeHTTP(w, req)

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if origin != tt.expectedOrigin {
				t.Errorf("예상 Origin: '%s', 실제: '%s'", tt.expectedOrigin, origin)
			}
		})
	}
}

// TestCORS_PreflightAbort는 preflight 요청이 c.Next()를 호출하지 않는지 테스트합니다.
func TestCORS_PreflightAbort(t *testing.T) {
	handlerCalled := false

	router := gin.New()
	router.Use(DefaultCORS())
	router.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	router.ServeHTTP(w, req)

	// OPTIONS 요청은 204로 abort되어야 함
	if w.Code != http.StatusNoContent {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusNoContent, w.Code)
	}

	// 핸들러가 호출되지 않아야 함
	if handlerCalled {
		t.Error("preflight 요청에서 핸들러가 호출되면 안됨")
	}
}
