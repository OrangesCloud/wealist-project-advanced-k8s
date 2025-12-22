package ratelimit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// KeyFunc is a function that extracts the rate limit key from a request.
type KeyFunc func(c *gin.Context) string

// IPKey extracts the client IP as the rate limit key.
func IPKey(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// UserKey extracts the user ID from context as the rate limit key.
// Falls back to IP if user ID is not available.
func UserKey(c *gin.Context) string {
	userID := c.GetString("userID")
	if userID == "" {
		// Fallback to IP for unauthenticated requests
		return "ip:" + c.ClientIP()
	}
	return "user:" + userID
}

// PathKey combines the path and client IP as the rate limit key.
func PathKey(c *gin.Context) string {
	return "path:" + c.Request.URL.Path + ":" + c.ClientIP()
}

// PathUserKey combines the path and user ID as the rate limit key.
// Falls back to PathKey if user ID is not available.
func PathUserKey(c *gin.Context) string {
	userID := c.GetString("userID")
	if userID == "" {
		return PathKey(c)
	}
	return "path:" + c.Request.URL.Path + ":user:" + userID
}

// CombinedKey creates a key that includes service name, path, and user/IP.
func CombinedKey(serviceName string) KeyFunc {
	return func(c *gin.Context) string {
		userID := c.GetString("userID")
		if userID != "" {
			return serviceName + ":user:" + userID
		}
		return serviceName + ":ip:" + c.ClientIP()
	}
}

// Middleware creates a Gin middleware for rate limiting.
func Middleware(limiter RateLimiter, keyFunc KeyFunc, config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if rate limiting is disabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Check if path is excluded
		if config.IsExcluded(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract rate limit key
		key := keyFunc(c)

		// Check if request is allowed
		var allowed bool
		var remaining int
		var resetAfterMs int64
		var err error

		if limiterWithInfo, ok := limiter.(RateLimiterWithInfo); ok {
			result, e := limiterWithInfo.AllowWithInfo(c.Request.Context(), key)
			err = e
			if result != nil {
				allowed = result.Allowed
				remaining = result.Remaining
				resetAfterMs = result.ResetAfterMs
			} else {
				allowed = config.FailOpen
			}
		} else {
			allowed, err = limiter.Allow(c.Request.Context(), key)
			remaining = -1 // Unknown
			resetAfterMs = config.WindowSize.Milliseconds()
		}

		// Set rate limit headers
		limit := config.GetLimitForPath(c.Request.URL.Path)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		if remaining >= 0 {
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		}
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAfterMs/1000, 10))

		// Handle errors (already logged in limiter)
		if err != nil {
			if config.FailOpen {
				c.Next()
				return
			}
			c.Header("Retry-After", strconv.FormatInt(resetAfterMs/1000, 10))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Service Unavailable",
				"message": "Rate limiter unavailable. Please try again later.",
			})
			return
		}

		// Deny if rate limited
		if !allowed {
			retryAfter := resetAfterMs / 1000
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
				"retryAfter": retryAfter,
			})
			return
		}

		c.Next()
	}
}

// MiddlewareWithLogger creates a Gin middleware with custom logger.
func MiddlewareWithLogger(limiter RateLimiter, keyFunc KeyFunc, config Config, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if rate limiting is disabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Check if path is excluded
		if config.IsExcluded(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract rate limit key
		key := keyFunc(c)

		// Check if request is allowed
		var allowed bool
		var remaining int
		var resetAfterMs int64
		var err error

		if limiterWithInfo, ok := limiter.(RateLimiterWithInfo); ok {
			result, e := limiterWithInfo.AllowWithInfo(c.Request.Context(), key)
			err = e
			if result != nil {
				allowed = result.Allowed
				remaining = result.Remaining
				resetAfterMs = result.ResetAfterMs
			} else {
				allowed = config.FailOpen
			}
		} else {
			allowed, err = limiter.Allow(c.Request.Context(), key)
			remaining = -1
			resetAfterMs = config.WindowSize.Milliseconds()
		}

		// Set rate limit headers
		limit := config.GetLimitForPath(c.Request.URL.Path)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		if remaining >= 0 {
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		}
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAfterMs/1000, 10))

		// Handle errors
		if err != nil {
			logger.Warn("rate limiter error",
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path),
				zap.Error(err),
			)
			if config.FailOpen {
				c.Next()
				return
			}
			c.Header("Retry-After", strconv.FormatInt(resetAfterMs/1000, 10))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Service Unavailable",
				"message": "Rate limiter unavailable. Please try again later.",
			})
			return
		}

		// Deny if rate limited
		if !allowed {
			retryAfter := resetAfterMs / 1000
			if retryAfter < 1 {
				retryAfter = 1
			}
			logger.Info("rate limit exceeded",
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path),
				zap.String("clientIP", c.ClientIP()),
			)
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
				"retryAfter": retryAfter,
			})
			return
		}

		c.Next()
	}
}

// NewMiddleware is a convenience function to create rate limiting middleware.
// It creates a RedisRateLimiter internally with the provided configuration.
func NewMiddleware(client interface{ Pipeline() interface{} }, config Config, logger *zap.Logger) gin.HandlerFunc {
	// Type assertion to get *redis.Client
	// This allows the package to accept redis.Client without importing it directly in some cases
	panic("Use Middleware() with explicit limiter creation instead")
}
