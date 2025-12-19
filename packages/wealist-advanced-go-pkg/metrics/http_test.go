package metrics

import (
	"testing"
	"time"
)

func TestRecordHTTPRequest(t *testing.T) {
	m := NewForTest("test_service")

	// Should not panic
	m.RecordHTTPRequest("GET", "/api/users", 200, 100*time.Millisecond)
	m.RecordHTTPRequest("POST", "/api/users", 201, 200*time.Millisecond)
	m.RecordHTTPRequest("GET", "/api/users", 404, 50*time.Millisecond)
	m.RecordHTTPRequest("POST", "/api/users", 500, 1*time.Second)
}

func TestCategorizeStatus(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{204, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{302, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{503, "5xx"},
		{599, "5xx"},
		{100, "unknown"},
		{0, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			result := CategorizeStatus(tt.code)
			if result != tt.expected {
				t.Errorf("CategorizeStatus(%d) = %s, want %s", tt.code, result, tt.expected)
			}
		})
	}
}

func TestShouldSkipEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/metrics", true},
		{"/health", true},
		{"/health/live", true},
		{"/health/ready", true},
		{"/readyz", true},
		{"/livez", true},
		{"/api/users", false},
		{"/api/boards", false},
		{"/api/metrics", false}, // different from /metrics
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ShouldSkipEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldSkipEndpoint(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestShouldSkipEndpointWithBasePath(t *testing.T) {
	tests := []struct {
		path     string
		basePath string
		expected bool
	}{
		{"/metrics", "", true},
		{"/health", "", true},
		{"/api/boards/metrics", "/api/boards", true},
		{"/api/boards/health", "/api/boards", true},
		{"/api/boards/health/live", "/api/boards", true},
		{"/api/boards/users", "/api/boards", false},
		{"/api/users", "/api/boards", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ShouldSkipEndpointWithBasePath(tt.path, tt.basePath)
			if result != tt.expected {
				t.Errorf("ShouldSkipEndpointWithBasePath(%s, %s) = %v, want %v", tt.path, tt.basePath, result, tt.expected)
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/api/users", "/api/users"},
		{"/api/users/", "/api/users"},
		{"/api/users/123e4567-e89b-12d3-a456-426614174000", "/api/users/:id"},
		{"/api/boards/123e4567-e89b-12d3-a456-426614174000/tasks", "/api/boards/:id/tasks"},
		{"/api/boards/123e4567-e89b-12d3-a456-426614174000/tasks/987e6543-e21b-12d3-a456-426614174000", "/api/boards/:id/tasks/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := NormalizeEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("NormalizeEndpoint(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		s        string
		expected bool
	}{
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"ABCDEF12-3456-7890-ABCD-EF1234567890", true},
		{"not-a-uuid", false},
		{"123e4567e89b12d3a456426614174000", false}, // no dashes
		{"123e4567-e89b-12d3-a456-42661417400", false}, // too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := isUUID(tt.s)
			if result != tt.expected {
				t.Errorf("isUUID(%s) = %v, want %v", tt.s, result, tt.expected)
			}
		})
	}
}
