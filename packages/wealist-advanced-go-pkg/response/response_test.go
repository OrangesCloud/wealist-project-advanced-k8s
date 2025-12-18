// Package response는 HTTP 응답 유틸리티를 제공합니다.
// 이 파일은 response.go의 테스트를 포함합니다.
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// 테스트 모드 설정
	gin.SetMode(gin.TestMode)
}

// TestSuccess는 Success 함수가 올바른 응답을 반환하는지 테스트합니다.
func TestSuccess(t *testing.T) {
	tests := []struct {
		name       string      // 테스트 케이스 이름
		statusCode int         // HTTP 상태 코드
		data       interface{} // 응답 데이터
	}{
		{
			name:       "200 OK with map data",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "success"},
		},
		{
			name:       "201 Created with struct data",
			statusCode: http.StatusCreated,
			data:       struct{ ID string }{ID: "test-id"},
		},
		{
			name:       "200 OK with nil data",
			statusCode: http.StatusOK,
			data:       nil,
		},
		{
			name:       "200 OK with array data",
			statusCode: http.StatusOK,
			data:       []string{"item1", "item2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Success(c, tt.statusCode, tt.data)

			// 상태 코드 확인
			if w.Code != tt.statusCode {
				t.Errorf("예상 상태 코드: %d, 실제: %d", tt.statusCode, w.Code)
			}

			// JSON 응답 파싱
			var response SuccessResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON 파싱 실패: %v", err)
			}

			// requestId가 존재하는지 확인 (빈 문자열이 아닌지)
			if response.RequestID == "" {
				t.Error("requestId가 비어있으면 안됨")
			}
		})
	}
}

// TestSuccess_WithContextRequestID는 컨텍스트에 설정된 requestId를 사용하는지 테스트합니다.
func TestSuccess_WithContextRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	expectedRequestID := "test-request-id-12345"
	c.Set("requestId", expectedRequestID)

	Success(c, http.StatusOK, nil)

	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if response.RequestID != expectedRequestID {
		t.Errorf("예상 requestId: '%s', 실제: '%s'", expectedRequestID, response.RequestID)
	}
}

// TestSuccessWithMessage는 SuccessWithMessage 함수가 올바르게 동작하는지 테스트합니다.
func TestSuccessWithMessage(t *testing.T) {
	t.Run("data와 message 모두 있는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := map[string]string{"key": "value"}
		SuccessWithMessage(c, http.StatusOK, data, "success message")

		if w.Code != http.StatusOK {
			t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
		}

		var response SuccessResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		// data가 그대로 반환되어야 함
		dataMap, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("data가 map이어야 함")
		}
		if dataMap["key"] != "value" {
			t.Errorf("data.key가 'value'여야 함")
		}
	})

	t.Run("data가 nil이고 message가 있는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		SuccessWithMessage(c, http.StatusOK, nil, "operation completed")

		var response SuccessResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		// message만 있으면 {message: "..."} 형태로 반환
		dataMap, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("data가 map이어야 함")
		}
		if dataMap["message"] != "operation completed" {
			t.Errorf("data.message가 'operation completed'여야 함")
		}
	})

	t.Run("data가 nil이고 message도 빈 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		SuccessWithMessage(c, http.StatusOK, nil, "")

		var response SuccessResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		// 둘 다 없으면 nil
		if response.Data != nil {
			t.Errorf("data가 nil이어야 함")
		}
	})
}

// TestError는 Error 함수가 올바른 에러 응답을 반환하는지 테스트합니다.
func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	// error 필드 확인
	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "VALIDATION_ERROR" {
		t.Errorf("error.code가 'VALIDATION_ERROR'여야 함, 실제: %v", errorDetail["code"])
	}
	if errorDetail["message"] != "Invalid input" {
		t.Errorf("error.message가 'Invalid input'여야 함, 실제: %v", errorDetail["message"])
	}

	// requestId 확인
	if response.RequestID == "" {
		t.Error("requestId가 비어있으면 안됨")
	}
}

// TestErrorWithDetails는 ErrorWithDetails 함수가 상세 정보를 포함하는지 테스트합니다.
func TestErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", "field 'email' is required")

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["details"] != "field 'email' is required" {
		t.Errorf("error.details가 설정되어야 함, 실제: %v", errorDetail["details"])
	}
}

// TestBadRequest는 BadRequest 함수가 400 응답을 반환하는지 테스트합니다.
func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "Invalid request body")

	if w.Code != http.StatusBadRequest {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "BAD_REQUEST" {
		t.Errorf("error.code가 'BAD_REQUEST'여야 함")
	}
	if errorDetail["message"] != "Invalid request body" {
		t.Errorf("error.message가 'Invalid request body'여야 함")
	}
}

// TestUnauthorized는 Unauthorized 함수가 401 응답을 반환하는지 테스트합니다.
func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "Token expired")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "UNAUTHORIZED" {
		t.Errorf("error.code가 'UNAUTHORIZED'여야 함")
	}
}

// TestForbidden는 Forbidden 함수가 403 응답을 반환하는지 테스트합니다.
func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusForbidden, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "FORBIDDEN" {
		t.Errorf("error.code가 'FORBIDDEN'여야 함")
	}
}

// TestNotFound는 NotFound 함수가 404 응답을 반환하는지 테스트합니다.
func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "Resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusNotFound, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "NOT_FOUND" {
		t.Errorf("error.code가 'NOT_FOUND'여야 함")
	}
}

// TestConflict는 Conflict 함수가 409 응답을 반환하는지 테스트합니다.
func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Conflict(c, "Resource already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusConflict, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "CONFLICT" {
		t.Errorf("error.code가 'CONFLICT'여야 함")
	}
}

// TestInternalError는 InternalError 함수가 500 응답을 반환하는지 테스트합니다.
func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalError(c, "Something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusInternalServerError, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	errorDetail, ok := response.Error.(map[string]interface{})
	if !ok {
		t.Fatal("error가 map이어야 함")
	}

	if errorDetail["code"] != "INTERNAL_ERROR" {
		t.Errorf("error.code가 'INTERNAL_ERROR'여야 함")
	}
}

// TestGetRequestID는 getRequestID 함수가 올바르게 동작하는지 테스트합니다.
func TestGetRequestID(t *testing.T) {
	t.Run("컨텍스트에 requestId가 있는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expectedID := "existing-request-id"
		c.Set("requestId", expectedID)

		id := getRequestID(c)
		if id != expectedID {
			t.Errorf("예상 requestId: '%s', 실제: '%s'", expectedID, id)
		}
	})

	t.Run("컨텍스트에 requestId가 없는 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id := getRequestID(c)
		if id == "" {
			t.Error("requestId가 생성되어야 함")
		}

		// UUID 형식인지 간단히 확인 (길이 36)
		if len(id) != 36 {
			t.Errorf("requestId가 UUID 형식이어야 함 (길이 36), 실제 길이: %d", len(id))
		}
	})

	t.Run("requestId가 문자열이 아닌 경우", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("requestId", 12345) // 정수 설정

		id := getRequestID(c)
		// 타입이 맞지 않으면 새 UUID를 생성해야 함
		if id == "" {
			t.Error("requestId가 생성되어야 함")
		}
		if len(id) != 36 {
			t.Errorf("requestId가 UUID 형식이어야 함")
		}
	})
}

// TestResponseStructs는 응답 구조체가 올바르게 JSON으로 직렬화되는지 테스트합니다.
func TestResponseStructs(t *testing.T) {
	t.Run("SuccessResponse 직렬화", func(t *testing.T) {
		resp := SuccessResponse{
			Data:      map[string]string{"key": "value"},
			RequestID: "test-request-id",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("JSON 직렬화 실패: %v", err)
		}

		// 예상 필드 확인
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		if _, ok := parsed["data"]; !ok {
			t.Error("'data' 필드가 있어야 함")
		}
		if _, ok := parsed["requestId"]; !ok {
			t.Error("'requestId' 필드가 있어야 함")
		}
	})

	t.Run("ErrorResponse 직렬화", func(t *testing.T) {
		resp := ErrorResponse{
			Error: ErrorDetail{
				Code:    "TEST_ERROR",
				Message: "Test error message",
				Details: "Some details",
			},
			RequestID: "test-request-id",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("JSON 직렬화 실패: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}

		if _, ok := parsed["error"]; !ok {
			t.Error("'error' 필드가 있어야 함")
		}
		if _, ok := parsed["requestId"]; !ok {
			t.Error("'requestId' 필드가 있어야 함")
		}
	})

	t.Run("ErrorDetail omitempty 동작", func(t *testing.T) {
		// details가 빈 경우 JSON에 포함되지 않아야 함
		detail := ErrorDetail{
			Code:    "TEST_ERROR",
			Message: "Test message",
			Details: "", // 빈 문자열
		}

		data, err := json.Marshal(detail)
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

// TestContentType은 응답 Content-Type이 올바른지 테스트합니다.
func TestContentType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, nil)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("예상 Content-Type: 'application/json; charset=utf-8', 실제: '%s'", contentType)
	}
}
