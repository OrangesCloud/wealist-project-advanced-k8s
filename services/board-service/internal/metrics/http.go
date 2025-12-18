package metrics

import (
	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// ShouldSkipEndpoint checks if endpoint should be excluded from metrics
// Delegates to common module with board-service base path
func ShouldSkipEndpoint(path string) bool {
	return commonmetrics.ShouldSkipEndpointWithBasePath(path, "/api/boards")
}

// NormalizeEndpoint normalizes endpoint path for consistent metrics
// Delegates to common module
func NormalizeEndpoint(path string) string {
	return commonmetrics.NormalizeEndpoint(path)
}
