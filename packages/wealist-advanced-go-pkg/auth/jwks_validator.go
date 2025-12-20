// Package auth는 JWT 인증 미들웨어를 제공합니다.
// 이 파일은 JWKS 기반 RSA 토큰 검증을 구현합니다.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JWK는 JSON Web Key 구조체입니다.
type JWK struct {
	Kty string `json:"kty"` // Key Type (RSA)
	Use string `json:"use"` // Usage (sig)
	Alg string `json:"alg"` // Algorithm (RS256)
	Kid string `json:"kid"` // Key ID
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// JWKS는 JSON Web Key Set 구조체입니다.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSValidator는 JWKS 엔드포인트에서 공개키를 가져와 JWT를 검증합니다.
type JWKSValidator struct {
	jwksURL    string
	issuer     string
	httpClient *http.Client
	logger     *zap.Logger

	// 캐시된 키
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
	cacheTTL   time.Duration
}

// NewJWKSValidator는 새 JWKSValidator를 생성합니다.
// jwksURL: JWKS 엔드포인트 URL (예: http://auth-service:8080/.well-known/jwks.json)
// issuer: 예상되는 JWT issuer (예: wealist-auth-service)
func NewJWKSValidator(jwksURL, issuer string, logger *zap.Logger) *JWKSValidator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JWKSValidator{
		jwksURL: jwksURL,
		issuer:  issuer,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:   logger,
		keys:     make(map[string]*rsa.PublicKey),
		cacheTTL: 5 * time.Minute, // 키 캐시 5분
	}
}

// ValidateToken은 JWKS를 사용하여 JWT 토큰을 검증합니다.
func (v *JWKSValidator) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	// 토큰 파싱 (검증 없이 헤더만 확인)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// kid (Key ID) 추출
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("token missing kid header")
	}

	// 공개키 가져오기 (캐시 또는 JWKS에서)
	publicKey, err := v.getPublicKey(ctx, kid)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// 토큰 검증
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// RS256 알고리즘 확인
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !parsedToken.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	// 클레임 추출
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	// issuer 검증
	if v.issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != v.issuer {
			return uuid.Nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, iss)
		}
	}

	// 사용자 ID 추출
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

// getPublicKey는 캐시에서 키를 가져오거나 JWKS에서 새로 가져옵니다.
func (v *JWKSValidator) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, exists := v.keys[kid]
	needsRefresh := time.Since(v.lastFetch) > v.cacheTTL
	v.mu.RUnlock()

	if exists && !needsRefresh {
		return key, nil
	}

	// JWKS 새로고침
	if err := v.refreshKeys(ctx); err != nil {
		// 캐시된 키가 있으면 사용
		if exists {
			v.logger.Warn("Failed to refresh JWKS, using cached key",
				zap.Error(err),
				zap.String("kid", kid))
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	key, exists = v.keys[kid]
	v.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// refreshKeys는 JWKS 엔드포인트에서 키를 새로 가져옵니다.
func (v *JWKSValidator) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		publicKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			v.logger.Warn("Failed to parse JWK",
				zap.String("kid", jwk.Kid),
				zap.Error(err))
			continue
		}

		newKeys[jwk.Kid] = publicKey
	}

	v.mu.Lock()
	v.keys = newKeys
	v.lastFetch = time.Now()
	v.mu.Unlock()

	v.logger.Debug("JWKS refreshed", zap.Int("key_count", len(newKeys)))
	return nil
}

// jwkToRSAPublicKey는 JWK를 RSA 공개키로 변환합니다.
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Base64 URL decode
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// BigInt로 변환
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// SetCacheTTL은 키 캐시 TTL을 설정합니다.
func (v *JWKSValidator) SetCacheTTL(ttl time.Duration) {
	v.mu.Lock()
	v.cacheTTL = ttl
	v.mu.Unlock()
}
