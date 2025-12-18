package handler

import (
	"chat-service/internal/domain"
	"chat-service/internal/response"
	"chat-service/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MessageHandler struct {
	chatService *service.ChatService
	logger      *zap.Logger
}

func NewMessageHandler(chatService *service.ChatService, logger *zap.Logger) *MessageHandler {
	return &MessageHandler{
		chatService: chatService,
		logger:      logger,
	}
}

// GetMessages returns messages from a chat
func (h *MessageHandler) GetMessages(c *gin.Context) {
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

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var before *uuid.UUID
	if beforeStr := c.Query("before"); beforeStr != "" {
		if b, err := uuid.Parse(beforeStr); err == nil {
			before = &b
		}
	}

	messages, err := h.chatService.GetMessages(c.Request.Context(), chatID, limit, before)
	if err != nil {
		h.logger.Error("failed to get messages", zap.Error(err))
		response.InternalError(c, "Failed to get messages")
		return
	}

	response.Success(c, messages)
}

// SendMessage sends a message to a chat
func (h *MessageHandler) SendMessage(c *gin.Context) {
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

	var req domain.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	message, err := h.chatService.SendMessage(c.Request.Context(), chatID, userID, &req)
	if err != nil {
		h.logger.Error("failed to send message", zap.Error(err))
		response.InternalError(c, "Failed to send message")
		return
	}

	response.Created(c, message)
}

// DeleteMessage soft deletes a message
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		response.BadRequest(c, "Invalid message ID")
		return
	}

	if err := h.chatService.DeleteMessage(c.Request.Context(), messageID); err != nil {
		h.logger.Error("failed to delete message", zap.Error(err))
		response.InternalError(c, "Failed to delete message")
		return
	}

	response.NoContent(c)
}

// MarkMessagesAsRead marks messages as read
func (h *MessageHandler) MarkMessagesAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		MessageIDs []uuid.UUID `json:"messageIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.chatService.MarkMessagesAsRead(c.Request.Context(), req.MessageIDs, userID); err != nil {
		h.logger.Error("failed to mark messages as read", zap.Error(err))
		response.InternalError(c, "Failed to mark as read")
		return
	}

	response.Success(c, "Messages marked as read")
}

// GetUnreadCount returns unread count for a chat
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	count, err := h.chatService.GetUnreadCount(c.Request.Context(), chatID, userID)
	if err != nil {
		h.logger.Error("failed to get unread count", zap.Error(err))
		response.InternalError(c, "Failed to get unread count")
		return
	}

	response.OK(c, map[string]int64{"unreadCount": count})
}

// UpdateLastRead updates last read timestamp
func (h *MessageHandler) UpdateLastRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	if err := h.chatService.UpdateLastReadAt(c.Request.Context(), chatID, userID); err != nil {
		h.logger.Error("failed to update last read", zap.Error(err))
		response.InternalError(c, "Failed to update last read")
		return
	}

	response.Success(c, "Last read updated")
}
