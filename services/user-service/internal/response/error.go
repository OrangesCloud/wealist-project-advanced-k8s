// Package response provides HTTP response utilities for the user-service.
// Handler용 응답 함수와 Service용 에러 생성 함수를 제공합니다.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// ============================================================
// 타입 및 상수 (공통 모듈 재사용)
// ============================================================

// AppError는 공통 모듈의 AppError 타입 alias입니다.
// Service 레이어에서 타입화된 에러를 반환할 때 사용합니다.
type AppError = apperrors.AppError

// 에러 코드 상수 - 공통 모듈 상수 재사용
const (
	ErrCodeNotFound      = apperrors.ErrCodeNotFound
	ErrCodeAlreadyExists = apperrors.ErrCodeAlreadyExists
	ErrCodeValidation    = apperrors.ErrCodeValidation
	ErrCodeInternal      = apperrors.ErrCodeInternal
	ErrCodeUnauthorized  = apperrors.ErrCodeUnauthorized
	ErrCodeForbidden     = apperrors.ErrCodeForbidden
	ErrCodeBadRequest    = apperrors.ErrCodeBadRequest
	ErrCodeConflict      = apperrors.ErrCodeConflict
)

// ============================================================
// Service 레이어용 에러 생성 함수
// ============================================================

// NewNotFoundError는 리소스를 찾을 수 없을 때 사용합니다. (404)
func NewNotFoundError(message, details string) *AppError {
	return apperrors.NotFound(message, details)
}

// NewAlreadyExistsError는 리소스가 이미 존재할 때 사용합니다. (409)
func NewAlreadyExistsError(message, details string) *AppError {
	return apperrors.AlreadyExists(message, details)
}

// NewValidationError는 유효성 검증 실패 시 사용합니다. (400)
func NewValidationError(message, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewInternalError는 내부 서버 오류 시 사용합니다. (500)
func NewInternalError(message, details string) *AppError {
	return apperrors.Internal(message, details)
}

// NewUnauthorizedError는 인증 실패 시 사용합니다. (401)
func NewUnauthorizedError(message, details string) *AppError {
	return apperrors.Unauthorized(message, details)
}

// NewForbiddenError는 권한 부족 시 사용합니다. (403)
func NewForbiddenError(message, details string) *AppError {
	return apperrors.Forbidden(message, details)
}

// NewBadRequestError는 잘못된 요청 시 사용합니다. (400)
func NewBadRequestError(message, details string) *AppError {
	return apperrors.BadRequest(message, details)
}

// NewConflictError는 충돌 발생 시 사용합니다. (409)
func NewConflictError(message, details string) *AppError {
	return apperrors.Conflict(message, details)
}

// NewAppError는 사용자 정의 에러 코드로 에러를 생성합니다.
func NewAppError(code, message, details string) *AppError {
	return apperrors.New(code, message, details)
}

// ============================================================
// Handler 레이어용 응답 함수
// ============================================================

// Error sends an error response using the common AppError type.
func Error(c *gin.Context, err *apperrors.AppError) {
	status := apperrors.GetHTTPStatus(err.Code)
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// ErrorWithDetails sends an error response with additional details.
func ErrorWithDetails(c *gin.Context, err *apperrors.AppError) {
	status := apperrors.GetHTTPStatus(err.Code)
	response := gin.H{
		"code":    err.Code,
		"message": err.Message,
	}
	if err.Details != "" {
		response["details"] = err.Details
	}
	c.JSON(status, gin.H{"error": response})
}

// HandleError converts a generic error to an appropriate HTTP response.
// If the error is an AppError, it uses the error's code.
// Otherwise, it returns a 500 Internal Server Error.
func HandleError(c *gin.Context, err error) {
	if appErr := apperrors.AsAppError(err); appErr != nil {
		Error(c, appErr)
		return
	}

	// Default to internal error
	Error(c, apperrors.Internal("An unexpected error occurred", err.Error()))
}

// Common error response helpers

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	Error(c, apperrors.BadRequest(message, ""))
}

// BadRequestWithDetails sends a 400 Bad Request response with details.
func BadRequestWithDetails(c *gin.Context, message, details string) {
	Error(c, apperrors.BadRequest(message, details))
}

// ValidationError sends a 400 response for validation errors.
func ValidationError(c *gin.Context, message string) {
	Error(c, apperrors.Validation(message, ""))
}

// ValidationErrorWithDetails sends a 400 response for validation errors with details.
func ValidationErrorWithDetails(c *gin.Context, message, details string) {
	Error(c, apperrors.Validation(message, details))
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

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	Error(c, apperrors.Conflict(message, ""))
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.Internal(message, ""))
}

// InternalErrorWithDetails sends a 500 response with error details.
func InternalErrorWithDetails(c *gin.Context, message string, err error) {
	details := ""
	if err != nil {
		details = err.Error()
	}
	Error(c, apperrors.Internal(message, details))
}

// Success response helpers

// Success sends a success response with a message.
func Success(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"message": message})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// OK sends a 200 OK response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
