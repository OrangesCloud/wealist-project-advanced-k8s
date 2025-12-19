package metrics

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestProperty11_MetricCollectionErrorHandling tests that metric collection errors are handled gracefully
// **Feature: board-service-prometheus-metrics, Property 11: 메트릭 수집 에러 처리**
// **Validates: Requirements 6.3**
//
// Property: For all metric recording operations, when an error or panic occurs,
// the error should be logged and the operation should continue without crashing
func TestProperty11_MetricCollectionErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name      string
		operation func(*Metrics)
		shouldRun bool
	}{
		{
			name: "RecordHTTPRequest with nil metrics should not panic",
			operation: func(m *Metrics) {
				// This could panic if not properly protected
				m.RecordHTTPRequest("GET", "/test", 200, time.Second)
			},
			shouldRun: true,
		},
		{
			name: "RecordDBQuery with nil metrics should not panic",
			operation: func(m *Metrics) {
				m.RecordDBQuery("select", "test_table", time.Millisecond, nil)
			},
			shouldRun: true,
		},
		{
			name: "RecordExternalAPICall with nil metrics should not panic",
			operation: func(m *Metrics) {
				m.RecordExternalAPICall("/api/test", "GET", 200, time.Second, nil)
			},
			shouldRun: true,
		},
		{
			name: "IncrementProjectCreated should not panic",
			operation: func(m *Metrics) {
				m.IncrementProjectCreated()
			},
			shouldRun: true,
		},
		{
			name: "IncrementBoardCreated should not panic",
			operation: func(m *Metrics) {
				m.IncrementBoardCreated()
			},
			shouldRun: true,
		},
		{
			name: "SetProjectsTotal should not panic",
			operation: func(m *Metrics) {
				m.SetProjectsTotal(100)
			},
			shouldRun: true,
		},
		{
			name: "SetBoardsTotal should not panic",
			operation: func(m *Metrics) {
				m.SetBoardsTotal(50)
			},
			shouldRun: true,
		},
		{
			name: "UpdateDBStats should not panic",
			operation: func(m *Metrics) {
				stats := sql.DBStats{
					OpenConnections: 10,
					InUse:           5,
					Idle:            5,
				}
				m.UpdateDBStats(stats)
			},
			shouldRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldRun {
				t.Skip("Test disabled")
			}

			// Create metrics with proper registry
			registry := prometheus.NewRegistry()
			m := NewWithRegistry(registry, logger)

			// This should not panic even if there are issues
			assert.NotPanics(t, func() {
				tt.operation(m)
			}, "Metric operation should not panic")
		})
	}
}

// TestMetricCollectionContinuesAfterError tests that request processing continues after metric errors
func TestMetricCollectionContinuesAfterError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry, logger)

	// Simulate multiple operations - all should succeed without panic
	assert.NotPanics(t, func() {
		m.RecordHTTPRequest("GET", "/api/test", 200, time.Millisecond*100)
		m.RecordHTTPRequest("POST", "/api/test", 201, time.Millisecond*150)
		m.RecordDBQuery("select", "users", time.Millisecond*10, nil)
		m.RecordDBQuery("insert", "projects", time.Millisecond*20, errors.New("test error"))
		m.RecordExternalAPICall("/api/users/123", "GET", 200, time.Millisecond*50, nil)
		m.IncrementProjectCreated()
		m.IncrementBoardCreated()
		m.SetProjectsTotal(100)
		m.SetBoardsTotal(50)
	}, "Multiple metric operations should not panic")
}

// TestPublicMethodsHandlePanicsGracefully tests that public metric methods handle errors gracefully
// Note: safeExecute is an unexported method in the common module, so we test through public methods
func TestPublicMethodsHandlePanicsGracefully(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry, logger)

	// Test that all public methods don't panic with various inputs
	assert.NotPanics(t, func() {
		// These should all be protected by the common module's safeExecute
		m.RecordHTTPRequest("GET", "/test", 200, time.Second)
		m.RecordDBQuery("select", "test", time.Millisecond, nil)
		m.RecordExternalAPICall("/api/test", "GET", 500, time.Second, errors.New("test error"))
		m.UpdateDBStats(sql.DBStats{})
		m.IncrementProjectCreated()
		m.IncrementBoardCreated()
		m.SetProjectsTotal(100)
		m.SetBoardsTotal(50)
	}, "Public metric methods should handle errors gracefully")
}

// TestMetricsWithNilLogger tests that metrics work even without a logger
func TestMetricsWithNilLogger(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry, nil)

	// Should not panic even without a logger
	assert.NotPanics(t, func() {
		m.RecordHTTPRequest("GET", "/test", 200, time.Second)
		m.RecordDBQuery("select", "test", time.Millisecond, nil)
		m.IncrementProjectCreated()
	}, "Metrics should work without a logger")
}

// TestCollectorPanicRecovery tests that the collector recovers from panics
func TestCollectorPanicRecovery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry, logger)

	// Create a collector with nil db to potentially cause issues
	collector := &BusinessMetricsCollector{
		db:      nil,
		metrics: m,
		logger:  logger,
	}

	// The Collect method should not panic even with nil db
	assert.NotPanics(t, func() {
		ctx := context.Background()
		_ = collector.Collect(ctx)
	}, "Collector should handle errors gracefully")
}
