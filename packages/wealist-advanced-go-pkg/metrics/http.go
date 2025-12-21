package metrics

import (
	"strings"
	"time"
)

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.safeExecute("RecordHTTPRequest", func() {
		status := CategorizeStatus(statusCode)
		m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	})
}

// CategorizeStatus converts status code to category (2xx, 3xx, 4xx, 5xx)
func CategorizeStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// DefaultSkipPaths contains common paths to exclude from metrics
var DefaultSkipPaths = []string{
	"/metrics",
	"/health",
	"/health/live",
	"/health/ready",
	"/readyz",
	"/livez",
}

// ShouldSkipEndpoint checks if endpoint should be excluded from metrics
func ShouldSkipEndpoint(path string) bool {
	for _, skip := range DefaultSkipPaths {
		if path == skip {
			return true
		}
	}
	return false
}

// ShouldSkipEndpointWithBasePath checks if endpoint should be excluded,
// considering the service base path (e.g., "/api/boards/metrics")
func ShouldSkipEndpointWithBasePath(path, basePath string) bool {
	if ShouldSkipEndpoint(path) {
		return true
	}

	// Check with base path prefix
	if basePath != "" {
		for _, skip := range DefaultSkipPaths {
			if path == basePath+skip {
				return true
			}
		}
	}
	return false
}

// NormalizeEndpoint normalizes endpoint path for consistent metrics
// e.g., /api/boards/123 -> /api/boards/:id
func NormalizeEndpoint(path string) string {
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Replace UUIDs with :id placeholder
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if isUUID(part) {
			parts[i] = ":id"
		}
	}

	return strings.Join(parts, "/")
}

// isUUID checks if string looks like a UUID
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !isHexChar(c) {
			return false
		}
	}
	return true
}

// isHexChar checks if a rune is a valid hexadecimal character
func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
