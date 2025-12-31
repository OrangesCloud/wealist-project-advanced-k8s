// Package otel provides OpenTelemetry integration utilities.
// This file contains Redis tracing helpers.
package otel

import (
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// RedisTracingConfig holds configuration options for Redis tracing.
type RedisTracingConfig struct {
	// ServiceName is the service name to include in span attributes.
	ServiceName string
	// WithDBStatement includes Redis commands in spans (disabled for security).
	WithDBStatement bool
}

// DefaultRedisTracingConfig returns a default Redis tracing configuration.
func DefaultRedisTracingConfig(serviceName string) *RedisTracingConfig {
	return &RedisTracingConfig{
		ServiceName:     serviceName,
		WithDBStatement: false, // Disabled for security
	}
}

// EnableRedisTracing enables OpenTelemetry tracing for Redis operations.
// This wraps every Redis command with a span for distributed tracing.
//
// Example:
//
//	client := redis.NewClient(&redis.Options{...})
//	if err := otel.EnableRedisTracing(client); err != nil {
//	    log.Warn("Failed to enable Redis tracing", zap.Error(err))
//	}
func EnableRedisTracing(client *redis.Client) error {
	return redisotel.InstrumentTracing(client)
}

// EnableRedisTracingWithConfig enables Redis tracing with custom configuration.
func EnableRedisTracingWithConfig(client *redis.Client, cfg *RedisTracingConfig) error {
	opts := []redisotel.TracingOption{}

	if cfg.WithDBStatement {
		opts = append(opts, redisotel.WithDBStatement(true))
	}

	return redisotel.InstrumentTracing(client, opts...)
}

// EnableRedisClusterTracing enables OpenTelemetry tracing for Redis Cluster operations.
func EnableRedisClusterTracing(client *redis.ClusterClient) error {
	return redisotel.InstrumentTracing(client)
}
