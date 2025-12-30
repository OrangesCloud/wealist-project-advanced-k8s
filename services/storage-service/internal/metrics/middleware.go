package metrics

import (
	"github.com/gin-gonic/gin"

	commonmiddleware "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
)

// HTTPMiddleware returns a Gin middleware that records HTTP request metrics.
// It tracks request duration and count per method/endpoint/status combination.
// Health check and metrics endpoints are automatically skipped.
// Uses the common middleware package for consistent metrics recording.
func HTTPMiddleware(m *Metrics) gin.HandlerFunc {
	// storage-service의 basePath는 "/api/storage"
	return commonmiddleware.MetricsMiddleware(m.Metrics, "/api/storage")
}
