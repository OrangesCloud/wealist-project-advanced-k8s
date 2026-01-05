package handler

import (
	"github.com/gin-gonic/gin"

	"ops-service/internal/domain"
	"ops-service/internal/middleware"
	"ops-service/internal/response"
	"ops-service/internal/service"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	userService  *service.PortalUserService
	auditService *service.AuditLogService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *service.PortalUserService, auditService *service.AuditLogService) *AuthHandler {
	return &AuthHandler{
		userService:  userService,
		auditService: auditService,
	}
}

// OAuthCallback handles OAuth callback from auth-service
// @Summary OAuth callback handler
// @Description Called after successful Google OAuth authentication
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} domain.PortalUserResponse
// @Router /api/auth/callback [get]
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	email := middleware.GetEmail(c)
	name := middleware.GetName(c)
	picture := middleware.GetPicture(c)

	if email == "" {
		response.Unauthorized(c, "Email not found in token")
		return
	}

	user, isNew, err := h.userService.GetOrCreateFromOAuth(email, name, picture)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	// Log login action
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	h.auditService.LogWithContext(
		user.ID,
		user.Email,
		domain.ActionLogin,
		domain.ResourceUser,
		user.ID.String(),
		"OAuth login",
		ipAddress,
		userAgent,
	)

	resp := map[string]interface{}{
		"user":  user.ToResponse(),
		"isNew": isNew,
	}

	response.Success(c, resp)
}

// Logout handles logout
// @Summary Logout handler
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser != nil {
		// Log logout action
		ipAddress := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		h.auditService.LogWithContext(
			portalUser.ID,
			portalUser.Email,
			domain.ActionLogout,
			domain.ResourceUser,
			portalUser.ID.String(),
			"Logout",
			ipAddress,
			userAgent,
		)
	}

	response.SuccessWithMessage(c, "Logged out successfully", nil)
}
