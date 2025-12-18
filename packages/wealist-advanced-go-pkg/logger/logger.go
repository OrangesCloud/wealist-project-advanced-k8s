// Package logger는 zap 기반의 구조화된 로깅 유틸리티를 제공합니다.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger는 zap.Logger를 래핑합니다.
type Logger struct {
	*zap.Logger
}

// Config는 로거 설정입니다.
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // stdout, stderr, or file path
	Encoding   string // json or console
}

// DefaultConfig는 기본 로거 설정을 반환합니다.
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		OutputPath: "stdout",
		Encoding:   "json",
	}
}

// New는 새 로거 인스턴스를 생성합니다.
func New(level, outputPath string) (*Logger, error) {
	return NewWithConfig(&Config{
		Level:      level,
		OutputPath: outputPath,
		Encoding:   "json",
	})
}

// NewWithConfig는 설정으로 새 로거 인스턴스를 생성합니다.
func NewWithConfig(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 로그 레벨 파싱
	zapLevel, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// 인코더 설정
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// zap 설정
	encoding := cfg.Encoding
	if encoding == "" {
		encoding = "json"
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{cfg.OutputPath},
		ErrorOutputPaths: []string{cfg.OutputPath},
	}

	// 로거 빌드
	zapLogger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("로거 빌드 실패: %w", err)
	}

	return &Logger{Logger: zapLogger}, nil
}

// parseLevel은 문자열 로그 레벨을 파싱합니다.
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("잘못된 로그 레벨: %s", level)
	}
}

// WithFields는 추가 필드가 있는 로거를 반환합니다.
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.With(fields...)}
}

// WithRequestID는 요청 ID 필드가 있는 로거를 반환합니다.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithFields(zap.String("request_id", requestID))
}

// WithUserID는 사용자 ID 필드가 있는 로거를 반환합니다.
func (l *Logger) WithUserID(userID string) *Logger {
	return l.WithFields(zap.String("user_id", userID))
}

// WithError는 에러 필드가 있는 로거를 반환합니다.
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(zap.Error(err))
}

// WithService는 서비스 이름 필드가 있는 로거를 반환합니다.
func (l *Logger) WithService(serviceName string) *Logger {
	return l.WithFields(zap.String("service", serviceName))
}
