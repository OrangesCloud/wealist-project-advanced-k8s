package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const (
	namespace = "board_service"
)

// Metrics는 공통 메트릭과 board-service 전용 비즈니스 메트릭을 래핑합니다.
// 공통 모듈을 사용하면서 하위 호환성을 유지합니다.
type Metrics struct {
	// 공통 메트릭 임베딩
	*commonmetrics.Metrics

	// 비즈니스 메트릭 (board-service 전용)
	ProjectsTotal       prometheus.Gauge   // 활성 프로젝트 총 개수
	BoardsTotal         prometheus.Gauge   // 보드 총 개수
	ProjectCreatedTotal prometheus.Counter // 프로젝트 생성 이벤트 총 횟수
	BoardCreatedTotal   prometheus.Counter // 보드 생성 이벤트 총 횟수
}

// New는 기본 레지스트리에 모든 메트릭을 생성하고 등록합니다.
func New() *Metrics {
	return NewWithLogger(nil)
}

// NewTestMetrics는 테스트용 격리된 레지스트리로 메트릭을 생성합니다.
// 테스트에서 "duplicate metrics collector registration" 에러를 방지합니다.
func NewTestMetrics() *Metrics {
	common := commonmetrics.NewForTest(namespace)
	return newWithCommon(common)
}

// NewWithLogger는 기본 레지스트리와 로거로 모든 메트릭을 생성하고 등록합니다.
func NewWithLogger(logger *zap.Logger) *Metrics {
	common := commonmetrics.New(&commonmetrics.Config{
		Namespace: namespace,
		Logger:    logger,
		Registry:  prometheus.DefaultRegisterer,
	})
	return newWithCommon(common)
}

// NewWithRegistry는 커스텀 레지스트리로 모든 메트릭을 생성하고 등록합니다.
func NewWithRegistry(registerer prometheus.Registerer, logger *zap.Logger) *Metrics {
	common := commonmetrics.New(&commonmetrics.Config{
		Namespace: namespace,
		Logger:    logger,
		Registry:  registerer,
	})
	return newWithCommon(common)
}

// newWithCommon은 공통 메트릭을 래핑하고 비즈니스 메트릭을 추가하여 Metrics를 생성합니다.
func newWithCommon(common *commonmetrics.Metrics) *Metrics {
	return &Metrics{
		Metrics: common,

		// 공통 모듈의 헬퍼를 사용하여 비즈니스 메트릭 등록
		ProjectsTotal:       common.RegisterGauge("projects_total", "Total number of active projects"),
		BoardsTotal:         common.RegisterGauge("boards_total", "Total number of boards"),
		ProjectCreatedTotal: common.RegisterCounter("project_created_total", "Total number of project creation events"),
		BoardCreatedTotal:   common.RegisterCounter("board_created_total", "Total number of board creation events"),
	}
}
