// Package testutil은 테스트 유틸리티를 제공합니다.
// 이 파일은 테스트에서 사용할 수 있는 로거 mock을 제공합니다.
package testutil

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// NewTestLogger는 테스트용 로거를 생성합니다.
// 테스트가 실패하거나 -v 플래그가 있을 때만 로그가 출력됩니다.
// uber-go/zap의 zaptest 패턴을 따릅니다.
//
// 사용 예:
//
//	func TestSomething(t *testing.T) {
//	    logger := testutil.NewTestLogger(t)
//	    // logger 사용...
//	}
func NewTestLogger(t testing.TB) *zap.Logger {
	return zaptest.NewLogger(t)
}

// NewNopLogger는 아무것도 출력하지 않는 로거를 반환합니다.
// 로그 출력이 필요 없는 테스트에서 사용합니다.
//
// 사용 예:
//
//	logger := testutil.NewNopLogger()
//	service := NewService(logger)
func NewNopLogger() *zap.Logger {
	return zap.NewNop()
}

// ObservedLogger는 로그 출력을 관찰(검증)할 수 있는 로거입니다.
// 특정 로그가 기록되었는지 확인할 때 사용합니다.
type ObservedLogger struct {
	Logger   *zap.Logger
	Observer *observer.ObservedLogs
}

// NewObservedLogger는 로그 출력을 관찰할 수 있는 로거를 생성합니다.
// 테스트에서 특정 로그 메시지가 기록되었는지 검증할 때 사용합니다.
//
// 사용 예:
//
//	observed := testutil.NewObservedLogger(zapcore.InfoLevel)
//	service := NewService(observed.Logger)
//	service.DoSomething()
//	logs := observed.Observer.All()
//	if len(logs) == 0 {
//	    t.Error("expected log entry")
//	}
func NewObservedLogger(level zapcore.Level) *ObservedLogger {
	core, recorded := observer.New(level)
	logger := zap.New(core)
	return &ObservedLogger{
		Logger:   logger,
		Observer: recorded,
	}
}

// AllLogs는 기록된 모든 로그 엔트리를 반환합니다.
func (o *ObservedLogger) AllLogs() []observer.LoggedEntry {
	return o.Observer.All()
}

// FilterByLevel은 특정 레벨의 로그만 필터링합니다.
func (o *ObservedLogger) FilterByLevel(level zapcore.Level) []observer.LoggedEntry {
	return o.Observer.FilterLevelExact(level).All()
}

// FilterByMessage는 특정 메시지를 포함하는 로그를 필터링합니다.
func (o *ObservedLogger) FilterByMessage(msg string) []observer.LoggedEntry {
	return o.Observer.FilterMessage(msg).All()
}

// HasLog는 특정 레벨과 메시지의 로그가 있는지 확인합니다.
func (o *ObservedLogger) HasLog(level zapcore.Level, message string) bool {
	logs := o.Observer.FilterLevelExact(level).FilterMessage(message).All()
	return len(logs) > 0
}

// LogCount는 기록된 로그의 총 개수를 반환합니다.
func (o *ObservedLogger) LogCount() int {
	return o.Observer.Len()
}

// Reset은 기록된 로그를 초기화합니다.
// 여러 테스트 케이스에서 같은 ObservedLogger를 재사용할 때 유용합니다.
func (o *ObservedLogger) Reset() {
	o.Observer.TakeAll()
}

// AssertHasLog는 특정 레벨과 메시지의 로그가 있는지 검증합니다.
// 테스트 실패 시 t.Error를 호출합니다.
func (o *ObservedLogger) AssertHasLog(t testing.TB, level zapcore.Level, message string) {
	t.Helper()
	if !o.HasLog(level, message) {
		t.Errorf("expected log with level=%s, message=%q but not found", level.String(), message)
	}
}

// AssertLogCount는 로그 개수가 예상값과 일치하는지 검증합니다.
func (o *ObservedLogger) AssertLogCount(t testing.TB, expected int) {
	t.Helper()
	actual := o.LogCount()
	if actual != expected {
		t.Errorf("expected %d logs, got %d", expected, actual)
	}
}
