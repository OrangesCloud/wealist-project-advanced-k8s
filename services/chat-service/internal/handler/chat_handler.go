package handler

import (
	"chat-service/internal/domain"
	"chat-service/internal/response"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChatHandler struct {
	chatService     *service.ChatService
	presenceService *service.PresenceService
	logger          *zap.Logger
}

func NewChatHandler(
	chatService *service.ChatService,
	presenceService *service.PresenceService,
	logger *zap.Logger,
) *ChatHandler {
	return &ChatHandler{
		chatService:     chatService,
		presenceService: presenceService,
		logger:          logger,
	}
}

// CreateChat creates a new chat
func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	chat, err := h.chatService.CreateChat(c.Request.Context(), &req, userID)
	if err != nil {
		h.logger.Error("failed to create chat", zap.Error(err))
		response.InternalError(c, "Failed to create chat")
		return
	}

	response.Created(c, chat)
}

// GetMyChats returns user's chats
func (h *ChatHandler) GetMyChats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chats, err := h.chatService.GetUserChats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user chats", zap.Error(err))
		response.InternalError(c, "Failed to get chats")
		return
	}

	response.Success(c, chats)
}

// GetWorkspaceChats returns chats in a workspace
func (h *ChatHandler) GetWorkspaceChats(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	chats, err := h.chatService.GetWorkspaceChats(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.Error("failed to get workspace chats", zap.Error(err))
		response.InternalError(c, "Failed to get chats")
		return
	}

	response.Success(c, chats)
}

// GetChat returns a specific chat
func (h *ChatHandler) GetChat(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	// Verify user is in chat
	inChat, err := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if err != nil || !inChat {
		response.Forbidden(c, "Not a participant")
		return
	}

	chat, err := h.chatService.GetChatByID(c.Request.Context(), chatID)
	if err != nil {
		response.NotFound(c, "Chat not found")
		return
	}

	response.Success(c, chat)
}

// DeleteChat soft deletes a chat
func (h *ChatHandler) DeleteChat(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	// Verify user is in chat
	inChat, _ := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if !inChat {
		response.Forbidden(c, "Not a participant")
		return
	}

	if err := h.chatService.DeleteChat(c.Request.Context(), chatID); err != nil {
		h.logger.Error("failed to delete chat", zap.Error(err))
		response.InternalError(c, "Failed to delete chat")
		return
	}

	response.NoContent(c)
}

// AddParticipants adds participants to a chat
func (h *ChatHandler) AddParticipants(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	var req struct {
		UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Verify user is in chat
	inChat, _ := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if !inChat {
		response.Forbidden(c, "Not a participant")
		return
	}

	if err := h.chatService.AddParticipants(c.Request.Context(), chatID, req.UserIDs); err != nil {
		h.logger.Error("failed to add participants", zap.Error(err))
		response.InternalError(c, "Failed to add participants")
		return
	}

	chat, _ := h.chatService.GetChatByID(c.Request.Context(), chatID)
	response.Success(c, chat)
}

// RemoveParticipant removes a participant from a chat
func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	currentUserID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// Verify current user is in chat
	inChat, _ := h.chatService.IsUserInChat(c.Request.Context(), chatID, currentUserID)
	if !inChat {
		response.Forbidden(c, "Not a participant")
		return
	}

	if err := h.chatService.RemoveParticipant(c.Request.Context(), chatID, userID); err != nil {
		h.logger.Error("failed to remove participant", zap.Error(err))
		response.InternalError(c, "Failed to remove participant")
		return
	}

	response.NoContent(c)
}
