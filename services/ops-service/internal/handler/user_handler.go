package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ops-service/internal/domain"
	"ops-service/internal/middleware"
	"ops-service/internal/response"
	"ops-service/internal/service"
)

// UserHandler handles portal user HTTP requests
type UserHandler struct {
	userService *service.PortalUserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.PortalUserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetMe returns the current user's profile
// @Summary Get current user profile
// @Tags users
// @Security BearerAuth
// @Success 200 {object} domain.PortalUserResponse
// @Router /api/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	response.Success(c, portalUser.ToResponse())
}

// GetAll returns all portal users
// @Summary Get all portal users
// @Tags users
// @Security BearerAuth
// @Success 200 {array} domain.PortalUserResponse
// @Router /api/admin/users [get]
func (h *UserHandler) GetAll(c *gin.Context) {
	users, err := h.userService.GetAll()
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	responses := make([]domain.PortalUserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	response.Success(c, responses)
}

// GetByID returns a portal user by ID
// @Summary Get portal user by ID
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} domain.PortalUserResponse
// @Router /api/admin/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Success(c, user.ToResponse())
}

// InviteUser invites a new user
// @Summary Invite a new user
// @Tags users
// @Security BearerAuth
// @Param body body domain.InviteUserRequest true "Invite request"
// @Success 201 {object} domain.PortalUserResponse
// @Router /api/admin/users/invite [post]
func (h *UserHandler) InviteUser(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req domain.InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	user, err := h.userService.InviteUser(portalUser.ID, portalUser.Email, req)
	if err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.Created(c, user.ToResponse())
}

// UpdateRole updates a user's role
// @Summary Update user role
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param body body domain.UpdateUserRoleRequest true "Update role request"
// @Success 200 {object} response.Response
// @Router /api/admin/users/{id}/role [put]
func (h *UserHandler) UpdateRole(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	idStr := c.Param("id")
	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req domain.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.userService.UpdateRole(portalUser.ID, portalUser.Email, targetUserID, req.Role); err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.SuccessWithMessage(c, "Role updated successfully", nil)
}

// DeactivateUser deactivates a user
// @Summary Deactivate a user
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response
// @Router /api/admin/users/{id}/deactivate [post]
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	idStr := c.Param("id")
	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.userService.DeactivateUser(portalUser.ID, portalUser.Email, targetUserID); err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.SuccessWithMessage(c, "User deactivated successfully", nil)
}

// ReactivateUser reactivates a user
// @Summary Reactivate a user
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response
// @Router /api/admin/users/{id}/reactivate [post]
func (h *UserHandler) ReactivateUser(c *gin.Context) {
	portalUser := middleware.GetPortalUser(c)
	if portalUser == nil {
		response.Unauthorized(c, "User not found in context")
		return
	}

	idStr := c.Param("id")
	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.userService.ReactivateUser(portalUser.ID, portalUser.Email, targetUserID); err != nil {
		response.HandleServiceError(c, err)
		return
	}

	response.SuccessWithMessage(c, "User reactivated successfully", nil)
}
