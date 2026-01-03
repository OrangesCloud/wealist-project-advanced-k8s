package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenValidator interface for token validation
type TokenValidator interface {
	ValidateToken(token string) (map[string]interface{}, error)
}

// JWTParser parses JWT without validation (for Istio mode)
type JWTParser struct {
	logger *zap.Logger
}

// NewJWTParser creates a new JWT parser
func NewJWTParser(logger *zap.Logger) *JWTParser {
	return &JWTParser{logger: logger}
}

// ParseToken parses JWT payload without validation
func (p *JWTParser) ParseToken(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// AuthWithValidator creates middleware that validates tokens using the provided validator
func AuthWithValidator(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := validator.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		setClaimsToContext(c, claims)
		c.Next()
	}
}

// IstioAuthMiddleware creates middleware for Istio JWT mode (parse only)
func IstioAuthMiddleware(parser *JWTParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := parser.ParseToken(parts[1])
		if err != nil || claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		setClaimsToContext(c, claims)
		c.Next()
	}
}

func setClaimsToContext(c *gin.Context, claims map[string]interface{}) {
	if sub, ok := claims["sub"].(string); ok {
		c.Set("userID", sub)
	}
	if email, ok := claims["email"].(string); ok {
		c.Set("email", email)
	}
	if name, ok := claims["name"].(string); ok {
		c.Set("name", name)
	}
	if picture, ok := claims["picture"].(string); ok {
		c.Set("picture", picture)
	}
	c.Set("claims", claims)
}

// GetUserID gets user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("userID"); exists {
		return userID.(string)
	}
	return ""
}

// GetEmail gets email from context
func GetEmail(c *gin.Context) string {
	if email, exists := c.Get("email"); exists {
		return email.(string)
	}
	return ""
}

// GetName gets name from context
func GetName(c *gin.Context) string {
	if name, exists := c.Get("name"); exists {
		return name.(string)
	}
	return ""
}

// GetPicture gets picture from context
func GetPicture(c *gin.Context) string {
	if picture, exists := c.Get("picture"); exists {
		return picture.(string)
	}
	return ""
}
