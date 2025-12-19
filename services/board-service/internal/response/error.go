// Package response provides HTTP response utilities and error handling.
package response

import (
	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// Error codes - using common module constants for consistency
const (
	ErrCodeNotFound      = apperrors.ErrCodeNotFound
	ErrCodeAlreadyExists = apperrors.ErrCodeAlreadyExists
	ErrCodeValidation    = apperrors.ErrCodeValidation
	ErrCodeInternal      = apperrors.ErrCodeInternal
	ErrCodeUnauthorized  = apperrors.ErrCodeUnauthorized
	ErrCodeForbidden     = apperrors.ErrCodeForbidden
)

// AppError is an alias for the common module's AppError
// This maintains backwards compatibility with existing code
type AppError = apperrors.AppError

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, details string) *AppError {
	return apperrors.NotFound(message, details)
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(message string, details string) *AppError {
	return apperrors.AlreadyExists(message, details)
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewInternalError creates a new internal error
func NewInternalError(message string, details string) *AppError {
	return apperrors.Internal(message, details)
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, details string) *AppError {
	return apperrors.Unauthorized(message, details)
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string, details string) *AppError {
	return apperrors.Forbidden(message, details)
}

// NewAppError creates a new application error with the given code, message, and details
func NewAppError(code string, message string, details string) *AppError {
	return apperrors.New(code, message, details)
}
