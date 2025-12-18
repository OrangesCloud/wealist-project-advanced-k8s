// Package metrics provides Prometheus metrics for user-service.
//
// This package extends the common metrics package with business-specific metrics
// for tracking users, workspaces, profiles, and join requests. It follows the
// Prometheus naming convention with the "user_service" namespace.
//
// Example usage:
//
//	m := metrics.New()
//	m.RecordUserCreated()
//	m.SetUsersTotal(100)
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const namespace = "user_service"

// Metrics holds all application metrics for user-service.
// It embeds the common Metrics struct for HTTP, database, and external API metrics,
// and adds business-specific metrics for the user-service domain.
type Metrics struct {
	// Embedded common metrics for HTTP requests, database operations, etc.
	*commonmetrics.Metrics

	// UsersTotal tracks the current number of active users in the system.
	UsersTotal prometheus.Gauge
	// WorkspacesTotal tracks the current number of workspaces.
	WorkspacesTotal prometheus.Gauge
	// ProfilesTotal tracks the current number of user profiles.
	ProfilesTotal prometheus.Gauge
	// UserCreatedTotal counts the total number of user creation events.
	UserCreatedTotal prometheus.Counter
	// WorkspaceCreatedTotal counts the total number of workspace creation events.
	WorkspaceCreatedTotal prometheus.Counter
	// ProfileCreatedTotal counts the total number of profile creation events.
	ProfileCreatedTotal prometheus.Counter
	// JoinRequestsTotal tracks the current number of pending join requests.
	JoinRequestsTotal prometheus.Gauge
}

// New creates and registers all metrics with the default Prometheus registerer.
// This should typically be called once during application startup.
func New() *Metrics {
	return NewWithRegistry(prometheus.DefaultRegisterer)
}

// NewWithRegistry creates metrics with a custom registry.
// This is useful for testing to avoid metric registration conflicts.
func NewWithRegistry(registerer prometheus.Registerer) *Metrics {
	cfg := &commonmetrics.Config{
		Namespace: namespace,
		Registry:  registerer,
	}

	factory := promauto.With(registerer)

	return &Metrics{
		Metrics: commonmetrics.New(cfg),

		// Business metrics
		UsersTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "users_total",
				Help:      "Total number of active users",
			},
		),
		WorkspacesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "workspaces_total",
				Help:      "Total number of workspaces",
			},
		),
		ProfilesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "profiles_total",
				Help:      "Total number of user profiles",
			},
		),
		UserCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "user_created_total",
				Help:      "Total number of user creation events",
			},
		),
		WorkspaceCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workspace_created_total",
				Help:      "Total number of workspace creation events",
			},
		),
		ProfileCreatedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "profile_created_total",
				Help:      "Total number of profile creation events",
			},
		),
		JoinRequestsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "join_requests_pending_total",
				Help:      "Total number of pending join requests",
			},
		),
	}
}

// NewForTest creates metrics with an isolated registry for testing
func NewForTest() *Metrics {
	return NewWithRegistry(prometheus.NewRegistry())
}

// RecordUserCreated increments user creation counter
func (m *Metrics) RecordUserCreated() {
	m.UserCreatedTotal.Inc()
}

// RecordWorkspaceCreated increments workspace creation counter
func (m *Metrics) RecordWorkspaceCreated() {
	m.WorkspaceCreatedTotal.Inc()
}

// RecordProfileCreated increments profile creation counter
func (m *Metrics) RecordProfileCreated() {
	m.ProfileCreatedTotal.Inc()
}

// SetUsersTotal sets the total number of users
func (m *Metrics) SetUsersTotal(count int64) {
	m.UsersTotal.Set(float64(count))
}

// SetWorkspacesTotal sets the total number of workspaces
func (m *Metrics) SetWorkspacesTotal(count int64) {
	m.WorkspacesTotal.Set(float64(count))
}

// SetProfilesTotal sets the total number of profiles
func (m *Metrics) SetProfilesTotal(count int64) {
	m.ProfilesTotal.Set(float64(count))
}

// SetJoinRequestsTotal sets the number of pending join requests
func (m *Metrics) SetJoinRequestsTotal(count int64) {
	m.JoinRequestsTotal.Set(float64(count))
}
