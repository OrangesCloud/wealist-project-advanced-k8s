// Package otel provides OpenTelemetry integration utilities.
// This file contains logger helpers for trace context correlation.
package otel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TraceFields extracts trace context from context and returns zap fields.
// Use this to add trace_id and span_id to log entries for correlation.
func TraceFields(ctx context.Context) []zap.Field {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	return []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
		zap.Bool("trace_sampled", spanCtx.IsSampled()),
	}
}

// WithTraceContext creates a new logger with trace context fields.
// This is the primary way to add trace correlation to logs.
//
// Example:
//
//	func (h *Handler) CreateBoard(c *gin.Context) {
//	    logger := otel.WithTraceContext(c.Request.Context(), h.logger)
//	    logger.Info("Creating board")
//	}
func WithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	fields := TraceFields(ctx)
	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields...)
}

// InjectTraceHeaders injects trace context headers into an HTTP request.
// Use this when making outbound HTTP calls to propagate trace context.
//
// Example:
//
//	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
//	otel.InjectTraceHeaders(ctx, req)
//	resp, err := client.Do(req)
func InjectTraceHeaders(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// ExtractTraceContext extracts trace context from HTTP request headers.
// Use this in middleware to continue an existing trace.
//
// Example:
//
//	func TracingMiddleware() gin.HandlerFunc {
//	    return func(c *gin.Context) {
//	        ctx := otel.ExtractTraceContext(c.Request.Context(), c.Request)
//	        c.Request = c.Request.WithContext(ctx)
//	        c.Next()
//	    }
//	}
func ExtractTraceContext(ctx context.Context, req *http.Request) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))
}

// StartSpan starts a new span and returns the context with the span.
// The caller should defer span.End() to ensure the span is properly closed.
//
// Example:
//
//	ctx, span := otel.StartSpan(ctx, "board-service", "CreateBoard")
//	defer span.End()
func StartSpan(ctx context.Context, serviceName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	return tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext returns the current span from context.
// Returns a no-op span if no span is found.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// RecordError records an error on the current span.
// Use this to add error information to traces.
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
}

// SetSpanStatus sets the status of the current span.
// Use codes.Error for errors and codes.Ok for success.
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// SetSpanError marks the span as error and records the error.
// This is a convenience function combining RecordError and SetStatus.
func SetSpanError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanOk marks the span as successful.
func SetSpanOk(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Ok, "")
}

// AddSpanEvent adds an event to the current span.
// Events are useful for marking significant points in a span's lifecycle.
func AddSpanEvent(ctx context.Context, name string, attrs ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, attrs...)
}

// GetTraceID returns the trace ID from the context, or empty string if not available.
func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}

// GetSpanID returns the span ID from the context, or empty string if not available.
func GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.SpanID().String()
}
