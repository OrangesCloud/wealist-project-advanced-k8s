// Package otel provides OpenTelemetry SDK initialization utilities for wealist services.
// It sets up traces, logs, and metrics exporters with OTLP protocol support.
package otel

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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
	OTLPEndpoint   string  // e.g., "otel-collector:4317"
	SamplingRatio  float64 // 0.0 to 1.0, default 1.0
	Enabled        bool
}

// DefaultConfig returns a default configuration with values from environment variables.
func DefaultConfig(serviceName string) *Config {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
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
// Uses resource.New() instead of Merge to avoid Schema URL conflicts between
// resource.Default() and semconv.SchemaURL.
func newResource(cfg *Config) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
		// Include standard resource detectors
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
	)
}

// newPropagator creates a propagator for W3C Trace Context and Baggage.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTraceProvider creates a new trace provider with OTLP exporter.
func newTraceProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
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

// newLoggerProvider creates a new logger provider with OTLP exporter.
func newLoggerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*log.LoggerProvider, error) {
	// Create OTLP log exporter
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlploggrpc.WithInsecure(),
	)
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
