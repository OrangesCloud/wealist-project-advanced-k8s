package handler

import (
	"ops-service/internal/client"
	"ops-service/internal/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SLOHandler handles SLO dashboard requests
type SLOHandler struct {
	prometheusClient *client.PrometheusClient
	namespace        string
	logger           *zap.Logger
}

// NewSLOHandler creates a new SLO handler
func NewSLOHandler(prometheusClient *client.PrometheusClient, namespace string, logger *zap.Logger) *SLOHandler {
	return &SLOHandler{
		prometheusClient: prometheusClient,
		namespace:        namespace,
		logger:           logger,
	}
}

// GetSLOOverview returns SLO metrics for all services
// @Summary Get SLO overview
// @Description Returns SLO metrics for all services including availability, latency, and error budget
// @Tags SLO
// @Produce json
// @Success 200 {object} client.SLOOverview
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/slo/overview [get]
func (h *SLOHandler) GetSLOOverview(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	overview, err := h.prometheusClient.GetSLOOverview(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get SLO overview", zap.Error(err))
		response.InternalError(c, "Failed to get SLO overview")
		return
	}

	response.Success(c, overview)
}

// GetBurnRates returns error budget burn rates for all services
// @Summary Get burn rates
// @Description Returns error budget burn rates (1h, 6h, 24h) for all services
// @Tags SLO
// @Produce json
// @Success 200 {array} client.BurnRate
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/slo/burn-rates [get]
func (h *SLOHandler) GetBurnRates(c *gin.Context) {
	if h.prometheusClient == nil {
		response.InternalError(c, "Prometheus client not configured")
		return
	}

	burnRates, err := h.prometheusClient.GetBurnRates(c.Request.Context(), h.namespace)
	if err != nil {
		h.logger.Error("Failed to get burn rates", zap.Error(err))
		response.InternalError(c, "Failed to get burn rates")
		return
	}

	response.Success(c, burnRates)
}
