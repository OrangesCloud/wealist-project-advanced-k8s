package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ops-service/internal/domain"
	"ops-service/internal/repository"
	"ops-service/internal/response"
	"ops-service/internal/service"
)

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	auditService *service.AuditLogService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *service.AuditLogService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// List returns audit logs with filtering and pagination
// @Summary List audit logs
// @Tags audit
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param user_id query string false "Filter by user ID"
// @Param resource_type query string false "Filter by resource type"
// @Param action query string false "Filter by action"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Success 200 {object} response.PaginatedResponse
// @Router /api/admin/audit-logs [get]
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	opts := repository.ListOptions{
		Page:  page,
		Limit: limit,
	}

	// Parse user_id filter
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			opts.UserID = &userID
		}
	}

	// Parse resource_type filter
	if resourceType := c.Query("resource_type"); resourceType != "" {
		rt := domain.ResourceType(resourceType)
		opts.ResourceType = &rt
	}

	// Parse action filter
	if action := c.Query("action"); action != "" {
		a := domain.ActionType(action)
		opts.Action = &a
	}

	// Parse time filters
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			opts.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err == nil {
			opts.EndTime = &endTime
		}
	}

	logs, total, err := h.auditService.List(opts)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	responses := make([]domain.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = log.ToResponse()
	}

	response.Paginated(c, responses, page, limit, total)
}

// GetByID returns an audit log by ID
// @Summary Get audit log by ID
// @Tags audit
// @Security BearerAuth
// @Param id path string true "Audit log ID"
// @Success 200 {object} domain.AuditLogResponse
// @Router /api/admin/audit-logs/{id} [get]
func (h *AuditHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid audit log ID")
		return
	}

	log, err := h.auditService.GetByID(id)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Success(c, log.ToResponse())
}
