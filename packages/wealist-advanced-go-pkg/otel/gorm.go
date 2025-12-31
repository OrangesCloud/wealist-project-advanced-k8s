// Package otel provides OpenTelemetry integration utilities.
// This file contains GORM database tracing helpers.
package otel

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// GORMTracingConfig holds configuration options for GORM tracing.
type GORMTracingConfig struct {
	// DBName is the database name to include in span attributes.
	DBName string
	// WithMetrics enables metrics collection (disabled by default for simplicity).
	WithMetrics bool
	// WithDBStatement includes SQL statements in spans (disabled for security).
	WithDBStatement bool
}

// DefaultGORMTracingConfig returns a default GORM tracing configuration.
func DefaultGORMTracingConfig(dbName string) *GORMTracingConfig {
	return &GORMTracingConfig{
		DBName:          dbName,
		WithMetrics:     false,
		WithDBStatement: false, // Disabled for security (SQL may contain sensitive data)
	}
}

// EnableGORMTracing enables OpenTelemetry tracing for GORM database operations.
// This wraps every DB query with a span for distributed tracing.
//
// Example:
//
//	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
//	if err := otel.EnableGORMTracing(db, "wealist"); err != nil {
//	    log.Warn("Failed to enable GORM tracing", zap.Error(err))
//	}
func EnableGORMTracing(db *gorm.DB, dbName string) error {
	cfg := DefaultGORMTracingConfig(dbName)
	return EnableGORMTracingWithConfig(db, cfg)
}

// EnableGORMTracingWithConfig enables GORM tracing with custom configuration.
func EnableGORMTracingWithConfig(db *gorm.DB, cfg *GORMTracingConfig) error {
	opts := []tracing.Option{
		tracing.WithDBName(cfg.DBName),
	}

	if !cfg.WithMetrics {
		opts = append(opts, tracing.WithoutMetrics())
	}

	if !cfg.WithDBStatement {
		opts = append(opts, tracing.WithoutQueryVariables())
	}

	return db.Use(tracing.NewPlugin(opts...))
}
