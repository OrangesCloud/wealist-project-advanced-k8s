// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 recovery.go의 테스트를 포함합니다.
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestRecovery는 Recovery 미들웨어가 panic을 복구하는지 테스트합니다.
func TestRecovery(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic!")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	// panic이 발생해도 서버가 죽지 않아야 함
	router.ServeHTTP(w, req)

	// 500 상태 코드 확인
	if w.Code != http.StatusInternalServerError {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusInternalServerError, w.Code)
	}

	// 에러 로그가 기록되었는지 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("1개의 에러 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
	}

	if logs[0].Level != zapcore.ErrorLevel {
		t.Errorf("예상 로그 레벨: Error, 실제: %s", logs[0].Level)
	}

	if logs[0].Message != "Panic recovered" {
		t.Errorf("예상 메시지: 'Panic recovered', 실제: '%s'", logs[0].Message)
	}
}

// TestRecovery_ResponseBody는 Recovery가 올바른 에러 응답을 반환하는지 테스트합니다.
func TestRecovery_ResponseBody(t *testing.T) {
	core, _ := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic!")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	router.ServeHTTP(w, req)

	// JSON 응답 파싱
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	// error 필드 확인
	errorField, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error 필드가 map이어야 함")
	}

	if errorField["code"] != "INTERNAL_ERROR" {
		t.Errorf("error.code가 'INTERNAL_ERROR'여야 함, 실제: %v", errorField["code"])
	}

	if errorField["message"] != "Internal server error" {
		t.Errorf("error.message가 'Internal server error'여야 함, 실제: %v", errorField["message"])
	}

	// requestId 필드 확인
	if _, ok := response["requestId"]; !ok {
		t.Error("requestId 필드가 있어야 함")
	}
}

// TestRecovery_LogFields는 Recovery가 올바른 로그 필드를 기록하는지 테스트합니다.
func TestRecovery_LogFields(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.POST("/api/panic", func(c *gin.Context) {
		panic("detailed panic message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/panic", nil)

	router.ServeHTTP(w, req)

	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("1개의 로그가 기록되어야 함")
	}

	contextMap := logs[0].ContextMap()

	// request_id 필드 확인
	if _, ok := contextMap["request_id"]; !ok {
		t.Error("request_id 필드가 있어야 함")
	}

	// error 필드 확인
	if contextMap["error"] != "detailed panic message" {
		t.Errorf("error 필드가 'detailed panic message'여야 함, 실제: %v", contextMap["error"])
	}

	// stack 필드 확인
	if _, ok := contextMap["stack"]; !ok {
		t.Error("stack 필드가 있어야 함")
	}

	// path 필드 확인
	if contextMap["path"] != "/api/panic" {
		t.Errorf("path 필드가 '/api/panic'이어야 함, 실제: %v", contextMap["path"])
	}

	// method 필드 확인
	if contextMap["method"] != "POST" {
		t.Errorf("method 필드가 'POST'이어야 함, 실제: %v", contextMap["method"])
	}
}

// TestRecovery_NoPanic은 panic이 없을 때 정상 동작하는지 테스트합니다.
func TestRecovery_NoPanic(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.GET("/normal", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/normal", nil)

	router.ServeHTTP(w, req)

	// 정상 응답 확인
	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}

	// 에러 로그가 없어야 함
	if len(recorded.All()) != 0 {
		t.Errorf("에러 로그가 없어야 하는데 %d개가 기록됨", len(recorded.All()))
	}
}

// TestRecovery_WithRequestID는 Recovery가 기존 request_id를 사용하는지 테스트합니다.
func TestRecovery_WithRequestID(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	expectedRequestID := "test-request-id-12345"

	router := gin.New()
	// 먼저 request_id를 설정하는 미들웨어
	router.Use(func(c *gin.Context) {
		c.Set(RequestIDKey, expectedRequestID)
		c.Next()
	})
	router.Use(Recovery(zapLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic!")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	router.ServeHTTP(w, req)

	// 응답에서 requestId 확인
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response["requestId"] != expectedRequestID {
		t.Errorf("예상 requestId: '%s', 실제: '%v'", expectedRequestID, response["requestId"])
	}

	// 로그에서 request_id 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("1개의 로그가 기록되어야 함")
	}

	contextMap := logs[0].ContextMap()
	if contextMap["request_id"] != expectedRequestID {
		t.Errorf("로그의 request_id가 '%s'여야 함, 실제: %v", expectedRequestID, contextMap["request_id"])
	}
}

// TestRecovery_ContentType은 응답 Content-Type이 올바른지 테스트합니다.
func TestRecovery_ContentType(t *testing.T) {
	core, _ := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic!")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("예상 Content-Type: 'application/json; charset=utf-8', 실제: '%s'", contentType)
	}
}

// TestRecovery_ErrorType은 다양한 panic 타입을 처리하는지 테스트합니다.
func TestRecovery_ErrorType(t *testing.T) {
	tests := []struct {
		name       string      // 테스트 케이스 이름
		panicValue interface{} // panic에 전달할 값
	}{
		{
			name:       "string panic",
			panicValue: "string error",
		},
		{
			name:       "error type panic",
			panicValue: http.ErrAbortHandler,
		},
		{
			name:       "integer panic",
			panicValue: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.ErrorLevel)
			zapLogger := zap.New(core)

			router := gin.New()
			router.Use(Recovery(zapLogger))
			router.GET("/panic", func(c *gin.Context) {
				panic(tt.panicValue)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/panic", nil)

			router.ServeHTTP(w, req)

			// 500 상태 코드 확인
			if w.Code != http.StatusInternalServerError {
				t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusInternalServerError, w.Code)
			}

			// 로그가 기록되었는지 확인
			if len(recorded.All()) != 1 {
				t.Errorf("1개의 로그가 기록되어야 함")
			}
		})
	}
}
