// Package errors provides common error handling utilities for Go microservices.
package errors

// Standard error codes for HTTP responses
const (
	// ErrCodeNotFound indicates the requested resource was not found
	ErrCodeNotFound = "NOT_FOUND"

	// ErrCodeAlreadyExists indicates the resource already exists
	ErrCodeAlreadyExists = "ALREADY_EXISTS"

	// ErrCodeValidation indicates a validation error in the request
	ErrCodeValidation = "VALIDATION_ERROR"

	// ErrCodeInternal indicates an internal server error
	ErrCodeInternal = "INTERNAL_ERROR"

	// ErrCodeUnauthorized indicates the request lacks valid authentication
	ErrCodeUnauthorized = "UNAUTHORIZED"

	// ErrCodeForbidden indicates the user doesn't have permission
	ErrCodeForbidden = "FORBIDDEN"

	// ErrCodeBadRequest indicates a malformed or invalid request
	ErrCodeBadRequest = "BAD_REQUEST"

	// ErrCodeConflict indicates a conflict with the current state
	ErrCodeConflict = "CONFLICT"

	// ErrCodeTimeout indicates the operation timed out
	ErrCodeTimeout = "TIMEOUT"

	// ErrCodeServiceUnavailable indicates the service is temporarily unavailable
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// HTTP status code mapping for error codes
var ErrorCodeToHTTPStatus = map[string]int{
	ErrCodeNotFound:           404,
	ErrCodeAlreadyExists:      409,
	ErrCodeValidation:         400,
	ErrCodeInternal:           500,
	ErrCodeUnauthorized:       401,
	ErrCodeForbidden:          403,
	ErrCodeBadRequest:         400,
	ErrCodeConflict:           409,
	ErrCodeTimeout:            408,
	ErrCodeServiceUnavailable: 503,
}

// GetHTTPStatus returns the HTTP status code for the given error code.
// Returns 500 (Internal Server Error) if the code is not recognized.
func GetHTTPStatus(code string) int {
	if status, ok := ErrorCodeToHTTPStatus[code]; ok {
		return status
	}
	return 500
}
