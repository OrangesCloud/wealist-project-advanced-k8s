package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonclient "github.com/OrangesCloud/wealist-advanced-go-pkg/client"
	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
	"project-board-api/internal/metrics"
)

// NotificationType defines notification types matching noti-service
type NotificationType string

const (
	NotificationTypeTaskAssigned    NotificationType = "TASK_ASSIGNED"
	NotificationTypeTaskUpdated     NotificationType = "TASK_STATUS_CHANGED"
	NotificationTypeCommentAdded    NotificationType = "COMMENT_ADDED"
)

// ResourceType defines resource types matching noti-service
type ResourceType string

const (
	ResourceTypeBoard ResourceType = "board"
	ResourceTypeTask  ResourceType = "task"
)

// NotificationEvent represents the payload for creating a notification
type NotificationEvent struct {
	Type         NotificationType       `json:"type"`
	ActorID      uuid.UUID              `json:"actorId"`
	TargetUserID uuid.UUID              `json:"targetUserId"`
	WorkspaceID  uuid.UUID              `json:"workspaceId"`
	ResourceType ResourceType           `json:"resourceType"`
	ResourceID   uuid.UUID              `json:"resourceId"`
	ResourceName *string                `json:"resourceName,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotiClient defines the interface for notification service interactions
type NotiClient interface {
	SendNotification(ctx context.Context, event *NotificationEvent) error
	SendBulkNotifications(ctx context.Context, events []*NotificationEvent) error
}

// notiClient implements NotiClient interface
type notiClient struct {
	*commonclient.BaseHTTPClient
	internalAPIKey string
	metrics        *metrics.Metrics
}

// NewNotiClient creates a new Notification API client
func NewNotiClient(baseURL string, internalAPIKey string, timeout time.Duration, logger *zap.Logger, m *metrics.Metrics) NotiClient {
	return &notiClient{
		BaseHTTPClient: commonclient.NewBaseHTTPClient(baseURL, timeout, logger),
		internalAPIKey: internalAPIKey,
		metrics:        m,
	}
}

// SendNotification sends a notification to noti-service
// This is designed to be called asynchronously (in a goroutine) so notification
// failures don't affect the main business logic
func (c *notiClient) SendNotification(ctx context.Context, event *NotificationEvent) error {
	log := c.log(ctx)
	url := c.BuildURL("/api/internal/notifications")

	log.Debug("Sending notification",
		zap.String("peer.service", "noti-service"),
		zap.String("http.url", url),
		zap.String("notification.type", string(event.Type)),
		zap.String("target.user.id", event.TargetUserID.String()),
	)

	return c.doRequest(ctx, url, event)
}

// SendBulkNotifications sends multiple notifications to noti-service
func (c *notiClient) SendBulkNotifications(ctx context.Context, events []*NotificationEvent) error {
	if len(events) == 0 {
		return nil
	}

	log := c.log(ctx)
	url := c.BuildURL("/api/internal/notifications/bulk")

	payload := map[string]interface{}{
		"notifications": events,
	}

	log.Debug("Sending bulk notifications",
		zap.String("peer.service", "noti-service"),
		zap.String("http.url", url),
		zap.Int("count", len(events)),
	)

	return c.doRequest(ctx, url, payload)
}

// log returns a trace-context aware logger
func (c *notiClient) log(ctx context.Context) *zap.Logger {
	return commnotel.WithTraceContext(ctx, c.Logger)
}

// doRequest performs the HTTP POST request to noti-service
func (c *notiClient) doRequest(ctx context.Context, url string, payload interface{}) error {
	startTime := time.Now()
	log := c.log(ctx)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal notification payload", zap.Error(err))
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Failed to create notification request", zap.Error(err))
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Inject W3C Trace Context headers for distributed tracing
	commnotel.InjectTraceHeaders(ctx, req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-api-key", c.internalAPIKey)

	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(startTime)

	// Record metrics
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if c.metrics != nil {
		c.metrics.RecordExternalAPICall(url, "POST", statusCode, duration, err)
	}

	if err != nil {
		log.Error("Failed to send notification",
			zap.Error(err),
			zap.String("http.url", url),
			zap.Duration("http.duration", duration),
		)
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Warn("Noti service returned error status",
			zap.Int("http.status_code", resp.StatusCode),
			zap.String("http.url", url),
			zap.String("response.body", string(respBody)),
			zap.Duration("http.duration", duration),
		)
		// 알림 전송 실패는 치명적이지 않으므로 로그만 남기고 에러 반환하지 않음
		return nil
	}

	log.Debug("Notification sent successfully",
		zap.Int("http.status_code", resp.StatusCode),
		zap.Duration("http.duration", duration),
	)

	return nil
}

// Helper function to create a task assignment notification
func NewTaskAssignedNotification(actorID, targetUserID, workspaceID, boardID uuid.UUID, boardTitle string) *NotificationEvent {
	title := boardTitle
	return &NotificationEvent{
		Type:         NotificationTypeTaskAssigned,
		ActorID:      actorID,
		TargetUserID: targetUserID,
		WorkspaceID:  workspaceID,
		ResourceType: ResourceTypeTask,
		ResourceID:   boardID,
		ResourceName: &title,
		Metadata: map[string]interface{}{
			"boardTitle": boardTitle,
		},
	}
}

// Helper function to create a task update notification
func NewTaskUpdatedNotification(actorID, targetUserID, workspaceID, boardID uuid.UUID, boardTitle string, changeType string) *NotificationEvent {
	title := boardTitle
	return &NotificationEvent{
		Type:         NotificationTypeTaskUpdated,
		ActorID:      actorID,
		TargetUserID: targetUserID,
		WorkspaceID:  workspaceID,
		ResourceType: ResourceTypeTask,
		ResourceID:   boardID,
		ResourceName: &title,
		Metadata: map[string]interface{}{
			"boardTitle": boardTitle,
			"changeType": changeType,
		},
	}
}
