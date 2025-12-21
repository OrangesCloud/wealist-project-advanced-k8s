package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 테스트용 JWT 토큰 생성 헬퍼
func createTestTokenForValidator(t *testing.T, secret string, userID uuid.UUID, claimKey string) string {
	t.Helper()
	claims := jwt.MapClaims{
		claimKey: userID.String(),
		"exp":    time.Now().Add(time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}
	return tokenString
}

func TestNewAuthServiceValidator(t *testing.T) {
	tests := []struct {
		name           string
		authServiceURL string
		secretKey      string
		logger         *zap.Logger
	}{
		{
			name:           "with all parameters",
			authServiceURL: "http://auth-service:8080",
			secretKey:      "test-secret",
			logger:         zap.NewNop(),
		},
		{
			name:           "with nil logger",
			authServiceURL: "http://auth-service:8080",
			secretKey:      "test-secret",
			logger:         nil,
		},
		{
			name:           "without auth service URL",
			authServiceURL: "",
			secretKey:      "test-secret",
			logger:         zap.NewNop(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewAuthServiceValidator(tt.authServiceURL, tt.secretKey, tt.logger)
			if validator == nil {
				t.Error("Expected non-nil validator")
			}
			if validator.authServiceURL != tt.authServiceURL {
				t.Errorf("Expected authServiceURL %s, got %s", tt.authServiceURL, validator.authServiceURL)
			}
			if validator.secretKey != tt.secretKey {
				t.Errorf("Expected secretKey %s, got %s", tt.secretKey, validator.secretKey)
			}
		})
	}
}

func TestAuthServiceValidator_ValidateToken_LocalFallback(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()

	tests := []struct {
		name      string
		claimKey  string
		wantError bool
	}{
		{"sub claim", "sub", false},
		{"userId claim", "userId", false},
		{"user_id claim", "user_id", false},
		{"uid claim", "uid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewAuthServiceValidator("", secret, zap.NewNop())
			token := createTestTokenForValidator(t, secret, userID, tt.claimKey)

			gotUserID, err := validator.ValidateToken(context.Background(), token)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if gotUserID != userID {
				t.Errorf("Expected userID %s, got %s", userID, gotUserID)
			}
		})
	}
}

func TestAuthServiceValidator_ValidateToken_WithAuthService(t *testing.T) {
	userID := uuid.New()

	// Mock auth-service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/validate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"userId": userID.String()})
	}))
	defer server.Close()

	validator := NewAuthServiceValidator(server.URL, "fallback-secret", zap.NewNop())
	gotUserID, err := validator.ValidateToken(context.Background(), "any-token")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, gotUserID)
	}
}

func TestAuthServiceValidator_ValidateToken_AuthServiceFallback(t *testing.T) {
	secret := "fallback-secret"
	userID := uuid.New()

	// Mock auth-service that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	validator := NewAuthServiceValidator(server.URL, secret, zap.NewNop())
	token := createTestTokenForValidator(t, secret, userID, "sub")

	gotUserID, err := validator.ValidateToken(context.Background(), token)
	if err != nil {
		t.Errorf("Unexpected error (should fallback to local): %v", err)
	}
	if gotUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, gotUserID)
	}
}

func TestAuthServiceValidator_InvalidToken(t *testing.T) {
	validator := NewAuthServiceValidator("", "test-secret", zap.NewNop())

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "not-a-jwt-token"},
		{"wrong secret", createTestTokenForValidator(&testing.T{}, "wrong-secret", uuid.New(), "sub")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateToken(context.Background(), tt.token)
			if err == nil {
				t.Error("Expected error for invalid token")
			}
		})
	}
}

func TestLocalValidator(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	validator := NewLocalValidator(secret)
	token := createTestTokenForValidator(t, secret, userID, "sub")

	gotUserID, err := validator.ValidateToken(context.Background(), token)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, gotUserID)
	}
}

func TestAuthMiddlewareWithValidator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	userID := uuid.New()

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantUserID bool
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + createTestTokenForValidator(t, secret, userID, "sub"),
			wantStatus: http.StatusOK,
			wantUserID: true,
		},
		{
			name:       "no header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantUserID: false,
		},
		{
			name:       "invalid format - no Bearer",
			authHeader: "Basic token",
			wantStatus: http.StatusUnauthorized,
			wantUserID: false,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			wantStatus: http.StatusUnauthorized,
			wantUserID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewLocalValidator(secret)

			r := gin.New()
			r.Use(AuthMiddlewareWithValidator(validator))
			r.GET("/test", func(c *gin.Context) {
				if tt.wantUserID {
					id, exists := c.Get("user_id")
					if !exists {
						t.Error("Expected user_id in context")
					}
					if id.(uuid.UUID) != userID {
						t.Errorf("Expected userID %s, got %s", userID, id)
					}
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
