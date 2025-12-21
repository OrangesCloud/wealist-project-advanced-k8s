// Package response는 HTTP 응답 유틸리티를 제공합니다.
package response

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SuccessResponse는 성공 API 응답 구조체입니다.
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
}

// ErrorResponse는 에러 API 응답 구조체입니다.
type ErrorResponse struct {
	Error     interface{} `json:"error"`
	RequestID string      `json:"requestId"`
}

// ErrorDetail은 에러 상세 정보 구조체입니다.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// getRequestID는 컨텍스트에서 요청 ID를 가져오거나 생성합니다.
func getRequestID(c *gin.Context) string {
	// 미들웨어에서 설정한 요청 ID 확인
	if requestID, exists := c.Get("requestId"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	// 없으면 새로 생성
	return uuid.New().String()
}

// Success는 성공 응답을 전송합니다.
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Data:      data,
		RequestID: getRequestID(c),
	})
}

// SuccessWithMessage는 메시지와 함께 성공 응답을 전송합니다. (하위 호환성)
func SuccessWithMessage(c *gin.Context, statusCode int, data interface{}, message string) {
	responseData := data
	if message != "" && data == nil {
		responseData = map[string]string{"message": message}
	}
	c.JSON(statusCode, SuccessResponse{
		Data:      responseData,
		RequestID: getRequestID(c),
	})
}

// Error는 에러 응답을 전송합니다.
func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		RequestID: getRequestID(c),
	})
}

// ErrorWithDetails는 상세 정보와 함께 에러 응답을 전송합니다.
func ErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details string) {
	c.JSON(statusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		RequestID: getRequestID(c),
	})
}

// BadRequest는 400 에러 응답을 전송합니다.
func BadRequest(c *gin.Context, message string) {
	Error(c, 400, "BAD_REQUEST", message)
}

// Unauthorized는 401 에러 응답을 전송합니다.
func Unauthorized(c *gin.Context, message string) {
	Error(c, 401, "UNAUTHORIZED", message)
}

// Forbidden는 403 에러 응답을 전송합니다.
func Forbidden(c *gin.Context, message string) {
	Error(c, 403, "FORBIDDEN", message)
}

// NotFound는 404 에러 응답을 전송합니다.
func NotFound(c *gin.Context, message string) {
	Error(c, 404, "NOT_FOUND", message)
}

// Conflict는 409 에러 응답을 전송합니다.
func Conflict(c *gin.Context, message string) {
	Error(c, 409, "CONFLICT", message)
}

// InternalError는 500 에러 응답을 전송합니다.
func InternalError(c *gin.Context, message string) {
	Error(c, 500, "INTERNAL_ERROR", message)
}
