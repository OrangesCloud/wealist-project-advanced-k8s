package errors

import "testing"

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{ErrCodeNotFound, 404},
		{ErrCodeAlreadyExists, 409},
		{ErrCodeValidation, 400},
		{ErrCodeInternal, 500},
		{ErrCodeUnauthorized, 401},
		{ErrCodeForbidden, 403},
		{ErrCodeBadRequest, 400},
		{ErrCodeConflict, 409},
		{ErrCodeTimeout, 408},
		{ErrCodeServiceUnavailable, 503},
		{"UNKNOWN_CODE", 500}, // Unknown code should return 500
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := GetHTTPStatus(tt.code); got != tt.expected {
				t.Errorf("GetHTTPStatus(%v) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify constants are defined correctly
	expectedCodes := map[string]string{
		"ErrCodeNotFound":           "NOT_FOUND",
		"ErrCodeAlreadyExists":      "ALREADY_EXISTS",
		"ErrCodeValidation":         "VALIDATION_ERROR",
		"ErrCodeInternal":           "INTERNAL_ERROR",
		"ErrCodeUnauthorized":       "UNAUTHORIZED",
		"ErrCodeForbidden":          "FORBIDDEN",
		"ErrCodeBadRequest":         "BAD_REQUEST",
		"ErrCodeConflict":           "CONFLICT",
		"ErrCodeTimeout":            "TIMEOUT",
		"ErrCodeServiceUnavailable": "SERVICE_UNAVAILABLE",
	}

	actualCodes := map[string]string{
		"ErrCodeNotFound":           ErrCodeNotFound,
		"ErrCodeAlreadyExists":      ErrCodeAlreadyExists,
		"ErrCodeValidation":         ErrCodeValidation,
		"ErrCodeInternal":           ErrCodeInternal,
		"ErrCodeUnauthorized":       ErrCodeUnauthorized,
		"ErrCodeForbidden":          ErrCodeForbidden,
		"ErrCodeBadRequest":         ErrCodeBadRequest,
		"ErrCodeConflict":           ErrCodeConflict,
		"ErrCodeTimeout":            ErrCodeTimeout,
		"ErrCodeServiceUnavailable": ErrCodeServiceUnavailable,
	}

	for name, expected := range expectedCodes {
		if actual, ok := actualCodes[name]; !ok {
			t.Errorf("Missing constant: %s", name)
		} else if actual != expected {
			t.Errorf("%s = %v, want %v", name, actual, expected)
		}
	}
}
