package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

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
	log := getLogger(c)
	log.Debug("CreateUser started")

	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("CreateUser validation failed", zap.Error(err))
		response.ValidationError(c, err.Error())
		return
	}

	log.Debug("CreateUser calling service", zap.String("user.email", req.Email))

	user, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		log.Error("CreateUser service error", zap.Error(err))
		response.HandleError(c, err)
		return
	}

	log.Info("User created", zap.String("enduser.id", user.ID.String()))
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
	log := getLogger(c)
	log.Debug("GetMe started")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		log.Warn("GetMe user not authenticated")
		response.Unauthorized(c, "User not authenticated")
		return
	}

	log.Debug("GetMe fetching user", zap.String("enduser.id", userID.String()))

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		log.Debug("GetMe user not found", zap.String("enduser.id", userID.String()))
		response.NotFound(c, "User not found")
		return
	}

	log.Debug("GetMe completed", zap.String("enduser.id", userID.String()))
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
	log := getLogger(c)
	log.Debug("GetUser started")

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Warn("GetUser invalid user ID", zap.String("userId", userIDStr))
		response.BadRequest(c, "Invalid user ID")
		return
	}

	log.Debug("GetUser fetching user", zap.String("enduser.id", userID.String()))

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		log.Debug("GetUser user not found", zap.String("enduser.id", userID.String()))
		response.NotFound(c, "User not found")
		return
	}

	log.Debug("GetUser completed", zap.String("enduser.id", userID.String()))
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
	log := getLogger(c)
	log.Debug("UpdateUser started")

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Warn("UpdateUser invalid user ID", zap.String("userId", userIDStr))
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// Check if user is updating themselves
	currentUserID, ok := middleware.GetUserID(c)
	if !ok || currentUserID != userID {
		log.Warn("UpdateUser forbidden - cannot update other user",
			zap.String("enduser.id", userID.String()),
			zap.String("current_user_id", currentUserID.String()))
		response.Forbidden(c, "Cannot update other user")
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("UpdateUser validation failed", zap.Error(err))
		response.ValidationError(c, err.Error())
		return
	}

	log.Debug("UpdateUser calling service", zap.String("enduser.id", userID.String()))

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, req)
	if err != nil {
		log.Error("UpdateUser service error", zap.Error(err))
		response.HandleError(c, err)
		return
	}

	log.Info("User updated", zap.String("enduser.id", user.ID.String()))
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
	log := getLogger(c)
	log.Debug("DeleteMe started")

	userID, ok := middleware.GetUserID(c)
	if !ok {
		log.Warn("DeleteMe user not authenticated")
		response.Unauthorized(c, "User not authenticated")
		return
	}

	log.Debug("DeleteMe calling service", zap.String("enduser.id", userID.String()))

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		log.Error("DeleteMe service error", zap.Error(err))
		response.InternalErrorWithDetails(c, "Failed to delete user", err)
		return
	}

	log.Info("User deleted", zap.String("enduser.id", userID.String()))
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
	log := getLogger(c)
	log.Debug("RestoreUser started")

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Warn("RestoreUser invalid user ID", zap.String("userId", userIDStr))
		response.BadRequest(c, "Invalid user ID")
		return
	}

	log.Debug("RestoreUser calling service", zap.String("enduser.id", userID.String()))

	user, err := h.userService.RestoreUser(c.Request.Context(), userID)
	if err != nil {
		log.Debug("RestoreUser user not found", zap.String("enduser.id", userID.String()))
		response.NotFound(c, "User not found")
		return
	}

	log.Info("User restored", zap.String("enduser.id", user.ID.String()))
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
	log := getLogger(c)
	log.Debug("OAuthLogin started")

	var req domain.OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("OAuthLogin validation failed", zap.Error(err))
		response.ValidationError(c, err.Error())
		return
	}

	log.Debug("OAuthLogin calling service",
		zap.String("user.email", req.Email),
		zap.String("oauth.provider", req.Provider))

	user, err := h.userService.FindOrCreateOAuthUser(c.Request.Context(), req.Email, req.Name, req.Provider)
	if err != nil {
		log.Error("OAuthLogin service error", zap.Error(err))
		response.InternalErrorWithDetails(c, "OAuth login failed", err)
		return
	}

	log.Info("OAuth login successful", zap.String("enduser.id", user.ID.String()))
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
	log := getLogger(c)
	log.Debug("UserExists started")

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Warn("UserExists invalid user ID", zap.String("userId", userIDStr))
		response.BadRequest(c, "Invalid user ID")
		return
	}

	log.Debug("UserExists checking", zap.String("enduser.id", userID.String()))

	exists, err := h.userService.UserExists(c.Request.Context(), userID)
	if err != nil {
		log.Error("UserExists service error", zap.Error(err))
		response.InternalErrorWithDetails(c, "Failed to check user existence", err)
		return
	}

	log.Debug("UserExists completed",
		zap.String("enduser.id", userID.String()),
		zap.Bool("exists", exists))
	response.OK(c, gin.H{"exists": exists})
}
