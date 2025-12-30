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

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
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

// log returns a trace-context aware logger
func (s *NotificationService) log(ctx context.Context) *zap.Logger {
	return commnotel.WithTraceContext(ctx, s.logger)
}

// CreateNotification creates a new notification from an event and publishes it via Redis.
func (s *NotificationService) CreateNotification(ctx context.Context, event *domain.NotificationEvent) (*domain.Notification, error) {
	log := s.log(ctx)
	log.Debug("CreateNotification service started",
		zap.String("notification.type", string(event.Type)),
		zap.String("target.user.id", event.TargetUserID.String()))

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
		log.Error("CreateNotification failed to save", zap.Error(err))
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

	log.Info("Notification created",
		zap.String("notification.id", notification.ID.String()),
		zap.String("notification.type", string(notification.Type)),
		zap.String("target.user.id", notification.TargetUserID.String()))

	return notification, nil
}

// CreateBulkNotifications creates multiple notifications from a list of events.
// Errors for individual notifications are logged but don't stop the batch.
func (s *NotificationService) CreateBulkNotifications(ctx context.Context, events []domain.NotificationEvent) ([]domain.Notification, error) {
	log := s.log(ctx)
	log.Debug("CreateBulkNotifications service started", zap.Int("event.count", len(events)))

	notifications := make([]domain.Notification, 0, len(events))

	for _, event := range events {
		notification, err := s.CreateNotification(ctx, &event)
		if err != nil {
			log.Error("CreateBulkNotifications failed for one", zap.Error(err))
			continue
		}
		notifications = append(notifications, *notification)
	}

	log.Debug("CreateBulkNotifications completed",
		zap.Int("created.count", len(notifications)),
		zap.Int("total.count", len(events)))

	return notifications, nil
}

// GetNotifications returns paginated notifications for a user in a workspace.
func (s *NotificationService) GetNotifications(ctx context.Context, userID, workspaceID uuid.UUID, page, limit int, unreadOnly bool) (*domain.PaginatedNotifications, error) {
	log := s.log(ctx)
	log.Debug("GetNotifications service started",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit))

	notifications, total, err := s.repo.GetByUserAndWorkspace(userID, workspaceID, page, limit, unreadOnly)
	if err != nil {
		log.Error("GetNotifications failed", zap.Error(err))
		return nil, err
	}

	hasMore := int64(page*limit) < total

	log.Debug("GetNotifications completed",
		zap.Int64("total", total),
		zap.Int("fetched.count", len(notifications)))

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
	log := s.log(ctx)
	log.Debug("GetNotificationByID service started",
		zap.String("notification.id", id.String()),
		zap.String("enduser.id", userID.String()))

	notification, err := s.repo.GetByIDAndUserID(id, userID)
	if err != nil {
		log.Warn("GetNotificationByID not found",
			zap.String("notification.id", id.String()),
			zap.String("enduser.id", userID.String()),
			zap.Error(err))
		return nil, response.ErrNotificationNotFound
	}

	log.Debug("GetNotificationByID completed", zap.String("notification.id", id.String()))
	return notification, nil
}

// MarkAsRead marks a single notification as read and invalidates the cache.
// 읽음 처리 성공 시 메트릭을 기록합니다.
// userID로 소유권을 검증합니다.
func (s *NotificationService) MarkAsRead(ctx context.Context, id, userID uuid.UUID) (*domain.Notification, error) {
	log := s.log(ctx)
	log.Debug("MarkAsRead service started",
		zap.String("notification.id", id.String()),
		zap.String("enduser.id", userID.String()))

	notification, err := s.repo.MarkAsRead(id, userID)
	if err != nil {
		log.Warn("MarkAsRead failed",
			zap.String("notification.id", id.String()),
			zap.String("enduser.id", userID.String()),
			zap.Error(err))
		return nil, response.ErrNotificationNotFound
	}

	// Invalidate cache
	s.invalidateUnreadCountCache(ctx, notification.TargetUserID, notification.WorkspaceID)

	// 메트릭 기록: 알림 읽음 처리
	if s.metrics != nil {
		s.metrics.RecordNotificationRead()
	}

	log.Info("Notification marked as read",
		zap.String("notification.id", id.String()),
		zap.String("enduser.id", userID.String()))

	return notification, nil
}

// MarkAllAsRead marks all notifications as read for a user in a workspace.
// 전체 읽음 처리된 알림 수만큼 메트릭을 기록합니다.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID, workspaceID uuid.UUID) (int64, error) {
	log := s.log(ctx)
	log.Debug("MarkAllAsRead service started",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()))

	count, err := s.repo.MarkAllAsRead(userID, workspaceID)
	if err != nil {
		log.Error("MarkAllAsRead failed",
			zap.String("enduser.id", userID.String()),
			zap.String("workspace.id", workspaceID.String()),
			zap.Error(err))
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

	log.Info("All notifications marked as read",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()),
		zap.Int64("marked.count", count))

	return count, nil
}

// GetUnreadCount returns the unread notification count, using cache when available.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.UnreadCount, error) {
	log := s.log(ctx)
	log.Debug("GetUnreadCount service started",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", workspaceID.String()))

	cacheKey := fmt.Sprintf("unread:%s:%s", userID.String(), workspaceID.String())

	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Int64()
		if err == nil {
			log.Debug("GetUnreadCount cache hit", zap.Int64("unread.count", cached))
			return &domain.UnreadCount{
				Count:       cached,
				WorkspaceID: workspaceID,
			}, nil
		}
	}

	// Get from DB
	count, err := s.repo.GetUnreadCount(userID, workspaceID)
	if err != nil {
		log.Error("GetUnreadCount failed", zap.Error(err))
		return nil, err
	}

	// Cache the result
	if s.redis != nil {
		ttl := time.Duration(s.config.App.CacheUnreadTTL) * time.Second
		s.redis.Set(ctx, cacheKey, count, ttl)
	}

	log.Debug("GetUnreadCount completed", zap.Int64("unread.count", count))
	return &domain.UnreadCount{
		Count:       count,
		WorkspaceID: workspaceID,
	}, nil
}

// DeleteNotification deletes a notification and invalidates the cache if it was unread.
// 삭제 성공 시 메트릭을 기록합니다.
func (s *NotificationService) DeleteNotification(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	log := s.log(ctx)
	log.Debug("DeleteNotification service started",
		zap.String("notification.id", id.String()),
		zap.String("enduser.id", userID.String()))

	// Get notification to find workspace ID for cache invalidation
	notification, _ := s.repo.GetByIDAndUserID(id, userID)

	deleted, wasUnread, err := s.repo.Delete(id, userID)
	if err != nil {
		log.Error("DeleteNotification failed",
			zap.String("notification.id", id.String()),
			zap.String("enduser.id", userID.String()),
			zap.Error(err))
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
		log.Info("Notification deleted",
			zap.String("notification.id", id.String()),
			zap.String("enduser.id", userID.String()))
	}

	return deleted, nil
}

// CleanupOldNotifications removes read notifications older than configured days.
func (s *NotificationService) CleanupOldNotifications(ctx context.Context) (int64, error) {
	log := s.log(ctx)
	log.Debug("CleanupOldNotifications started", zap.Int("cleanup.days", s.config.App.CleanupDays))

	count, err := s.repo.CleanupOld(s.config.App.CleanupDays)
	if err != nil {
		log.Error("CleanupOldNotifications failed", zap.Error(err))
		return 0, err
	}

	log.Info("Old notifications cleaned up", zap.Int64("deleted.count", count))
	return count, err
}

// publishNotification publishes a notification to Redis for SSE delivery.
func (s *NotificationService) publishNotification(ctx context.Context, notification *domain.Notification) {
	log := s.log(ctx)
	if s.redis == nil {
		return
	}

	channel := fmt.Sprintf("notifications:user:%s", notification.TargetUserID.String())
	data, err := json.Marshal(notification)
	if err != nil {
		log.Error("publishNotification marshal failed", zap.Error(err))
		return
	}

	if err := s.redis.Publish(ctx, channel, data).Err(); err != nil {
		log.Error("publishNotification Redis publish failed", zap.Error(err))
	} else {
		log.Debug("Notification published to Redis",
			zap.String("channel", channel),
			zap.String("notification.id", notification.ID.String()))
	}
}

// invalidateUnreadCountCache removes the cached unread count for a user/workspace.
func (s *NotificationService) invalidateUnreadCountCache(ctx context.Context, userID, workspaceID uuid.UUID) {
	log := s.log(ctx)
	if s.redis == nil {
		return
	}

	cacheKey := fmt.Sprintf("unread:%s:%s", userID.String(), workspaceID.String())
	if err := s.redis.Del(ctx, cacheKey).Err(); err != nil {
		log.Error("invalidateUnreadCountCache failed", zap.Error(err))
	} else {
		log.Debug("Unread count cache invalidated", zap.String("cache.key", cacheKey))
	}
}
