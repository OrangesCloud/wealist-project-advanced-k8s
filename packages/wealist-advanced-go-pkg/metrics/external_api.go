package metrics

import (
	"time"
)

// RecordExternalAPIRequest records external API request metrics
func (m *Metrics) RecordExternalAPIRequest(endpoint, method string, statusCode int, duration time.Duration) {
	m.safeExecute("RecordExternalAPIRequest", func() {
		status := CategorizeStatus(statusCode)
		m.ExternalAPIRequestsTotal.WithLabelValues(endpoint, method, status).Inc()
		m.ExternalAPIRequestDuration.WithLabelValues(endpoint, status).Observe(duration.Seconds())
	})
}

// RecordExternalAPIError records external API error
func (m *Metrics) RecordExternalAPIError(endpoint, errorType string) {
	m.safeExecute("RecordExternalAPIError", func() {
		m.ExternalAPIErrors.WithLabelValues(endpoint, errorType).Inc()
	})
}

// ExternalAPIErrorType constants for common error types
const (
	ErrorTypeTimeout     = "timeout"
	ErrorTypeConnection  = "connection"
	ErrorTypeDNS         = "dns"
	ErrorTypeTLS         = "tls"
	ErrorTypeRateLimit   = "rate_limit"
	ErrorTypeServerError = "server_error"
	ErrorTypeClientError = "client_error"
	ErrorTypeUnknown     = "unknown"
)

// CategorizeError categorizes an error into a type for metrics
func CategorizeError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	switch {
	case containsAny(errStr, "timeout", "deadline exceeded"):
		return ErrorTypeTimeout
	case containsAny(errStr, "no such host", "dns"):
		return ErrorTypeDNS
	case containsAny(errStr, "connection refused", "connection reset"):
		return ErrorTypeConnection
	case containsAny(errStr, "tls", "certificate"):
		return ErrorTypeTLS
	case containsAny(errStr, "rate limit", "too many requests", "429"):
		return ErrorTypeRateLimit
	default:
		return ErrorTypeUnknown
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// RecordExternalAPICall records external API call with automatic error handling.
// This is a convenience method that combines request and error recording.
// Use this when you want to record both the request metrics and any errors in one call.
func (m *Metrics) RecordExternalAPICall(endpoint, method string, statusCode int, duration time.Duration, err error) {
	m.safeExecute("RecordExternalAPICall", func() {
		// Normalize endpoint to avoid high cardinality
		normalizedEndpoint := NormalizeEndpoint(endpoint)
		status := CategorizeStatus(statusCode)

		m.ExternalAPIRequestsTotal.WithLabelValues(normalizedEndpoint, method, status).Inc()
		m.ExternalAPIRequestDuration.WithLabelValues(normalizedEndpoint, status).Observe(duration.Seconds())

		// Record errors for both network errors and HTTP error status codes
		if err != nil || statusCode >= 400 {
			errorType := CategorizeExternalError(statusCode, err)
			m.ExternalAPIErrors.WithLabelValues(normalizedEndpoint, errorType).Inc()
		}
	})
}

// CategorizeExternalError categorizes errors based on status code and error message.
// This provides more detailed error classification than CategorizeError.
func CategorizeExternalError(statusCode int, err error) string {
	// First, check HTTP status codes
	switch statusCode {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 408:
		return "request_timeout"
	case 429:
		return ErrorTypeRateLimit
	case 500:
		return "internal_server_error"
	case 502:
		return "bad_gateway"
	case 503:
		return "service_unavailable"
	case 504:
		return "gateway_timeout"
	}

	// Check status code ranges
	if statusCode >= 400 && statusCode < 500 {
		return ErrorTypeClientError
	}
	if statusCode >= 500 && statusCode < 600 {
		return ErrorTypeServerError
	}

	// Check network errors
	if err != nil {
		return CategorizeError(err)
	}

	return ErrorTypeUnknown
}
