package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisRateLimiter implements RateLimiter using Redis with sliding window algorithm.
type RedisRateLimiter struct {
	client *redis.Client
	config Config
	logger *zap.Logger
}

// NewRedisRateLimiter creates a new Redis-based rate limiter.
func NewRedisRateLimiter(client *redis.Client, config Config, logger *zap.Logger) *RedisRateLimiter {
	if config.WindowSize == 0 {
		config.WindowSize = time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "rl:"
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RedisRateLimiter{
		client: client,
		config: config,
		logger: logger,
	}
}

// Allow checks if a single request is allowed for the given key.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed for the given key.
func (r *RedisRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	result, err := r.AllowWithInfo(ctx, key)
	if err != nil {
		return r.config.FailOpen, err
	}
	return result.Allowed, nil
}

// AllowWithInfo checks if a request is allowed and returns detailed rate limit info.
func (r *RedisRateLimiter) AllowWithInfo(ctx context.Context, key string) (*Result, error) {
	fullKey := r.config.KeyPrefix + key
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := r.config.WindowSize.Milliseconds()
	windowStart := nowMs - windowMs

	// Use Redis pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Remove entries outside the current window
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// Add current request timestamp
	pipe.ZAdd(ctx, fullKey, redis.Z{
		Score:  float64(nowMs),
		Member: fmt.Sprintf("%d", nowMs),
	})

	// Count entries in current window
	countCmd := pipe.ZCard(ctx, fullKey)

	// Set expiration on the key
	pipe.Expire(ctx, fullKey, r.config.WindowSize+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Warn("rate limiter redis error",
			zap.String("key", key),
			zap.Error(err),
		)
		if r.config.FailOpen {
			return &Result{
				Allowed:      true,
				Remaining:    -1,
				ResetAfterMs: windowMs,
				RetryAfterMs: 0,
			}, err
		}
		return &Result{
			Allowed:      false,
			Remaining:    0,
			ResetAfterMs: windowMs,
			RetryAfterMs: windowMs,
		}, err
	}

	count := countCmd.Val()
	limit := int64(r.config.RequestsPerMinute + r.config.BurstSize)
	allowed := count <= limit
	remaining := int(limit - count)
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (when oldest entry expires)
	resetAfterMs := windowMs

	result := &Result{
		Allowed:      allowed,
		Remaining:    remaining,
		ResetAfterMs: resetAfterMs,
		RetryAfterMs: 0,
	}

	if !allowed {
		// Calculate retry time based on when space will be available
		result.RetryAfterMs = windowMs / limit
		r.logger.Debug("rate limit exceeded",
			zap.String("key", key),
			zap.Int64("count", count),
			zap.Int64("limit", limit),
		)
	}

	return result, nil
}

// GetRemaining returns the remaining requests allowed for the given key.
func (r *RedisRateLimiter) GetRemaining(ctx context.Context, key string) (int, error) {
	fullKey := r.config.KeyPrefix + key
	now := time.Now()
	windowStart := now.Add(-r.config.WindowSize).UnixMilli()

	// Count entries in current window
	count, err := r.client.ZCount(ctx, fullKey, fmt.Sprintf("%d", windowStart), "+inf").Result()
	if err != nil {
		if err == redis.Nil {
			return r.config.RequestsPerMinute, nil
		}
		return -1, err
	}

	remaining := r.config.RequestsPerMinute + r.config.BurstSize - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// Reset clears the rate limit for the given key.
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := r.config.KeyPrefix + key
	return r.client.Del(ctx, fullKey).Err()
}

// GetConfig returns the current configuration.
func (r *RedisRateLimiter) GetConfig() Config {
	return r.config
}

// UpdateConfig updates the rate limiter configuration.
// Note: This only affects new requests, existing windows are not modified.
func (r *RedisRateLimiter) UpdateConfig(config Config) {
	r.config = config
}

// Close closes the Redis client connection.
// Only call this if you want to close the shared Redis client.
func (r *RedisRateLimiter) Close() error {
	return r.client.Close()
}
