package handler

import (
	"strconv"

	"ops-service/internal/client"
	"ops-service/internal/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorTrackerHandler handles error tracking requests
type ErrorTrackerHandler struct {
	prometheusClient *client.PrometheusClient
	namespace        string
	logger           *zap.Logger
}

// NewErrorTrackerHandler creates a new error tracker handler
func NewErrorTrackerHandler(prometheusClient *client.PrometheusClient, namespace string, logger *zap.Logger) *ErrorTrackerHandler {
	return &ErrorTrackerHandler{
		prometheusClient: prometheusClient,
		namespace:        namespace,
		logger:           logger,
	}
}

// GetErrorOverview returns overall error statistics for dashboard
// @Summary Get error overview
// @Description Returns summary error statistics including total errors, error rate, and most affected service
// @Tags ErrorTracker
// @Produce json
// @Success 200 {object} client.ErrorOverview
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/errors/overview [get]
func (h *ErrorTrackerHandler) GetErrorOverview(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	overview, err := h.prometheusClient.GetErrorOverview(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get error overview", zap.Error(err))
		response.InternalError(c, "Failed to get error overview")
		return
	}

	response.Success(c, overview)
}

// GetRecentErrors returns recent 5xx errors
// @Summary Get recent errors
// @Description Returns recent 5xx errors from the last hour
// @Tags ErrorTracker
// @Produce json
// @Param limit query int false "Maximum number of errors to return (default: 50)"
// @Success 200 {array} client.RecentError
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/errors/recent [get]
func (h *ErrorTrackerHandler) GetRecentErrors(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	errors, err := h.prometheusClient.GetRecentErrors(c.Request.Context(), h.namespace, limit)
	if err != nil {
		h.logger.Error("Failed to get recent errors", zap.Error(err))
		response.InternalError(c, "Failed to get recent errors")
		return
	}

	response.Success(c, errors)
}

// GetErrorTrend returns error rate trend over time
// @Summary Get error trend
// @Description Returns error rate trend over the last hour with 5-minute intervals
// @Tags ErrorTracker
// @Produce json
// @Success 200 {array} client.ErrorTrendPoint
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/errors/trend [get]
func (h *ErrorTrackerHandler) GetErrorTrend(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	trend, err := h.prometheusClient.GetErrorTrend(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get error trend", zap.Error(err))
		response.InternalError(c, "Failed to get error trend")
		return
	}

	response.Success(c, trend)
}

// GetErrorsByService returns error statistics per service
// @Summary Get errors by service
// @Description Returns error statistics for each service including 4xx and 5xx counts
// @Tags ErrorTracker
// @Produce json
// @Success 200 {array} client.ServiceErrorSummary
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/errors/by-service [get]
func (h *ErrorTrackerHandler) GetErrorsByService(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	summaries, err := h.prometheusClient.GetErrorsByService(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get errors by service", zap.Error(err))
		response.InternalError(c, "Failed to get errors by service")
		return
	}

	response.Success(c, summaries)
}
