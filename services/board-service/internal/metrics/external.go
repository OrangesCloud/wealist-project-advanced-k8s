package metrics

import (
	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// Re-export common module error type constants for backwards compatibility
const (
	ErrorTypeTimeout     = commonmetrics.ErrorTypeTimeout
	ErrorTypeConnection  = commonmetrics.ErrorTypeConnection
	ErrorTypeDNS         = commonmetrics.ErrorTypeDNS
	ErrorTypeTLS         = commonmetrics.ErrorTypeTLS
	ErrorTypeRateLimit   = commonmetrics.ErrorTypeRateLimit
	ErrorTypeServerError = commonmetrics.ErrorTypeServerError
	ErrorTypeClientError = commonmetrics.ErrorTypeClientError
	ErrorTypeUnknown     = commonmetrics.ErrorTypeUnknown
)

// CategorizeError categorizes an error into a type for metrics
// Delegates to common module
func CategorizeError(err error) string {
	return commonmetrics.CategorizeError(err)
}

// CategorizeExternalError categorizes errors based on status code and error message
// Delegates to common module
func CategorizeExternalError(statusCode int, err error) string {
	return commonmetrics.CategorizeExternalError(statusCode, err)
}
