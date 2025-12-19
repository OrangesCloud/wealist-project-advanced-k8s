// Package response provides HTTP response utilities for the chat-service.
// 이 파일은 chat-service의 비즈니스 에러 정의와 처리 함수를 포함합니다.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// AppError는 공통 에러 패키지의 타입 alias입니다.
type AppError = apperrors.AppError

// Service-level sentinel errors for chat-service
// 채팅 서비스 비즈니스 로직에서 발생하는 에러들
var (
	// 채팅방 관련 에러
	ErrChatNotFound       = errors.New("chat not found")
	ErrNotChatParticipant = errors.New("user is not a participant of this chat")
	ErrNotChatCreator     = errors.New("only chat creator can perform this action")
	ErrAlreadyParticipant = errors.New("user is already a participant")

	// 메시지 관련 에러
	ErrMessageNotFound = errors.New("message not found")
	ErrNotMessageOwner = errors.New("only message owner can perform this action")
	ErrEmptyMessage    = errors.New("message content cannot be empty")

	// 워크스페이스 에러
	ErrNotWorkspaceMember = errors.New("user is not a member of this workspace")
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
func NewValidationErrorTyped(message, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewConflictError는 409 CONFLICT 에러를 생성합니다.
func NewConflictError(message, details string) *AppError {
	return apperrors.Conflict(message, details)
}

// NewBadRequestError는 400 BAD_REQUEST 에러를 생성합니다.
func NewBadRequestError(message, details string) *AppError {
	return apperrors.BadRequest(message, details)
}

// NewInternalError는 500 INTERNAL 에러를 생성합니다.
func NewInternalError(message, details string) *AppError {
	return apperrors.Internal(message, details)
}

// HandleServiceError maps service errors to HTTP responses.
// sentinel error와 타입화된 AppError 모두 처리합니다.
func HandleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrChatNotFound):
		NotFound(c, "Chat not found")

	case errors.Is(err, ErrNotChatParticipant):
		Forbidden(c, "You are not a participant of this chat")

	case errors.Is(err, ErrNotChatCreator):
		Forbidden(c, "Only the chat creator can perform this action")

	case errors.Is(err, ErrAlreadyParticipant):
		Conflict(c, "User is already a participant of this chat")

	case errors.Is(err, ErrMessageNotFound):
		NotFound(c, "Message not found")

	case errors.Is(err, ErrNotMessageOwner):
		Forbidden(c, "Only the message owner can perform this action")

	case errors.Is(err, ErrEmptyMessage):
		BadRequest(c, "Message content cannot be empty")

	case errors.Is(err, ErrNotWorkspaceMember):
		Forbidden(c, "You are not a member of this workspace")

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

// Error sends an error response using the common AppError type.
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

// ValidationError sends a 400 response for validation errors.
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

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	Error(c, apperrors.Conflict(message, ""))
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.Internal(message, ""))
}

// Success sends a 200 OK response with success wrapper and message.
func Success(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// OK sends a 200 OK response with success wrapper and data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
