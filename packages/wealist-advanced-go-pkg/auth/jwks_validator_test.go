package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestJWKSValidator_ValidateToken(t *testing.T) {
	// RSA 키 쌍 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	// 테스트용 JWKS 서버
	keyID := "test-key-1"
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: keyID,
					N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), // 65537
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	logger := zap.NewNop()
	validator := NewJWKSValidator(jwksServer.URL, "test-issuer", logger)

	t.Run("valid token", func(t *testing.T) {
		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": userID.String(),
			"iss": "test-issuer",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		token.Header["kid"] = keyID

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		result, err := validator.ValidateToken(context.Background(), tokenString)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != userID {
			t.Errorf("expected userID %s, got %s", userID, result)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": userID.String(),
			"iss": "test-issuer",
			"exp": time.Now().Add(-time.Hour).Unix(), // 만료됨
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		})
		token.Header["kid"] = keyID

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": userID.String(),
			"iss": "wrong-issuer",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		token.Header["kid"] = keyID

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for wrong issuer")
		}
	})

	t.Run("missing kid", func(t *testing.T) {
		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": userID.String(),
			"iss": "test-issuer",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		// kid 없음

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for missing kid")
		}
	})

	t.Run("unknown kid", func(t *testing.T) {
		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": userID.String(),
			"iss": "test-issuer",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		token.Header["kid"] = "unknown-key"

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = validator.ValidateToken(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for unknown kid")
		}
	})
}

func TestJWKSValidator_KeyCaching(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	keyID := "test-key-1"

	fetchCount := 0
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: keyID,
					N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	validator := NewJWKSValidator(jwksServer.URL, "test-issuer", nil)
	validator.SetCacheTTL(time.Hour) // 긴 캐시 TTL

	// 토큰 생성
	createToken := func() string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": uuid.New().String(),
			"iss": "test-issuer",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = keyID
		tokenString, _ := token.SignedString(privateKey)
		return tokenString
	}

	// 여러 번 검증
	for i := 0; i < 5; i++ {
		_, err := validator.ValidateToken(context.Background(), createToken())
		if err != nil {
			t.Errorf("validation %d failed: %v", i, err)
		}
	}

	// JWKS는 한 번만 fetch되어야 함
	if fetchCount != 1 {
		t.Errorf("expected 1 JWKS fetch, got %d", fetchCount)
	}
}
