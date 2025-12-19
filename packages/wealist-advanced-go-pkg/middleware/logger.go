// Package middleware는 Gin 기반 서비스를 위한 HTTP 미들웨어를 제공합니다.
// 이 파일은 요청 로깅 미들웨어를 포함합니다.
package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 기본적으로 로깅에서 제외할 경로들 (health check, metrics 등)
var defaultSkipPaths = []string{
	"/health",
	"/health/live",
	"/health/ready",
	"/metrics",
	"/readyz",
	"/livez",
}

// RequestIDKey는 컨텍스트에서 요청 ID를 저장/조회하기 위한 키입니다.
const RequestIDKey = "request_id"

// Logger는 HTTP 요청을 구조화된 로그로 기록하는 미들웨어를 반환합니다.
// 각 요청에 대해 UUID 기반 request_id를 생성하고, 요청/응답 정보를 로깅합니다.
// 상태 코드에 따라 로그 레벨이 결정됩니다:
//   - 5xx: Error
//   - 4xx: Warn
//   - 그 외: Info
//
// Health check, metrics 등의 **성공 응답(2xx)**은 자동으로 로깅에서 제외됩니다.
// 에러 응답(4xx, 5xx)은 항상 로깅됩니다 (디버깅용).
// 제외되는 경로: /health, /health/live, /health/ready, /metrics, /readyz, /livez
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 ID 생성 및 설정 (스킵 경로에서도 request_id는 설정)
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		path := c.Request.URL.Path
		isHealthPath := isHealthCheckPath(path)

		// 타이머 시작
		start := time.Now()
		query := c.Request.URL.RawQuery

		// 요청 처리
		c.Next()

		// 처리 시간 계산
		duration := time.Since(start)

		// 상태 코드 가져오기
		statusCode := c.Writer.Status()

		// Health check 성공(2xx)은 로깅 스킵 (노이즈 감소)
		// 에러(4xx, 5xx)는 항상 로깅 (디버깅용)
		if isHealthPath && statusCode < 400 {
			return
		}

		// 로그 필드 구성
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// 인증 미들웨어에서 설정한 user_id가 있으면 추가
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// 에러가 있으면 추가
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// 상태 코드에 따른 로그 레벨 결정
		if statusCode >= 500 {
			logger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}
	}
}

// isHealthCheckPath는 주어진 경로가 health check 또는 metrics 경로인지 확인합니다.
func isHealthCheckPath(path string) bool {
	for _, skipPath := range defaultSkipPaths {
		if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
			return true
		}
		// basePath가 있는 경우도 처리 (e.g., /api/boards/health/live)
		if strings.HasSuffix(path, skipPath) {
			return true
		}
	}
	return false
}

// SkipPathLogger는 특정 경로를 로깅에서 제외하는 미들웨어를 반환합니다.
// /health, /metrics 등 노이즈가 많은 경로를 제외할 때 사용합니다.
func SkipPathLogger(logger *zap.Logger, skipPaths ...string) gin.HandlerFunc {
	// 스킵할 경로를 map으로 변환하여 O(1) 조회
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 스킵 경로면 로깅 없이 다음 핸들러로 진행
		if skipMap[path] {
			c.Next()
			return
		}

		// 일반 로거 사용
		Logger(logger)(c)
	}
}

// GetRequestID는 컨텍스트에서 요청 ID를 가져옵니다.
// 요청 ID가 없으면 새로운 UUID를 생성하여 반환합니다.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return uuid.New().String()
}
