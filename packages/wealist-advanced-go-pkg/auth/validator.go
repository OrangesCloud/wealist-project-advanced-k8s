// Package auth는 JWT 인증 미들웨어를 제공합니다.
// 이 파일은 토큰 검증 인터페이스와 auth-service 연동 구현을 포함합니다.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TokenValidator는 JWT 토큰 검증을 위한 인터페이스입니다.
// 다양한 검증 전략(로컬, auth-service 등)을 추상화합니다.
type TokenValidator interface {
	// ValidateToken은 JWT 토큰을 검증하고 사용자 ID를 반환합니다.
	// 토큰이 유효하지 않거나 만료된 경우 에러를 반환합니다.
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// AuthServiceValidator는 auth-service를 통한 토큰 검증과
// 로컬 JWT 검증 fallback을 구현합니다.
type AuthServiceValidator struct {
	authServiceURL string       // auth-service URL (예: http://auth-service:8080)
	secretKey      string       // JWT 서명 키 (로컬 검증용)
	httpClient     *http.Client // HTTP 클라이언트
	logger         *zap.Logger  // 로거
}

// NewAuthServiceValidator는 새 AuthServiceValidator를 생성합니다.
// authServiceURL이 비어있으면 로컬 검증만 사용합니다.
// secretKey는 로컬 JWT 검증에 사용됩니다.
func NewAuthServiceValidator(authServiceURL, secretKey string, logger *zap.Logger) *AuthServiceValidator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuthServiceValidator{
		authServiceURL: authServiceURL,
		secretKey:      secretKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// ValidateToken은 JWT 토큰을 검증합니다.
// 먼저 auth-service를 통해 검증을 시도하고, 실패하면 로컬 JWT 검증으로 fallback합니다.
// 검증 성공 시 사용자 ID(UUID)를 반환합니다.
func (v *AuthServiceValidator) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	// auth-service URL이 설정된 경우 먼저 시도
	if v.authServiceURL != "" {
		userID, err := v.validateWithAuthService(ctx, tokenString)
		if err == nil {
			return userID, nil
		}
		v.logger.Debug("Auth service validation failed, falling back to local",
			zap.Error(err),
			zap.String("auth_service_url", v.authServiceURL))
	}

	// 로컬 JWT 검증으로 fallback
	return v.validateLocally(tokenString)
}

// validateWithAuthService는 auth-service의 /api/auth/validate 엔드포인트를 호출하여
// 토큰을 검증합니다.
func (v *AuthServiceValidator) validateWithAuthService(ctx context.Context, token string) (uuid.UUID, error) {
	url := v.authServiceURL + "/api/auth/validate"

	reqBody, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return uuid.Nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	var result struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, err
	}

	return uuid.Parse(result.UserID)
}

// validateLocally는 JWT 토큰을 로컬에서 secretKey를 사용하여 검증합니다.
// 'sub', 'userId', 'user_id' 클레임에서 사용자 ID를 추출합니다.
func (v *AuthServiceValidator) validateLocally(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// HMAC 서명 방식 검증
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(v.secretKey), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	// 다양한 클레임 키에서 사용자 ID 추출
	var userIDStr string
	for _, key := range []string{"sub", "userId", "user_id", "uid"} {
		if val, exists := claims[key]; exists {
			if str, ok := val.(string); ok {
				userIDStr = str
				break
			}
		}
	}

	if userIDStr == "" {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return uuid.Parse(userIDStr)
}

// AuthMiddlewareWithValidator는 TokenValidator를 사용하여 JWT 토큰을 검증하는
// Gin 미들웨어를 반환합니다.
// Authorization 헤더에서 Bearer 토큰을 추출하고 검증합니다.
// 검증 성공 시 'user_id'와 'jwtToken'을 Gin 컨텍스트에 저장합니다.
func AuthMiddlewareWithValidator(validator TokenValidator) gin.HandlerFunc {
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
		userID, err := validator.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
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

// LocalValidator는 로컬 JWT 검증만 수행하는 간단한 validator입니다.
// auth-service 없이 JWT secret만으로 검증할 때 사용합니다.
// Deprecated: auth-service가 RS256을 사용하므로 SmartValidator 사용 권장
type LocalValidator struct {
	secretKey string
}

// NewLocalValidator는 새 LocalValidator를 생성합니다.
// Deprecated: auth-service가 RS256을 사용하므로 NewSmartValidator 사용 권장
func NewLocalValidator(secretKey string) *LocalValidator {
	return &LocalValidator{secretKey: secretKey}
}

// ValidateToken은 로컬에서 JWT 토큰을 검증합니다.
func (v *LocalValidator) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(v.secretKey), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	var userIDStr string
	for _, key := range []string{"sub", "userId", "user_id", "uid"} {
		if val, exists := claims[key]; exists {
			if str, ok := val.(string); ok {
				userIDStr = str
				break
			}
		}
	}

	if userIDStr == "" {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return uuid.Parse(userIDStr)
}

// SmartValidator는 여러 검증 전략을 체이닝합니다.
// 1. auth-service HTTP 검증 (/api/auth/validate)
// 2. JWKS (RSA) 검증 fallback (/.well-known/jwks.json)
// auth-service가 RS256으로 JWT를 서명하므로 JWKS 검증이 필수입니다.
type SmartValidator struct {
	authServiceURL string         // auth-service URL (예: http://auth-service:8080)
	issuer         string         // JWT issuer (예: wealist-auth-service)
	httpClient     *http.Client   // HTTP 클라이언트
	logger         *zap.Logger    // 로거
	jwksValidator  *JWKSValidator // JWKS 검증기 (RSA)
}

// NewSmartValidator는 새 SmartValidator를 생성합니다.
// authServiceURL: auth-service URL (예: http://auth-service:8080)
// issuer: JWT issuer (예: wealist-auth-service), 빈 문자열이면 issuer 검증 생략
func NewSmartValidator(authServiceURL, issuer string, logger *zap.Logger) *SmartValidator {
	if logger == nil {
		logger = zap.NewNop()
	}

	jwksURL := authServiceURL + "/.well-known/jwks.json"

	return &SmartValidator{
		authServiceURL: authServiceURL,
		issuer:         issuer,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:        logger,
		jwksValidator: NewJWKSValidator(jwksURL, issuer, logger),
	}
}

// ValidateToken은 JWT 토큰을 검증합니다.
// 1. auth-service HTTP 검증 시도
// 2. JWKS (RSA) 검증 fallback
func (v *SmartValidator) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	// 1. auth-service HTTP 검증 시도
	if v.authServiceURL != "" {
		userID, err := v.validateWithAuthService(ctx, tokenString)
		if err == nil {
			return userID, nil
		}
		v.logger.Debug("Auth service HTTP validation failed, trying JWKS",
			zap.Error(err),
			zap.String("auth_service_url", v.authServiceURL))
	}

	// 2. JWKS (RSA) 검증 fallback
	return v.jwksValidator.ValidateToken(ctx, tokenString)
}

// validateWithAuthService는 auth-service의 /api/auth/validate 엔드포인트를 호출하여
// 토큰을 검증합니다.
func (v *SmartValidator) validateWithAuthService(ctx context.Context, token string) (uuid.UUID, error) {
	url := v.authServiceURL + "/api/auth/validate"

	reqBody, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return uuid.Nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	var result struct {
		UserID string `json:"userId"`
		Valid  bool   `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, err
	}

	// valid 필드 확인 (auth-service가 false를 반환할 수 있음)
	if !result.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return uuid.Parse(result.UserID)
}
