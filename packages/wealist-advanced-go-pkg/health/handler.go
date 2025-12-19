// Package health는 K8s 헬스체크 엔드포인트를 제공합니다.
// 이 파일은 Kubernetes liveness/readiness probe를 위한 핸들러를 포함합니다.
package health

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthChecker는 K8s 표준 헬스체크 엔드포인트를 제공하는 구조체입니다.
type HealthChecker struct {
	db    *gorm.DB       // 데이터베이스 연결 (nil 가능)
	redis *redis.Client  // Redis 연결 (nil 가능, 선택적)
}

// NewHealthChecker는 새 HealthChecker 인스턴스를 생성합니다.
// db: 데이터베이스 연결 (nil 가능 - DB가 없는 서비스용)
// redis: Redis 연결 (nil 가능 - Redis가 없는 서비스용)
func NewHealthChecker(db *gorm.DB, redis *redis.Client) *HealthChecker {
	return &HealthChecker{
		db:    db,
		redis: redis,
	}
}

// RegisterRoutes는 헬스체크 엔드포인트를 루트 레벨과 basePath 아래에 등록합니다.
// Pod 직접 접근과 Ingress를 통한 접근 모두에서 헬스체크가 동작하도록 합니다.
//
// 등록되는 엔드포인트:
//   - /health/live: liveness probe
//   - /health/ready: readiness probe
//   - /health: readiness probe (하위 호환성)
//   - {basePath}/health/live, {basePath}/health/ready, {basePath}/health (basePath가 있는 경우)
func (h *HealthChecker) RegisterRoutes(router gin.IRouter, basePath string) {
	// 루트 레벨 엔드포인트 (K8s probe의 기본 경로)
	router.GET("/health/live", h.Liveness)
	router.GET("/health/ready", h.Readiness)
	router.GET("/health", h.Readiness) // 하위 호환성

	// basePath 아래 엔드포인트 (Ingress 라우팅용)
	if basePath != "" {
		group := router.Group(basePath)
		group.GET("/health/live", h.Liveness)
		group.GET("/health/ready", h.Readiness)
		group.GET("/health", h.Readiness)
	}
}

// Liveness는 애플리케이션 프로세스가 실행 중인지 확인합니다.
// K8s liveness probe용 - 의존성(DB, Redis)은 확인하지 않습니다.
// 프로세스가 살아있으면 200을 반환합니다 (항상 200을 반환해야 함).
func (h *HealthChecker) Liveness(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "alive",
	})
}

// Readiness는 서비스가 트래픽을 받을 준비가 되었는지 확인합니다.
// K8s readiness probe용 - DB와 Redis 연결을 확인합니다.
// 준비되면 200, 준비 안됨이면 503을 반환합니다.
func (h *HealthChecker) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := gin.H{}
	isReady := true

	// 데이터베이스 연결 확인
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			checks["database"] = "error"
			isReady = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			checks["database"] = "disconnected"
			isReady = false
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "not_configured"
	}

	// Redis 연결 확인 (선택적)
	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = "disconnected"
			isReady = false
		} else {
			checks["redis"] = "ok"
		}
	} else {
		checks["redis"] = "not_configured"
	}

	// 결과 반환
	if isReady {
		c.JSON(200, gin.H{
			"status": "ready",
			"checks": checks,
		})
	} else {
		c.JSON(503, gin.H{
			"status": "not_ready",
			"checks": checks,
		})
	}
}
