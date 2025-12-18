// Package logger는 구조화된 로깅 유틸리티를 제공합니다.
// 공통 모듈을 래핑하여 서비스에서 일관된 로깅을 제공합니다.
package logger

import (
	commonlogger "github.com/OrangesCloud/wealist-advanced-go-pkg/logger"
)

// Logger는 공통 모듈의 Logger 타입 별칭입니다.
type Logger = commonlogger.Logger

// New는 새 로거 인스턴스를 생성합니다.
// level: 로그 레벨 (debug, info, warn, error)
// outputPath: 로그 출력 경로 (stdout, stderr, 파일 경로)
func New(level, outputPath string) (*Logger, error) {
	return commonlogger.New(level, outputPath)
}
