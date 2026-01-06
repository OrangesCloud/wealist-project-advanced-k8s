package handler

import (
	"ops-service/internal/client"
	"ops-service/internal/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MetricsHandler handles metrics-related requests
type MetricsHandler struct {
	prometheusClient *client.PrometheusClient
	namespace        string
	logger           *zap.Logger
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(prometheusClient *client.PrometheusClient, namespace string, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		prometheusClient: prometheusClient,
		namespace:        namespace,
		logger:           logger,
	}
}

// GetSystemOverview returns system overview metrics for dashboard
// @Summary Get system overview metrics
// @Description Returns overall system metrics including request rates, error rates, and active services
// @Tags Metrics
// @Produce json
// @Success 200 {object} client.SystemOverview
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/metrics/overview [get]
func (h *MetricsHandler) GetSystemOverview(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	overview, err := h.prometheusClient.GetSystemOverview(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get system overview", zap.Error(err))
		response.InternalError(c, "Failed to get system overview metrics")
		return
	}

	response.Success(c, overview)
}

// GetServiceMetrics returns metrics for all services
// @Summary Get service metrics
// @Description Returns detailed metrics for each service including request rate, latency, and error rate
// @Tags Metrics
// @Produce json
// @Success 200 {array} client.ServiceMetrics
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/metrics/services [get]
func (h *MetricsHandler) GetServiceMetrics(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	metrics, err := h.prometheusClient.GetServiceMetrics(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get service metrics", zap.Error(err))
		response.InternalError(c, "Failed to get service metrics")
		return
	}

	response.Success(c, metrics)
}

// GetClusterMetrics returns cluster-level metrics
// @Summary Get cluster metrics
// @Description Returns cluster-level metrics including node count, CPU, and memory usage
// @Tags Metrics
// @Produce json
// @Success 200 {object} client.ClusterMetrics
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/metrics/cluster [get]
func (h *MetricsHandler) GetClusterMetrics(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	metrics, err := h.prometheusClient.GetClusterMetrics(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get cluster metrics", zap.Error(err))
		response.InternalError(c, "Failed to get cluster metrics")
		return
	}

	response.Success(c, metrics)
}

// GetServiceDetail returns detailed metrics for a specific service
// @Summary Get detailed service metrics
// @Description Returns detailed metrics for a specific service
// @Tags Metrics
// @Produce json
// @Param serviceName path string true "Service name"
// @Success 200 {object} client.ServiceMetrics
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/metrics/services/{serviceName} [get]
func (h *MetricsHandler) GetServiceDetail(c *gin.Context) {
	serviceName := c.Param("serviceName")
	if serviceName == "" {
		response.BadRequest(c, "Service name is required")
		return
	}

	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	// Get metrics for all services and find the requested one
	metrics, err := h.prometheusClient.GetServiceMetrics(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get service metrics", zap.Error(err))
		response.InternalError(c, "Failed to get service metrics")
		return
	}

	for _, m := range metrics {
		if m.ServiceName == serviceName {
			response.Success(c, m)
			return
		}
	}

	response.NotFound(c, "Service not found: "+serviceName)
}
