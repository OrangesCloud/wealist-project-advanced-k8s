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
// Deprecated: SmartValidator 사용 권장 (RS256 JWKS 지원)
type AuthServiceValidator = commonauth.AuthServiceValidator

// SmartValidator는 공통 모듈의 SmartValidator 타입 별칭입니다.
// auth-service HTTP 검증 + JWKS (RSA) fallback을 지원합니다.
type SmartValidator = commonauth.SmartValidator

// JWTParser는 공통 모듈의 JWTParser 타입 별칭입니다.
// Istio JWT 모드에서 사용: 검증 없이 파싱만 수행합니다.
type JWTParser = commonauth.JWTParser

// NewAuthServiceValidator는 새 AuthServiceValidator를 생성합니다.
// Deprecated: NewSmartValidator 사용 권장
func NewAuthServiceValidator(authServiceURL, secretKey string, logger *zap.Logger) *AuthServiceValidator {
	return commonauth.NewAuthServiceValidator(authServiceURL, secretKey, logger)
}

// NewSmartValidator는 새 SmartValidator를 생성합니다.
// authServiceURL: auth-service URL (예: http://auth-service:8080)
// issuer: JWT issuer (예: wealist-auth-service)
func NewSmartValidator(authServiceURL, issuer string, logger *zap.Logger) *SmartValidator {
	return commonauth.NewSmartValidator(authServiceURL, issuer, logger)
}

// NewJWTParser는 새 JWTParser를 생성합니다.
// Istio JWT 모드에서 사용: Istio가 검증을 완료했다고 가정하고 파싱만 수행합니다.
func NewJWTParser(logger *zap.Logger) *JWTParser {
	return commonauth.NewJWTParser(logger)
}

// IstioAuthMiddleware는 Istio JWT 모드용 미들웨어입니다.
// Istio가 JWT를 검증한 후 Go 서비스는 파싱만 수행합니다.
func IstioAuthMiddleware(parser *JWTParser) gin.HandlerFunc {
	return commonauth.IstioAuthMiddleware(parser)
}

// AuthMiddleware는 JWT 토큰을 검증하는 Gin 미들웨어입니다.
// TokenValidator를 사용하여 토큰을 검증하고 user_id를 컨텍스트에 저장합니다.
func AuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return commonauth.AuthMiddlewareWithValidator(validator)
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
// WebSocket 연결 등에서 사용할 수 있습니다.
func ValidateTokenFromContext(ctx context.Context, validator TokenValidator, token string) (uuid.UUID, error) {
	return validator.ValidateToken(ctx, token)
}
