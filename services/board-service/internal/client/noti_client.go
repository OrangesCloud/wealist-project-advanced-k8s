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

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
)

// NotificationEvent represents a notification to be sent
type NotificationEvent struct {
	Type         string                 `json:"type"`
	ActorID      string                 `json:"actorId"`
	TargetUserID string                 `json:"targetUserId"`
	WorkspaceID  string                 `json:"workspaceId"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId"`
	ResourceName *string                `json:"resourceName,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotiClient defines the interface for notification service interactions
type NotiClient interface {
	SendNotification(ctx context.Context, event NotificationEvent) error
	SendBulkNotifications(ctx context.Context, events []NotificationEvent) error
}

// notiClient implements NotiClient interface
type notiClient struct {
	baseURL       string
	internalAPIKey string
	httpClient    *http.Client
	logger        *zap.Logger
}

// NewNotiClient creates a new Notification service client
func NewNotiClient(baseURL string, internalAPIKey string, timeout time.Duration, logger *zap.Logger) NotiClient {
	return &notiClient{
		baseURL:       baseURL,
		internalAPIKey: internalAPIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendNotification sends a single notification to noti-service
func (c *notiClient) SendNotification(ctx context.Context, event NotificationEvent) error {
	if c.baseURL == "" {
		c.logger.Debug("NotiClient: baseURL not configured, skipping notification")
		return nil
	}

	url := fmt.Sprintf("%s/api/internal/notifications", c.baseURL)

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal notification event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-api-key", c.internalAPIKey)

	log := commnotel.WithTraceContext(ctx, c.logger)
	log.Debug("Sending notification to noti-service",
		zap.String("type", event.Type),
		zap.String("targetUserId", event.TargetUserID),
		zap.String("resourceType", event.ResourceType))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("Failed to send notification", zap.Error(err))
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Error("Notification service returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	log.Info("Notification sent successfully",
		zap.String("type", event.Type),
		zap.String("targetUserId", event.TargetUserID))

	return nil
}

// SendBulkNotifications sends multiple notifications to noti-service
func (c *notiClient) SendBulkNotifications(ctx context.Context, events []NotificationEvent) error {
	if c.baseURL == "" || len(events) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/api/internal/notifications/bulk", c.baseURL)

	body, err := json.Marshal(map[string]interface{}{
		"notifications": events,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal bulk notifications: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-api-key", c.internalAPIKey)

	log := commnotel.WithTraceContext(ctx, c.logger)
	log.Debug("Sending bulk notifications to noti-service", zap.Int("count", len(events)))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("Failed to send bulk notifications", zap.Error(err))
		return fmt.Errorf("failed to send bulk notifications: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Error("Notification service returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	log.Info("Bulk notifications sent successfully", zap.Int("count", len(events)))
	return nil
}

// Helper function to create a task assignment notification
func NewTaskAssignedNotification(actorID, targetUserID, workspaceID, boardID uuid.UUID, boardTitle string) NotificationEvent {
	title := boardTitle
	return NotificationEvent{
		Type:         "TASK_ASSIGNED",
		ActorID:      actorID.String(),
		TargetUserID: targetUserID.String(),
		WorkspaceID:  workspaceID.String(),
		ResourceType: "task",
		ResourceID:   boardID.String(),
		ResourceName: &title,
		Metadata: map[string]interface{}{
			"boardTitle": boardTitle,
		},
	}
}

// Helper function to create a task update notification
func NewTaskUpdatedNotification(actorID, targetUserID, workspaceID, boardID uuid.UUID, boardTitle string, changeType string) NotificationEvent {
	title := boardTitle
	return NotificationEvent{
		Type:         "TASK_STATUS_CHANGED",
		ActorID:      actorID.String(),
		TargetUserID: targetUserID.String(),
		WorkspaceID:  workspaceID.String(),
		ResourceType: "task",
		ResourceID:   boardID.String(),
		ResourceName: &title,
		Metadata: map[string]interface{}{
			"boardTitle": boardTitle,
			"changeType": changeType,
		},
	}
}
