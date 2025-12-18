// Package middleware는 HTTP 미들웨어를 제공합니다.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonauth "github.com/OrangesCloud/wealist-advanced-go-pkg/auth"
)

// Auth는 JWT 토큰을 검증하는 미들웨어를 반환합니다.
// 공통 모듈을 사용하여 하위 호환성을 유지합니다.
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
