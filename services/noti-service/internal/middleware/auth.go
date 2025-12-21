// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 JWT 인증 미들웨어와 서비스 전용 미들웨어를 포함합니다.
package middleware

import (
	"context"
	"noti-service/internal/response"
	"strings"

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
func ValidateTokenFromContext(ctx context.Context, validator TokenValidator, token string) (uuid.UUID, error) {
	return validator.ValidateToken(ctx, token)
}

// ===== noti-service 전용 미들웨어 =====

// InternalAuthMiddleware는 서비스 간 내부 API 키를 검증하는 미들웨어입니다.
// x-internal-api-key 또는 X-Internal-Api-Key 헤더를 확인합니다.
func InternalAuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		providedKey := c.GetHeader("x-internal-api-key")
		if providedKey == "" {
			providedKey = c.GetHeader("X-Internal-Api-Key")
		}

		if providedKey == "" || providedKey != apiKey {
			response.Unauthorized(c, "Invalid internal API key")
			c.Abort()
			return
		}

		c.Next()
	}
}

// WorkspaceMiddleware는 요청 헤더에서 워크스페이스 ID를 추출하는 미들웨어입니다.
// x-workspace-id 또는 X-Workspace-Id 헤더에서 추출하여 컨텍스트에 저장합니다.
func WorkspaceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceIDStr := c.GetHeader("x-workspace-id")
		if workspaceIDStr == "" {
			workspaceIDStr = c.GetHeader("X-Workspace-Id")
		}

		if workspaceIDStr != "" {
			workspaceID, err := uuid.Parse(workspaceIDStr)
			if err == nil {
				c.Set("workspace_id", workspaceID)
			}
		}

		c.Next()
	}
}

// RequireWorkspace는 워크스페이스 ID가 컨텍스트에 있는지 확인하는 미들웨어입니다.
// WorkspaceMiddleware 이후에 사용해야 합니다.
func RequireWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("workspace_id")
		if !exists {
			response.BadRequest(c, "x-workspace-id header is required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SSEAuthMiddleware는 SSE 연결을 위한 인증 미들웨어입니다.
// EventSource API는 커스텀 헤더를 지원하지 않으므로 쿼리 파라미터로 토큰을 전달받습니다.
// 쿼리 파라미터가 없으면 Authorization 헤더를 확인합니다.
func SSEAuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 쿼리 파라미터에서 토큰 확인 (SSE용)
		tokenString := c.Query("token")

		// Authorization 헤더로 fallback
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString = parts[1]
				}
			}
		}

		if tokenString == "" {
			response.Unauthorized(c, "No token provided")
			c.Abort()
			return
		}

		userID, err := validator.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			response.Unauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
