// Package metrics provides Prometheus metrics for video-service.
//
// This package extends the common metrics package with business-specific metrics
// for tracking video room operations, participants, and call durations.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const namespace = "video_service"

// Metrics holds all application metrics for video-service.
type Metrics struct {
	// Embedded common metrics for HTTP requests, database operations, etc.
	*commonmetrics.Metrics

	// RoomsTotal tracks the current number of active rooms.
	RoomsTotal prometheus.Gauge
	// ParticipantsTotal tracks the current number of participants across all rooms.
	ParticipantsTotal prometheus.Gauge

	// RoomsCreatedTotal counts room creation operations.
	RoomsCreatedTotal prometheus.Counter
	// RoomsEndedTotal counts room end operations.
	RoomsEndedTotal prometheus.Counter
	// ParticipantsJoinedTotal counts participant join operations.
	ParticipantsJoinedTotal prometheus.Counter
	// ParticipantsLeftTotal counts participant leave operations.
	ParticipantsLeftTotal prometheus.Counter

	// CallDurationSeconds tracks call durations.
	CallDurationSeconds prometheus.Histogram
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

		RoomsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "rooms_active_total",
				Help:      "Total number of active rooms",
			},
		),
		ParticipantsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "participants_active_total",
				Help:      "Total number of active participants across all rooms",
			},
		),
		RoomsCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rooms_created_total",
				Help:      "Total number of rooms created",
			},
		),
		RoomsEndedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rooms_ended_total",
				Help:      "Total number of rooms ended",
			},
		),
		ParticipantsJoinedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "participants_joined_total",
				Help:      "Total number of participants joined",
			},
		),
		ParticipantsLeftTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "participants_left_total",
				Help:      "Total number of participants left",
			},
		),
		CallDurationSeconds: factory.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "call_duration_seconds",
				Help:      "Duration of video calls in seconds",
				Buckets:   []float64{60, 300, 600, 1800, 3600, 7200}, // 1m, 5m, 10m, 30m, 1h, 2h
			},
		),
	}
}

// NewForTest creates metrics with an isolated registry for testing.
func NewForTest() *Metrics {
	return NewWithRegistry(prometheus.NewRegistry())
}

// RecordRoomCreated increments room creation counter.
func (m *Metrics) RecordRoomCreated() {
	m.RoomsCreatedTotal.Inc()
	m.RoomsTotal.Inc()
}

// RecordRoomEnded increments room end counter.
func (m *Metrics) RecordRoomEnded() {
	m.RoomsEndedTotal.Inc()
	m.RoomsTotal.Dec()
}

// RecordParticipantJoined increments participant join counter.
func (m *Metrics) RecordParticipantJoined() {
	m.ParticipantsJoinedTotal.Inc()
	m.ParticipantsTotal.Inc()
}

// RecordParticipantLeft increments participant leave counter.
func (m *Metrics) RecordParticipantLeft() {
	m.ParticipantsLeftTotal.Inc()
	m.ParticipantsTotal.Dec()
}

// RecordCallDuration records the duration of a video call.
func (m *Metrics) RecordCallDuration(duration time.Duration) {
	m.CallDurationSeconds.Observe(duration.Seconds())
}

// SetRoomsTotal sets the total number of active rooms.
func (m *Metrics) SetRoomsTotal(count int64) {
	m.RoomsTotal.Set(float64(count))
}

// SetParticipantsTotal sets the total number of active participants.
func (m *Metrics) SetParticipantsTotal(count int64) {
	m.ParticipantsTotal.Set(float64(count))
}

// RecordHTTPRequest delegates to the embedded common metrics.
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.Metrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
}
