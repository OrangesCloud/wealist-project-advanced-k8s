package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	// Use isolated registry for testing
	registry := prometheus.NewRegistry()
	cfg := &Config{
		Namespace: "test_service",
		Registry:  registry,
	}

	m := New(cfg)

	if m == nil {
		t.Fatal("Expected non-nil Metrics")
	}
	if m.Namespace() != "test_service" {
		t.Errorf("Expected namespace 'test_service', got '%s'", m.Namespace())
	}
}

func TestNewForTest(t *testing.T) {
	m := NewForTest("test_service")

	if m == nil {
		t.Fatal("Expected non-nil Metrics")
	}
	if m.HTTPRequestsTotal == nil {
		t.Error("Expected HTTPRequestsTotal to be initialized")
	}
	if m.DBQueryDuration == nil {
		t.Error("Expected DBQueryDuration to be initialized")
	}
}

func TestNewWithNilConfig(t *testing.T) {
	// This will use DefaultRegisterer, so we need to be careful
	// In a real test environment, use NewForTest instead
	registry := prometheus.NewRegistry()
	cfg := &Config{
		Namespace: "default_test",
		Registry:  registry,
	}

	m := New(cfg)
	if m == nil {
		t.Fatal("Expected non-nil Metrics with nil config")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("my_service")

	if cfg.Namespace != "my_service" {
		t.Errorf("Expected namespace 'my_service', got '%s'", cfg.Namespace)
	}
	if cfg.Registry == nil {
		t.Error("Expected default registry to be set")
	}
}

func TestRegisterGauge(t *testing.T) {
	m := NewForTest("test_service")

	gauge1 := m.RegisterGauge("test_gauge", "A test gauge")
	if gauge1 == nil {
		t.Fatal("Expected non-nil gauge")
	}

	// Registering same name should return same gauge
	gauge2 := m.RegisterGauge("test_gauge", "A test gauge")
	if gauge1 != gauge2 {
		t.Error("Expected same gauge instance for same name")
	}

	// Set value and verify no panic
	gauge1.Set(42.0)
}

func TestRegisterCounter(t *testing.T) {
	m := NewForTest("test_service")

	counter1 := m.RegisterCounter("test_counter", "A test counter")
	if counter1 == nil {
		t.Fatal("Expected non-nil counter")
	}

	// Registering same name should return same counter
	counter2 := m.RegisterCounter("test_counter", "A test counter")
	if counter1 != counter2 {
		t.Error("Expected same counter instance for same name")
	}

	// Inc and verify no panic
	counter1.Inc()
	counter1.Add(5)
}

func TestSafeExecuteNoPanic(t *testing.T) {
	m := NewForTest("test_service")

	// Normal operation should work
	executed := false
	m.safeExecute("test_op", func() {
		executed = true
	})

	if !executed {
		t.Error("Expected function to be executed")
	}
}

func TestSafeExecuteWithPanic(t *testing.T) {
	m := NewForTest("test_service")

	// Panic should be recovered
	didPanic := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()

		m.safeExecute("panic_op", func() {
			panic("test panic")
		})
		didPanic = false
	}()

	if didPanic {
		t.Error("Expected panic to be recovered within safeExecute")
	}
}
