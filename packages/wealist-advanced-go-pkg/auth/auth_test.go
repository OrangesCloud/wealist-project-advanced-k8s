// Package auth는 JWT 인증 미들웨어를 제공합니다.
// 이 파일은 auth.go의 테스트를 포함합니다.
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// testJWTSecret은 테스트에서 사용하는 JWT 시크릿입니다.
const testJWTSecret = "test-jwt-secret-key-for-testing-purposes"

func init() {
	// 테스트 모드 설정
	gin.SetMode(gin.TestMode)
}

// createTestToken은 테스트용 JWT 토큰을 생성합니다.
func createTestToken(userID string, expiresIn time.Duration, secret string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(expiresIn).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// createTestTokenWithClaim은 특정 클레임 키로 테스트용 JWT 토큰을 생성합니다.
func createTestTokenWithClaim(claimKey, userID string, expiresIn time.Duration, secret string) string {
	claims := jwt.MapClaims{
		claimKey: userID,
		"exp":    time.Now().Add(expiresIn).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// setupTestRouter는 테스트용 라우터를 설정합니다.
func setupTestRouter(jwtSecret string) *gin.Engine {
	r := gin.New()
	r.Use(JWTMiddleware(jwtSecret))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID.String()})
	})
	return r
}

// TestJWTMiddleware_ValidToken은 유효한 토큰으로 인증 성공하는지 테스트합니다.
func TestJWTMiddleware_ValidToken(t *testing.T) {
	userID := uuid.New().String()
	token := createTestToken(userID, time.Hour, testJWTSecret)

	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusOK, w.Code)
	}
}

// TestJWTMiddleware_NoAuthHeader는 Authorization 헤더가 없을 때 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_NoAuthHeader(t *testing.T) {
	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	// Authorization 헤더 없음

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}

// TestJWTMiddleware_InvalidFormat은 잘못된 헤더 형식일 때 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name       string // 테스트 케이스 이름
		authHeader string // Authorization 헤더 값
	}{
		{
			name:       "Bearer 없이 토큰만",
			authHeader: "some-token",
		},
		{
			name:       "Basic 인증",
			authHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:       "빈 Bearer",
			authHeader: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter(testJWTSecret)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

// TestJWTMiddleware_ExpiredToken은 만료된 토큰으로 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	userID := uuid.New().String()
	// 이미 만료된 토큰 생성
	token := createTestToken(userID, -time.Hour, testJWTSecret)

	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}

// TestJWTMiddleware_InvalidSecret은 잘못된 시크릿으로 서명된 토큰으로 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_InvalidSecret(t *testing.T) {
	userID := uuid.New().String()
	// 다른 시크릿으로 토큰 생성
	token := createTestToken(userID, time.Hour, "wrong-secret")

	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}

// TestJWTMiddleware_InvalidUserID는 잘못된 사용자 ID 형식일 때 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_InvalidUserID(t *testing.T) {
	// UUID가 아닌 사용자 ID로 토큰 생성
	token := createTestToken("not-a-uuid", time.Hour, testJWTSecret)

	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}

// TestJWTMiddleware_DifferentClaimKeys는 다양한 클레임 키를 지원하는지 테스트합니다.
func TestJWTMiddleware_DifferentClaimKeys(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name     string // 테스트 케이스 이름
		claimKey string // 클레임 키
	}{
		{
			name:     "user_id 클레임",
			claimKey: "user_id",
		},
		{
			name:     "sub 클레임",
			claimKey: "sub",
		},
		{
			name:     "uid 클레임",
			claimKey: "uid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTestTokenWithClaim(tt.claimKey, userID, time.Hour, testJWTSecret)

			router := setupTestRouter(testJWTSecret)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("예상 상태 코드: %d, 실제: %d (클레임: %s)", http.StatusOK, w.Code, tt.claimKey)
			}
		})
	}
}

// TestJWTMiddleware_NoUserIDClaim은 사용자 ID 클레임이 없을 때 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_NoUserIDClaim(t *testing.T) {
	// 사용자 ID 클레임 없이 토큰 생성
	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testJWTSecret))

	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}

// TestGetUserID는 컨텍스트에서 사용자 ID를 올바르게 추출하는지 테스트합니다.
func TestGetUserID(t *testing.T) {
	t.Run("사용자 ID가 설정된 경우", func(t *testing.T) {
		userID := uuid.New()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", userID)

		got, ok := GetUserID(c)
		if !ok {
			t.Error("GetUserID가 true를 반환해야 합니다")
		}
		if got != userID {
			t.Errorf("예상 사용자 ID: %s, 실제: %s", userID, got)
		}
	})

	t.Run("사용자 ID가 설정되지 않은 경우", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		got, ok := GetUserID(c)
		if ok {
			t.Error("GetUserID가 false를 반환해야 합니다")
		}
		if got != uuid.Nil {
			t.Errorf("예상 사용자 ID: %s, 실제: %s", uuid.Nil, got)
		}
	})

	t.Run("잘못된 타입이 설정된 경우", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "not-a-uuid-type")

		_, ok := GetUserID(c)
		if ok {
			t.Error("GetUserID가 false를 반환해야 합니다")
		}
	})
}

// TestGetJWTToken은 컨텍스트에서 JWT 토큰을 올바르게 추출하는지 테스트합니다.
func TestGetJWTToken(t *testing.T) {
	t.Run("토큰이 설정된 경우", func(t *testing.T) {
		expectedToken := "test-jwt-token"
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("jwtToken", expectedToken)

		got, ok := GetJWTToken(c)
		if !ok {
			t.Error("GetJWTToken이 true를 반환해야 합니다")
		}
		if got != expectedToken {
			t.Errorf("예상 토큰: %s, 실제: %s", expectedToken, got)
		}
	})

	t.Run("토큰이 설정되지 않은 경우", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		got, ok := GetJWTToken(c)
		if ok {
			t.Error("GetJWTToken이 false를 반환해야 합니다")
		}
		if got != "" {
			t.Errorf("예상 토큰: 빈 문자열, 실제: %s", got)
		}
	})

	t.Run("잘못된 타입이 설정된 경우", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("jwtToken", 12345)

		_, ok := GetJWTToken(c)
		if ok {
			t.Error("GetJWTToken이 false를 반환해야 합니다")
		}
	})
}

// TestJWTMiddleware_InvalidToken은 완전히 잘못된 토큰으로 401을 반환하는지 테스트합니다.
func TestJWTMiddleware_InvalidToken(t *testing.T) {
	router := setupTestRouter(testJWTSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("예상 상태 코드: %d, 실제: %d", http.StatusUnauthorized, w.Code)
	}
}
