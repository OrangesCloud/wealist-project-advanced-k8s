package handler

import (
	"strconv"
	"time"

	"ops-service/internal/client"
	"ops-service/internal/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogsHandler handles log-related requests
type LogsHandler struct {
	lokiClient *client.LokiClient
	namespace  string
	logger     *zap.Logger
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(lokiClient *client.LokiClient, namespace string, logger *zap.Logger) *LogsHandler {
	return &LogsHandler{
		lokiClient: lokiClient,
		namespace:  namespace,
		logger:     logger,
	}
}

// GetLogs queries logs from Loki
// @Summary Get logs
// @Description Query logs from Loki with filters
// @Tags Logs
// @Produce json
// @Param service query string false "Service name filter"
// @Param level query string false "Log level filter (debug, info, warn, error)"
// @Param query query string false "Text search query"
// @Param start query string false "Start time (RFC3339 or Unix timestamp)"
// @Param end query string false "End time (RFC3339 or Unix timestamp)"
// @Param limit query int false "Max number of entries (default 100, max 1000)"
// @Success 200 {object} client.LogQueryResult
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/logs [get]
func (h *LogsHandler) GetLogs(c *gin.Context) {
	if h.lokiClient == nil {
		response.Success(c, client.LogQueryResult{
			Entries:    []client.LogEntry{},
			TotalCount: 0,
		})
		return
	}

	params := client.LogQueryParams{
		Service: c.Query("service"),
		Level:   c.Query("level"),
		Query:   c.Query("query"),
	}

	// Parse time range
	now := time.Now()
	params.End = now
	params.Start = now.Add(-1 * time.Hour) // Default: last 1 hour

	if startStr := c.Query("start"); startStr != "" {
		if t, err := parseTime(startStr); err == nil {
			params.Start = t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := parseTime(endStr); err == nil {
			params.End = t
		}
	}

	// Parse limit
	params.Limit = 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = l
		}
	}

	result, err := h.lokiClient.GetLogs(params)
	if err != nil {
		h.logger.Error("Failed to query logs", zap.Error(err))
		response.InternalError(c, "Failed to query logs: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetServices returns list of available services
// @Summary Get available services
// @Description Returns list of services that have logs in Loki
// @Tags Logs
// @Produce json
// @Success 200 {array} client.ServiceInfo
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/logs/services [get]
func (h *LogsHandler) GetServices(c *gin.Context) {
	if h.lokiClient == nil {
		response.Success(c, []client.ServiceInfo{})
		return
	}

	services, err := h.lokiClient.GetServices()
	if err != nil {
		h.logger.Error("Failed to get services", zap.Error(err))
		response.InternalError(c, "Failed to get services: "+err.Error())
		return
	}

	response.Success(c, services)
}

// parseTime parses time from string (RFC3339 or Unix timestamp)
func parseTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try Unix timestamp (seconds)
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	// Try Unix timestamp (milliseconds)
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil && ts > 1e12 {
		return time.Unix(ts/1000, (ts%1000)*1e6), nil
	}

	return time.Time{}, nil
}
