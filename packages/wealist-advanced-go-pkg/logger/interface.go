// Package logger는 zap 기반의 구조화된 로깅 유틸리티를 제공합니다.
// 이 파일은 로거 인터페이스를 정의합니다.
package logger

import "go.uber.org/zap"

// ILogger는 구조화된 로깅을 위한 인터페이스입니다.
// 이 인터페이스를 사용하면 테스트에서 로거를 mock할 수 있습니다.
//
// 사용 예:
//
//	type Service struct {
//	    logger logger.ILogger
//	}
//
//	// 테스트에서 mock 로거 사용
//	type mockLogger struct{}
//	func (m *mockLogger) Info(msg string, fields ...zap.Field) {}
//	// ... 다른 메서드 구현
type ILogger interface {
	// Debug는 디버그 레벨 로그를 기록합니다.
	Debug(msg string, fields ...zap.Field)

	// Info는 정보 레벨 로그를 기록합니다.
	Info(msg string, fields ...zap.Field)

	// Warn는 경고 레벨 로그를 기록합니다.
	Warn(msg string, fields ...zap.Field)

	// Error는 에러 레벨 로그를 기록합니다.
	Error(msg string, fields ...zap.Field)

	// With는 추가 필드가 있는 새 로거를 반환합니다.
	// ILogger 대신 *zap.Logger를 반환하는 이유는 zap.Logger의 메서드 시그니처를 따르기 위함입니다.
	With(fields ...zap.Field) *zap.Logger

	// Sync는 버퍼된 로그를 플러시합니다.
	Sync() error
}

// 컴파일 타임에 Logger가 ILogger 인터페이스를 구현하는지 확인합니다.
var _ ILogger = (*Logger)(nil)

// NopLogger는 아무것도 출력하지 않는 로거입니다.
// 테스트에서 로그 출력이 필요 없을 때 사용합니다.
type NopLogger struct{}

// Debug는 아무것도 하지 않습니다.
func (n *NopLogger) Debug(msg string, fields ...zap.Field) {}

// Info는 아무것도 하지 않습니다.
func (n *NopLogger) Info(msg string, fields ...zap.Field) {}

// Warn는 아무것도 하지 않습니다.
func (n *NopLogger) Warn(msg string, fields ...zap.Field) {}

// Error는 아무것도 하지 않습니다.
func (n *NopLogger) Error(msg string, fields ...zap.Field) {}

// With는 자기 자신을 반환합니다.
func (n *NopLogger) With(fields ...zap.Field) *zap.Logger {
	return zap.NewNop()
}

// Sync는 아무것도 하지 않습니다.
func (n *NopLogger) Sync() error { return nil }

// 컴파일 타임에 NopLogger가 ILogger 인터페이스를 구현하는지 확인합니다.
var _ ILogger = (*NopLogger)(nil)

// NewNopLogger는 아무것도 출력하지 않는 로거를 반환합니다.
// 테스트에서 로그 출력이 필요 없을 때 사용합니다.
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}
