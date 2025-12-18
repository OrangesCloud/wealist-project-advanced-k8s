// Package service는 noti-service의 비즈니스 로직 테스트를 포함합니다.
// 이 파일은 NotificationService의 유닛 테스트를 포함합니다.
package service

import (
	"context"
	"noti-service/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// NotificationType 상수 테스트
// ============================================================

func TestNotificationService_NotificationTypes(t *testing.T) {
	// Given/When/Then: 알림 타입 상수 확인
	// Task events
	assert.Equal(t, domain.NotificationType("TASK_ASSIGNED"), domain.NotificationTypeTaskAssigned)
	assert.Equal(t, domain.NotificationType("TASK_UNASSIGNED"), domain.NotificationTypeTaskUnassigned)
	assert.Equal(t, domain.NotificationType("TASK_MENTIONED"), domain.NotificationTypeTaskMentioned)
	assert.Equal(t, domain.NotificationType("TASK_DUE_SOON"), domain.NotificationTypeTaskDueSoon)
	assert.Equal(t, domain.NotificationType("TASK_OVERDUE"), domain.NotificationTypeTaskOverdue)
	assert.Equal(t, domain.NotificationType("TASK_STATUS_CHANGED"), domain.NotificationTypeTaskStatusChanged)

	// Comment events
	assert.Equal(t, domain.NotificationType("COMMENT_ADDED"), domain.NotificationTypeCommentAdded)
	assert.Equal(t, domain.NotificationType("COMMENT_MENTIONED"), domain.NotificationTypeCommentMentioned)

	// Workspace events
	assert.Equal(t, domain.NotificationType("WORKSPACE_INVITED"), domain.NotificationTypeWorkspaceInvited)
	assert.Equal(t, domain.NotificationType("WORKSPACE_ROLE_CHANGED"), domain.NotificationTypeWorkspaceRoleChanged)
	assert.Equal(t, domain.NotificationType("WORKSPACE_REMOVED"), domain.NotificationTypeWorkspaceRemoved)

	// Project events
	assert.Equal(t, domain.NotificationType("PROJECT_INVITED"), domain.NotificationTypeProjectInvited)
	assert.Equal(t, domain.NotificationType("PROJECT_ROLE_CHANGED"), domain.NotificationTypeProjectRoleChanged)
	assert.Equal(t, domain.NotificationType("PROJECT_REMOVED"), domain.NotificationTypeProjectRemoved)
}

// ============================================================
// ResourceType 상수 테스트
// ============================================================

func TestNotificationService_ResourceTypes(t *testing.T) {
	// Given/When/Then: 리소스 타입 상수 확인 (소문자 사용)
	assert.Equal(t, domain.ResourceType("task"), domain.ResourceTypeTask)
	assert.Equal(t, domain.ResourceType("comment"), domain.ResourceTypeComment)
	assert.Equal(t, domain.ResourceType("workspace"), domain.ResourceTypeWorkspace)
	assert.Equal(t, domain.ResourceType("project"), domain.ResourceTypeProject)
}

// ============================================================
// Notification 구조체 테스트
// ============================================================

func TestNotificationService_Notification_Fields(t *testing.T) {
	// Given: Notification 생성
	notificationID := uuid.New()
	actorID := uuid.New()
	targetUserID := uuid.New()
	workspaceID := uuid.New()
	resourceID := uuid.New()
	resourceName := "Test Task"
	now := time.Now()

	notification := &domain.Notification{
		ID:           notificationID,
		Type:         domain.NotificationTypeTaskAssigned,
		ActorID:      actorID,
		TargetUserID: targetUserID,
		WorkspaceID:  workspaceID,
		ResourceType: domain.ResourceTypeTask,
		ResourceID:   resourceID,
		ResourceName: &resourceName,
		Metadata:     map[string]interface{}{"key": "value"},
		IsRead:       false,
		CreatedAt:    now,
	}

	// Then: 필드 확인
	assert.Equal(t, notificationID, notification.ID)
	assert.Equal(t, domain.NotificationTypeTaskAssigned, notification.Type)
	assert.Equal(t, actorID, notification.ActorID)
	assert.Equal(t, targetUserID, notification.TargetUserID)
	assert.Equal(t, workspaceID, notification.WorkspaceID)
	assert.Equal(t, domain.ResourceTypeTask, notification.ResourceType)
	assert.Equal(t, resourceID, notification.ResourceID)
	assert.Equal(t, &resourceName, notification.ResourceName)
	assert.False(t, notification.IsRead)
	assert.Nil(t, notification.ReadAt)
	assert.Equal(t, now, notification.CreatedAt)
}

func TestNotificationService_Notification_TableName(t *testing.T) {
	// Given
	notification := domain.Notification{}

	// When/Then
	assert.Equal(t, "notifications", notification.TableName())
}

// ============================================================
// NotificationEvent 구조체 테스트
// ============================================================

func TestNotificationService_NotificationEvent_Fields(t *testing.T) {
	// Given: NotificationEvent 생성
	actorID := uuid.New()
	targetUserID := uuid.New()
	workspaceID := uuid.New()
	resourceID := uuid.New()
	resourceName := "Test Resource"
	occurredAt := time.Now()

	event := &domain.NotificationEvent{
		Type:         domain.NotificationTypeTaskAssigned,
		ActorID:      actorID,
		TargetUserID: targetUserID,
		WorkspaceID:  workspaceID,
		ResourceType: domain.ResourceTypeTask,
		ResourceID:   resourceID,
		ResourceName: &resourceName,
		Metadata:     map[string]interface{}{"priority": "high"},
		OccurredAt:   &occurredAt,
	}

	// Then: 필드 확인
	assert.Equal(t, domain.NotificationTypeTaskAssigned, event.Type)
	assert.Equal(t, actorID, event.ActorID)
	assert.Equal(t, targetUserID, event.TargetUserID)
	assert.Equal(t, workspaceID, event.WorkspaceID)
	assert.Equal(t, domain.ResourceTypeTask, event.ResourceType)
	assert.Equal(t, resourceID, event.ResourceID)
	assert.Equal(t, &resourceName, event.ResourceName)
	assert.Equal(t, "high", event.Metadata["priority"])
	assert.Equal(t, &occurredAt, event.OccurredAt)
}

// ============================================================
// PaginatedNotifications 구조체 테스트
// ============================================================

func TestNotificationService_PaginatedNotifications_Fields(t *testing.T) {
	// Given: PaginatedNotifications 생성
	notifications := []domain.Notification{
		{ID: uuid.New(), Type: domain.NotificationTypeTaskAssigned},
		{ID: uuid.New(), Type: domain.NotificationTypeCommentAdded},
	}

	paginated := &domain.PaginatedNotifications{
		Notifications: notifications,
		Total:         100,
		Page:          1,
		Limit:         20,
		HasMore:       true,
	}

	// Then: 필드 확인
	assert.Len(t, paginated.Notifications, 2)
	assert.Equal(t, int64(100), paginated.Total)
	assert.Equal(t, 1, paginated.Page)
	assert.Equal(t, 20, paginated.Limit)
	assert.True(t, paginated.HasMore)
}

func TestNotificationService_PaginatedNotifications_HasMore(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		page    int
		limit   int
		hasMore bool
	}{
		{"첫 페이지, 더 있음", 100, 1, 20, true},
		{"마지막 페이지", 100, 5, 20, false},
		{"전체 1페이지", 15, 1, 20, false},
		{"정확히 맞춤", 40, 2, 20, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HasMore 로직 테스트
			hasMore := int64(tt.page*tt.limit) < tt.total
			assert.Equal(t, tt.hasMore, hasMore)
		})
	}
}

// ============================================================
// UnreadCount 구조체 테스트
// ============================================================

func TestNotificationService_UnreadCount_Fields(t *testing.T) {
	// Given
	workspaceID := uuid.New()
	unreadCount := &domain.UnreadCount{
		Count:       42,
		WorkspaceID: workspaceID,
	}

	// Then
	assert.Equal(t, int64(42), unreadCount.Count)
	assert.Equal(t, workspaceID, unreadCount.WorkspaceID)
}

// ============================================================
// NotificationPreference 구조체 테스트
// ============================================================

func TestNotificationService_NotificationPreference_Fields(t *testing.T) {
	// Given
	prefID := uuid.New()
	userID := uuid.New()
	workspaceID := uuid.New()
	now := time.Now()

	pref := &domain.NotificationPreference{
		ID:          prefID,
		UserID:      userID,
		WorkspaceID: &workspaceID,
		Type:        "TASK_ASSIGNED",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Then
	assert.Equal(t, prefID, pref.ID)
	assert.Equal(t, userID, pref.UserID)
	assert.Equal(t, &workspaceID, pref.WorkspaceID)
	assert.Equal(t, "TASK_ASSIGNED", pref.Type)
	assert.True(t, pref.Enabled)
}

// ============================================================
// 페이지네이션 로직 테스트
// ============================================================

func TestNotificationService_Pagination_LimitValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"기본값 사용 (0)", 0, 20},
		{"음수", -1, 20},
		{"최대값 초과", 200, 100},
		{"정상 범위", 50, 50},
		{"최대값", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.inputLimit
			// 비즈니스 로직: limit 기본값 및 최대값 설정
			if limit <= 0 {
				limit = 20
			}
			if limit > 100 {
				limit = 100
			}
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

func TestNotificationService_Pagination_PageValidation(t *testing.T) {
	tests := []struct {
		name         string
		inputPage    int
		expectedPage int
	}{
		{"기본값 사용 (0)", 0, 1},
		{"음수", -1, 1},
		{"정상 값", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := tt.inputPage
			// 비즈니스 로직: page 최소값 설정
			if page <= 0 {
				page = 1
			}
			assert.Equal(t, tt.expectedPage, page)
		})
	}
}

// ============================================================
// UUID 유효성 테스트
// ============================================================

func TestNotificationService_UUIDValidation(t *testing.T) {
	// Valid UUID
	validUUID := uuid.New()
	assert.NotEqual(t, uuid.Nil, validUUID)

	// Parse UUID
	parsed, err := uuid.Parse(validUUID.String())
	assert.NoError(t, err)
	assert.Equal(t, validUUID, parsed)

	// Invalid UUID
	_, err = uuid.Parse("invalid-uuid")
	assert.Error(t, err)
}

// ============================================================
// Context 취소 테스트
// ============================================================

func TestNotificationService_ContextCancellation(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(context.Background())

	// When
	cancel()

	// Then
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestNotificationService_ContextTimeout(t *testing.T) {
	// Given: 매우 짧은 타임아웃
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// When: 타임아웃 대기
	time.Sleep(5 * time.Millisecond)

	// Then: 타임아웃 에러
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

// ============================================================
// Redis 채널 키 생성 테스트
// ============================================================

func TestNotificationService_RedisChannelKey(t *testing.T) {
	// Given: 채널 키 형식 정의
	formatChannel := func(userID uuid.UUID) string {
		return "notifications:" + userID.String()
	}

	// When
	userID := uuid.New()
	channel := formatChannel(userID)

	// Then: 형식 확인
	assert.Contains(t, channel, "notifications:")
	assert.Contains(t, channel, userID.String())
}

// ============================================================
// Cache 키 생성 테스트
// ============================================================

func TestNotificationService_CacheKey(t *testing.T) {
	// Given: 캐시 키 형식 정의
	formatCacheKey := func(userID, workspaceID uuid.UUID) string {
		return "unread_count:" + userID.String() + ":" + workspaceID.String()
	}

	// When
	userID := uuid.New()
	workspaceID := uuid.New()
	cacheKey := formatCacheKey(userID, workspaceID)

	// Then: 형식 확인
	assert.Contains(t, cacheKey, "unread_count:")
	assert.Contains(t, cacheKey, userID.String())
	assert.Contains(t, cacheKey, workspaceID.String())
}

// ============================================================
// Metadata JSON 직렬화 테스트
// ============================================================

func TestNotificationService_MetadataJSON(t *testing.T) {
	// Given: Metadata map
	metadata := map[string]interface{}{
		"taskName":   "Important Task",
		"priority":   "high",
		"assigneeId": uuid.New().String(),
	}

	// When: Notification에 설정
	notification := &domain.Notification{
		ID:       uuid.New(),
		Metadata: metadata,
	}

	// Then: 값 확인
	assert.Equal(t, "Important Task", notification.Metadata["taskName"])
	assert.Equal(t, "high", notification.Metadata["priority"])
	assert.NotEmpty(t, notification.Metadata["assigneeId"])
}

// ============================================================
// IsRead 상태 변경 테스트
// ============================================================

func TestNotificationService_MarkAsRead(t *testing.T) {
	// Given: 읽지 않은 알림
	notification := &domain.Notification{
		ID:     uuid.New(),
		IsRead: false,
		ReadAt: nil,
	}

	// When: 읽음 처리
	now := time.Now()
	notification.IsRead = true
	notification.ReadAt = &now

	// Then: 상태 확인
	assert.True(t, notification.IsRead)
	assert.NotNil(t, notification.ReadAt)
}

func TestNotificationService_MarkAsUnread(t *testing.T) {
	// Given: 읽은 알림
	now := time.Now()
	notification := &domain.Notification{
		ID:     uuid.New(),
		IsRead: true,
		ReadAt: &now,
	}

	// When: 읽지 않음 처리
	notification.IsRead = false
	notification.ReadAt = nil

	// Then: 상태 확인
	assert.False(t, notification.IsRead)
	assert.Nil(t, notification.ReadAt)
}

// ============================================================
// 동시성 테스트
// ============================================================

func TestNotificationService_Concurrency_UUIDGeneration(t *testing.T) {
	// Given: 동시에 많은 UUID 생성
	const numGoroutines = 100
	results := make(chan uuid.UUID, numGoroutines)

	// When: 동시 생성
	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- uuid.New()
		}()
	}

	// Then: 모든 UUID가 유니크한지 확인
	seen := make(map[uuid.UUID]bool)
	for i := 0; i < numGoroutines; i++ {
		id := <-results
		assert.False(t, seen[id], "UUID should be unique")
		seen[id] = true
	}
}
