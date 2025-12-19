// Package response는 HTTP 응답 유틸리티를 제공합니다.
// 공통 모듈을 래핑하여 서비스에서 일관된 응답 형식을 제공합니다.
package response

import (
	"github.com/gin-gonic/gin"

	commonresponse "github.com/OrangesCloud/wealist-advanced-go-pkg/response"
)

// 공통 모듈 타입 재export
type (
	// SuccessResponse는 성공 API 응답 구조체입니다.
	SuccessResponse = commonresponse.SuccessResponse
	// ErrorResponse는 에러 API 응답 구조체입니다.
	ErrorResponse = commonresponse.ErrorResponse
)

// SendSuccess는 성공 응답을 전송합니다. (공통 모듈 사용)
func SendSuccess(c *gin.Context, statusCode int, data interface{}) {
	commonresponse.Success(c, statusCode, data)
}

// SendSuccessMessage는 메시지와 함께 성공 응답을 전송합니다. (공통 모듈 사용)
func SendSuccessMessage(c *gin.Context, statusCode int, data interface{}, message string) {
	commonresponse.SuccessWithMessage(c, statusCode, data, message)
}

// SendError는 에러 응답을 전송합니다. (공통 모듈 사용)
func SendError(c *gin.Context, statusCode int, code string, message string) {
	commonresponse.Error(c, statusCode, code, message)
}

// SendErrorWithDetails는 상세 정보와 함께 에러 응답을 전송합니다. (공통 모듈 사용)
func SendErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details string) {
	commonresponse.ErrorWithDetails(c, statusCode, code, message, details)
}
