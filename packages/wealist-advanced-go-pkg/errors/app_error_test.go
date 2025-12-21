package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name:     "with details",
			appErr:   &AppError{Code: ErrCodeNotFound, Message: "Resource not found", Details: "user_id=123"},
			expected: "NOT_FOUND: Resource not found (user_id=123)",
		},
		{
			name:     "without details",
			appErr:   &AppError{Code: ErrCodeInternal, Message: "Internal error"},
			expected: "INTERNAL_ERROR: Internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.expected {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	appErr := &AppError{Code: ErrCodeInternal, Message: "wrapper", Cause: cause}

	if appErr.Unwrap() != cause {
		t.Error("AppError.Unwrap() should return the cause")
	}
}

func TestAppError_WithCause(t *testing.T) {
	cause := errors.New("original error")
	appErr := &AppError{Code: ErrCodeInternal, Message: "wrapper"}

	result := appErr.WithCause(cause)

	if result != appErr {
		t.Error("WithCause should return the same error for chaining")
	}
	if appErr.Cause != cause {
		t.Error("WithCause should set the cause")
	}
}

func TestNew(t *testing.T) {
	appErr := New(ErrCodeNotFound, "not found", "details")

	if appErr.Code != ErrCodeNotFound {
		t.Errorf("New() Code = %v, want %v", appErr.Code, ErrCodeNotFound)
	}
	if appErr.Message != "not found" {
		t.Errorf("New() Message = %v, want %v", appErr.Message, "not found")
	}
	if appErr.Details != "details" {
		t.Errorf("New() Details = %v, want %v", appErr.Details, "details")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	appErr := Wrap(ErrCodeInternal, "wrapped error", cause)

	if appErr.Code != ErrCodeInternal {
		t.Errorf("Wrap() Code = %v, want %v", appErr.Code, ErrCodeInternal)
	}
	if appErr.Message != "wrapped error" {
		t.Errorf("Wrap() Message = %v, want %v", appErr.Message, "wrapped error")
	}
	if appErr.Cause != cause {
		t.Error("Wrap() should set the cause")
	}
	if appErr.Details != "original error" {
		t.Errorf("Wrap() Details = %v, want %v", appErr.Details, "original error")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string, string) *AppError
		expectedCode string
	}{
		{"NotFound", NotFound, ErrCodeNotFound},
		{"AlreadyExists", AlreadyExists, ErrCodeAlreadyExists},
		{"Validation", Validation, ErrCodeValidation},
		{"Internal", Internal, ErrCodeInternal},
		{"Unauthorized", Unauthorized, ErrCodeUnauthorized},
		{"Forbidden", Forbidden, ErrCodeForbidden},
		{"BadRequest", BadRequest, ErrCodeBadRequest},
		{"Conflict", Conflict, ErrCodeConflict},
		{"Timeout", Timeout, ErrCodeTimeout},
		{"ServiceUnavailable", ServiceUnavailable, ErrCodeServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := tt.constructor("message", "details")
			if appErr.Code != tt.expectedCode {
				t.Errorf("%s() Code = %v, want %v", tt.name, appErr.Code, tt.expectedCode)
			}
			if appErr.Message != "message" {
				t.Errorf("%s() Message = %v, want %v", tt.name, appErr.Message, "message")
			}
			if appErr.Details != "details" {
				t.Errorf("%s() Details = %v, want %v", tt.name, appErr.Details, "details")
			}
		})
	}
}

func TestIsAppError(t *testing.T) {
	appErr := &AppError{Code: ErrCodeNotFound, Message: "test"}
	regularErr := errors.New("regular error")

	if !IsAppError(appErr) {
		t.Error("IsAppError should return true for AppError")
	}
	if IsAppError(regularErr) {
		t.Error("IsAppError should return false for regular error")
	}
}

func TestAsAppError(t *testing.T) {
	appErr := &AppError{Code: ErrCodeNotFound, Message: "test"}
	regularErr := errors.New("regular error")

	if result := AsAppError(appErr); result != appErr {
		t.Error("AsAppError should return the AppError")
	}
	if result := AsAppError(regularErr); result != nil {
		t.Error("AsAppError should return nil for regular error")
	}
}

func TestGetCode(t *testing.T) {
	appErr := &AppError{Code: ErrCodeNotFound, Message: "test"}
	regularErr := errors.New("regular error")

	if code := GetCode(appErr); code != ErrCodeNotFound {
		t.Errorf("GetCode() = %v, want %v", code, ErrCodeNotFound)
	}
	if code := GetCode(regularErr); code != ErrCodeInternal {
		t.Errorf("GetCode() = %v, want %v", code, ErrCodeInternal)
	}
}
