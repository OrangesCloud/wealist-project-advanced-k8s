package ratelimit

import (
	"time"
)

// Config holds the configuration for rate limiting.
type Config struct {
	// Enabled determines if rate limiting is active.
	Enabled bool

	// RequestsPerMinute is the maximum number of requests allowed per minute.
	// This is the primary rate limit setting.
	RequestsPerMinute int

	// RequestsPerSecond is an optional secondary limit for burst control.
	// If set to 0, only RequestsPerMinute is used.
	RequestsPerSecond int

	// BurstSize allows temporary bursts above the rate limit.
	// Default is 0 (no burst allowed).
	BurstSize int

	// WindowSize is the sliding window duration.
	// Default is 1 minute.
	WindowSize time.Duration

	// KeyPrefix is the prefix for Redis keys.
	// Default is "rl:".
	KeyPrefix string

	// FailOpen determines behavior when Redis is unavailable.
	// If true, requests are allowed when Redis fails (fail-open).
	// If false, requests are denied when Redis fails (fail-closed).
	// Default is true (fail-open) for availability.
	FailOpen bool

	// ExcludePaths are paths that bypass rate limiting.
	// Supports exact match and prefix match with trailing *.
	ExcludePaths []string

	// EndpointLimits allows different limits per endpoint.
	// Key is the path pattern, value is requests per minute.
	EndpointLimits map[string]int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		RequestsPerMinute: 60,
		RequestsPerSecond: 0,
		BurstSize:         0,
		WindowSize:        time.Minute,
		KeyPrefix:         "rl:",
		FailOpen:          true,
		ExcludePaths: []string{
			"/health/*",
			"/metrics",
			"/ready",
			"/live",
		},
		EndpointLimits: make(map[string]int),
	}
}

// WithRequestsPerMinute sets the requests per minute limit.
func (c Config) WithRequestsPerMinute(rpm int) Config {
	c.RequestsPerMinute = rpm
	return c
}

// WithRequestsPerSecond sets the requests per second limit.
func (c Config) WithRequestsPerSecond(rps int) Config {
	c.RequestsPerSecond = rps
	return c
}

// WithBurstSize sets the burst size.
func (c Config) WithBurstSize(burst int) Config {
	c.BurstSize = burst
	return c
}

// WithWindowSize sets the sliding window size.
func (c Config) WithWindowSize(d time.Duration) Config {
	c.WindowSize = d
	return c
}

// WithKeyPrefix sets the Redis key prefix.
func (c Config) WithKeyPrefix(prefix string) Config {
	c.KeyPrefix = prefix
	return c
}

// WithFailOpen sets the fail-open behavior.
func (c Config) WithFailOpen(failOpen bool) Config {
	c.FailOpen = failOpen
	return c
}

// WithExcludePaths sets paths to exclude from rate limiting.
func (c Config) WithExcludePaths(paths []string) Config {
	c.ExcludePaths = paths
	return c
}

// WithEndpointLimit adds a custom limit for a specific endpoint.
func (c Config) WithEndpointLimit(path string, rpm int) Config {
	if c.EndpointLimits == nil {
		c.EndpointLimits = make(map[string]int)
	}
	c.EndpointLimits[path] = rpm
	return c
}

// GetLimitForPath returns the rate limit for a specific path.
// Returns the endpoint-specific limit if configured, otherwise the default.
func (c Config) GetLimitForPath(path string) int {
	if limit, ok := c.EndpointLimits[path]; ok {
		return limit
	}
	return c.RequestsPerMinute
}

// IsExcluded checks if a path should be excluded from rate limiting.
func (c Config) IsExcluded(path string) bool {
	for _, excluded := range c.ExcludePaths {
		if matchPath(excluded, path) {
			return true
		}
	}
	return false
}

// matchPath checks if a path matches a pattern.
// Supports exact match and prefix match with trailing *.
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// Check for wildcard prefix match
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	return false
}
