// Package errors는 애플리케이션 에러 처리를 제공합니다.
// 이 파일은 handler.go의 테스트를 포함합니다.
package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
)

func init() {
	// 테스트 모드 설정
	gin.SetMode(gin.TestMode)
}

// TestNewHandler는 NewHandler가 올바르게 Handler를 생성하는지 테스트합니다.
func TestNewHandler(t *testing.T) {
	t.Run("logger가 있는 경우", func(t *testing.T) {
		core, _ := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		handler := NewHandler(logger)
		if handler == nil {
			t.Error("handler가 nil이면 안됨")
		}
	})

	t.Run("logger가 nil인 경우", func(t *testing.T) {
		handler := NewHandler(nil)
		if handler == nil {
			t.Error("handler가 nil이면 안됨")
		}
		// nil logger는 zap.NewNop()으로 대체됨
	})
}

// TestHandleError_AppError는 HandleError가 AppError를 올바르게 처리하는지 테스트합니다.
func TestHandleError_AppError(t *testing.T) {
	tests := []struct {
		name           string // 테스트 케이스 이름
		appError       *AppError
		expectedStatus int    // 예상 HTTP 상태 코드
		expectedCode   string // 예상 에러 코드
	}{
		{
			name:           "NotFound 에러",
			appError:       NotFound("User", "user-id-123"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrCodeNotFound,
		},
		{
			name:           "Validation 에러",
			appError:       Validation("email is required", ""),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name:           "Unauthorized 에러",
			appError:       Unauthorized("invalid token", ""),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   ErrCodeUnauthorized,
		},
		{
			name:           "Forbidden 에러",
			appError:       Forbidden("access denied", ""),
			expectedStatus: http.StatusForbidden,
			expectedCode:   ErrCodeForbidden,
		},
		{
			name:           "Conflict 에러",
			appError:       Conflict("resource already exists", ""),
			expectedStatus: http.StatusConflict,
			expectedCode:   ErrCodeConflict,
		},
		{
			name:           "Internal 에러",
			appError:       Internal("unexpected error", ""),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, _ := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)
			handler := NewHandler(logger)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			handler.HandleError(c, tt.appError)

			// 상태 코드 확인
			if w.Code != tt.expectedStatus {
				t.Errorf("예상 상태 코드: %d, 실제: %d", tt.expectedStatus, w.Code)
			}

			// JSON 응답 파싱
			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON 파싱 실패: %v", err)
			}

			if response.Code != tt.expectedCode {
				t.Errorf("예상 에러 코드: '%s', 실제: '%s'", tt.expectedCode, response.Code)
			}
		})
	}
}

// TestHandleError_GormNotFound는 gorm.ErrRecordNotFound를 올바르게 처리하는지 테스트합니다.
func TestHandleError_GormNotFound(t *testing.T) {
	core, _ := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	handler := NewHandler(logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler.HandleError(c, gorm.ErrRecordNotFound)

	// 404 상태 코드 확인
	if w.Code != http.StatusNotFound {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusNotFound, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.Code != ErrCodeNotFound {
		t.Errorf("예상 에러 코드: '%s', 실제: '%s'", ErrCodeNotFound, response.Code)
	}
}

// TestHandleError_UnknownError는 알 수 없는 에러를 500으로 처리하는지 테스트합니다.
func TestHandleError_UnknownError(t *testing.T) {
	core, _ := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	handler := NewHandler(logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler.HandleError(c, errors.New("some unknown error"))

	// 500 상태 코드 확인
	if w.Code != http.StatusInternalServerError {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusInternalServerError, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.Code != ErrCodeInternal {
		t.Errorf("예상 에러 코드: '%s', 실제: '%s'", ErrCodeInternal, response.Code)
	}
}

// TestHandleError_NilError는 nil 에러를 무시하는지 테스트합니다.
func TestHandleError_NilError(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	handler := NewHandler(logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler.HandleError(c, nil)

	// 응답이 작성되지 않아야 함 (상태 코드 200이 기본값)
	if w.Code != http.StatusOK {
		t.Errorf("nil 에러는 응답을 작성하면 안됨, 실제 상태 코드: %d", w.Code)
	}

	// 로그가 기록되지 않아야 함
	if len(recorded.All()) != 0 {
		t.Errorf("nil 에러는 로그가 없어야 함, 실제 로그 수: %d", len(recorded.All()))
	}
}

// TestHandleError_WithDetails는 에러 details가 응답에 포함되는지 테스트합니다.
func TestHandleError_WithDetails(t *testing.T) {
	core, _ := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	handler := NewHandler(logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	appErr := &AppError{
		Code:    ErrCodeValidation,
		Message: "Validation failed",
		Details: "field 'email' is invalid",
	}
	handler.HandleError(c, appErr)

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.Details != "field 'email' is invalid" {
		t.Errorf("예상 details: 'field 'email' is invalid', 실제: '%s'", response.Details)
	}
}

// TestSendError는 SendError 함수가 올바른 응답을 반환하는지 테스트합니다.
func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.Code != "BAD_REQUEST" {
		t.Errorf("예상 코드: 'BAD_REQUEST', 실제: '%s'", response.Code)
	}
	if response.Message != "Invalid input" {
		t.Errorf("예상 메시지: 'Invalid input', 실제: '%s'", response.Message)
	}
}

// TestSendErrorWithDetails는 SendErrorWithDetails 함수가 details를 포함하는지 테스트합니다.
func TestSendErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", "email is required")

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.Details != "email is required" {
		t.Errorf("예상 details: 'email is required', 실제: '%s'", response.Details)
	}
}

// TestHandleAppError는 HandleAppError가 올바르게 동작하는지 테스트합니다.
func TestHandleAppError(t *testing.T) {
	t.Run("정상적인 AppError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		appErr := NotFound("User", "user-123")
		HandleAppError(c, appErr)

		if w.Code != http.StatusNotFound {
			t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("nil AppError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		HandleAppError(c, nil)

		// nil이면 아무것도 하지 않아야 함
		if w.Code != http.StatusOK {
			t.Errorf("nil AppError는 응답을 작성하면 안됨")
		}
	})
}

// TestGetLoggerFromContext는 GetLoggerFromContext가 올바르게 동작하는지 테스트합니다.
func TestGetLoggerFromContext(t *testing.T) {
	t.Run("logger가 컨텍스트에 있는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		core, _ := observer.New(zapcore.InfoLevel)
		expectedLogger := zap.New(core)
		c.Set("logger", expectedLogger)

		logger := GetLoggerFromContext(c)
		if logger != expectedLogger {
			t.Error("컨텍스트에서 logger를 가져와야 함")
		}
	})

	t.Run("logger가 컨텍스트에 없는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		logger := GetLoggerFromContext(c)
		if logger == nil {
			t.Error("logger가 nil이면 안됨, nop logger를 반환해야 함")
		}
	})

	t.Run("logger가 잘못된 타입인 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("logger", "not a logger")

		logger := GetLoggerFromContext(c)
		if logger == nil {
			t.Error("logger가 nil이면 안됨, nop logger를 반환해야 함")
		}
	})
}

// TestHandleServiceError는 HandleServiceError가 올바르게 동작하는지 테스트합니다.
func TestHandleServiceError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// logger를 컨텍스트에 설정
	core, _ := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	c.Set("logger", logger)

	HandleServiceError(c, NotFound("User", "123"))

	if w.Code != http.StatusNotFound {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusNotFound, w.Code)
	}
}

// TestErrorResponse_JSON은 ErrorResponse가 올바르게 JSON으로 직렬화되는지 테스트합니다.
func TestErrorResponse_JSON(t *testing.T) {
	t.Run("모든 필드가 있는 경우", func(t *testing.T) {
		resp := ErrorResponse{
			Code:    "TEST_ERROR",
			Message: "Test message",
			Details: "Test details",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("JSON 직렬화 실패: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		if parsed["code"] != "TEST_ERROR" {
			t.Errorf("code 필드가 잘못됨")
		}
		if parsed["message"] != "Test message" {
			t.Errorf("message 필드가 잘못됨")
		}
		if parsed["details"] != "Test details" {
			t.Errorf("details 필드가 잘못됨")
		}
	})

	t.Run("details가 빈 경우 omitempty", func(t *testing.T) {
		resp := ErrorResponse{
			Code:    "TEST_ERROR",
			Message: "Test message",
			Details: "", // 빈 문자열
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("JSON 직렬화 실패: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		// details가 omitempty이므로 빈 값이면 포함되지 않아야 함
		if _, ok := parsed["details"]; ok {
			t.Error("빈 details는 JSON에 포함되면 안됨")
		}
	})
}

// TestHandleError_Logging은 HandleError가 올바르게 로그를 기록하는지 테스트합니다.
func TestHandleError_Logging(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	handler := NewHandler(logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/users", nil)

	handler.HandleError(c, Validation("email is required", ""))

	// Error 로그가 기록되었는지 확인
	logs := recorded.All()
	if len(logs) < 1 {
		t.Fatal("최소 1개의 로그가 기록되어야 함")
	}

	// 첫 번째 로그 확인 (Error 레벨)
	errorLog := logs[0]
	if errorLog.Level != zapcore.ErrorLevel {
		t.Errorf("에러 로그 레벨이어야 함, 실제: %s", errorLog.Level)
	}

	contextMap := errorLog.ContextMap()
	if contextMap["path"] != "/api/users" {
		t.Errorf("path 필드가 '/api/users'여야 함")
	}
	if contextMap["method"] != "POST" {
		t.Errorf("method 필드가 'POST'여야 함")
	}
}
