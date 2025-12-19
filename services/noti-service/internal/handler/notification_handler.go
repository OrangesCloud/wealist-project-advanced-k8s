// Package handler provides HTTP handlers for noti-service endpoints.
//
// This package implements Gin handlers for managing notifications,
// including CRUD operations, SSE streaming, and bulk notification creation.
package handler

import (
	"noti-service/internal/domain"
	"noti-service/internal/response"
	"noti-service/internal/service"
	"noti-service/internal/sse"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationHandler handles HTTP requests for notification operations.
// It provides endpoints for listing, reading, deleting notifications,
// and SSE streaming for real-time updates.
type NotificationHandler struct {
	service    *service.NotificationService
	sseService *sse.SSEService
	logger     *zap.Logger
}

// NewNotificationHandler creates a new NotificationHandler with the given dependencies.
func NewNotificationHandler(
	service *service.NotificationService,
	sseService *sse.SSEService,
	logger *zap.Logger,
) *NotificationHandler {
	return &NotificationHandler{
		service:    service,
		sseService: sseService,
		logger:     logger,
	}
}

// GetNotifications returns paginated notifications for the user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	workspaceID := c.MustGet("workspace_id").(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unreadOnly") == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	result, err := h.service.GetNotifications(c.Request.Context(), userID, workspaceID, page, limit, unreadOnly)
	if err != nil {
		h.logger.Error("failed to get notifications", zap.Error(err))
		response.InternalError(c, "Failed to get notifications")
		return
	}

	c.JSON(200, result)
}

// GetUnreadCount returns unread notification count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	workspaceID := c.MustGet("workspace_id").(uuid.UUID)

	result, err := h.service.GetUnreadCount(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.logger.Error("failed to get unread count", zap.Error(err))
		response.InternalError(c, "Failed to get unread count")
		return
	}

	c.JSON(200, result)
}

// StreamNotifications handles SSE connection
func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	h.sseService.AddClient(c, userID)
}

// MarkAsRead marks a single notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	notification, err := h.service.MarkAsRead(c.Request.Context(), notificationID, userID)
	if err != nil {
		h.logger.Error("failed to mark as read",
			zap.String("notification_id", notificationID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	c.JSON(200, notification)
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	workspaceID := c.MustGet("workspace_id").(uuid.UUID)

	count, err := h.service.MarkAllAsRead(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.logger.Error("failed to mark all as read", zap.Error(err))
		response.InternalError(c, "Failed to mark all as read")
		return
	}

	c.JSON(200, gin.H{"markedAsRead": count})
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	deleted, err := h.service.DeleteNotification(c.Request.Context(), notificationID, userID)
	if err != nil {
		h.logger.Error("failed to delete notification",
			zap.String("notification_id", notificationID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	if !deleted {
		response.NotFound(c, "Notification not found")
		return
	}

	response.NoContent(c)
}

// CreateNotification creates a new notification (internal API)
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var event domain.NotificationEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	notification, err := h.service.CreateNotification(c.Request.Context(), &event)
	if err != nil {
		h.logger.Error("failed to create notification",
			zap.String("type", string(event.Type)),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	c.JSON(201, notification)
}

// CreateBulkNotifications creates multiple notifications (internal API)
func (h *NotificationHandler) CreateBulkNotifications(c *gin.Context) {
	var req struct {
		Notifications []domain.NotificationEvent `json:"notifications" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	notifications, err := h.service.CreateBulkNotifications(c.Request.Context(), req.Notifications)
	if err != nil {
		h.logger.Error("failed to create bulk notifications", zap.Error(err))
		response.InternalError(c, "Failed to create notifications")
		return
	}

	c.JSON(201, gin.H{
		"created":       len(notifications),
		"notifications": notifications,
	})
}
