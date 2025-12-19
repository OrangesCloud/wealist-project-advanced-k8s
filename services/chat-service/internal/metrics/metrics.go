// Package metrics provides Prometheus metrics for chat-service.
//
// This package extends the common metrics package with business-specific metrics
// for tracking chat operations, messages, and WebSocket connections.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const namespace = "chat_service"

// Metrics holds all application metrics for chat-service.
type Metrics struct {
	// Embedded common metrics for HTTP requests, database operations, etc.
	*commonmetrics.Metrics

	// ChatsTotal tracks the current number of chats.
	ChatsTotal prometheus.Gauge
	// MessagesTotal tracks the total number of messages.
	MessagesTotal prometheus.Gauge
	// OnlineUsersTotal tracks the current number of online users.
	OnlineUsersTotal prometheus.Gauge

	// WebSocketConnectionsActive tracks active WebSocket connections.
	WebSocketConnectionsActive prometheus.Gauge
	// MessagesSentTotal counts messages sent.
	MessagesSentTotal prometheus.Counter
	// MessagesReadTotal counts messages read.
	MessagesReadTotal prometheus.Counter

	// WebSocketMessagesSent counts WebSocket messages sent.
	WebSocketMessagesSent prometheus.Counter
	// WebSocketMessagesReceived counts WebSocket messages received.
	WebSocketMessagesReceived prometheus.Counter
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

		ChatsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "chats_total",
				Help:      "Total number of chats",
			},
		),
		MessagesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "messages_total",
				Help:      "Total number of messages",
			},
		),
		OnlineUsersTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "online_users_total",
				Help:      "Total number of online users",
			},
		),
		WebSocketConnectionsActive: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "websocket_connections_active",
				Help:      "Number of active WebSocket connections",
			},
		),
		MessagesSentTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_sent_total",
				Help:      "Total number of messages sent",
			},
		),
		MessagesReadTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_read_total",
				Help:      "Total number of messages read",
			},
		),
		WebSocketMessagesSent: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_messages_sent_total",
				Help:      "Total number of WebSocket messages sent",
			},
		),
		WebSocketMessagesReceived: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_messages_received_total",
				Help:      "Total number of WebSocket messages received",
			},
		),
	}
}

// NewForTest creates metrics with an isolated registry for testing.
func NewForTest() *Metrics {
	return NewWithRegistry(prometheus.NewRegistry())
}

// RecordMessageSent increments message sent counter.
func (m *Metrics) RecordMessageSent() {
	m.MessagesSentTotal.Inc()
}

// RecordMessageRead increments message read counter.
func (m *Metrics) RecordMessageRead() {
	m.MessagesReadTotal.Inc()
}

// RecordWebSocketMessageSent increments WebSocket message sent counter.
func (m *Metrics) RecordWebSocketMessageSent() {
	m.WebSocketMessagesSent.Inc()
}

// RecordWebSocketMessageReceived increments WebSocket message received counter.
func (m *Metrics) RecordWebSocketMessageReceived() {
	m.WebSocketMessagesReceived.Inc()
}

// IncrementWebSocketConnections increments active WebSocket connections.
func (m *Metrics) IncrementWebSocketConnections() {
	m.WebSocketConnectionsActive.Inc()
}

// DecrementWebSocketConnections decrements active WebSocket connections.
func (m *Metrics) DecrementWebSocketConnections() {
	m.WebSocketConnectionsActive.Dec()
}

// SetChatsTotal sets the total number of chats.
func (m *Metrics) SetChatsTotal(count int64) {
	m.ChatsTotal.Set(float64(count))
}

// SetMessagesTotal sets the total number of messages.
func (m *Metrics) SetMessagesTotal(count int64) {
	m.MessagesTotal.Set(float64(count))
}

// SetOnlineUsersTotal sets the total number of online users.
func (m *Metrics) SetOnlineUsersTotal(count int64) {
	m.OnlineUsersTotal.Set(float64(count))
}

// RecordHTTPRequest delegates to the embedded common metrics.
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.Metrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
}
