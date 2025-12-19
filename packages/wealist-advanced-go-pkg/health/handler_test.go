// Package health는 K8s 헬스체크 엔드포인트를 제공합니다.
// 이 파일은 handler.go의 테스트를 포함합니다.
package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// 테스트 모드 설정
	gin.SetMode(gin.TestMode)
}

// TestLiveness는 liveness 엔드포인트가 항상 200을 반환하는지 테스트합니다.
func TestLiveness(t *testing.T) {
	// DB, Redis 없이 생성
	checker := NewHealthChecker(nil, nil)

	router := gin.New()
	checker.RegisterRoutes(router, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("예상 status: 'alive', 실제: %v", response["status"])
	}
}

// TestReadiness_AllHealthy는 모든 의존성이 정상일 때 200을 반환하는지 테스트합니다.
func TestReadiness_AllHealthy(t *testing.T) {
	// 모든 의존성 없이 (not_configured로 표시됨)
	checker := NewHealthChecker(nil, nil)

	router := gin.New()
	checker.RegisterRoutes(router, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("예상 status: 'ready', 실제: %v", response["status"])
	}
}

// TestRegisterRoutes_WithBasePath는 basePath가 있을 때 라우트가 등록되는지 테스트합니다.
func TestRegisterRoutes_WithBasePath(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	router := gin.New()
	checker.RegisterRoutes(router, "/api")

	tests := []struct {
		name string // 테스트 케이스 이름
		path string // 요청 경로
	}{
		{"루트 레벨 liveness", "/health/live"},
		{"루트 레벨 readiness", "/health/ready"},
		{"루트 레벨 health", "/health"},
		{"basePath liveness", "/api/health/live"},
		{"basePath readiness", "/api/health/ready"},
		{"basePath health", "/api/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("경로 %s: 예상 상태 코드 %d, 실제: %d", tt.path, http.StatusOK, w.Code)
			}
		})
	}
}

// TestRegisterRoutes_EmptyBasePath는 빈 basePath일 때 루트 레벨만 등록되는지 테스트합니다.
func TestRegisterRoutes_EmptyBasePath(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	router := gin.New()
	checker.RegisterRoutes(router, "")

	// 루트 레벨 라우트만 존재해야 함
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}
}

// TestHealthResponse는 응답 구조가 올바른지 테스트합니다.
func TestHealthResponse(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	router := gin.New()
	checker.RegisterRoutes(router, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	// status 필드 확인
	if _, ok := response["status"]; !ok {
		t.Error("응답에 'status' 필드가 있어야 합니다")
	}

	// checks 필드 확인
	checks, ok := response["checks"].(map[string]interface{})
	if !ok {
		t.Error("응답에 'checks' 필드가 있어야 합니다")
	}

	// database와 redis 상태 확인
	if _, ok := checks["database"]; !ok {
		t.Error("checks에 'database' 필드가 있어야 합니다")
	}
	if _, ok := checks["redis"]; !ok {
		t.Error("checks에 'redis' 필드가 있어야 합니다")
	}
}

// TestMockPinger는 MockPinger가 올바르게 동작하는지 테스트합니다.
func TestMockPinger(t *testing.T) {
	t.Run("성공하는 pinger", func(t *testing.T) {
		pinger := &MockPinger{Err: nil}
		if err := pinger.Ping(context.Background()); err != nil {
			t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
		}
	})

	t.Run("실패하는 pinger", func(t *testing.T) {
		expectedErr := errors.New("connection failed")
		pinger := &MockPinger{Err: expectedErr}
		if err := pinger.Ping(context.Background()); err != expectedErr {
			t.Errorf("예상 에러: %v, 실제: %v", expectedErr, err)
		}
	})
}

// TestDBPinger는 DBPinger가 올바르게 동작하는지 테스트합니다.
func TestDBPinger_NilDB(t *testing.T) {
	pinger := NewDBPinger(nil)
	if err := pinger.Ping(context.Background()); err != nil {
		t.Errorf("nil DB에서 에러가 없어야 하는데 %v가 반환됨", err)
	}
}

// TestRedisPinger는 RedisPinger가 올바르게 동작하는지 테스트합니다.
func TestRedisPinger_NilRedis(t *testing.T) {
	pinger := NewRedisPinger(nil)
	if err := pinger.Ping(context.Background()); err != nil {
		t.Errorf("nil Redis에서 에러가 없어야 하는데 %v가 반환됨", err)
	}
}
