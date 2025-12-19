// Package response provides centralized HTTP response helpers for noti-service.
//
// This package standardizes API responses across all handlers, ensuring
// consistent error formats and success responses throughout the service.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// AppError는 공통 에러 패키지의 타입 alias입니다.
type AppError = apperrors.AppError

// ============================================================
// Service-level sentinel errors for noti-service
// 알림 서비스 비즈니스 로직에서 발생하는 에러들
// ============================================================

var (
	// 알림 관련 에러
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotOwner             = errors.New("user is not the owner of this notification")

	// 워크스페이스 에러
	ErrNotWorkspaceMember = errors.New("user is not a member of this workspace")

	// 유효성 검사 에러
	ErrInvalidNotificationType = errors.New("invalid notification type")
)

// ============================================================
// 타입화된 에러 생성 함수들 (Service Layer용)
// Handler에서 HandleServiceError()로 자동 HTTP 상태 매핑
// ============================================================

// NewNotFoundError는 404 NOT_FOUND 에러를 생성합니다.
func NewNotFoundError(message, details string) *AppError {
	return apperrors.NotFound(message, details)
}

// NewForbiddenError는 403 FORBIDDEN 에러를 생성합니다.
func NewForbiddenError(message, details string) *AppError {
	return apperrors.Forbidden(message, details)
}

// NewValidationError는 400 VALIDATION 에러를 생성합니다.
func NewValidationError(message, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewInternalError는 500 INTERNAL 에러를 생성합니다.
func NewInternalError(message, details string) *AppError {
	return apperrors.Internal(message, details)
}

// HandleServiceError는 서비스 레이어 에러를 HTTP 응답으로 매핑합니다.
// sentinel error와 타입화된 AppError 모두 처리합니다.
func HandleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotificationNotFound):
		NotFound(c, "Notification not found")

	case errors.Is(err, ErrNotOwner):
		Forbidden(c, "You are not the owner of this notification")

	case errors.Is(err, ErrNotWorkspaceMember):
		Forbidden(c, "You are not a member of this workspace")

	case errors.Is(err, ErrInvalidNotificationType):
		BadRequest(c, "Invalid notification type")

	default:
		// 타입화된 AppError 처리
		if appErr := apperrors.AsAppError(err); appErr != nil {
			Error(c, appErr)
			return
		}
		// 기본 내부 에러
		InternalError(c, "An internal error occurred")
	}
}

// Error sends an error response based on AppError.
// It maps the error code to appropriate HTTP status.
func Error(c *gin.Context, err *apperrors.AppError) {
	status := apperrors.GetHTTPStatus(err.Code)
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	Error(c, apperrors.BadRequest(message, ""))
}

// ValidationError sends a 400 Bad Request response for validation failures.
func ValidationError(c *gin.Context, message string) {
	Error(c, apperrors.Validation(message, ""))
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	Error(c, apperrors.Unauthorized(message, ""))
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	Error(c, apperrors.Forbidden(message, ""))
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	Error(c, apperrors.NotFound(message, ""))
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.Internal(message, ""))
}

// Success sends a success response with a message.
func Success(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// OK sends a 200 OK response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// OKWithPagination sends a 200 OK response with pagination info.
func OKWithPagination(c *gin.Context, data interface{}, total int64, page, limit int) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// CustomError sends a custom error response with specific status code and code.
func CustomError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
