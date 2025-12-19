package errors

import (
	"fmt"
)

// AppError represents a structured application error with code, message, and optional details.
// It implements the error interface and can be used throughout the application
// for consistent error handling.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Cause   error  `json:"-"` // Original error, not serialized to JSON
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause of the error for errors.Is/As support
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error and returns the same error for chaining
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// New creates a new AppError with the given code, message, and details
func New(code string, message string, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap wraps an existing error with an AppError
func Wrap(code string, message string, cause error) *AppError {
	details := ""
	if cause != nil {
		details = cause.Error()
	}
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// NotFound creates a new not found error
func NotFound(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeNotFound,
		Message: message,
		Details: details,
	}
}

// AlreadyExists creates a new already exists error
func AlreadyExists(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeAlreadyExists,
		Message: message,
		Details: details,
	}
}

// Validation creates a new validation error
func Validation(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeValidation,
		Message: message,
		Details: details,
	}
}

// Internal creates a new internal error
func Internal(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: message,
		Details: details,
	}
}

// Unauthorized creates a new unauthorized error
func Unauthorized(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeUnauthorized,
		Message: message,
		Details: details,
	}
}

// Forbidden creates a new forbidden error
func Forbidden(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeForbidden,
		Message: message,
		Details: details,
	}
}

// BadRequest creates a new bad request error
func BadRequest(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeBadRequest,
		Message: message,
		Details: details,
	}
}

// Conflict creates a new conflict error
func Conflict(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeConflict,
		Message: message,
		Details: details,
	}
}

// Timeout creates a new timeout error
func Timeout(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeTimeout,
		Message: message,
		Details: details,
	}
}

// ServiceUnavailable creates a new service unavailable error
func ServiceUnavailable(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeServiceUnavailable,
		Message: message,
		Details: details,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to convert an error to an AppError.
// Returns nil if the error is not an AppError.
func AsAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

// GetCode returns the error code if the error is an AppError,
// otherwise returns ErrCodeInternal.
func GetCode(err error) string {
	if appErr := AsAppError(err); appErr != nil {
		return appErr.Code
	}
	return ErrCodeInternal
}
