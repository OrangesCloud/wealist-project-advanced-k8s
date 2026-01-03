// Package auth provides JWT authentication utilities.
// This file defines context key constants for consistent usage across all services.
package auth

// Context key constants for storing authentication data in Gin context.
// All middleware and handlers should use these constants for consistency.
const (
	// UserIDContextKey is the key for storing user ID (uuid.UUID) in context.
	UserIDContextKey = "user_id"

	// TokenContextKey is the key for storing JWT token string in context.
	// Note: "jwtToken" is used for backward compatibility.
	TokenContextKey = "jwtToken"
)
