package metrics

import (
	"time"

	"github.com/gin-gonic/gin"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// HTTPMiddleware returns a Gin middleware that records HTTP request metrics.
// It tracks request duration and count per method/endpoint/status combination.
// Health check and metrics endpoints are automatically skipped.
func HTTPMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics for health check and metrics endpoints
		if commonmetrics.ShouldSkipEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request is processed
		duration := time.Since(start)
		endpoint := commonmetrics.NormalizeEndpoint(c.FullPath())
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		m.RecordHTTPRequest(c.Request.Method, endpoint, c.Writer.Status(), duration)
	}
}
