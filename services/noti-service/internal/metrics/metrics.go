// Package metrics provides Prometheus metrics for noti-service.
//
// This package extends the common metrics package with business-specific metrics
// for tracking notification operations, SSE connections, and delivery rates.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const namespace = "noti_service"

// Metrics holds all application metrics for noti-service.
type Metrics struct {
	// Embedded common metrics for HTTP requests, database operations, etc.
	*commonmetrics.Metrics

	// NotificationsCreatedTotal counts notification creation operations.
	NotificationsCreatedTotal prometheus.Counter
	// NotificationsDeliveredTotal counts successfully delivered notifications.
	NotificationsDeliveredTotal prometheus.Counter
	// NotificationsReadTotal counts notifications marked as read.
	NotificationsReadTotal prometheus.Counter
	// NotificationsDeletedTotal counts deleted notifications.
	NotificationsDeletedTotal prometheus.Counter

	// SSEConnectionsTotal tracks the current number of active SSE connections.
	SSEConnectionsTotal prometheus.Gauge
	// SSEConnectionsCreatedTotal counts SSE connection open events.
	SSEConnectionsCreatedTotal prometheus.Counter
	// SSEConnectionsClosedTotal counts SSE connection close events.
	SSEConnectionsClosedTotal prometheus.Counter

	// NotificationDeliveryDuration tracks notification delivery latency.
	NotificationDeliveryDuration prometheus.Histogram
}

// New creates and registers all metrics with the default Prometheus registerer.
func New() *Metrics {
	return NewWithRegistry(prometheus.DefaultRegisterer)
}

// NewWithRegistry creates metrics with a custom registry.
func NewWithRegistry(registerer prometheus.Registerer) *Metrics {
	cfg := &commonmetrics.Config{
		Namespace: namespace,
		Registry:  registerer,
	}

	factory := promauto.With(registerer)

	return &Metrics{
		Metrics: commonmetrics.New(cfg),

		NotificationsCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "notifications_created_total",
				Help:      "Total number of notifications created",
			},
		),
		NotificationsDeliveredTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "notifications_delivered_total",
				Help:      "Total number of notifications delivered via SSE",
			},
		),
		NotificationsReadTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "notifications_read_total",
				Help:      "Total number of notifications marked as read",
			},
		),
		NotificationsDeletedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "notifications_deleted_total",
				Help:      "Total number of notifications deleted",
			},
		),
		SSEConnectionsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "sse_connections_active_total",
				Help:      "Current number of active SSE connections",
			},
		),
		SSEConnectionsCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sse_connections_created_total",
				Help:      "Total number of SSE connections opened",
			},
		),
		SSEConnectionsClosedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sse_connections_closed_total",
				Help:      "Total number of SSE connections closed",
			},
		),
		NotificationDeliveryDuration: factory.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "notification_delivery_duration_seconds",
				Help:      "Time taken to deliver notifications via SSE",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
		),
	}
}

// NewForTest creates metrics with an isolated registry for testing.
func NewForTest() *Metrics {
	return NewWithRegistry(prometheus.NewRegistry())
}

// RecordNotificationCreated increments the notification creation counter.
func (m *Metrics) RecordNotificationCreated() {
	m.NotificationsCreatedTotal.Inc()
}

// RecordNotificationDelivered increments the notification delivery counter.
func (m *Metrics) RecordNotificationDelivered() {
	m.NotificationsDeliveredTotal.Inc()
}

// RecordNotificationRead increments the notification read counter.
func (m *Metrics) RecordNotificationRead() {
	m.NotificationsReadTotal.Inc()
}

// RecordNotificationDeleted increments the notification deletion counter.
func (m *Metrics) RecordNotificationDeleted() {
	m.NotificationsDeletedTotal.Inc()
}

// RecordSSEConnectionOpened increments SSE connection counters.
func (m *Metrics) RecordSSEConnectionOpened() {
	m.SSEConnectionsCreatedTotal.Inc()
	m.SSEConnectionsTotal.Inc()
}

// RecordSSEConnectionClosed decrements active SSE connections.
func (m *Metrics) RecordSSEConnectionClosed() {
	m.SSEConnectionsClosedTotal.Inc()
	m.SSEConnectionsTotal.Dec()
}

// SetSSEConnectionsTotal sets the total number of active SSE connections.
func (m *Metrics) SetSSEConnectionsTotal(count int64) {
	m.SSEConnectionsTotal.Set(float64(count))
}

// RecordNotificationDeliveryDuration records the time taken to deliver a notification.
func (m *Metrics) RecordNotificationDeliveryDuration(duration time.Duration) {
	m.NotificationDeliveryDuration.Observe(duration.Seconds())
}

// RecordHTTPRequest delegates to the embedded common metrics.
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.Metrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
}
