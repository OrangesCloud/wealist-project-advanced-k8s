// Package otel provides OpenTelemetry SDK initialization utilities for wealist services.
// It sets up traces, logs, and metrics exporters with OTLP protocol support.
//
// Protocol: HTTP/protobuf (recommended by OpenTelemetry spec)
// - Endpoint format: "http://otel-collector:4318" (with scheme)
// - Compatible with all languages (Go, Java, Python, etc.)
// - Firewall-friendly and easier to debug
package otel

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration options.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string  // e.g., "http://otel-collector:4318" (HTTP/protobuf)
	SamplingRatio  float64 // 0.0 to 1.0, default 1.0
	Enabled        bool
}

// DefaultConfig returns a default configuration with values from environment variables.
func DefaultConfig(serviceName string) *Config {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	enabled := os.Getenv("OTEL_SDK_DISABLED") != "true"

	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnvOrDefault("DEPLOYMENT_ENV", "development"),
		OTLPEndpoint:   endpoint,
		SamplingRatio:  1.0,
		Enabled:        enabled,
	}
}

// InitProvider initializes the OpenTelemetry SDK with traces and logs.
// It returns a shutdown function that should be called when the application exits.
func InitProvider(ctx context.Context, cfg *Config) (shutdown func(context.Context) error, err error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if !cfg.Enabled {
		// Return a no-op shutdown function when disabled
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Create resource with service information
	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	// Set up propagator (W3C Trace Context + Baggage)
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider
	tracerProvider, err := newTraceProvider(ctx, cfg, res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up logger provider
	loggerProvider, err := newLoggerProvider(ctx, cfg, res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return shutdown, nil
}

// newResource creates an OpenTelemetry resource with service metadata.
// Uses resource.NewSchemaless() to avoid Schema URL conflicts entirely.
// The standard detectors (WithHost, WithOS, etc.) each use their own schema URLs
// which conflict when merged. Using schemaless resource avoids this issue.
func newResource(cfg *Config) (*resource.Resource, error) {
	return resource.NewSchemaless(
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentName(cfg.Environment),
		semconv.TelemetrySDKName("opentelemetry"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersion("1.32.0"),
	), nil
}

// newPropagator creates a propagator for W3C Trace Context and Baggage.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTraceProvider creates a new trace provider with OTLP HTTP exporter.
// Uses environment variables for endpoint configuration:
// - OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4318" (SDK parses scheme)
// - OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
func newTraceProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// Let the SDK use OTEL_EXPORTER_OTLP_ENDPOINT environment variable directly.
	// The SDK correctly parses the scheme (http:// vs https://) and sets insecure mode.
	// Passing WithEndpoint() would override the env var and require manual scheme handling,
	// which can cause "too many colons in address" errors.
	// See: https://github.com/open-telemetry/opentelemetry-go/issues/5706
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	// Configure sampler
	var sampler sdktrace.Sampler
	if cfg.SamplingRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SamplingRatio <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRatio))
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return traceProvider, nil
}

// newLoggerProvider creates a new logger provider with OTLP HTTP exporter.
// Uses environment variables for endpoint configuration (same as trace provider).
func newLoggerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*log.LoggerProvider, error) {
	// Let the SDK use OTEL_EXPORTER_OTLP_ENDPOINT environment variable directly.
	// See newTraceProvider for details on why we don't use WithEndpoint().
	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)

	return loggerProvider, nil
}

// getEnvOrDefault returns the environment variable value or the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Tracer returns a tracer for the given service name.
// This is a convenience wrapper around otel.Tracer.
func Tracer(serviceName string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(serviceName)
}
