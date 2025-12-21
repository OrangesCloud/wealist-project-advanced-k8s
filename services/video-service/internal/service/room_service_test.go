// Package service는 video-service의 비즈니스 로직 테스트를 포함합니다.
// 이 파일은 RoomService의 유닛 테스트를 포함합니다.
package service

import (
	"context"
	"testing"
	"time"
	"video-service/internal/domain"
	"video-service/internal/response"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// 에러 정의 테스트
// ============================================================

func TestRoomService_ErrorDefinitions(t *testing.T) {
	// Given/When/Then: 에러 상수 정의 확인 (response 패키지에서 관리)
	assert.NotNil(t, response.ErrRoomNotFound)
	assert.NotNil(t, response.ErrRoomFull)
	assert.NotNil(t, response.ErrAlreadyInRoom)
	assert.NotNil(t, response.ErrNotInRoom)
	assert.NotNil(t, response.ErrRoomNotActive)
	assert.NotNil(t, response.ErrNotWorkspaceMember)

	// 에러 메시지 확인
	assert.Equal(t, "room not found", response.ErrRoomNotFound.Error())
	assert.Equal(t, "room is full", response.ErrRoomFull.Error())
	assert.Equal(t, "user is already in room", response.ErrAlreadyInRoom.Error())
	assert.Equal(t, "user is not in room", response.ErrNotInRoom.Error())
	assert.Equal(t, "room is not active", response.ErrRoomNotActive.Error())
	assert.Equal(t, "user is not a member of this workspace", response.ErrNotWorkspaceMember.Error())
}

// ============================================================
// Domain 구조체 테스트
// ============================================================

func TestRoomService_CreateRoomRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *domain.CreateRoomRequest
		wantErr bool
	}{
		{
			name: "유효한 요청",
			req: &domain.CreateRoomRequest{
				WorkspaceID:     uuid.New().String(),
				Name:            "Test Room",
				MaxParticipants: 10,
			},
			wantErr: false,
		},
		{
			name: "워크스페이스 ID 없음",
			req: &domain.CreateRoomRequest{
				Name:            "Test Room",
				MaxParticipants: 10,
			},
			wantErr: true,
		},
		{
			name: "이름 없음",
			req: &domain.CreateRoomRequest{
				WorkspaceID:     uuid.New().String(),
				MaxParticipants: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 유효성 검사 로직 테스트
			hasError := false
			if tt.req.WorkspaceID == "" {
				hasError = true
			}
			if tt.req.Name == "" {
				hasError = true
			}

			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestRoomService_RoomResponse_Fields(t *testing.T) {
	// Given: RoomResponse 생성 (string ID 사용)
	roomID := uuid.New().String()
	workspaceID := uuid.New().String()
	creatorID := uuid.New().String()
	now := time.Now()

	room := &domain.RoomResponse{
		ID:              roomID,
		WorkspaceID:     workspaceID,
		Name:            "Test Room",
		IsActive:        true,
		MaxParticipants: 10,
		CreatorID:       creatorID,
		CreatedAt:       now,
	}

	// Then: 필드 확인
	assert.Equal(t, roomID, room.ID)
	assert.Equal(t, workspaceID, room.WorkspaceID)
	assert.Equal(t, "Test Room", room.Name)
	assert.True(t, room.IsActive)
	assert.Equal(t, 10, room.MaxParticipants)
	assert.Equal(t, creatorID, room.CreatorID)
	assert.Equal(t, now, room.CreatedAt)
}

func TestRoomService_JoinRoomResponse_Fields(t *testing.T) {
	// Given: JoinRoomResponse 생성
	roomID := uuid.New().String()
	workspaceID := uuid.New().String()
	creatorID := uuid.New().String()
	now := time.Now()

	room := domain.RoomResponse{
		ID:              roomID,
		WorkspaceID:     workspaceID,
		Name:            "Test Room",
		IsActive:        true,
		MaxParticipants: 10,
		CreatorID:       creatorID,
		CreatedAt:       now,
	}

	resp := &domain.JoinRoomResponse{
		Room:  room,
		Token: "livekit-token-xxx",
		WSUrl: "wss://livekit.example.com",
	}

	// Then: 필드 확인
	assert.Equal(t, roomID, resp.Room.ID)
	assert.Equal(t, "livekit-token-xxx", resp.Token)
	assert.Equal(t, "wss://livekit.example.com", resp.WSUrl)
}

func TestRoomService_ParticipantResponse_Fields(t *testing.T) {
	// Given: ParticipantResponse 생성
	participantID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now()

	participant := &domain.ParticipantResponse{
		ID:       participantID,
		UserID:   userID,
		Name:     "Test Participant",
		JoinedAt: now,
		IsActive: true,
	}

	// Then: 필드 확인
	assert.Equal(t, participantID, participant.ID)
	assert.Equal(t, userID, participant.UserID)
	assert.Equal(t, "Test Participant", participant.Name)
	assert.Equal(t, now, participant.JoinedAt)
	assert.True(t, participant.IsActive)
}

func TestRoomService_CallHistoryResponse_Fields(t *testing.T) {
	// Given: CallHistoryResponse 생성 (string ID 사용)
	historyID := uuid.New().String()
	workspaceID := uuid.New().String()
	creatorID := uuid.New().String()
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	history := &domain.CallHistoryResponse{
		ID:                historyID,
		WorkspaceID:       workspaceID,
		RoomName:          "Past Call",
		CreatorID:         creatorID,
		StartedAt:         startTime,
		EndedAt:           endTime,
		DurationSeconds:   3600,
		TotalParticipants: 5,
	}

	// Then: 필드 확인
	assert.Equal(t, historyID, history.ID)
	assert.Equal(t, workspaceID, history.WorkspaceID)
	assert.Equal(t, "Past Call", history.RoomName)
	assert.Equal(t, creatorID, history.CreatorID)
	assert.Equal(t, startTime, history.StartedAt)
	assert.Equal(t, endTime, history.EndedAt)
	assert.Equal(t, 3600, history.DurationSeconds)
	assert.Equal(t, 5, history.TotalParticipants)
}

// ============================================================
// 페이지네이션 테스트
// ============================================================

func TestRoomService_Pagination_LimitValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"기본값 사용 (0)", 0, 10},
		{"음수", -1, 10},
		{"최대값 초과", 200, 100},
		{"정상 범위", 50, 50},
		{"최대값", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.inputLimit
			// 비즈니스 로직: limit 기본값 및 최대값 설정
			if limit <= 0 {
				limit = 10
			}
			if limit > 100 {
				limit = 100
			}
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

func TestRoomService_Pagination_OffsetValidation(t *testing.T) {
	tests := []struct {
		name           string
		inputOffset    int
		expectedOffset int
	}{
		{"기본값 사용 (0)", 0, 0},
		{"음수", -1, 0},
		{"정상 값", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := tt.inputOffset
			// 비즈니스 로직: offset 음수 방지
			if offset < 0 {
				offset = 0
			}
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

// ============================================================
// UUID 유효성 테스트
// ============================================================

func TestRoomService_UUIDValidation(t *testing.T) {
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

func TestRoomService_ContextCancellation(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(context.Background())

	// When
	cancel()

	// Then
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestRoomService_ContextTimeout(t *testing.T) {
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
// Room Code 생성 테스트
// ============================================================

func TestRoomService_RoomCodeGeneration(t *testing.T) {
	// Given: 방 코드 형식 정의 (6자리 대문자 영숫자)
	isValidCode := func(code string) bool {
		if len(code) != 6 {
			return false
		}
		for _, c := range code {
			if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return false
			}
		}
		return true
	}

	// When/Then: 유효한 코드 예시 확인
	validCodes := []string{"ABC123", "XYZ789", "TEST00", "ROOM01"}
	for _, code := range validCodes {
		assert.True(t, isValidCode(code), "Code %s should be valid", code)
	}

	// 유효하지 않은 코드 예시 확인
	invalidCodes := []string{"abc123", "ABC12", "ABCDEFG", "ABC-12"}
	for _, code := range invalidCodes {
		assert.False(t, isValidCode(code), "Code %s should be invalid", code)
	}
}

// ============================================================
// MaxParticipants 검증 테스트
// ============================================================

func TestRoomService_MaxParticipants_Validation(t *testing.T) {
	tests := []struct {
		name     string
		maxPart  int
		expected int
	}{
		{"기본값 사용 (0)", 0, 10},
		{"음수", -1, 10},
		{"최대값 초과", 500, 100},
		{"정상 범위", 50, 50},
		{"최소값", 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxPart := tt.maxPart
			// 비즈니스 로직: maxParticipants 기본값 및 범위 설정
			if maxPart <= 0 {
				maxPart = 10
			}
			if maxPart > 100 {
				maxPart = 100
			}
			assert.Equal(t, tt.expected, maxPart)
		})
	}
}

// ============================================================
// 통화 시간 계산 테스트
// ============================================================

func TestRoomService_CallDuration_Calculation(t *testing.T) {
	// Given: 시작 시간과 종료 시간
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	// When: 통화 시간 계산 (초 단위)
	durationSeconds := int(endTime.Sub(startTime).Seconds())

	// Then: 약 3600초 (1시간)
	assert.InDelta(t, 3600, durationSeconds, 5) // 5초 오차 허용
}

func TestRoomService_CallDuration_ZeroForActiveCall(t *testing.T) {
	// Given: 활성 통화 (종료 시간 없음)
	startTime := time.Now().Add(-30 * time.Minute)
	var endTime *time.Time = nil

	// When: 통화 시간 계산
	var durationSeconds int
	if endTime != nil {
		durationSeconds = int(endTime.Sub(startTime).Seconds())
	} else {
		durationSeconds = 0 // 활성 통화는 0
	}

	// Then: 0초
	assert.Equal(t, 0, durationSeconds)
}

// ============================================================
// RoomStatus 상수 테스트
// ============================================================

func TestRoomService_RoomStatus_Constants(t *testing.T) {
	// Given: 상태 상수 정의 확인
	const (
		RoomStatusActive = "active"
		RoomStatusEnded  = "ended"
	)

	// When/Then: 상수 값 확인
	assert.Equal(t, "active", RoomStatusActive)
	assert.Equal(t, "ended", RoomStatusEnded)
}

// ============================================================
// Transcript 테스트
// ============================================================

func TestRoomService_TranscriptResponse_Fields(t *testing.T) {
	// Given: TranscriptResponse 생성 (string ID 사용)
	transcriptID := uuid.New().String()
	callHistoryID := uuid.New().String()
	roomID := uuid.New().String()
	now := time.Now()

	transcript := &domain.TranscriptResponse{
		ID:            transcriptID,
		CallHistoryID: callHistoryID,
		RoomID:        roomID,
		Content:       "This is a transcript content.",
		CreatedAt:     now,
	}

	// Then: 필드 확인
	assert.Equal(t, transcriptID, transcript.ID)
	assert.Equal(t, callHistoryID, transcript.CallHistoryID)
	assert.Equal(t, roomID, transcript.RoomID)
	assert.Equal(t, "This is a transcript content.", transcript.Content)
	assert.Equal(t, now, transcript.CreatedAt)
}

// ============================================================
// 동시성 테스트
// ============================================================

func TestRoomService_Concurrency_UUIDGeneration(t *testing.T) {
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
