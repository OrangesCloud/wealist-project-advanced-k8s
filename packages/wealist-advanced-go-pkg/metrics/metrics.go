// Package metrics provides common Prometheus metrics utilities for Go microservices.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// Config holds configuration for metrics initialization
type Config struct {
	// Namespace is the metric namespace (e.g., "board_service", "user_service")
	Namespace string
	// Logger for error reporting
	Logger *zap.Logger
	// Registry is the prometheus registry to use (nil for default)
	Registry prometheus.Registerer
}

// DefaultConfig returns default metrics configuration
func DefaultConfig(namespace string) *Config {
	return &Config{
		Namespace: namespace,
		Logger:    nil,
		Registry:  prometheus.DefaultRegisterer,
	}
}

// Metrics holds all application metrics
type Metrics struct {
	namespace string
	factory   promauto.Factory
	logger    *zap.Logger

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Database metrics
	DBConnectionsOpen        prometheus.Gauge
	DBConnectionsInUse       prometheus.Gauge
	DBConnectionsIdle        prometheus.Gauge
	DBConnectionsMax         prometheus.Gauge
	DBConnectionWaitTotal    prometheus.Counter
	DBConnectionWaitDuration prometheus.Counter
	DBQueryDuration          *prometheus.HistogramVec
	DBQueryErrors            *prometheus.CounterVec

	// External API metrics
	ExternalAPIRequestDuration *prometheus.HistogramVec
	ExternalAPIRequestsTotal   *prometheus.CounterVec
	ExternalAPIErrors          *prometheus.CounterVec

	// Custom metrics storage for service-specific metrics
	customGauges   map[string]prometheus.Gauge
	customCounters map[string]prometheus.Counter
}

// New creates and registers all metrics with the given configuration
func New(cfg *Config) *Metrics {
	if cfg == nil {
		cfg = DefaultConfig("service")
	}

	registry := cfg.Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(registry)

	logger := cfg.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &Metrics{
		namespace: cfg.Namespace,
		factory:   factory,
		logger:    logger,

		// HTTP metrics
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),

		// Database connection pool metrics
		DBConnectionsOpen: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connections_open",
				Help:      "Current number of open database connections",
			},
		),
		DBConnectionsInUse: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connections_in_use",
				Help:      "Current number of in-use database connections",
			},
		),
		DBConnectionsIdle: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle database connections",
			},
		),
		DBConnectionsMax: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connections_max",
				Help:      "Maximum number of open database connections configured",
			},
		),
		DBConnectionWaitTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connection_wait_total",
				Help:      "Total number of times waited for a database connection",
			},
		),
		DBConnectionWaitDuration: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "db_connection_wait_duration_seconds_total",
				Help:      "Total duration waited for database connections in seconds",
			},
		),

		// Database query metrics
		DBQueryDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "table"},
		),

		// External API metrics
		ExternalAPIRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Name:      "external_api_request_duration_seconds",
				Help:      "External API request duration in seconds",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"endpoint", "status"},
		),
		ExternalAPIRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "external_api_requests_total",
				Help:      "Total number of external API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		ExternalAPIErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "external_api_errors_total",
				Help:      "Total number of external API errors",
			},
			[]string{"endpoint", "error_type"},
		),

		customGauges:   make(map[string]prometheus.Gauge),
		customCounters: make(map[string]prometheus.Counter),
	}
}

// NewForTest creates metrics with an isolated registry for testing.
// This prevents "duplicate metrics collector registration" errors in tests.
func NewForTest(namespace string) *Metrics {
	return New(&Config{
		Namespace: namespace,
		Registry:  prometheus.NewRegistry(),
	})
}

// safeExecute wraps metric operations with panic recovery
func (m *Metrics) safeExecute(operation string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic in metrics operation",
					zap.String("operation", operation),
					zap.Any("panic", r),
				)
			}
		}
	}()
	fn()
}

// RegisterGauge registers a custom gauge metric
func (m *Metrics) RegisterGauge(name, help string) prometheus.Gauge {
	if g, exists := m.customGauges[name]; exists {
		return g
	}

	gauge := m.factory.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Name:      name,
		Help:      help,
	})
	m.customGauges[name] = gauge
	return gauge
}

// RegisterCounter registers a custom counter metric
func (m *Metrics) RegisterCounter(name, help string) prometheus.Counter {
	if c, exists := m.customCounters[name]; exists {
		return c
	}

	counter := m.factory.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Name:      name,
		Help:      help,
	})
	m.customCounters[name] = counter
	return counter
}

// Namespace returns the metrics namespace
func (m *Metrics) Namespace() string {
	return m.namespace
}
