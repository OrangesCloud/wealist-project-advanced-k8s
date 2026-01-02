// Package auth provides JWT authentication utilities.
// This file contains JWTParser for Istio JWT mode where Istio handles validation.
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JWTParser는 검증 없이 JWT를 파싱하여 user_id만 추출합니다.
// Istio RequestAuthentication이 이미 JWT를 검증한 경우 사용합니다.
// ⚠️ 주의: 반드시 Istio가 JWT를 검증한 후에만 사용해야 합니다.
type JWTParser struct {
	logger *zap.Logger
}

// NewJWTParser는 새 JWTParser를 생성합니다.
func NewJWTParser(logger *zap.Logger) *JWTParser {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JWTParser{logger: logger}
}

// ParseToken은 검증 없이 JWT claims를 추출합니다.
// Istio RequestAuthentication이 먼저 검증했다고 가정합니다.
// 'sub', 'userId', 'user_id', 'uid' 클레임에서 user_id를 추출합니다.
func (p *JWTParser) ParseToken(tokenString string) (uuid.UUID, error) {
	// 서명 검증 없이 파싱 (Istio가 이미 검증함)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		p.logger.Debug("Failed to parse JWT token", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims format")
	}

	// user_id 추출 (sub, userId, user_id, uid 순서)
	var userIDStr string
	for _, key := range []string{"sub", "userId", "user_id", "uid"} {
		if val, exists := claims[key]; exists {
			if str, ok := val.(string); ok && str != "" {
				userIDStr = str
				break
			}
		}
	}

	if userIDStr == "" {
		return uuid.Nil, errors.New("user_id not found in claims")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	p.logger.Debug("JWT parsed successfully (Istio mode)",
		zap.String("user_id", userID.String()))

	return userID, nil
}

// IstioAuthMiddleware는 Istio가 JWT를 검증한 경우 사용하는 Gin 미들웨어입니다.
// JWT 파싱만 수행하고, 서명 검증은 스킵합니다.
// Authorization 헤더에서 Bearer 토큰을 추출하고 claims에서 user_id를 가져옵니다.
// 'user_id'와 'jwtToken'을 Gin 컨텍스트에 저장합니다.
func IstioAuthMiddleware(parser *JWTParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// Bearer 토큰 추출
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// JWT 파싱 (검증 없음 - Istio가 이미 검증함)
		userID, err := parser.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Failed to parse token",
				},
			})
			c.Abort()
			return
		}

		// 컨텍스트에 사용자 정보 저장 (상수는 keys.go 참조)
		c.Set(UserIDContextKey, userID)
		c.Set(TokenContextKey, tokenString)
		c.Next()
	}
}

// Note: GetUserID and GetJWTToken are defined in auth.go
// They work with both AuthMiddlewareWithValidator and IstioAuthMiddleware
