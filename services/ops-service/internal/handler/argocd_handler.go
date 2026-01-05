package handler

import (
	"ops-service/internal/response"
	"ops-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ArgoCDHandler handles ArgoCD-related requests
type ArgoCDHandler struct {
	argoCDService *service.ArgoCDRBACService
	logger        *zap.Logger
}

// NewArgoCDHandler creates a new ArgoCD handler
func NewArgoCDHandler(argoCDService *service.ArgoCDRBACService, logger *zap.Logger) *ArgoCDHandler {
	return &ArgoCDHandler{
		argoCDService: argoCDService,
		logger:        logger,
	}
}

// GetRBAC returns current ArgoCD RBAC configuration
// @Summary Get ArgoCD RBAC configuration
// @Description Returns the current ArgoCD RBAC policy and admin users
// @Tags ArgoCD
// @Produce json
// @Success 200 {object} service.RBACInfo
// @Failure 500 {object} response.ErrorResponse
// @Router /api/admin/argocd/rbac [get]
func (h *ArgoCDHandler) GetRBAC(c *gin.Context) {
	rbac, err := h.argoCDService.GetRBAC(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get RBAC", zap.Error(err))
		response.InternalError(c, "Failed to get ArgoCD RBAC configuration")
		return
	}

	response.Success(c, rbac)
}

// AddAdminRequest represents the request body for adding an admin
type AddAdminRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// AddAdmin adds an admin user to ArgoCD RBAC
// @Summary Add ArgoCD admin
// @Description Adds a user as an ArgoCD admin
// @Tags ArgoCD
// @Accept json
// @Produce json
// @Param request body AddAdminRequest true "Admin email"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/admin/argocd/rbac/admins [post]
func (h *ArgoCDHandler) AddAdmin(c *gin.Context) {
	var req AddAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Get the current user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.argoCDService.AddAdmin(c.Request.Context(), req.Email, userID); err != nil {
		if err.Error() == "user "+req.Email+" is already an admin" {
			response.Conflict(c, err.Error())
			return
		}
		h.logger.Error("Failed to add admin", zap.String("email", req.Email), zap.Error(err))
		response.InternalError(c, "Failed to add ArgoCD admin")
		return
	}

	response.Success(c, gin.H{
		"message": "Admin added successfully",
		"email":   req.Email,
	})
}

// RemoveAdmin removes an admin user from ArgoCD RBAC
// @Summary Remove ArgoCD admin
// @Description Removes a user from ArgoCD admins
// @Tags ArgoCD
// @Produce json
// @Param email path string true "Admin email"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/admin/argocd/rbac/admins/{email} [delete]
func (h *ArgoCDHandler) RemoveAdmin(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		response.BadRequest(c, "Email is required")
		return
	}

	// Get the current user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.argoCDService.RemoveAdmin(c.Request.Context(), email, userID); err != nil {
		if err.Error() == "user "+email+" is not an admin" {
			response.NotFound(c, err.Error())
			return
		}
		h.logger.Error("Failed to remove admin", zap.String("email", email), zap.Error(err))
		response.InternalError(c, "Failed to remove ArgoCD admin")
		return
	}

	response.Success(c, gin.H{
		"message": "Admin removed successfully",
		"email":   email,
	})
}

// GetApplications returns ArgoCD applications
// @Summary Get ArgoCD applications
// @Description Returns all ArgoCD applications for monitoring
// @Tags ArgoCD
// @Produce json
// @Success 200 {array} client.Application
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/applications [get]
func (h *ArgoCDHandler) GetApplications(c *gin.Context) {
	apps, err := h.argoCDService.GetApplications(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get applications", zap.Error(err))
		response.InternalError(c, "Failed to get ArgoCD applications")
		return
	}

	// Transform to a simpler format for the frontend
	var result []gin.H
	for _, app := range apps {
		result = append(result, gin.H{
			"name":      app.Metadata.Name,
			"namespace": app.Metadata.Namespace,
			"project":   app.Spec.Project,
			"sync":      app.Status.Sync.Status,
			"health":    app.Status.Health.Status,
		})
	}

	response.Success(c, result)
}

// SyncApplicationRequest represents the request to sync an application
type SyncApplicationRequest struct {
	Name string `json:"name" binding:"required"`
}

// SyncApplication triggers a sync for an ArgoCD application
// @Summary Sync ArgoCD application
// @Description Triggers a sync for the specified ArgoCD application
// @Tags ArgoCD
// @Accept json
// @Produce json
// @Param request body SyncApplicationRequest true "Application name"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/monitoring/applications/sync [post]
func (h *ArgoCDHandler) SyncApplication(c *gin.Context) {
	var req SyncApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Get the current user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.argoCDService.SyncApplication(c.Request.Context(), req.Name, userID); err != nil {
		h.logger.Error("Failed to sync application", zap.String("name", req.Name), zap.Error(err))
		response.InternalError(c, "Failed to sync ArgoCD application")
		return
	}

	response.Success(c, gin.H{
		"message":     "Application sync triggered",
		"application": req.Name,
	})
}
