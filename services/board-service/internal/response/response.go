// Package response는 HTTP 응답 유틸리티를 제공합니다.
package response

import (
	"github.com/gin-gonic/gin"

	commonresponse "github.com/OrangesCloud/wealist-advanced-go-pkg/response"
)

// 공통 모듈 타입 재export
type (
	SuccessResponse = commonresponse.SuccessResponse
	ErrorResponse   = commonresponse.ErrorResponse
)

// SendSuccess는 성공 응답을 전송합니다.
func SendSuccess(c *gin.Context, statusCode int, data interface{}) {
	commonresponse.Success(c, statusCode, data)
}

// SendSuccessMessage는 메시지와 함께 성공 응답을 전송합니다. (하위 호환성)
func SendSuccessMessage(c *gin.Context, statusCode int, data interface{}, message string) {
	commonresponse.SuccessWithMessage(c, statusCode, data, message)
}

// SendError는 에러 응답을 전송합니다.
func SendError(c *gin.Context, statusCode int, code string, message string) {
	commonresponse.Error(c, statusCode, code, message)
}

// SendErrorWithDetails는 상세 정보와 함께 에러 응답을 전송합니다.
func SendErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details string) {
	commonresponse.ErrorWithDetails(c, statusCode, code, message, details)
}
