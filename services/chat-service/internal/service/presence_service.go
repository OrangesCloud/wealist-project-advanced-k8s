// Package service는 chat-service의 비즈니스 로직을 구현합니다.
package service

import (
	"chat-service/internal/domain"
	"chat-service/internal/metrics"
	"chat-service/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PresenceService는 사용자 온라인 상태를 관리합니다.
type PresenceService struct {
	repo        *repository.PresenceRepository
	redis       *redis.Client
	logger      *zap.Logger
	metrics     *metrics.Metrics
	onlineUsers map[uuid.UUID]map[uuid.UUID]bool // workspaceID -> userID -> online
	mu          sync.RWMutex
}

// NewPresenceService는 새 PresenceService를 생성합니다.
func NewPresenceService(
	repo *repository.PresenceRepository,
	redis *redis.Client,
	logger *zap.Logger,
	m *metrics.Metrics,
) *PresenceService {
	return &PresenceService{
		repo:        repo,
		redis:       redis,
		logger:      logger,
		metrics:     m,
		onlineUsers: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

// SetUserOnline은 사용자를 온라인 상태로 설정합니다.
func (s *PresenceService) SetUserOnline(ctx context.Context, userID, workspaceID uuid.UUID) error {
	// 메모리 내 상태 업데이트
	s.mu.Lock()
	if s.onlineUsers[workspaceID] == nil {
		s.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
	}
	s.onlineUsers[workspaceID][userID] = true
	onlineCount := s.countTotalOnlineUsers()
	s.mu.Unlock()

	// 데이터베이스 업데이트
	if err := s.repo.SetStatus(userID, workspaceID, domain.PresenceStatusOnline); err != nil {
		s.logger.Error("DB 온라인 상태 설정 실패",
			zap.String("user_id", userID.String()),
			zap.Error(err))
	}

	// 📊 메트릭: 온라인 사용자 수 업데이트
	if s.metrics != nil {
		s.metrics.SetOnlineUsersTotal(int64(onlineCount))
	}

	s.logger.Debug("사용자 온라인 설정",
		zap.String("user_id", userID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("total_online", onlineCount))

	// 상태 변경 브로드캐스트
	s.broadcastStatus(ctx, userID, workspaceID, domain.PresenceStatusOnline)

	return nil
}

// SetUserOffline은 사용자를 오프라인 상태로 설정합니다.
func (s *PresenceService) SetUserOffline(ctx context.Context, userID, workspaceID uuid.UUID) error {
	// 메모리 내 상태 업데이트
	s.mu.Lock()
	if s.onlineUsers[workspaceID] != nil {
		delete(s.onlineUsers[workspaceID], userID)
		if len(s.onlineUsers[workspaceID]) == 0 {
			delete(s.onlineUsers, workspaceID)
		}
	}
	onlineCount := s.countTotalOnlineUsers()
	s.mu.Unlock()

	// 데이터베이스 업데이트
	if err := s.repo.SetOffline(userID); err != nil {
		s.logger.Error("DB 오프라인 상태 설정 실패",
			zap.String("user_id", userID.String()),
			zap.Error(err))
	}

	// 📊 메트릭: 온라인 사용자 수 업데이트
	if s.metrics != nil {
		s.metrics.SetOnlineUsersTotal(int64(onlineCount))
	}

	s.logger.Debug("사용자 오프라인 설정",
		zap.String("user_id", userID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("total_online", onlineCount))

	// 상태 변경 브로드캐스트
	s.broadcastStatus(ctx, userID, workspaceID, domain.PresenceStatusOffline)

	return nil
}

// countTotalOnlineUsers는 전체 온라인 사용자 수를 계산합니다.
// 호출 시 mu 락이 이미 획득된 상태여야 합니다.
func (s *PresenceService) countTotalOnlineUsers() int {
	count := 0
	for _, users := range s.onlineUsers {
		count += len(users)
	}
	return count
}

func (s *PresenceService) SetUserAway(ctx context.Context, userID, workspaceID uuid.UUID) error {
	if err := s.repo.SetStatus(userID, workspaceID, domain.PresenceStatusAway); err != nil {
		return err
	}

	s.broadcastStatus(ctx, userID, workspaceID, domain.PresenceStatusAway)
	return nil
}

func (s *PresenceService) GetUserStatus(ctx context.Context, userID uuid.UUID) (*domain.UserPresence, error) {
	return s.repo.GetUserStatus(userID)
}

func (s *PresenceService) GetOnlineUsers(ctx context.Context, workspaceID *uuid.UUID) ([]domain.UserPresence, error) {
	return s.repo.GetOnlineUsers(workspaceID)
}

func (s *PresenceService) IsUserOnline(userID, workspaceID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ws, ok := s.onlineUsers[workspaceID]; ok {
		return ws[userID]
	}
	return false
}

func (s *PresenceService) GetOnlineUsersInMemory(workspaceID uuid.UUID) []uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []uuid.UUID
	if ws, ok := s.onlineUsers[workspaceID]; ok {
		for userID := range ws {
			users = append(users, userID)
		}
	}
	return users
}

func (s *PresenceService) broadcastStatus(ctx context.Context, userID, workspaceID uuid.UUID, status domain.PresenceStatus) {
	if s.redis == nil {
		return
	}

	channel := fmt.Sprintf("presence:workspace:%s", workspaceID.String())
	data, err := json.Marshal(map[string]interface{}{
		"type":   "USER_STATUS",
		"userId": userID.String(),
		"status": status,
	})
	if err != nil {
		s.logger.Error("failed to marshal status for broadcast", zap.Error(err))
		return
	}

	if err := s.redis.Publish(ctx, channel, data).Err(); err != nil {
		s.logger.Error("failed to broadcast status", zap.Error(err))
	}
}
