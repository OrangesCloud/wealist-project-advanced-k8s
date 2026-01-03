package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the ops service
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveUsers     prometheus.Gauge
	AuditLogsTotal  prometheus.Counter
}

// New creates new metrics
func New() *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ops_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ops_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		ActiveUsers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "ops_active_users",
				Help: "Number of active portal users",
			},
		),
		AuditLogsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "ops_audit_logs_total",
				Help: "Total number of audit log entries created",
			},
		),
	}
}

// HTTPMiddleware creates HTTP metrics middleware
func HTTPMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		m.RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
