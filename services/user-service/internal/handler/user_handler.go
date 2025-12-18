package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"user-service/internal/domain"
	"user-service/internal/middleware"
	"user-service/internal/response"
	"user-service/internal/service"
)

// UserHandler handles user HTTP requests
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// CreateUser godoc
// @Summary Create a new user
// @Tags Users
// @Accept json
// @Produce json
// @Param request body domain.CreateUserRequest true "Create user request"
// @Success 201 {object} domain.UserResponse
// @Failure 400 {object} ErrorResponse
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.CreateUser(req)
	if err != nil {
		response.BadRequestWithDetails(c, "Failed to create user", err.Error())
		return
	}

	response.Created(c, user.ToResponse())
}

// GetMe godoc
// @Summary Get current user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userService.GetUser(userID)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.OK(c, user.ToResponse())
}

// GetUser godoc
// @Summary Get user by ID
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Success 200 {object} domain.UserResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{userId} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetUser(userID)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.OK(c, user.ToResponse())
}

// UpdateUser godoc
// @Summary Update user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Param request body domain.UpdateUserRequest true "Update user request"
// @Success 200 {object} domain.UserResponse
// @Failure 400 {object} ErrorResponse
// @Router /users/{userId} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// Check if user is updating themselves
	currentUserID, ok := middleware.GetUserID(c)
	if !ok || currentUserID != userID {
		response.Forbidden(c, "Cannot update other user")
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.UpdateUser(userID, req)
	if err != nil {
		response.BadRequestWithDetails(c, "Failed to update user", err.Error())
		return
	}

	response.OK(c, user.ToResponse())
}

// DeleteMe godoc
// @Summary Delete current user (soft delete)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/me [delete]
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.userService.DeleteUser(userID); err != nil {
		response.InternalErrorWithDetails(c, "Failed to delete user", err)
		return
	}

	response.Success(c, "User deleted successfully")
}

// RestoreUser godoc
// @Summary Restore deleted user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Success 200 {object} domain.UserResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{userId}/restore [put]
func (h *UserHandler) RestoreUser(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.RestoreUser(userID)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.OK(c, user.ToResponse())
}

// OAuthLogin godoc
// @Summary Find or create user for OAuth login (internal)
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body domain.OAuthLoginRequest true "OAuth login request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /internal/oauth/login [post]
func (h *UserHandler) OAuthLogin(c *gin.Context) {
	var req domain.OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.FindOrCreateOAuthUser(req.Email, req.Name, req.Provider)
	if err != nil {
		response.InternalErrorWithDetails(c, "OAuth login failed", err)
		return
	}

	response.OK(c, gin.H{"userId": user.ID.String()})
}

// UserExists godoc
// @Summary Check if user exists (internal)
// @Tags Internal
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]bool
// @Router /internal/users/{userId}/exists [get]
func (h *UserHandler) UserExists(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	exists, err := h.userService.UserExists(userID)
	if err != nil {
		response.InternalErrorWithDetails(c, "Failed to check user existence", err)
		return
	}

	response.OK(c, gin.H{"exists": exists})
}
