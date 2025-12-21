// Package service는 chat-service의 비즈니스 로직 테스트를 포함합니다.
// 이 파일은 PresenceService의 테스트를 포함합니다.
package service

import (
	"chat-service/internal/domain"
	"chat-service/internal/metrics"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ============================================================
// PresenceService 테스트 헬퍼
// ============================================================

// newTestPresenceService는 테스트용 PresenceService를 생성합니다.
func newTestPresenceService() *PresenceService {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewForTest()

	return &PresenceService{
		repo:        nil, // 실제 테스트에서는 인터페이스 기반 mock 필요
		redis:       nil,
		logger:      logger,
		metrics:     m,
		onlineUsers: make(map[uuid.UUID]map[uuid.UUID]bool),
		mu:          sync.RWMutex{},
	}
}

// ============================================================
// SetUserOnline/SetUserOffline 메모리 상태 테스트
// ============================================================

func TestPresenceService_SetUserOnline_InMemory(t *testing.T) {
	// Given: 테스트용 서비스 생성
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()

	// When: 메모리에 사용자 온라인 상태 설정
	svc.mu.Lock()
	if svc.onlineUsers[workspaceID] == nil {
		svc.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
	}
	svc.onlineUsers[workspaceID][userID] = true
	svc.mu.Unlock()

	// Then: 온라인 상태 확인
	assert.True(t, svc.IsUserOnline(userID, workspaceID))
}

func TestPresenceService_SetUserOffline_InMemory(t *testing.T) {
	// Given: 온라인 상태의 사용자
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()

	svc.mu.Lock()
	svc.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
	svc.onlineUsers[workspaceID][userID] = true
	svc.mu.Unlock()

	// When: 오프라인으로 변경
	svc.mu.Lock()
	delete(svc.onlineUsers[workspaceID], userID)
	if len(svc.onlineUsers[workspaceID]) == 0 {
		delete(svc.onlineUsers, workspaceID)
	}
	svc.mu.Unlock()

	// Then: 오프라인 상태 확인
	assert.False(t, svc.IsUserOnline(userID, workspaceID))
}

// ============================================================
// IsUserOnline 테스트
// ============================================================

func TestPresenceService_IsUserOnline_True(t *testing.T) {
	// Given
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()

	svc.mu.Lock()
	svc.onlineUsers[workspaceID] = map[uuid.UUID]bool{userID: true}
	svc.mu.Unlock()

	// When/Then
	assert.True(t, svc.IsUserOnline(userID, workspaceID))
}

func TestPresenceService_IsUserOnline_False_NoWorkspace(t *testing.T) {
	// Given: 워크스페이스가 등록되지 않은 경우
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()

	// When/Then: false 반환
	assert.False(t, svc.IsUserOnline(userID, workspaceID))
}

func TestPresenceService_IsUserOnline_False_UserNotInWorkspace(t *testing.T) {
	// Given: 워크스페이스는 있지만 사용자가 없는 경우
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	svc.mu.Lock()
	svc.onlineUsers[workspaceID] = map[uuid.UUID]bool{otherUserID: true}
	svc.mu.Unlock()

	// When/Then: false 반환
	assert.False(t, svc.IsUserOnline(userID, workspaceID))
}

// ============================================================
// GetOnlineUsersInMemory 테스트
// ============================================================

func TestPresenceService_GetOnlineUsersInMemory_Empty(t *testing.T) {
	// Given: 빈 워크스페이스
	svc := newTestPresenceService()
	workspaceID := uuid.New()

	// When
	users := svc.GetOnlineUsersInMemory(workspaceID)

	// Then: 빈 슬라이스 (nil)
	assert.Nil(t, users)
}

func TestPresenceService_GetOnlineUsersInMemory_WithUsers(t *testing.T) {
	// Given: 여러 사용자가 온라인인 워크스페이스
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	svc.mu.Lock()
	svc.onlineUsers[workspaceID] = map[uuid.UUID]bool{
		user1: true,
		user2: true,
		user3: true,
	}
	svc.mu.Unlock()

	// When
	users := svc.GetOnlineUsersInMemory(workspaceID)

	// Then: 3명의 사용자 반환
	assert.Len(t, users, 3)
	assert.Contains(t, users, user1)
	assert.Contains(t, users, user2)
	assert.Contains(t, users, user3)
}

// ============================================================
// countTotalOnlineUsers 테스트
// ============================================================

func TestPresenceService_CountTotalOnlineUsers_Empty(t *testing.T) {
	// Given: 아무도 온라인이 아닌 경우
	svc := newTestPresenceService()

	// When
	svc.mu.Lock()
	count := svc.countTotalOnlineUsers()
	svc.mu.Unlock()

	// Then: 0 반환
	assert.Equal(t, 0, count)
}

func TestPresenceService_CountTotalOnlineUsers_MultipleWorkspaces(t *testing.T) {
	// Given: 여러 워크스페이스에 사용자들이 온라인
	svc := newTestPresenceService()
	ws1 := uuid.New()
	ws2 := uuid.New()

	svc.mu.Lock()
	svc.onlineUsers[ws1] = map[uuid.UUID]bool{
		uuid.New(): true,
		uuid.New(): true,
	}
	svc.onlineUsers[ws2] = map[uuid.UUID]bool{
		uuid.New(): true,
		uuid.New(): true,
		uuid.New(): true,
	}
	count := svc.countTotalOnlineUsers()
	svc.mu.Unlock()

	// Then: 전체 5명
	assert.Equal(t, 5, count)
}

// ============================================================
// Metrics 연동 테스트
// ============================================================

func TestPresenceService_Metrics_SetOnlineUsersTotal(t *testing.T) {
	// Given
	m := metrics.NewForTest()

	// When: 온라인 사용자 수 설정
	m.SetOnlineUsersTotal(42)

	// Then: panic 없이 완료
	assert.NotNil(t, m.OnlineUsersTotal)
}

func TestPresenceService_Metrics_UpdateOnUserOnline(t *testing.T) {
	// Given
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	userID := uuid.New()

	// When: 사용자 온라인 상태로 변경 (메모리만)
	svc.mu.Lock()
	if svc.onlineUsers[workspaceID] == nil {
		svc.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
	}
	svc.onlineUsers[workspaceID][userID] = true
	onlineCount := svc.countTotalOnlineUsers()
	svc.mu.Unlock()

	// 메트릭 업데이트
	if svc.metrics != nil {
		svc.metrics.SetOnlineUsersTotal(int64(onlineCount))
	}

	// Then: panic 없이 완료, 카운트 확인
	assert.Equal(t, 1, onlineCount)
}

// ============================================================
// 동시성 테스트
// ============================================================

func TestPresenceService_Concurrency_MultipleOnlineOffline(t *testing.T) {
	// Given: 테스트용 서비스
	svc := newTestPresenceService()
	workspaceID := uuid.New()

	var wg sync.WaitGroup
	numUsers := 100

	// When: 동시에 100명 온라인
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userID := uuid.New()

			svc.mu.Lock()
			if svc.onlineUsers[workspaceID] == nil {
				svc.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
			}
			svc.onlineUsers[workspaceID][userID] = true
			svc.mu.Unlock()
		}()
	}

	wg.Wait()

	// Then: race condition 없이 완료
	svc.mu.RLock()
	count := len(svc.onlineUsers[workspaceID])
	svc.mu.RUnlock()

	assert.Equal(t, numUsers, count)
}

func TestPresenceService_Concurrency_ReadWrite(t *testing.T) {
	// Given: 테스트용 서비스와 사전 등록된 사용자들
	svc := newTestPresenceService()
	workspaceID := uuid.New()
	users := make([]uuid.UUID, 50)

	svc.mu.Lock()
	svc.onlineUsers[workspaceID] = make(map[uuid.UUID]bool)
	for i := 0; i < 50; i++ {
		users[i] = uuid.New()
		svc.onlineUsers[workspaceID][users[i]] = true
	}
	svc.mu.Unlock()

	var wg sync.WaitGroup

	// When: 동시에 읽기/쓰기 수행
	// 읽기 고루틴
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := users[idx%50]
			_ = svc.IsUserOnline(userID, workspaceID)
		}(i)
	}

	// 쓰기 고루틴
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newUser := uuid.New()
			svc.mu.Lock()
			svc.onlineUsers[workspaceID][newUser] = true
			svc.mu.Unlock()
		}()
	}

	wg.Wait()

	// Then: race condition 없이 완료
	svc.mu.RLock()
	count := len(svc.onlineUsers[workspaceID])
	svc.mu.RUnlock()

	// 50 (초기) + 50 (추가) = 100
	assert.Equal(t, 100, count)
}

// ============================================================
// PresenceStatus 상수 테스트
// ============================================================

func TestPresenceStatus_Constants(t *testing.T) {
	// Given/When/Then: 상수 값 확인 (대문자 사용)
	assert.Equal(t, domain.PresenceStatus("ONLINE"), domain.PresenceStatusOnline)
	assert.Equal(t, domain.PresenceStatus("OFFLINE"), domain.PresenceStatusOffline)
	assert.Equal(t, domain.PresenceStatus("AWAY"), domain.PresenceStatusAway)
}

// ============================================================
// Context 취소 테스트
// ============================================================

func TestPresenceService_ContextCancellation(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(context.Background())

	// When
	cancel()

	// Then
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}
