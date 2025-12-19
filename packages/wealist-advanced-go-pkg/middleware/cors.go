// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 CORS(Cross-Origin Resource Sharing) 미들웨어를 포함합니다.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig는 CORS 설정을 담는 구조체입니다.
type CORSConfig struct {
	AllowedOrigins   []string // 허용할 origin 목록 (예: ["http://example.com"])
	AllowedMethods   []string // 허용할 HTTP 메서드 (예: ["GET", "POST"])
	AllowedHeaders   []string // 허용할 요청 헤더
	ExposedHeaders   []string // 클라이언트에 노출할 응답 헤더
	AllowCredentials bool     // credentials(쿠키, 인증) 허용 여부
	MaxAge           int      // preflight 캐시 시간(초)
}

// DefaultCORSConfig는 기본 CORS 설정을 반환합니다.
// 개발 환경에 적합한 관대한 설정을 제공합니다.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Workspace-Id"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // 24시간
	}
}

// CORS는 CORS 처리를 위한 Gin 미들웨어를 반환합니다.
// config에 따라 CORS 헤더를 설정하고 preflight 요청을 처리합니다.
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// origin이 허용 목록에 있는지 확인
		allowOrigin := "*"
		if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] != "*" {
			for _, allowed := range config.AllowedOrigins {
				if allowed == origin {
					allowOrigin = origin
					break
				}
			}
		} else if origin != "" {
			allowOrigin = origin
		}

		// CORS 응답 헤더 설정
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// preflight 요청 처리 (OPTIONS 메서드)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// DefaultCORS는 기본 설정으로 CORS 미들웨어를 반환합니다.
func DefaultCORS() gin.HandlerFunc {
	return CORS(DefaultCORSConfig())
}

// CORSWithOrigins는 지정된 origin만 허용하는 CORS 미들웨어를 반환합니다.
// origins는 쉼표로 구분된 origin 목록입니다 (예: "http://a.com, http://b.com").
func CORSWithOrigins(origins string) gin.HandlerFunc {
	config := DefaultCORSConfig()
	if origins != "" && origins != "*" {
		config.AllowedOrigins = strings.Split(origins, ",")
		// 각 origin의 공백 제거
		for i := range config.AllowedOrigins {
			config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		}
	}
	return CORS(config)
}
