// Package middleware는 HTTP 미들웨어를 제공합니다.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// MetricsMiddleware는 HTTP 메트릭을 기록하는 미들웨어를 반환합니다.
// basePath는 서비스의 기본 경로입니다 (예: "/api/boards")
func MetricsMiddleware(m *commonmetrics.Metrics, basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 메트릭/헬스 엔드포인트 스킵
		if commonmetrics.ShouldSkipEndpointWithBasePath(c.Request.URL.Path, basePath) {
			c.Next()
			return
		}

		start := time.Now()

		// 요청 처리
		c.Next()

		// 메트릭 기록
		duration := time.Since(start)
		m.RecordHTTPRequest(
			c.Request.Method,
			c.FullPath(), // 실제 경로가 아닌 라우트 패턴 사용
			c.Writer.Status(),
			duration,
		)
	}
}

// MetricsMiddlewareSimple은 basePath 없이 메트릭을 기록하는 미들웨어입니다.
func MetricsMiddlewareSimple(m *commonmetrics.Metrics) gin.HandlerFunc {
	return MetricsMiddleware(m, "")
}
