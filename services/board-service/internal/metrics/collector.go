package metrics

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// BusinessMetricsCollector는 비즈니스 메트릭을 주기적으로 수집합니다.
// commonmetrics.MetricsCollector 인터페이스를 구현합니다.
type BusinessMetricsCollector struct {
	db      *gorm.DB
	metrics *Metrics
	logger  *zap.Logger
}

// NewBusinessMetricsCollector는 새 수집기를 생성합니다.
func NewBusinessMetricsCollector(db *gorm.DB, metrics *Metrics, logger *zap.Logger) *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		db:      db,
		metrics: metrics,
		logger:  logger,
	}
}

// Collect는 commonmetrics.MetricsCollector 인터페이스를 구현합니다.
// PeriodicCollector에 의해 주기적으로 호출됩니다.
func (c *BusinessMetricsCollector) Collect(ctx context.Context) error {
	// nil db를 우아하게 처리
	if c.db == nil {
		return nil
	}

	// 프로젝트 수 조회
	var projectCount int64
	if err := c.db.WithContext(ctx).Table("projects").Count(&projectCount).Error; err != nil {
		if c.logger != nil {
			c.logger.Error("프로젝트 수 조회 실패", zap.Error(err))
		}
	} else {
		c.metrics.SetProjectsTotal(projectCount)
	}

	// 보드 수 조회
	var boardCount int64
	if err := c.db.WithContext(ctx).Table("boards").Count(&boardCount).Error; err != nil {
		if c.logger != nil {
			c.logger.Error("보드 수 조회 실패", zap.Error(err))
		}
	} else {
		c.metrics.SetBoardsTotal(boardCount)
	}

	return nil
}

// StartPeriodicCollection은 이 수집기를 위한 PeriodicCollector를 생성하고 시작합니다.
// 나중에 중지할 수 있도록 수집기를 반환합니다.
func (c *BusinessMetricsCollector) StartPeriodicCollection() *commonmetrics.PeriodicCollector {
	collector := commonmetrics.NewPeriodicCollector(
		&commonmetrics.CollectorConfig{
			Logger: c.logger,
		},
		c,
	)
	collector.Start()
	return collector
}
