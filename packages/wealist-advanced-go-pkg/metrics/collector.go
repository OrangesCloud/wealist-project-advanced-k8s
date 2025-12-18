// Package metrics provides common Prometheus metrics utilities for Go microservices.
package metrics

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MetricsCollector defines the interface for periodic metrics collection.
// Implement this interface to collect service-specific business metrics.
type MetricsCollector interface {
	// Collect gathers metrics. Called periodically by the collector.
	// Context is provided for timeout handling.
	Collect(ctx context.Context) error
}

// PeriodicCollector runs metric collection at regular intervals.
type PeriodicCollector struct {
	collectors []MetricsCollector
	interval   time.Duration
	timeout    time.Duration
	logger     *zap.Logger

	ticker *time.Ticker
	done   chan struct{}
	wg     sync.WaitGroup
}

// CollectorConfig holds configuration for PeriodicCollector.
type CollectorConfig struct {
	// Interval between collections (default: 60s)
	Interval time.Duration
	// Timeout for each collection (default: 5s)
	Timeout time.Duration
	// Logger for error reporting
	Logger *zap.Logger
}

// DefaultCollectorConfig returns default collector configuration.
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		Interval: 60 * time.Second,
		Timeout:  5 * time.Second,
		Logger:   nil,
	}
}

// NewPeriodicCollector creates a new periodic metrics collector.
func NewPeriodicCollector(cfg *CollectorConfig, collectors ...MetricsCollector) *PeriodicCollector {
	if cfg == nil {
		cfg = DefaultCollectorConfig()
	}

	if cfg.Interval == 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewProduction()
	}

	return &PeriodicCollector{
		collectors: collectors,
		interval:   cfg.Interval,
		timeout:    cfg.Timeout,
		logger:     cfg.Logger,
		done:       make(chan struct{}),
	}
}

// AddCollector adds a new collector to the periodic collector.
func (p *PeriodicCollector) AddCollector(c MetricsCollector) {
	p.collectors = append(p.collectors, c)
}

// Start begins the periodic collection goroutine.
func (p *PeriodicCollector) Start() {
	p.ticker = time.NewTicker(p.interval)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		// Collect immediately on start
		p.collectAll()

		for {
			select {
			case <-p.ticker.C:
				p.collectAll()
			case <-p.done:
				return
			}
		}
	}()
}

// Stop stops the periodic collection.
func (p *PeriodicCollector) Stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)
	p.wg.Wait()
}

// collectAll runs all registered collectors with panic recovery.
func (p *PeriodicCollector) collectAll() {
	for _, collector := range p.collectors {
		p.collectOne(collector)
	}
}

// collectOne runs a single collector with timeout and panic recovery.
func (p *PeriodicCollector) collectOne(collector MetricsCollector) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Panic in metrics collection",
				zap.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	if err := collector.Collect(ctx); err != nil {
		p.logger.Error("Failed to collect metrics",
			zap.Error(err),
		)
	}
}

// SimpleCollector wraps a function as a MetricsCollector.
type SimpleCollector struct {
	collectFunc func(ctx context.Context) error
}

// NewSimpleCollector creates a collector from a function.
func NewSimpleCollector(fn func(ctx context.Context) error) *SimpleCollector {
	return &SimpleCollector{collectFunc: fn}
}

// Collect implements MetricsCollector interface.
func (s *SimpleCollector) Collect(ctx context.Context) error {
	if s.collectFunc != nil {
		return s.collectFunc(ctx)
	}
	return nil
}
