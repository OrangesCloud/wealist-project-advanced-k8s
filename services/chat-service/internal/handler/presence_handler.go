package handler

import (
	"chat-service/internal/response"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PresenceHandler struct {
	presenceService *service.PresenceService
	logger          *zap.Logger
}

func NewPresenceHandler(presenceService *service.PresenceService, logger *zap.Logger) *PresenceHandler {
	return &PresenceHandler{
		presenceService: presenceService,
		logger:          logger,
	}
}

// GetOnlineUsers returns online users
func (h *PresenceHandler) GetOnlineUsers(c *gin.Context) {
	var workspaceID *uuid.UUID
	if wsIDStr := c.Query("workspaceId"); wsIDStr != "" {
		if wsID, err := uuid.Parse(wsIDStr); err == nil {
			workspaceID = &wsID
		}
	}

	presences, err := h.presenceService.GetOnlineUsers(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.Error("failed to get online users", zap.Error(err))
		response.InternalError(c, "Failed to get online users")
		return
	}

	response.Success(c, presences)
}

// GetUserStatus returns a user's online status
func (h *PresenceHandler) GetUserStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	presence, err := h.presenceService.GetUserStatus(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "User presence not found")
		return
	}

	response.Success(c, presence)
}
