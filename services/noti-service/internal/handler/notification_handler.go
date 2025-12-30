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

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
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

// log returns a trace-context aware logger
func (h *NotificationHandler) log(c *gin.Context) *zap.Logger {
	return commnotel.WithTraceContext(c.Request.Context(), h.logger)
}

// GetNotifications returns paginated notifications for the user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	log := h.log(c)
	log.Debug("GetNotifications started")

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

	log.Debug("GetNotifications fetching",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.Bool("unreadOnly", unreadOnly))

	result, err := h.service.GetNotifications(c.Request.Context(), userID, workspaceID, page, limit, unreadOnly)
	if err != nil {
		log.Error("GetNotifications failed", zap.Error(err))
		response.InternalError(c, "Failed to get notifications")
		return
	}

	log.Debug("GetNotifications completed",
		zap.String("enduser.id", userID.String()),
		zap.Int("notification.count", len(result.Notifications)))
	c.JSON(200, result)
}

// GetUnreadCount returns unread notification count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	log := h.log(c)
	log.Debug("GetUnreadCount started")

	userID := c.MustGet("user_id").(uuid.UUID)
	workspaceID := c.MustGet("workspace_id").(uuid.UUID)

	log.Debug("GetUnreadCount fetching",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()))

	result, err := h.service.GetUnreadCount(c.Request.Context(), userID, workspaceID)
	if err != nil {
		log.Error("GetUnreadCount failed", zap.Error(err))
		response.InternalError(c, "Failed to get unread count")
		return
	}

	log.Debug("GetUnreadCount completed",
		zap.String("enduser.id", userID.String()),
		zap.Int64("unread.count", result.Count))
	c.JSON(200, result)
}

// StreamNotifications handles SSE connection
func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	log := h.log(c)
	userID := c.MustGet("user_id").(uuid.UUID)

	log.Info("SSE client connected", zap.String("enduser.id", userID.String()))
	h.sseService.AddClient(c, userID)
}

// MarkAsRead marks a single notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	log := h.log(c)
	log.Debug("MarkAsRead started")

	userID := c.MustGet("user_id").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("MarkAsRead invalid notification ID")
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	log.Debug("MarkAsRead calling service",
		zap.String("notification.id", notificationID.String()),
		zap.String("enduser.id", userID.String()))

	notification, err := h.service.MarkAsRead(c.Request.Context(), notificationID, userID)
	if err != nil {
		log.Error("MarkAsRead failed",
			zap.String("notification.id", notificationID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	log.Info("Notification marked as read", zap.String("notification.id", notificationID.String()))
	c.JSON(200, notification)
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	log := h.log(c)
	log.Debug("MarkAllAsRead started")

	userID := c.MustGet("user_id").(uuid.UUID)
	workspaceID := c.MustGet("workspace_id").(uuid.UUID)

	log.Debug("MarkAllAsRead calling service",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()))

	count, err := h.service.MarkAllAsRead(c.Request.Context(), userID, workspaceID)
	if err != nil {
		log.Error("MarkAllAsRead failed", zap.Error(err))
		response.InternalError(c, "Failed to mark all as read")
		return
	}

	log.Info("All notifications marked as read",
		zap.String("enduser.id", userID.String()),
		zap.Int64("marked.count", count))
	c.JSON(200, gin.H{"markedAsRead": count})
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	log := h.log(c)
	log.Debug("DeleteNotification started")

	userID := c.MustGet("user_id").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("DeleteNotification invalid notification ID")
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	log.Debug("DeleteNotification calling service",
		zap.String("notification.id", notificationID.String()),
		zap.String("enduser.id", userID.String()))

	deleted, err := h.service.DeleteNotification(c.Request.Context(), notificationID, userID)
	if err != nil {
		log.Error("DeleteNotification failed",
			zap.String("notification.id", notificationID.String()),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	if !deleted {
		log.Debug("DeleteNotification not found", zap.String("notification.id", notificationID.String()))
		response.NotFound(c, "Notification not found")
		return
	}

	log.Info("Notification deleted", zap.String("notification.id", notificationID.String()))
	response.NoContent(c)
}

// CreateNotification creates a new notification (internal API)
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	log := h.log(c)
	log.Debug("CreateNotification started")

	var event domain.NotificationEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		log.Warn("CreateNotification validation failed", zap.Error(err))
		response.BadRequest(c, err.Error())
		return
	}

	log.Debug("CreateNotification calling service",
		zap.String("notification.type", string(event.Type)),
		zap.String("target.user.id", event.TargetUserID.String()))

	notification, err := h.service.CreateNotification(c.Request.Context(), &event)
	if err != nil {
		log.Error("CreateNotification failed",
			zap.String("notification.type", string(event.Type)),
			zap.Error(err))
		response.HandleServiceError(c, err)
		return
	}

	log.Info("Notification created",
		zap.String("notification.id", notification.ID.String()),
		zap.String("notification.type", string(notification.Type)))
	c.JSON(201, notification)
}

// CreateBulkNotifications creates multiple notifications (internal API)
func (h *NotificationHandler) CreateBulkNotifications(c *gin.Context) {
	log := h.log(c)
	log.Debug("CreateBulkNotifications started")

	var req struct {
		Notifications []domain.NotificationEvent `json:"notifications" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("CreateBulkNotifications validation failed", zap.Error(err))
		response.BadRequest(c, err.Error())
		return
	}

	log.Debug("CreateBulkNotifications calling service",
		zap.Int("notification.count", len(req.Notifications)))

	notifications, err := h.service.CreateBulkNotifications(c.Request.Context(), req.Notifications)
	if err != nil {
		log.Error("CreateBulkNotifications failed", zap.Error(err))
		response.InternalError(c, "Failed to create notifications")
		return
	}

	log.Info("Bulk notifications created", zap.Int("created.count", len(notifications)))
	c.JSON(201, gin.H{
		"created":       len(notifications),
		"notifications": notifications,
	})
}
