package handler

import (
	"chat-service/internal/domain"
	"chat-service/internal/response"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
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

// log returns a trace-context aware logger
func (h *ChatHandler) log(c *gin.Context) *zap.Logger {
	return commnotel.WithTraceContext(c.Request.Context(), h.logger)
}

// CreateChat creates a new chat
func (h *ChatHandler) CreateChat(c *gin.Context) {
	log := h.log(c)
	log.Debug("CreateChat started")

	userID := c.MustGet("user_id").(uuid.UUID)

	var req domain.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("CreateChat validation failed", zap.Error(err))
		response.BadRequest(c, err.Error())
		return
	}

	log.Debug("CreateChat calling service",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", req.WorkspaceID.String()))

	chat, err := h.chatService.CreateChat(c.Request.Context(), &req, userID)
	if err != nil {
		log.Error("CreateChat service error", zap.Error(err))
		response.InternalError(c, "Failed to create chat")
		return
	}

	log.Info("Chat created", zap.String("chat.id", chat.ID.String()))
	response.Created(c, chat)
}

// GetMyChats returns user's chats
func (h *ChatHandler) GetMyChats(c *gin.Context) {
	log := h.log(c)
	log.Debug("GetMyChats started")

	userID := c.MustGet("user_id").(uuid.UUID)

	log.Debug("GetMyChats fetching chats", zap.String("enduser.id", userID.String()))

	chats, err := h.chatService.GetUserChats(c.Request.Context(), userID)
	if err != nil {
		log.Error("GetMyChats service error", zap.Error(err))
		response.InternalError(c, "Failed to get chats")
		return
	}

	log.Debug("GetMyChats completed",
		zap.String("enduser.id", userID.String()),
		zap.Int("chat.count", len(chats)))
	response.Success(c, chats)
}

// GetWorkspaceChats returns chats in a workspace
func (h *ChatHandler) GetWorkspaceChats(c *gin.Context) {
	log := h.log(c)
	log.Debug("GetWorkspaceChats started")

	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		log.Warn("GetWorkspaceChats invalid workspace ID")
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	log.Debug("GetWorkspaceChats fetching chats", zap.String("workspace.id", workspaceID.String()))

	chats, err := h.chatService.GetWorkspaceChats(c.Request.Context(), workspaceID)
	if err != nil {
		log.Error("GetWorkspaceChats service error", zap.Error(err))
		response.InternalError(c, "Failed to get chats")
		return
	}

	log.Debug("GetWorkspaceChats completed",
		zap.String("workspace.id", workspaceID.String()),
		zap.Int("chat.count", len(chats)))
	response.Success(c, chats)
}

// GetChat returns a specific chat
func (h *ChatHandler) GetChat(c *gin.Context) {
	log := h.log(c)
	log.Debug("GetChat started")

	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		log.Warn("GetChat invalid chat ID")
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	log.Debug("GetChat checking participant",
		zap.String("chat.id", chatID.String()),
		zap.String("enduser.id", userID.String()))

	// Verify user is in chat
	inChat, err := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if err != nil || !inChat {
		log.Warn("GetChat user not a participant",
			zap.String("chat.id", chatID.String()),
			zap.String("enduser.id", userID.String()))
		response.Forbidden(c, "Not a participant")
		return
	}

	chat, err := h.chatService.GetChatByID(c.Request.Context(), chatID)
	if err != nil {
		log.Debug("GetChat not found", zap.String("chat.id", chatID.String()))
		response.NotFound(c, "Chat not found")
		return
	}

	log.Debug("GetChat completed", zap.String("chat.id", chatID.String()))
	response.Success(c, chat)
}

// DeleteChat soft deletes a chat (creator only)
func (h *ChatHandler) DeleteChat(c *gin.Context) {
	log := h.log(c)
	log.Debug("DeleteChat started")

	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		log.Warn("DeleteChat invalid chat ID")
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	log.Debug("DeleteChat calling service",
		zap.String("chat.id", chatID.String()),
		zap.String("enduser.id", userID.String()))

	// Service layer validates creator permission
	if err := h.chatService.DeleteChat(c.Request.Context(), chatID, userID); err != nil {
		log.Error("DeleteChat service error",
			zap.String("chat.id", chatID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	log.Info("Chat deleted", zap.String("chat.id", chatID.String()))
	response.NoContent(c)
}

// AddParticipants adds participants to a chat
func (h *ChatHandler) AddParticipants(c *gin.Context) {
	log := h.log(c)
	log.Debug("AddParticipants started")

	userID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		log.Warn("AddParticipants invalid chat ID")
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	var req struct {
		UserIDs []uuid.UUID `json:"userIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("AddParticipants validation failed", zap.Error(err))
		response.BadRequest(c, err.Error())
		return
	}

	log.Debug("AddParticipants checking participant",
		zap.String("chat.id", chatID.String()),
		zap.String("enduser.id", userID.String()))

	// Verify user is in chat
	inChat, _ := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if !inChat {
		log.Warn("AddParticipants requester not a participant",
			zap.String("chat.id", chatID.String()),
			zap.String("enduser.id", userID.String()))
		response.Forbidden(c, "Not a participant")
		return
	}

	log.Debug("AddParticipants adding users",
		zap.String("chat.id", chatID.String()),
		zap.Int("participant.count", len(req.UserIDs)))

	if err := h.chatService.AddParticipants(c.Request.Context(), chatID, req.UserIDs); err != nil {
		log.Error("AddParticipants service error", zap.Error(err))
		response.InternalError(c, "Failed to add participants")
		return
	}

	chat, _ := h.chatService.GetChatByID(c.Request.Context(), chatID)
	log.Info("Participants added",
		zap.String("chat.id", chatID.String()),
		zap.Int("added.count", len(req.UserIDs)))
	response.Success(c, chat)
}

// RemoveParticipant removes a participant from a chat
// Users can remove themselves, or creators can remove other participants
func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	log := h.log(c)
	log.Debug("RemoveParticipant started")

	requesterID := c.MustGet("user_id").(uuid.UUID)

	chatID, err := uuid.Parse(c.Param("chatId"))
	if err != nil {
		log.Warn("RemoveParticipant invalid chat ID")
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		log.Warn("RemoveParticipant invalid user ID")
		response.BadRequest(c, "Invalid user ID")
		return
	}

	log.Debug("RemoveParticipant calling service",
		zap.String("chat.id", chatID.String()),
		zap.String("target.user.id", targetUserID.String()),
		zap.String("requester.id", requesterID.String()))

	// Service layer validates permission (self-removal or creator)
	if err := h.chatService.RemoveParticipant(c.Request.Context(), chatID, targetUserID, requesterID); err != nil {
		log.Error("RemoveParticipant service error",
			zap.String("chat.id", chatID.String()),
			zap.String("target.user.id", targetUserID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	log.Info("Participant removed",
		zap.String("chat.id", chatID.String()),
		zap.String("removed.user.id", targetUserID.String()))
	response.NoContent(c)
}
