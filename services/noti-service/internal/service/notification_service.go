// Package service implements business logic for noti-service.
//
// This package provides the NotificationService for managing notifications,
// including creation, delivery via Redis pub/sub, and cache management.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"noti-service/internal/config"
	"noti-service/internal/domain"
	"noti-service/internal/metrics"
	"noti-service/internal/repository"
	"noti-service/internal/response"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NotificationService provides notification management operations.
// It handles notification CRUD, Redis pub/sub for real-time delivery,
// and caching for unread count optimization.
// 메트릭과 로깅을 통해 모니터링을 지원합니다.
type NotificationService struct {
	repo    *repository.NotificationRepository
	redis   *redis.Client
	config  *config.Config
	logger  *zap.Logger
	metrics *metrics.Metrics // 메트릭 수집을 위한 필드
}

// NewNotificationService creates a new NotificationService with the given dependencies.
// metrics 파라미터가 nil인 경우에도 안전하게 동작합니다.
func NewNotificationService(
	repo *repository.NotificationRepository,
	redis *redis.Client,
	config *config.Config,
	logger *zap.Logger,
	m *metrics.Metrics,
) *NotificationService {
	return &NotificationService{
		repo:    repo,
		redis:   redis,
		config:  config,
		logger:  logger,
		metrics: m,
	}
}

// CreateNotification creates a new notification from an event and publishes it via Redis.
func (s *NotificationService) CreateNotification(ctx context.Context, event *domain.NotificationEvent) (*domain.Notification, error) {
	notification := &domain.Notification{
		ID:           uuid.New(),
		Type:         event.Type,
		ActorID:      event.ActorID,
		TargetUserID: event.TargetUserID,
		WorkspaceID:  event.WorkspaceID,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		ResourceName: event.ResourceName,
		Metadata:     event.Metadata,
		IsRead:       false,
		CreatedAt:    time.Now(),
	}

	if event.OccurredAt != nil {
		notification.CreatedAt = *event.OccurredAt
	}

	if err := s.repo.Create(notification); err != nil {
		return nil, err
	}

	// Publish to Redis for SSE clients
	s.publishNotification(ctx, notification)

	// Invalidate cache
	s.invalidateUnreadCountCache(ctx, notification.TargetUserID, notification.WorkspaceID)

	// 메트릭 기록: 알림 생성 성공
	if s.metrics != nil {
		s.metrics.RecordNotificationCreated()
	}

	s.logger.Info("notification created",
		zap.String("id", notification.ID.String()),
		zap.String("type", string(notification.Type)),
		zap.String("targetUserId", notification.TargetUserID.String()),
	)

	return notification, nil
}

// CreateBulkNotifications creates multiple notifications from a list of events.
// Errors for individual notifications are logged but don't stop the batch.
func (s *NotificationService) CreateBulkNotifications(ctx context.Context, events []domain.NotificationEvent) ([]domain.Notification, error) {
	notifications := make([]domain.Notification, 0, len(events))

	for _, event := range events {
		notification, err := s.CreateNotification(ctx, &event)
		if err != nil {
			s.logger.Error("failed to create notification in bulk", zap.Error(err))
			continue
		}
		notifications = append(notifications, *notification)
	}

	return notifications, nil
}

// GetNotifications returns paginated notifications for a user in a workspace.
func (s *NotificationService) GetNotifications(ctx context.Context, userID, workspaceID uuid.UUID, page, limit int, unreadOnly bool) (*domain.PaginatedNotifications, error) {
	notifications, total, err := s.repo.GetByUserAndWorkspace(userID, workspaceID, page, limit, unreadOnly)
	if err != nil {
		return nil, err
	}

	hasMore := int64(page*limit) < total

	return &domain.PaginatedNotifications{
		Notifications: notifications,
		Total:         total,
		Page:          page,
		Limit:         limit,
		HasMore:       hasMore,
	}, nil
}

// GetNotificationByID retrieves a single notification by ID for a specific user.
// userID를 함께 조회하여 소유권을 검증합니다.
func (s *NotificationService) GetNotificationByID(ctx context.Context, id, userID uuid.UUID) (*domain.Notification, error) {
	notification, err := s.repo.GetByIDAndUserID(id, userID)
	if err != nil {
		s.logger.Warn("notification not found",
			zap.String("notificationId", id.String()),
			zap.String("userId", userID.String()),
			zap.Error(err))
		return nil, response.ErrNotificationNotFound
	}
	return notification, nil
}

// MarkAsRead marks a single notification as read and invalidates the cache.
// 읽음 처리 성공 시 메트릭을 기록합니다.
// userID로 소유권을 검증합니다.
func (s *NotificationService) MarkAsRead(ctx context.Context, id, userID uuid.UUID) (*domain.Notification, error) {
	notification, err := s.repo.MarkAsRead(id, userID)
	if err != nil {
		s.logger.Warn("failed to mark notification as read",
			zap.String("notificationId", id.String()),
			zap.String("userId", userID.String()),
			zap.Error(err),
		)
		return nil, response.ErrNotificationNotFound
	}

	// Invalidate cache
	s.invalidateUnreadCountCache(ctx, notification.TargetUserID, notification.WorkspaceID)

	// 메트릭 기록: 알림 읽음 처리
	if s.metrics != nil {
		s.metrics.RecordNotificationRead()
	}

	s.logger.Info("notification marked as read",
		zap.String("notificationId", id.String()),
		zap.String("userId", userID.String()),
	)

	return notification, nil
}

// MarkAllAsRead marks all notifications as read for a user in a workspace.
// 전체 읽음 처리된 알림 수만큼 메트릭을 기록합니다.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID, workspaceID uuid.UUID) (int64, error) {
	count, err := s.repo.MarkAllAsRead(userID, workspaceID)
	if err != nil {
		s.logger.Error("failed to mark all notifications as read",
			zap.String("userId", userID.String()),
			zap.String("workspaceId", workspaceID.String()),
			zap.Error(err),
		)
		return 0, err
	}

	// Invalidate cache
	s.invalidateUnreadCountCache(ctx, userID, workspaceID)

	// 메트릭 기록: 읽음 처리된 알림 수만큼 카운터 증가
	if s.metrics != nil && count > 0 {
		for i := int64(0); i < count; i++ {
			s.metrics.RecordNotificationRead()
		}
	}

	s.logger.Info("all notifications marked as read",
		zap.String("userId", userID.String()),
		zap.String("workspaceId", workspaceID.String()),
		zap.Int64("count", count),
	)

	return count, nil
}

// GetUnreadCount returns the unread notification count, using cache when available.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.UnreadCount, error) {
	cacheKey := fmt.Sprintf("unread:%s:%s", userID.String(), workspaceID.String())

	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Int64()
		if err == nil {
			return &domain.UnreadCount{
				Count:       cached,
				WorkspaceID: workspaceID,
			}, nil
		}
	}

	// Get from DB
	count, err := s.repo.GetUnreadCount(userID, workspaceID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.redis != nil {
		ttl := time.Duration(s.config.App.CacheUnreadTTL) * time.Second
		s.redis.Set(ctx, cacheKey, count, ttl)
	}

	return &domain.UnreadCount{
		Count:       count,
		WorkspaceID: workspaceID,
	}, nil
}

// DeleteNotification deletes a notification and invalidates the cache if it was unread.
// 삭제 성공 시 메트릭을 기록합니다.
func (s *NotificationService) DeleteNotification(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	// Get notification to find workspace ID for cache invalidation
	notification, _ := s.repo.GetByIDAndUserID(id, userID)

	deleted, wasUnread, err := s.repo.Delete(id, userID)
	if err != nil {
		s.logger.Error("failed to delete notification",
			zap.String("notificationId", id.String()),
			zap.String("userId", userID.String()),
			zap.Error(err),
		)
		return false, err
	}

	// Invalidate cache if was unread
	if deleted && wasUnread && notification != nil {
		s.invalidateUnreadCountCache(ctx, userID, notification.WorkspaceID)
	}

	// 메트릭 기록: 알림 삭제 성공
	if deleted && s.metrics != nil {
		s.metrics.RecordNotificationDeleted()
	}

	if deleted {
		s.logger.Info("notification deleted",
			zap.String("notificationId", id.String()),
			zap.String("userId", userID.String()),
		)
	}

	return deleted, nil
}

// CleanupOldNotifications removes read notifications older than configured days.
func (s *NotificationService) CleanupOldNotifications(ctx context.Context) (int64, error) {
	return s.repo.CleanupOld(s.config.App.CleanupDays)
}

// publishNotification publishes a notification to Redis for SSE delivery.
func (s *NotificationService) publishNotification(ctx context.Context, notification *domain.Notification) {
	if s.redis == nil {
		return
	}

	channel := fmt.Sprintf("notifications:user:%s", notification.TargetUserID.String())
	data, err := json.Marshal(notification)
	if err != nil {
		s.logger.Error("failed to marshal notification for publish", zap.Error(err))
		return
	}

	if err := s.redis.Publish(ctx, channel, data).Err(); err != nil {
		s.logger.Error("failed to publish notification", zap.Error(err))
	}
}

// invalidateUnreadCountCache removes the cached unread count for a user/workspace.
func (s *NotificationService) invalidateUnreadCountCache(ctx context.Context, userID, workspaceID uuid.UUID) {
	if s.redis == nil {
		return
	}

	cacheKey := fmt.Sprintf("unread:%s:%s", userID.String(), workspaceID.String())
	if err := s.redis.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Error("failed to invalidate unread cache", zap.Error(err))
	}
}
