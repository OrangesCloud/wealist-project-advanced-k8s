// Package auth는 JWT 인증 미들웨어를 제공합니다.
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Config는 인증 미들웨어 설정입니다.
type Config struct {
	JWTSecret string
}

// JWTMiddleware는 JWT 토큰을 검증하는 미들웨어를 반환합니다.
func JWTMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization 헤더 확인
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
				"message": "인증이 필요합니다",
			})
			c.Abort()
			return
		}

		// "Bearer <token>" 형식에서 토큰 추출
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
				"message": "잘못된 인증 헤더 형식입니다",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 토큰 파싱 및 검증
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 서명 방식 검증
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
				"message": "유효하지 않거나 만료된 토큰입니다",
			})
			c.Abort()
			return
		}

		// 클레임에서 사용자 정보 추출
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid token claims",
				},
				"message": "유효하지 않은 토큰 정보입니다",
			})
			c.Abort()
			return
		}

		// 사용자 ID 추출 (여러 클레임 형식 지원)
		var userIDStr string
		if uid, ok := claims["user_id"].(string); ok {
			userIDStr = uid
		} else if sub, ok := claims["sub"].(string); ok {
			userIDStr = sub
		} else if uid, ok := claims["uid"].(string); ok {
			userIDStr = uid
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User ID not found in token",
				},
				"message": "토큰에서 사용자 ID를 찾을 수 없습니다",
			})
			c.Abort()
			return
		}

		// UUID 파싱
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid user ID format",
				},
				"message": "유효하지 않은 사용자 ID 형식입니다",
			})
			c.Abort()
			return
		}

		// 컨텍스트에 사용자 정보 저장
		c.Set("user_id", userID)
		c.Set("jwtToken", tokenString)

		c.Next()
	}
}

// GetUserID는 컨텍스트에서 사용자 ID를 추출합니다.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := userID.(uuid.UUID)
	return id, ok
}

// GetJWTToken은 컨텍스트에서 JWT 토큰을 추출합니다.
func GetJWTToken(c *gin.Context) (string, bool) {
	token, exists := c.Get("jwtToken")
	if !exists {
		return "", false
	}
	tokenStr, ok := token.(string)
	return tokenStr, ok
}
