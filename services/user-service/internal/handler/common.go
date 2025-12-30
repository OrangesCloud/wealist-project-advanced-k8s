package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
)

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// getLogger retrieves the zap logger from gin context with trace context
func getLogger(c *gin.Context) *zap.Logger {
	var baseLogger *zap.Logger
	if logger, exists := c.Get("logger"); exists {
		if log, ok := logger.(*zap.Logger); ok {
			baseLogger = log
		}
	}
	if baseLogger == nil {
		baseLogger = zap.NewNop()
	}
	return commnotel.WithTraceContext(c.Request.Context(), baseLogger)
}
