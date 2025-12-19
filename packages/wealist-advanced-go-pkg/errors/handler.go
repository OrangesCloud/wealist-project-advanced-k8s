// Package errors는 애플리케이션 에러 처리를 제공합니다.
// 이 파일은 HTTP 에러 핸들러와 응답 유틸리티를 포함합니다.
package errors

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ErrorResponse는 에러 API 응답의 JSON 구조체입니다.
type ErrorResponse struct {
	Code    string `json:"code"`              // 에러 코드 (예: NOT_FOUND, VALIDATION_ERROR)
	Message string `json:"message"`           // 사용자에게 보여줄 메시지
	Details string `json:"details,omitempty"` // 추가 상세 정보 (선택적)
}

// Handler는 HTTP 핸들러에서 에러를 처리하기 위한 구조체입니다.
// 로깅과 응답 생성을 담당합니다.
type Handler struct {
	logger *zap.Logger
}

// NewHandler는 주어진 로거로 새 에러 핸들러를 생성합니다.
// logger가 nil이면 zap.NewNop()을 사용합니다.
func NewHandler(logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{logger: logger}
}

// HandleError는 에러를 처리하고 적절한 HTTP 응답을 전송합니다.
// 에러를 로깅하고 적절한 HTTP 상태 코드와 응답 본문을 생성합니다.
// 처리 순서:
//  1. GORM ErrRecordNotFound → 404
//  2. AppError → 에러 코드에 따른 상태 코드
//  3. 기타 에러 → 500
func (h *Handler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 에러 로깅
	h.logger.Error("Request error",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method))

	// GORM의 레코드 없음 에러 확인
	if errors.Is(err, gorm.ErrRecordNotFound) {
		h.sendError(c, http.StatusNotFound, ErrCodeNotFound, "Resource not found", "")
		return
	}

	// AppError 타입 확인
	var appErr *AppError
	if errors.As(err, &appErr) {
		h.logger.Debug("AppError details",
			zap.String("code", appErr.Code),
			zap.String("message", appErr.Message),
			zap.String("details", appErr.Details))

		statusCode := GetHTTPStatus(appErr.Code)
		h.sendError(c, statusCode, appErr.Code, appErr.Message, appErr.Details)
		return
	}

	// 처리되지 않은 에러는 500 반환
	h.logger.Warn("Unhandled error type",
		zap.String("type", err.Error()),
		zap.Error(err))
	h.sendError(c, http.StatusInternalServerError, ErrCodeInternal, "Internal server error", "")
}

// sendError는 클라이언트에 에러 응답을 전송합니다.
func (h *Handler) sendError(c *gin.Context, statusCode int, code, message, details string) {
	c.JSON(statusCode, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// SendError는 Handler 인스턴스 없이 에러 응답을 전송하는 헬퍼 함수입니다.
func SendError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// SendErrorWithDetails는 상세 정보와 함께 에러 응답을 전송합니다.
func SendErrorWithDetails(c *gin.Context, statusCode int, code, message, details string) {
	c.JSON(statusCode, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// HandleAppError는 AppError를 직접 처리하는 편의 함수입니다.
// nil인 경우 아무것도 하지 않습니다.
func HandleAppError(c *gin.Context, err *AppError) {
	if err == nil {
		return
	}
	statusCode := GetHTTPStatus(err.Code)
	c.JSON(statusCode, ErrorResponse{
		Code:    err.Code,
		Message: err.Message,
		Details: err.Details,
	})
}

// GetLoggerFromContext는 Gin 컨텍스트에서 로거를 가져옵니다.
// 로거가 없으면 아무것도 출력하지 않는 nop 로거를 반환합니다.
func GetLoggerFromContext(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if log, ok := logger.(*zap.Logger); ok {
			return log
		}
	}
	return zap.NewNop()
}

// HandleServiceError는 서비스 레이어의 에러를 처리하는 편의 함수입니다.
// 컨텍스트에서 로거를 가져와서 에러를 적절히 처리합니다.
func HandleServiceError(c *gin.Context, err error) {
	logger := GetLoggerFromContext(c)
	handler := NewHandler(logger)
	handler.HandleError(c, err)
}
