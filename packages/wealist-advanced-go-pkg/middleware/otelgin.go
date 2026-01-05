// Package middleware provides HTTP middleware for Gin-based services.
// This file contains OpenTelemetry Gin instrumentation middleware.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// OTELTracing returns otelgin middleware for automatic HTTP span creation.
// It automatically instruments all HTTP handlers with OpenTelemetry tracing,
// creating spans with standard HTTP semantic conventions.
//
// The middleware filters out health check and metrics endpoints by default.
//
// Usage:
//
//	router.Use(middleware.OTELTracing("board-service"))
func OTELTracing(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName,
		otelgin.WithFilter(DefaultTracingFilter),
		otelgin.WithSpanNameFormatter(SpanNameFormatter),
	)
}

// OTELTracingWithOptions returns otelgin middleware with custom options.
// Use this when you need more control over the middleware behavior.
//
// Usage:
//
//	router.Use(middleware.OTELTracingWithOptions("board-service",
//	    otelgin.WithFilter(customFilter),
//	    otelgin.WithPropagators(customPropagator),
//	))
func OTELTracingWithOptions(serviceName string, opts ...otelgin.Option) gin.HandlerFunc {
	return otelgin.Middleware(serviceName, opts...)
}

// DefaultTracingFilter returns true for requests that should be traced.
// It filters out health check endpoints, metrics, and other noise.
func DefaultTracingFilter(r *http.Request) bool {
	path := r.URL.Path

	// Skip health checks (high frequency, low value for tracing)
	if path == "/health/live" || path == "/health/ready" || path == "/health" {
		return false
	}

	// Skip Prometheus metrics endpoint
	if path == "/metrics" {
		return false
	}

	// Skip Kubernetes probes
	if path == "/readyz" || path == "/livez" || path == "/healthz" {
		return false
	}

	return true
}

// SpanNameFormatter formats span names using HTTP method and route.
// Example: "GET /api/boards/:boardId"
func SpanNameFormatter(r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// TracingFilterFunc is a function type for filtering requests.
type TracingFilterFunc func(*http.Request) bool

// NewTracingFilterWithPaths creates a filter that skips specified paths.
// All paths are exact matches.
//
// Usage:
//
//	filter := middleware.NewTracingFilterWithPaths("/health", "/metrics", "/internal/status")
//	router.Use(middleware.OTELTracingWithOptions("my-service",
//	    otelgin.WithFilter(filter),
//	))
func NewTracingFilterWithPaths(skipPaths ...string) TracingFilterFunc {
	skipSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = true
	}

	return func(r *http.Request) bool {
		return !skipSet[r.URL.Path]
	}
}
