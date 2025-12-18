// Package logger는 zap 기반의 구조화된 로깅 유틸리티를 제공합니다.
// 이 파일은 logger.go의 테스트를 포함합니다.
package logger

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestDefaultConfig는 DefaultConfig가 올바른 기본값을 반환하는지 테스트합니다.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("예상 Level: 'info', 실제: '%s'", cfg.Level)
	}
	if cfg.OutputPath != "stdout" {
		t.Errorf("예상 OutputPath: 'stdout', 실제: '%s'", cfg.OutputPath)
	}
	if cfg.Encoding != "json" {
		t.Errorf("예상 Encoding: 'json', 실제: '%s'", cfg.Encoding)
	}
}

// TestNew는 New 함수가 로거를 올바르게 생성하는지 테스트합니다.
func TestNew(t *testing.T) {
	tests := []struct {
		name       string // 테스트 케이스 이름
		level      string // 로그 레벨
		outputPath string // 출력 경로
		wantErr    bool   // 에러 예상 여부
	}{
		{
			name:       "info 레벨로 생성",
			level:      "info",
			outputPath: "stdout",
			wantErr:    false,
		},
		{
			name:       "debug 레벨로 생성",
			level:      "debug",
			outputPath: "stdout",
			wantErr:    false,
		},
		{
			name:       "warn 레벨로 생성",
			level:      "warn",
			outputPath: "stdout",
			wantErr:    false,
		},
		{
			name:       "error 레벨로 생성",
			level:      "error",
			outputPath: "stdout",
			wantErr:    false,
		},
		{
			name:       "잘못된 레벨",
			level:      "invalid",
			outputPath: "stdout",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.level, tt.outputPath)

			if tt.wantErr {
				if err == nil {
					t.Error("에러가 발생해야 하는데 nil이 반환됨")
				}
				return
			}

			if err != nil {
				t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
				return
			}

			if logger == nil {
				t.Error("로거가 nil이면 안됨")
				return
			}

			// 로거가 정상 동작하는지 확인 (panic 없이)
			logger.Info("테스트 로그 메시지")
		})
	}
}

// TestNewWithConfig는 NewWithConfig 함수가 올바르게 동작하는지 테스트합니다.
func TestNewWithConfig(t *testing.T) {
	t.Run("nil config로 생성", func(t *testing.T) {
		// nil config는 DefaultConfig를 사용해야 함
		logger, err := NewWithConfig(nil)
		if err != nil {
			t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
		}
		if logger == nil {
			t.Error("로거가 nil이면 안됨")
		}
	})

	t.Run("custom config로 생성", func(t *testing.T) {
		cfg := &Config{
			Level:      "debug",
			OutputPath: "stdout",
			Encoding:   "json",
		}
		logger, err := NewWithConfig(cfg)
		if err != nil {
			t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
		}
		if logger == nil {
			t.Error("로거가 nil이면 안됨")
		}
	})

	t.Run("console encoding으로 생성", func(t *testing.T) {
		cfg := &Config{
			Level:      "info",
			OutputPath: "stdout",
			Encoding:   "console",
		}
		logger, err := NewWithConfig(cfg)
		if err != nil {
			t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
		}
		if logger == nil {
			t.Error("로거가 nil이면 안됨")
		}
	})

	t.Run("빈 encoding은 json으로 기본 설정", func(t *testing.T) {
		cfg := &Config{
			Level:      "info",
			OutputPath: "stdout",
			Encoding:   "",
		}
		logger, err := NewWithConfig(cfg)
		if err != nil {
			t.Errorf("에러가 없어야 하는데 %v가 반환됨", err)
		}
		if logger == nil {
			t.Error("로거가 nil이면 안됨")
		}
	})

	t.Run("잘못된 레벨로 생성 실패", func(t *testing.T) {
		cfg := &Config{
			Level:      "unknown",
			OutputPath: "stdout",
			Encoding:   "json",
		}
		_, err := NewWithConfig(cfg)
		if err == nil {
			t.Error("에러가 발생해야 함")
		}
	})
}

// TestWithFields는 WithFields가 새 로거를 올바르게 반환하는지 테스트합니다.
func TestWithFields(t *testing.T) {
	// 관찰 가능한 로거 생성
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	// 필드 추가
	newLogger := logger.WithFields(zap.String("key", "value"))

	// 새 로거가 nil이 아닌지 확인
	if newLogger == nil {
		t.Error("새 로거가 nil이면 안됨")
		return
	}

	// 원래 로거와 다른 인스턴스인지 확인
	if newLogger == logger {
		t.Error("새 로거는 원래 로거와 다른 인스턴스여야 함")
	}

	// 새 로거로 로그 기록
	newLogger.Info("테스트 메시지")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// 필드가 포함되어 있는지 확인
	contextMap := logs[0].ContextMap()
	if contextMap["key"] != "value" {
		t.Errorf("필드 'key'의 값이 'value'여야 하는데 '%v'임", contextMap["key"])
	}
}

// TestWithRequestID는 WithRequestID가 request_id 필드를 추가하는지 테스트합니다.
func TestWithRequestID(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	requestID := "test-request-id-12345"
	newLogger := logger.WithRequestID(requestID)

	// 로그 기록
	newLogger.Info("테스트 메시지")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// request_id 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["request_id"] != requestID {
		t.Errorf("request_id가 '%s'여야 하는데 '%v'임", requestID, contextMap["request_id"])
	}
}

// TestWithUserID는 WithUserID가 user_id 필드를 추가하는지 테스트합니다.
func TestWithUserID(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	userID := "user-uuid-67890"
	newLogger := logger.WithUserID(userID)

	// 로그 기록
	newLogger.Info("테스트 메시지")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// user_id 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["user_id"] != userID {
		t.Errorf("user_id가 '%s'여야 하는데 '%v'임", userID, contextMap["user_id"])
	}
}

// TestWithError는 WithError가 error 필드를 추가하는지 테스트합니다.
func TestWithError(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	testErr := errors.New("테스트 에러")
	newLogger := logger.WithError(testErr)

	// 로그 기록
	newLogger.Info("에러 발생")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// error 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["error"] != testErr.Error() {
		t.Errorf("error가 '%s'여야 하는데 '%v'임", testErr.Error(), contextMap["error"])
	}
}

// TestWithService는 WithService가 service 필드를 추가하는지 테스트합니다.
func TestWithService(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	serviceName := "board-service"
	newLogger := logger.WithService(serviceName)

	// 로그 기록
	newLogger.Info("서비스 시작")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// service 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["service"] != serviceName {
		t.Errorf("service가 '%s'여야 하는데 '%v'임", serviceName, contextMap["service"])
	}
}

// TestLoggerChaining은 여러 With* 메서드를 체이닝할 수 있는지 테스트합니다.
func TestLoggerChaining(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	// 여러 필드 체이닝
	chainedLogger := logger.
		WithService("test-service").
		WithRequestID("req-123").
		WithUserID("user-456")

	// 로그 기록
	chainedLogger.Info("체이닝 테스트")

	// 기록된 로그 확인
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("1개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// 모든 필드 확인
	contextMap := logs[0].ContextMap()
	if contextMap["service"] != "test-service" {
		t.Errorf("service 필드가 누락됨")
	}
	if contextMap["request_id"] != "req-123" {
		t.Errorf("request_id 필드가 누락됨")
	}
	if contextMap["user_id"] != "user-456" {
		t.Errorf("user_id 필드가 누락됨")
	}
}

// TestLogLevels는 다양한 로그 레벨이 올바르게 동작하는지 테스트합니다.
func TestLogLevels(t *testing.T) {
	// Debug 레벨로 설정하여 모든 로그가 기록되도록 함
	core, recorded := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	// 각 레벨로 로그 기록
	logger.Debug("debug 메시지")
	logger.Info("info 메시지")
	logger.Warn("warn 메시지")
	logger.Error("error 메시지")

	// 기록된 로그 수 확인
	logs := recorded.All()
	if len(logs) != 4 {
		t.Errorf("4개의 로그가 기록되어야 하는데 %d개가 기록됨", len(logs))
		return
	}

	// 각 로그 레벨 확인
	expectedLevels := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}

	for i, log := range logs {
		if log.Level != expectedLevels[i] {
			t.Errorf("로그 %d의 레벨이 %s여야 하는데 %s임", i, expectedLevels[i], log.Level)
		}
	}
}

// TestParseLevel은 parseLevel 함수를 테스트합니다.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string        // 테스트 케이스 이름
		input     string        // 입력 레벨 문자열
		wantLevel zapcore.Level // 예상 레벨
		wantErr   bool          // 에러 예상 여부
	}{
		{
			name:      "debug 레벨",
			input:     "debug",
			wantLevel: zapcore.DebugLevel,
			wantErr:   false,
		},
		{
			name:      "info 레벨",
			input:     "info",
			wantLevel: zapcore.InfoLevel,
			wantErr:   false,
		},
		{
			name:      "warn 레벨",
			input:     "warn",
			wantLevel: zapcore.WarnLevel,
			wantErr:   false,
		},
		{
			name:      "error 레벨",
			input:     "error",
			wantLevel: zapcore.ErrorLevel,
			wantErr:   false,
		},
		{
			name:      "잘못된 레벨",
			input:     "invalid",
			wantLevel: zapcore.InfoLevel, // 에러 시 기본값
			wantErr:   true,
		},
		{
			name:      "빈 문자열",
			input:     "",
			wantLevel: zapcore.InfoLevel,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLevel(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("에러가 발생해야 함")
				}
				return
			}

			if err != nil {
				t.Errorf("에러가 없어야 하는데 %v가 발생함", err)
				return
			}

			if level != tt.wantLevel {
				t.Errorf("레벨이 %s여야 하는데 %s임", tt.wantLevel, level)
			}
		})
	}
}

// TestNopLogger는 NopLogger가 인터페이스를 구현하고 올바르게 동작하는지 테스트합니다.
func TestNopLogger(t *testing.T) {
	nop := NewNopLogger()

	// 모든 메서드가 panic 없이 호출되는지 확인
	nop.Debug("debug")
	nop.Info("info")
	nop.Warn("warn")
	nop.Error("error")

	// With가 *zap.Logger를 반환하는지 확인
	withLogger := nop.With()
	if withLogger == nil {
		t.Error("With()가 nil을 반환하면 안됨")
	}

	// Sync가 에러 없이 동작하는지 확인
	if err := nop.Sync(); err != nil {
		t.Errorf("Sync()가 에러를 반환하면 안됨: %v", err)
	}
}

// TestLoggerImplementsILogger는 Logger가 ILogger 인터페이스를 구현하는지 테스트합니다.
func TestLoggerImplementsILogger(t *testing.T) {
	logger, err := New("info", "stdout")
	if err != nil {
		t.Fatalf("로거 생성 실패: %v", err)
	}

	// 인터페이스 구현 확인 (컴파일 타임 체크)
	var _ ILogger = logger
}

// TestNopLoggerImplementsILogger는 NopLogger가 ILogger 인터페이스를 구현하는지 테스트합니다.
func TestNopLoggerImplementsILogger(t *testing.T) {
	var _ ILogger = &NopLogger{}
}
