package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestProperty18_MetricHelpDescription(t *testing.T) {
	// Create a new registry to collect all metrics
	registry := prometheus.NewRegistry()

	// Use the factory method to create metrics with proper registration
	m := NewWithRegistry(registry, nil)

	// Register all collectors from both common metrics and business metrics
	collectors := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.DBConnectionsOpen,
		m.DBConnectionsInUse,
		m.DBConnectionsIdle,
		m.DBConnectionsMax,
		m.DBConnectionWaitTotal,
		m.DBConnectionWaitDuration,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.ExternalAPIRequestDuration,
		m.ExternalAPIRequestsTotal,
		m.ExternalAPIErrors,
		m.ProjectsTotal,
		m.BoardsTotal,
		m.ProjectCreatedTotal,
		m.BoardCreatedTotal,
	}

	// Verify all collectors are non-nil
	for i, collector := range collectors {
		if collector == nil {
			t.Errorf("Collector at index %d is nil", i)
		}
	}

	// Gather metrics from the registry
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check each metric has a non-empty help description
	for _, mf := range metricFamilies {
		name := mf.GetName()
		help := mf.GetHelp()

		if help == "" {
			t.Errorf("Metric '%s' has an empty help description", name)
		}

		if len(strings.TrimSpace(help)) == 0 {
			t.Errorf("Metric '%s' has a help description with only whitespace", name)
		}
	}
}
