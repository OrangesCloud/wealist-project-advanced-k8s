// Package metrics는 애플리케이션의 Prometheus 메트릭을 제공합니다.
package metrics

// IncrementProjectCreated는 프로젝트 생성 카운터를 증가시킵니다.
func (m *Metrics) IncrementProjectCreated() {
	if m.ProjectCreatedTotal != nil {
		m.ProjectCreatedTotal.Inc()
	}
}

// IncrementBoardCreated는 보드 생성 카운터를 증가시킵니다.
func (m *Metrics) IncrementBoardCreated() {
	if m.BoardCreatedTotal != nil {
		m.BoardCreatedTotal.Inc()
	}
}

// SetProjectsTotal은 총 프로젝트 수 게이지를 설정합니다.
func (m *Metrics) SetProjectsTotal(count int64) {
	if m.ProjectsTotal != nil {
		m.ProjectsTotal.Set(float64(count))
	}
}

// SetBoardsTotal은 총 보드 수 게이지를 설정합니다.
func (m *Metrics) SetBoardsTotal(count int64) {
	if m.BoardsTotal != nil {
		m.BoardsTotal.Set(float64(count))
	}
}
