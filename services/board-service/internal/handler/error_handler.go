package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/response"
)

// getErrorLogger retrieves the zap logger from gin context or returns a nop logger
func getErrorLogger(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if log, ok := logger.(*zap.Logger); ok {
			return log
		}
	}
	return zap.NewNop()
}

// handleServiceError maps service layer errors to appropriate HTTP responses
func handleServiceError(c *gin.Context, err error) {
	log := getErrorLogger(c)

	// Log the error for debugging
	log.Error("Service error", zap.Error(err))

	// Check for GORM errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.SendError(c, http.StatusNotFound, response.ErrCodeNotFound, "Resource not found")
		return
	}

	// Check for custom AppError
	var appErr *response.AppError
	if errors.As(err, &appErr) {
		log.Error("AppError",
			zap.String("code", appErr.Code),
			zap.String("message", appErr.Message),
			zap.String("details", appErr.Details))
		statusCode := mapErrorCodeToHTTPStatus(appErr.Code)
		response.SendError(c, statusCode, appErr.Code, appErr.Message)
		return
	}

	// Default to internal server error
	log.Error("Unhandled error type", zap.String("type", errors.Unwrap(err).Error()), zap.Error(err))
	response.SendError(c, http.StatusInternalServerError, response.ErrCodeInternal, "Internal server error")
}

// mapErrorCodeToHTTPStatus maps error codes to HTTP status codes
func mapErrorCodeToHTTPStatus(code string) int {
	switch code {
	case response.ErrCodeNotFound:
		return http.StatusNotFound
	case response.ErrCodeAlreadyExists:
		return http.StatusConflict
	case response.ErrCodeValidation:
		return http.StatusBadRequest
	case response.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case response.ErrCodeForbidden:
		return http.StatusForbidden
	case "ALREADY_MEMBER", "PENDING_REQUEST_EXISTS":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
