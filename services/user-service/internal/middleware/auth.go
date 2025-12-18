// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 JWT 인증 미들웨어를 포함합니다.
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonauth "github.com/OrangesCloud/wealist-advanced-go-pkg/auth"
)

// TokenValidator는 공통 모듈의 TokenValidator 타입 별칭입니다.
// 토큰 검증 전략을 추상화합니다.
type TokenValidator = commonauth.TokenValidator

// AuthServiceValidator는 공통 모듈의 AuthServiceValidator 타입 별칭입니다.
// auth-service 연동 + 로컬 JWT fallback을 지원합니다.
type AuthServiceValidator = commonauth.AuthServiceValidator

// NewAuthServiceValidator는 새 AuthServiceValidator를 생성합니다.
// authServiceURL: auth-service URL (비어있으면 로컬 검증만 사용)
// secretKey: JWT 서명 키 (로컬 검증용)
// logger: 로거 (nil이면 nop 로거 사용)
func NewAuthServiceValidator(authServiceURL, secretKey string, logger *zap.Logger) *AuthServiceValidator {
	return commonauth.NewAuthServiceValidator(authServiceURL, secretKey, logger)
}

// AuthWithValidator는 TokenValidator를 사용하여 JWT 토큰을 검증하는 미들웨어입니다.
// Authorization 헤더에서 Bearer 토큰을 추출하고 검증합니다.
func AuthWithValidator(validator TokenValidator) gin.HandlerFunc {
	return commonauth.AuthMiddlewareWithValidator(validator)
}

// Auth는 JWT 토큰을 로컬에서 검증하는 미들웨어입니다.
// 공통 모듈의 JWTMiddleware를 사용합니다.
func Auth(jwtSecret string) gin.HandlerFunc {
	return commonauth.JWTMiddleware(jwtSecret)
}

// GetUserID는 컨텍스트에서 사용자 ID를 추출합니다.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	return commonauth.GetUserID(c)
}

// GetJWTToken은 컨텍스트에서 JWT 토큰을 추출합니다.
func GetJWTToken(c *gin.Context) (string, bool) {
	return commonauth.GetJWTToken(c)
}

// ValidateTokenFromContext는 컨텍스트에서 토큰을 가져와 검증합니다.
func ValidateTokenFromContext(ctx context.Context, validator TokenValidator, token string) (uuid.UUID, error) {
	return validator.ValidateToken(ctx, token)
}
