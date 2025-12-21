// Package ratelimit provides rate limiting functionality for Go services.
// It supports Redis-based distributed rate limiting with sliding window algorithm.
package ratelimit

import (
	"context"
)

// RateLimiter defines the interface for rate limiting implementations.
type RateLimiter interface {
	// Allow checks if a request with the given key is allowed.
	// Returns true if allowed, false if rate limited.
	// The error is returned only for infrastructure failures (e.g., Redis down).
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if n requests with the given key are allowed.
	// Returns true if all n requests are allowed, false otherwise.
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// GetRemaining returns the remaining requests allowed for the given key.
	// Returns -1 if the key doesn't exist or on error.
	GetRemaining(ctx context.Context, key string) (int, error)

	// Reset clears the rate limit for the given key.
	Reset(ctx context.Context, key string) error
}

// Result contains detailed rate limiting information.
type Result struct {
	// Allowed indicates if the request was allowed.
	Allowed bool

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// ResetAfter is the duration until the rate limit resets.
	ResetAfterMs int64

	// RetryAfter is the duration to wait before retrying (only set when denied).
	RetryAfterMs int64
}

// RateLimiterWithInfo provides detailed rate limiting information.
type RateLimiterWithInfo interface {
	RateLimiter

	// AllowWithInfo checks if a request is allowed and returns detailed info.
	AllowWithInfo(ctx context.Context, key string) (*Result, error)
}
