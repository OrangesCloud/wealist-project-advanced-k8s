// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 logger.go의 테스트를 포함합니다.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestLogger는 Logger 미들웨어가 올바르게 동작하는지 테스트합니다.
func TestLogger(t *testing.T) {
	// 관찰 가능한 로거 생성
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Logger(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")

	router.ServeHTTP(w, req)

	// 로그가 기록되었는지 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
	}

	// 로그 레벨 확인 (200은 Info)
	if logs[0].Level != zapcore.InfoLevel {
		t.Errorf("예상 로그 레벨: Info, 실제: %s", logs[0].Level)
	}

	// 로그 메시지 확인
	if logs[0].Message != "Request completed" {
		t.Errorf("예상 메시지: 'Request completed', 실제: '%s'", logs[0].Message)
	}

	// 로그 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["method"] != "GET" {
		t.Errorf("method 필드가 'GET'이어야 함")
	}
	if contextMap["path"] != "/test" {
		t.Errorf("path 필드가 '/test'이어야 함")
	}
	if contextMap["query"] != "foo=bar" {
		t.Errorf("query 필드가 'foo=bar'이어야 함")
	}
	if contextMap["user_agent"] != "Test-Agent/1.0" {
		t.Errorf("user_agent 필드가 'Test-Agent/1.0'이어야 함")
	}
}

// TestLogger_RequestID는 Logger가 request_id를 설정하는지 테스트합니다.
func TestLogger_RequestID(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	var capturedRequestID string

	router := gin.New()
	router.Use(Logger(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		// request_id가 컨텍스트에 설정되어 있는지 확인
		if id, exists := c.Get(RequestIDKey); exists {
			capturedRequestID = id.(string)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	// request_id가 설정되었는지 확인
	if capturedRequestID == "" {
		t.Error("request_id가 컨텍스트에 설정되어야 함")
	}

	// X-Request-ID 헤더 확인
	responseRequestID := w.Header().Get("X-Request-ID")
	if responseRequestID == "" {
		t.Error("X-Request-ID 헤더가 설정되어야 함")
	}
	if responseRequestID != capturedRequestID {
		t.Error("헤더와 컨텍스트의 request_id가 일치해야 함")
	}
}

// TestLogger_StatusCodeLevels는 상태 코드에 따른 로그 레벨을 테스트합니다.
func TestLogger_StatusCodeLevels(t *testing.T) {
	tests := []struct {
		name          string        // 테스트 케이스 이름
		statusCode    int           // HTTP 상태 코드
		expectedLevel zapcore.Level // 예상 로그 레벨
		expectedMsg   string        // 예상 로그 메시지
	}{
		{
			name:          "200 OK는 Info 레벨",
			statusCode:    http.StatusOK,
			expectedLevel: zapcore.InfoLevel,
			expectedMsg:   "Request completed",
		},
		{
			name:          "201 Created는 Info 레벨",
			statusCode:    http.StatusCreated,
			expectedLevel: zapcore.InfoLevel,
			expectedMsg:   "Request completed",
		},
		{
			name:          "400 Bad Request는 Warn 레벨",
			statusCode:    http.StatusBadRequest,
			expectedLevel: zapcore.WarnLevel,
			expectedMsg:   "Client error",
		},
		{
			name:          "401 Unauthorized는 Warn 레벨",
			statusCode:    http.StatusUnauthorized,
			expectedLevel: zapcore.WarnLevel,
			expectedMsg:   "Client error",
		},
		{
			name:          "404 Not Found는 Warn 레벨",
			statusCode:    http.StatusNotFound,
			expectedLevel: zapcore.WarnLevel,
			expectedMsg:   "Client error",
		},
		{
			name:          "500 Internal Server Error는 Error 레벨",
			statusCode:    http.StatusInternalServerError,
			expectedLevel: zapcore.ErrorLevel,
			expectedMsg:   "Server error",
		},
		{
			name:          "503 Service Unavailable는 Error 레벨",
			statusCode:    http.StatusServiceUnavailable,
			expectedLevel: zapcore.ErrorLevel,
			expectedMsg:   "Server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.DebugLevel)
			zapLogger := zap.New(core)

			router := gin.New()
			router.Use(Logger(zapLogger))
			router.GET("/test", func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)

			router.ServeHTTP(w, req)

			logs := recorded.All()
			if len(logs) != 1 {
				t.Fatalf("1개의 로그가 기록되어야 함")
			}

			if logs[0].Level != tt.expectedLevel {
				t.Errorf("예상 로그 레벨: %s, 실제: %s", tt.expectedLevel, logs[0].Level)
			}

			if logs[0].Message != tt.expectedMsg {
				t.Errorf("예상 메시지: '%s', 실제: '%s'", tt.expectedMsg, logs[0].Message)
			}
		})
	}
}

// TestLogger_WithUserID는 user_id가 있을 때 로그에 포함되는지 테스트합니다.
func TestLogger_WithUserID(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Logger(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("1개의 로그가 기록되어야 함")
	}

	contextMap := logs[0].ContextMap()
	if contextMap["user_id"] != "test-user-123" {
		t.Errorf("user_id 필드가 'test-user-123'이어야 함, 실제: %v", contextMap["user_id"])
	}
}

// TestSkipPathLogger는 SkipPathLogger가 특정 경로를 스킵하는지 테스트합니다.
func TestSkipPathLogger(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(SkipPathLogger(zapLogger, "/health", "/metrics"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 스킵해야 할 경로 요청
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w1, req1)

	// 스킵 후 로그가 없어야 함
	if len(recorded.All()) != 0 {
		t.Errorf("/health 경로는 로그가 스킵되어야 함, 실제 로그 수: %d", len(recorded.All()))
	}

	// 일반 경로 요청
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w2, req2)

	// 일반 경로는 로그가 기록되어야 함
	if len(recorded.All()) != 1 {
		t.Errorf("/api/test 경로는 로그가 기록되어야 함, 실제 로그 수: %d", len(recorded.All()))
	}
}

// TestGetRequestID는 GetRequestID 함수가 올바르게 동작하는지 테스트합니다.
func TestGetRequestID(t *testing.T) {
	t.Run("컨텍스트에 request_id가 있는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expectedID := "existing-request-id"
		c.Set(RequestIDKey, expectedID)

		id := GetRequestID(c)
		if id != expectedID {
			t.Errorf("예상 request_id: '%s', 실제: '%s'", expectedID, id)
		}
	})

	t.Run("컨텍스트에 request_id가 없는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id := GetRequestID(c)
		if id == "" {
			t.Error("request_id가 생성되어야 함")
		}
		// UUID 형식인지 확인 (길이 36)
		if len(id) != 36 {
			t.Errorf("request_id가 UUID 형식이어야 함, 실제 길이: %d", len(id))
		}
	})

	t.Run("request_id가 문자열이 아닌 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set(RequestIDKey, 12345) // 정수 설정

		id := GetRequestID(c)
		// 타입이 맞지 않으면 새 UUID를 생성해야 함
		if id == "" {
			t.Error("request_id가 생성되어야 함")
		}
	})
}

// TestRequestIDKey는 RequestIDKey 상수가 올바른지 테스트합니다.
func TestRequestIDKey(t *testing.T) {
	if RequestIDKey != "request_id" {
		t.Errorf("RequestIDKey가 'request_id'여야 함, 실제: '%s'", RequestIDKey)
	}
}
