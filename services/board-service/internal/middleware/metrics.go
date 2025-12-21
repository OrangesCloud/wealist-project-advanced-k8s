package middleware

import (
	"github.com/gin-gonic/gin"

	commonmiddleware "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
	"project-board-api/internal/metrics"
)

// Metrics는 HTTP 메트릭을 기록하는 미들웨어를 반환합니다.
func Metrics(m *metrics.Metrics) gin.HandlerFunc {
	// board-service의 basePath는 "/api/boards"
	return commonmiddleware.MetricsMiddleware(m.Metrics, "/api/boards")
}
