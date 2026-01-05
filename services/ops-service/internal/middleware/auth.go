// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 JWT 인증 미들웨어를 포함합니다.
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonauth "github.com/OrangesCloud/wealist-advanced-go-pkg/auth"
)

// TokenValidator는 공통 모듈의 TokenValidator 타입 별칭입니다.
// 토큰 검증 전략을 추상화합니다.
type TokenValidator = commonauth.TokenValidator

// SmartValidator는 공통 모듈의 SmartValidator 타입 별칭입니다.
// auth-service HTTP 검증 + JWKS (RSA) fallback을 지원합니다.
type SmartValidator = commonauth.SmartValidator

// JWTParser는 공통 모듈의 JWTParser 타입 별칭입니다.
// Istio JWT 모드에서 사용: 검증 없이 파싱만 수행합니다.
type JWTParser = commonauth.JWTParser

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

// getClaimsFromToken은 토큰에서 claims를 추출합니다.
// 토큰은 이미 검증되었으므로 서명 검증 없이 파싱합니다.
func getClaimsFromToken(tokenString string) (jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// GetEmail은 컨텍스트에서 이메일을 추출합니다.
// JWT 토큰에서 "email" claim을 파싱합니다.
func GetEmail(c *gin.Context) string {
	token, ok := GetJWTToken(c)
	if !ok {
		return ""
	}
	claims, err := getClaimsFromToken(token)
	if err != nil {
		return ""
	}
	if email, ok := claims["email"].(string); ok {
		return email
	}
	return ""
}

// GetName은 컨텍스트에서 이름을 추출합니다.
// JWT 토큰에서 "name" claim을 파싱합니다.
func GetName(c *gin.Context) string {
	token, ok := GetJWTToken(c)
	if !ok {
		return ""
	}
	claims, err := getClaimsFromToken(token)
	if err != nil {
		return ""
	}
	if name, ok := claims["name"].(string); ok {
		return name
	}
	return ""
}

// GetPicture은 컨텍스트에서 사진 URL을 추출합니다.
// JWT 토큰에서 "picture" claim을 파싱합니다.
func GetPicture(c *gin.Context) string {
	token, ok := GetJWTToken(c)
	if !ok {
		return ""
	}
	claims, err := getClaimsFromToken(token)
	if err != nil {
		return ""
	}
	if picture, ok := claims["picture"].(string); ok {
		return picture
	}
	return ""
}

// ValidateTokenFromContext는 컨텍스트에서 토큰을 가져와 검증합니다.
func ValidateTokenFromContext(ctx context.Context, validator TokenValidator, token string) (uuid.UUID, error) {
	return validator.ValidateToken(ctx, token)
}

// ExtractBearerToken은 Authorization 헤더에서 Bearer 토큰을 추출합니다.
func ExtractBearerToken(authHeader string) (string, bool) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", false
	}
	return parts[1], true
}
