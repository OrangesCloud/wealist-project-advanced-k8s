// Package handler provides HTTP handlers for video-service endpoints.
//
// This package implements Gin handlers for managing video call rooms,
// participants, call history, and transcripts. All handlers use centralized
// error responses from the response package.
package handler

import (
	"errors"
	"fmt"
	"net/http"
	"video-service/internal/domain"
	"video-service/internal/response"
	"video-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RoomHandler handles HTTP requests for video room operations.
// It provides endpoints for creating rooms, joining/leaving calls,
// managing participants, and accessing call history.
type RoomHandler struct {
	roomService service.RoomService
	logger      *zap.Logger
}

// NewRoomHandler creates a new RoomHandler with the given service and logger.
func NewRoomHandler(roomService service.RoomService, logger *zap.Logger) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
		logger:      logger,
	}
}

// CreateRoom godoc
// @Summary Create a new video call room
// @Tags rooms
// @Accept json
// @Produce json
// @Param request body domain.CreateRoomRequest true "Room details"
// @Success 201 {object} domain.RoomResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms [post]
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	token := c.MustGet("token").(string)

	var req domain.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	room, err := h.roomService.CreateRoom(c.Request.Context(), &req, userID, token)
	if err != nil {
		if errors.Is(err, response.ErrNotWorkspaceMember) {
			response.Forbidden(c, "You are not a member of this workspace")
			return
		}
		h.logger.Error("Failed to create room", zap.Error(err))
		response.InternalError(c, "Failed to create room")
		return
	}

	response.Created(c, room)
}

// GetRoom godoc
// @Summary Get room details
// @Tags rooms
// @Produce json
// @Param roomId path string true "Room ID"
// @Success 200 {object} domain.RoomResponse
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms/{roomId} [get]
func (h *RoomHandler) GetRoom(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	room, err := h.roomService.GetRoom(c.Request.Context(), roomID)
	if err != nil {
		if errors.Is(err, response.ErrRoomNotFound) {
			response.NotFound(c, "Room not found")
			return
		}
		response.InternalError(c, "Failed to get room")
		return
	}

	response.OK(c, room)
}

// GetWorkspaceRooms godoc
// @Summary Get rooms for a workspace
// @Tags rooms
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param active query bool false "Filter active rooms only"
// @Success 200 {array} domain.RoomResponse
// @Security BearerAuth
// @Router /rooms/workspace/{workspaceId} [get]
func (h *RoomHandler) GetWorkspaceRooms(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	token := c.MustGet("token").(string)

	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	activeOnly := c.Query("active") == "true"

	rooms, err := h.roomService.GetWorkspaceRooms(c.Request.Context(), workspaceID, userID, token, activeOnly)
	if err != nil {
		if errors.Is(err, response.ErrNotWorkspaceMember) {
			response.Forbidden(c, "You are not a member of this workspace")
			return
		}
		h.logger.Error("Failed to get workspace rooms", zap.Error(err))
		response.InternalError(c, "Failed to get rooms")
		return
	}

	response.OK(c, rooms)
}

// JoinRoom godoc
// @Summary Join a video call room
// @Tags rooms
// @Accept json
// @Produce json
// @Param roomId path string true "Room ID"
// @Success 200 {object} domain.JoinRoomResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms/{roomId}/join [post]
func (h *RoomHandler) JoinRoom(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	token := c.MustGet("token").(string)

	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	// Get user name from query or use default
	userName := c.Query("userName")
	if userName == "" {
		userName = userID.String()[:8]
	}

	resp, err := h.roomService.JoinRoom(c.Request.Context(), roomID, userID, userName, token)
	if err != nil {
		switch {
		case errors.Is(err, response.ErrRoomNotFound):
			response.NotFound(c, "Room not found")
		case errors.Is(err, response.ErrRoomFull):
			response.CustomError(c, http.StatusConflict, "ROOM_FULL", "Room is full")
		case errors.Is(err, response.ErrAlreadyInRoom):
			response.CustomError(c, http.StatusConflict, "ALREADY_IN_ROOM", "User is already in room")
		case errors.Is(err, response.ErrRoomNotActive):
			response.CustomError(c, http.StatusGone, "ROOM_ENDED", "Room has ended")
		case errors.Is(err, response.ErrNotWorkspaceMember):
			response.Forbidden(c, "You are not a member of this workspace")
		default:
			h.logger.Error("Failed to join room", zap.Error(err))
			response.InternalError(c, "Failed to join room")
		}
		return
	}

	response.OK(c, resp)
}

// LeaveRoom godoc
// @Summary Leave a video call room
// @Tags rooms
// @Param roomId path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms/{roomId}/leave [post]
func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	if err := h.roomService.LeaveRoom(c.Request.Context(), roomID, userID); err != nil {
		if errors.Is(err, response.ErrNotInRoom) {
			response.CustomError(c, http.StatusBadRequest, "NOT_IN_ROOM", "User is not in room")
			return
		}
		h.logger.Error("Failed to leave room", zap.Error(err))
		response.InternalError(c, "Failed to leave room")
		return
	}

	response.Success(c, "Left room successfully")
}

// EndRoom godoc
// @Summary End a video call room (creator only)
// @Tags rooms
// @Param roomId path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms/{roomId}/end [post]
func (h *RoomHandler) EndRoom(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	if err := h.roomService.EndRoom(c.Request.Context(), roomID, userID); err != nil {
		if errors.Is(err, response.ErrRoomNotFound) {
			response.NotFound(c, "Room not found")
			return
		}
		h.logger.Error("Failed to end room", zap.Error(err))
		response.InternalError(c, "Failed to end room")
		return
	}

	response.Success(c, "Room ended successfully")
}

// GetParticipants godoc
// @Summary Get room participants
// @Tags rooms
// @Produce json
// @Param roomId path string true "Room ID"
// @Success 200 {array} domain.ParticipantResponse
// @Security BearerAuth
// @Router /rooms/{roomId}/participants [get]
func (h *RoomHandler) GetParticipants(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	participants, err := h.roomService.GetParticipants(c.Request.Context(), roomID)
	if err != nil {
		h.logger.Error("Failed to get participants", zap.Error(err))
		response.InternalError(c, "Failed to get participants")
		return
	}

	response.OK(c, participants)
}

// GetWorkspaceCallHistory godoc
// @Summary Get call history for a workspace
// @Tags history
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /history/workspace/{workspaceId} [get]
func (h *RoomHandler) GetWorkspaceCallHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	token := c.MustGet("token").(string)

	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 20
		}
	}
	if o := c.Query("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
			offset = 0
		}
	}

	histories, total, err := h.roomService.GetWorkspaceCallHistory(c.Request.Context(), workspaceID, userID, token, limit, offset)
	if err != nil {
		if errors.Is(err, response.ErrNotWorkspaceMember) {
			response.Forbidden(c, "You are not a member of this workspace")
			return
		}
		h.logger.Error("Failed to get call history", zap.Error(err))
		response.InternalError(c, "Failed to get call history")
		return
	}

	response.OKWithPagination(c, histories, total, limit, offset)
}

// GetMyCallHistory godoc
// @Summary Get current user's call history
// @Tags history
// @Produce json
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /history/me [get]
func (h *RoomHandler) GetMyCallHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 20
		}
	}
	if o := c.Query("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
			offset = 0
		}
	}

	histories, total, err := h.roomService.GetUserCallHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get call history", zap.Error(err))
		response.InternalError(c, "Failed to get call history")
		return
	}

	response.OKWithPagination(c, histories, total, limit, offset)
}

// GetCallHistory godoc
// @Summary Get a single call history by ID
// @Tags history
// @Produce json
// @Param historyId path string true "History ID"
// @Success 200 {object} domain.CallHistoryResponse
// @Security BearerAuth
// @Router /history/{historyId} [get]
func (h *RoomHandler) GetCallHistory(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid history ID")
		return
	}

	history, err := h.roomService.GetCallHistoryByID(c.Request.Context(), historyID)
	if err != nil {
		h.logger.Error("Failed to get call history", zap.Error(err))
		response.InternalError(c, "Failed to get call history")
		return
	}

	if history == nil {
		response.NotFound(c, "Call history not found")
		return
	}

	response.OK(c, history)
}

// SaveTranscript godoc
// @Summary Save transcript for a room
// @Tags transcript
// @Accept json
// @Produce json
// @Param roomId path string true "Room ID"
// @Param request body domain.SaveTranscriptRequest true "Transcript content"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /rooms/{roomId}/transcript [post]
func (h *RoomHandler) SaveTranscript(c *gin.Context) {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid room ID")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	transcript, err := h.roomService.SaveTranscript(c.Request.Context(), roomID, req.Content)
	if err != nil {
		h.logger.Error("Failed to save transcript", zap.Error(err))
		response.InternalError(c, "Failed to save transcript")
		return
	}

	response.OK(c, transcript)
}

// GetTranscript godoc
// @Summary Get transcript for a call history
// @Tags transcript
// @Produce json
// @Param historyId path string true "Call History ID"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /history/{historyId}/transcript [get]
func (h *RoomHandler) GetTranscript(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid history ID")
		return
	}

	transcript, err := h.roomService.GetTranscriptByCallHistoryID(c.Request.Context(), historyID)
	if err != nil {
		// Return empty content instead of error if not found
		response.OK(c, nil)
		return
	}

	response.OK(c, transcript)
}
