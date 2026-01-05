package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ops-service/internal/domain"
	"ops-service/internal/middleware"
	"ops-service/internal/response"
	"ops-service/internal/service"
)

// ConfigHandler handles app config HTTP requests
type ConfigHandler struct {
	configService *service.AppConfigService
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configService *service.AppConfigService) *ConfigHandler {
	return &ConfigHandler{configService: configService}
}

// GetAll returns all app configs
// @Summary Get all app configs
// @Tags config
// @Security BearerAuth
// @Success 200 {array} domain.AppConfigResponse
// @Router /api/config [get]
func (h *ConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.configService.GetAll()
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	responses := make([]domain.AppConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = config.ToResponse()
	}

	response.Success(c, responses)
}

// GetActive returns all active app configs (public endpoint for clients)
// @Summary Get active app configs
// @Tags config
// @Success 200 {array} domain.AppConfigResponse
// @Router /api/config/active [get]
func (h *ConfigHandler) GetActive(c *gin.Context) {
	configs, err := h.configService.GetActive()
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	// Return as key-value map for easier client consumption
	result := make(map[string]string)
	for _, config := range configs {
		result[config.Key] = config.Value
	}

	response.Success(c, result)
}

// GetByKey returns an app config by key
// @Summary Get app config by key
// @Tags config
// @Param key path string true "Config key"
// @Success 200 {object} domain.AppConfigResponse
// @Router /api/config/{key} [get]
func (h *ConfigHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Config key is required")
		return
	}

	config, err := h.configService.GetByKey(key)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Success(c, config.ToResponse())
}

// Create creates a new app config
// @Summary Create app config
// @Tags config
// @Security BearerAuth
// @Param body body domain.CreateAppConfigRequest true "Create request"
// @Success 201 {object} domain.AppConfigResponse
// @Router /api/admin/config [post]
func (h *ConfigHandler) Create(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req domain.CreateAppConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	config, err := h.configService.Create(portalUser.ID, portalUser.Email, req)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Created(c, config.ToResponse())
}

// Update updates an app config
// @Summary Update app config
// @Tags config
// @Security BearerAuth
// @Param id path string true "Config ID"
// @Param body body domain.UpdateAppConfigRequest true "Update request"
// @Success 200 {object} domain.AppConfigResponse
// @Router /api/admin/config/{id} [put]
func (h *ConfigHandler) Update(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid config ID")
		return
	}

	var req domain.UpdateAppConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	config, err := h.configService.Update(portalUser.ID, portalUser.Email, id, req)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Success(c, config.ToResponse())
}

// Delete deletes an app config
// @Summary Delete app config
// @Tags config
// @Security BearerAuth
// @Param id path string true "Config ID"
// @Success 204
// @Router /api/admin/config/{id} [delete]
func (h *ConfigHandler) Delete(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid config ID")
		return
	}

	if err := h.configService.Delete(portalUser.ID, portalUser.Email, id); err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.NoContent(c)
}
