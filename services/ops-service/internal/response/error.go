package response

import (
	"errors"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// Type alias for convenience
type AppError = apperrors.AppError

// Sentinel errors for ops-service
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidRole       = errors.New("invalid role")
	ErrSelfModification  = errors.New("cannot modify own account")
	ErrLastAdmin         = errors.New("cannot remove last admin")
	ErrConfigNotFound    = errors.New("config not found")
	ErrConfigExists      = errors.New("config key already exists")
	ErrAuditLogNotFound  = errors.New("audit log not found")
)

// NewNotFoundError creates a not found error
func NewNotFoundError(message, details string) *AppError {
	return apperrors.NotFound(message, details)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message, details string) *AppError {
	return apperrors.Forbidden(message, details)
}

// NewAlreadyExistsError creates an already exists error
func NewAlreadyExistsError(message, details string) *AppError {
	return apperrors.AlreadyExists(message, details)
}

// NewValidationError creates a validation error
func NewValidationError(message, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewConflictError creates a conflict error
func NewConflictError(message, details string) *AppError {
	return apperrors.Conflict(message, details)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message, details string) *AppError {
	return apperrors.Unauthorized(message, details)
}

// NewInternalError creates an internal error
func NewInternalError(message, details string) *AppError {
	return apperrors.Internal(message, details)
}

// HandleServiceError handles service layer errors and sends appropriate HTTP responses
func HandleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		NotFound(c, "User not found")
	case errors.Is(err, ErrUserAlreadyExists):
		Conflict(c, "User already exists")
	case errors.Is(err, ErrInvalidRole):
		BadRequest(c, "Invalid role")
	case errors.Is(err, ErrSelfModification):
		Forbidden(c, "Cannot modify own account")
	case errors.Is(err, ErrLastAdmin):
		Conflict(c, "Cannot remove the last admin")
	case errors.Is(err, ErrConfigNotFound):
		NotFound(c, "Config not found")
	case errors.Is(err, ErrConfigExists):
		Conflict(c, "Config key already exists")
	case errors.Is(err, ErrAuditLogNotFound):
		NotFound(c, "Audit log not found")
	default:
		if appErr := apperrors.AsAppError(err); appErr != nil {
			Error(c, appErr)
			return
		}
		InternalError(c, "An internal error occurred")
	}
}

// Error sends an error response based on AppError type
func Error(c *gin.Context, appErr *AppError) {
	status := apperrors.GetHTTPStatus(appErr.Code)
	c.JSON(status, Response{
		Success: false,
		Message: appErr.Message,
	})
}
