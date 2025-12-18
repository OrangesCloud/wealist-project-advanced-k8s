package metrics

import (
	"errors"
	"testing"
	"time"
)

func TestRecordExternalAPIRequest(t *testing.T) {
	m := NewForTest("test_service")

	// Should not panic
	m.RecordExternalAPIRequest("/users/validate", "GET", 200, 100*time.Millisecond)
	m.RecordExternalAPIRequest("/users/validate", "POST", 201, 200*time.Millisecond)
	m.RecordExternalAPIRequest("/users/validate", "GET", 500, 5*time.Second)
}

func TestRecordExternalAPIError(t *testing.T) {
	m := NewForTest("test_service")

	// Should not panic
	m.RecordExternalAPIError("/users/validate", ErrorTypeTimeout)
	m.RecordExternalAPIError("/users/validate", ErrorTypeConnection)
	m.RecordExternalAPIError("/storage/upload", ErrorTypeServerError)
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"timeout error", errors.New("context deadline exceeded"), ErrorTypeTimeout},
		{"timeout word", errors.New("request timeout"), ErrorTypeTimeout},
		{"connection refused", errors.New("connection refused"), ErrorTypeConnection},
		{"connection reset", errors.New("connection reset by peer"), ErrorTypeConnection},
		{"no such host", errors.New("no such host"), ErrorTypeDNS},
		{"dns error", errors.New("dns lookup failed"), ErrorTypeDNS},
		{"tls error", errors.New("tls handshake failed"), ErrorTypeTLS},
		{"certificate error", errors.New("certificate verify failed"), ErrorTypeTLS},
		{"rate limit", errors.New("rate limit exceeded"), ErrorTypeRateLimit},
		{"too many requests", errors.New("too many requests"), ErrorTypeRateLimit},
		{"429 status", errors.New("HTTP 429"), ErrorTypeRateLimit},
		{"unknown error", errors.New("some random error"), ErrorTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("CategorizeError(%v) = %s, want %s", tt.err, result, tt.expected)
			}
		})
	}
}

func TestErrorTypeConstants(t *testing.T) {
	// Verify constants are defined correctly
	expectedTypes := map[string]string{
		"ErrorTypeTimeout":     "timeout",
		"ErrorTypeConnection":  "connection",
		"ErrorTypeDNS":         "dns",
		"ErrorTypeTLS":         "tls",
		"ErrorTypeRateLimit":   "rate_limit",
		"ErrorTypeServerError": "server_error",
		"ErrorTypeClientError": "client_error",
		"ErrorTypeUnknown":     "unknown",
	}

	actualTypes := map[string]string{
		"ErrorTypeTimeout":     ErrorTypeTimeout,
		"ErrorTypeConnection":  ErrorTypeConnection,
		"ErrorTypeDNS":         ErrorTypeDNS,
		"ErrorTypeTLS":         ErrorTypeTLS,
		"ErrorTypeRateLimit":   ErrorTypeRateLimit,
		"ErrorTypeServerError": ErrorTypeServerError,
		"ErrorTypeClientError": ErrorTypeClientError,
		"ErrorTypeUnknown":     ErrorTypeUnknown,
	}

	for name, expected := range expectedTypes {
		if actual, ok := actualTypes[name]; !ok {
			t.Errorf("Missing constant: %s", name)
		} else if actual != expected {
			t.Errorf("%s = %v, want %v", name, actual, expected)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		expected   bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo", "bar", "world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"", []string{"foo"}, false},
		{"foo", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings...)
			if result != tt.expected {
				t.Errorf("containsAny(%s, %v) = %v, want %v", tt.s, tt.substrings, result, tt.expected)
			}
		})
	}
}
