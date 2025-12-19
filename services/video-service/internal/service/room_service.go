// Package service는 video-service의 비즈니스 로직을 구현합니다.
//
// 이 패키지는 비디오 통화 룸, 참가자, 통화 기록 및 트랜스크립트 관리를 위한
// RoomService 인터페이스와 구현을 제공합니다.
// LiveKit과 통합하여 WebRTC 기능을 제공하고,
// user-service를 통해 워크스페이스 멤버십을 검증합니다.
package service

import (
	"context"
	"fmt"
	"time"
	"video-service/internal/client"
	"video-service/internal/config"
	"video-service/internal/domain"
	"video-service/internal/metrics"
	"video-service/internal/repository"
	"video-service/internal/response"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RoomService는 비디오 룸 비즈니스 로직을 위한 인터페이스입니다.
// 룸 생명주기 관리, 참가자 추적, 통화 기록 조회 및 트랜스크립트 저장 메서드를 제공합니다.
type RoomService interface {
	// CreateRoom은 지정된 워크스페이스에 새 비디오 통화 룸을 생성합니다.
	// 워크스페이스 멤버십을 검증하고 LiveKit 룸을 초기화합니다.
	CreateRoom(ctx context.Context, req *domain.CreateRoomRequest, creatorID uuid.UUID, token string) (*domain.RoomResponse, error)

	// GetRoom은 ID로 단일 룸을 조회합니다.
	GetRoom(ctx context.Context, roomID uuid.UUID) (*domain.RoomResponse, error)

	// GetWorkspaceRooms는 워크스페이스의 모든 룸을 조회합니다.
	// activeOnly를 true로 설정하면 활성 룸만 필터링합니다.
	GetWorkspaceRooms(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID, token string, activeOnly bool) ([]domain.RoomResponse, error)

	// JoinRoom은 사용자를 룸에 추가하고 LiveKit 액세스 토큰을 반환합니다.
	// 룸이 가득 찬 경우 ErrRoomFull을 반환합니다.
	JoinRoom(ctx context.Context, roomID, userID uuid.UUID, userName string, token string) (*domain.JoinRoomResponse, error)

	// LeaveRoom은 사용자를 룸에서 제거합니다.
	// 생성자가 나가면 룸이 자동으로 종료됩니다.
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error

	// EndRoom은 룸을 닫고 통화 기록을 생성합니다.
	// 룸 생성자만 룸을 종료할 수 있습니다.
	EndRoom(ctx context.Context, roomID, userID uuid.UUID) error

	// GetParticipants는 룸의 모든 활성 참가자를 반환합니다.
	GetParticipants(ctx context.Context, roomID uuid.UUID) ([]domain.ParticipantResponse, error)

	// GetWorkspaceCallHistory는 워크스페이스의 페이지네이션된 통화 기록을 반환합니다.
	GetWorkspaceCallHistory(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID, token string, limit, offset int) ([]domain.CallHistoryResponse, int64, error)

	// GetUserCallHistory는 특정 사용자의 페이지네이션된 통화 기록을 반환합니다.
	GetUserCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.CallHistoryResponse, int64, error)

	// GetCallHistoryByID는 ID로 단일 통화 기록을 조회합니다.
	GetCallHistoryByID(ctx context.Context, historyID uuid.UUID) (*domain.CallHistoryResponse, error)

	// SaveTranscript는 룸의 트랜스크립트 내용을 저장하거나 업데이트합니다.
	SaveTranscript(ctx context.Context, roomID uuid.UUID, content string) (*domain.TranscriptResponse, error)

	// GetTranscriptByCallHistoryID는 완료된 통화의 트랜스크립트를 조회합니다.
	GetTranscriptByCallHistoryID(ctx context.Context, callHistoryID uuid.UUID) (*domain.TranscriptResponse, error)
}

// roomService는 RoomService 인터페이스를 구현합니다.
// repository, LiveKit, user-service 클라이언트 간의 조정을 담당합니다.
// 메트릭과 로깅을 통해 모니터링을 지원합니다.
type roomService struct {
	roomRepo    repository.RoomRepository
	userClient  client.UserClient
	lkClient    *lksdk.RoomServiceClient
	lkConfig    config.LiveKitConfig
	redisClient *redis.Client
	logger      *zap.Logger
	metrics     *metrics.Metrics // 메트릭 수집을 위한 필드
}

// NewRoomService는 주어진 의존성으로 새 RoomService를 생성합니다.
// LiveKit 설정이 제공되면 LiveKit 클라이언트를 초기화합니다.
// userClient가 nil이면 워크스페이스 검증을 건너뜁니다.
// metrics 파라미터가 nil인 경우에도 안전하게 동작합니다.
func NewRoomService(
	roomRepo repository.RoomRepository,
	userClient client.UserClient,
	lkConfig config.LiveKitConfig,
	redisClient *redis.Client,
	logger *zap.Logger,
	m *metrics.Metrics,
) RoomService {
	var lkClient *lksdk.RoomServiceClient
	if lkConfig.Host != "" && lkConfig.APIKey != "" && lkConfig.APISecret != "" {
		lkClient = lksdk.NewRoomServiceClient(lkConfig.Host, lkConfig.APIKey, lkConfig.APISecret)
	}

	return &roomService{
		roomRepo:    roomRepo,
		userClient:  userClient,
		lkClient:    lkClient,
		lkConfig:    lkConfig,
		redisClient: redisClient,
		logger:      logger,
		metrics:     m,
	}
}

// CreateRoom은 새 비디오 통화 룸을 생성합니다.
// 워크스페이스 멤버십을 검증하고 DB와 LiveKit에 룸을 생성합니다.
func (s *roomService) CreateRoom(ctx context.Context, req *domain.CreateRoomRequest, creatorID uuid.UUID, token string) (*domain.RoomResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	// 워크스페이스 멤버십 검증
	if s.userClient != nil {
		isMember, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, creatorID, token)
		if err != nil {
			s.logger.Error("워크스페이스 멤버십 검증 실패", zap.Error(err))
			return nil, fmt.Errorf("failed to validate workspace membership: %w", err)
		}
		if !isMember {
			return nil, response.ErrNotWorkspaceMember
		}
	}

	maxParticipants := req.MaxParticipants
	if maxParticipants <= 0 {
		maxParticipants = 10
	}

	room := &domain.Room{
		Name:            req.Name,
		WorkspaceID:     workspaceID,
		CreatorID:       creatorID,
		MaxParticipants: maxParticipants,
		IsActive:        true,
	}

	if err := s.roomRepo.Create(room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// LiveKit에 룸 생성
	if s.lkClient != nil {
		_, err := s.lkClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
			Name:            room.ID.String(),
			EmptyTimeout:    300, // 5분
			MaxParticipants: uint32(maxParticipants),
		})
		if err != nil {
			s.logger.Warn("LiveKit 룸 생성 실패", zap.Error(err))
		}
	}

	// 메트릭 기록: 룸 생성 성공
	if s.metrics != nil {
		s.metrics.RecordRoomCreated()
	}

	s.logger.Info("룸 생성 완료",
		zap.String("room_id", room.ID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("creator_id", creatorID.String()),
	)

	response := room.ToResponse()
	return &response, nil
}

// GetRoom은 ID로 룸을 조회합니다.
func (s *roomService) GetRoom(ctx context.Context, roomID uuid.UUID) (*domain.RoomResponse, error) {
	room, err := s.roomRepo.GetByID(roomID)
	if err != nil {
		return nil, response.ErrRoomNotFound
	}

	response := room.ToResponse()
	return &response, nil
}

// GetWorkspaceRooms는 워크스페이스의 룸 목록을 조회합니다.
func (s *roomService) GetWorkspaceRooms(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID, token string, activeOnly bool) ([]domain.RoomResponse, error) {
	// 워크스페이스 멤버십 검증
	if s.userClient != nil {
		isMember, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
		if err != nil {
			s.logger.Error("워크스페이스 멤버십 검증 실패", zap.Error(err))
			return nil, fmt.Errorf("failed to validate workspace membership: %w", err)
		}
		if !isMember {
			return nil, response.ErrNotWorkspaceMember
		}
	}

	var rooms []domain.Room
	var err error

	if activeOnly {
		rooms, err = s.roomRepo.GetActiveByWorkspaceID(workspaceID)
	} else {
		rooms, err = s.roomRepo.GetByWorkspaceID(workspaceID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get rooms: %w", err)
	}

	responses := make([]domain.RoomResponse, len(rooms))
	for i, room := range rooms {
		responses[i] = room.ToResponse()
	}

	return responses, nil
}

// EndRoom은 룸을 종료하고 통화 기록을 생성합니다.
// 룸 생성자만 룸을 종료할 수 있습니다.
// 종료 성공 시 메트릭과 통화 시간을 기록합니다.
func (s *roomService) EndRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	room, err := s.roomRepo.GetByID(roomID)
	if err != nil {
		return response.ErrRoomNotFound
	}

	// 생성자만 룸을 종료할 수 있음
	if room.CreatorID != userID {
		return response.ErrNotRoomCreator
	}

	// 통화 시간 계산 (메트릭 기록용)
	callDuration := time.Since(room.CreatedAt)

	room.IsActive = false
	if err := s.roomRepo.Update(room); err != nil {
		return fmt.Errorf("failed to end room: %w", err)
	}

	// 통화 기록 생성
	s.createCallHistory(room)

	// LiveKit에서 룸 삭제
	if s.lkClient != nil {
		_, _ = s.lkClient.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
			Room: room.ID.String(),
		})
	}

	// 메트릭 기록: 룸 종료 및 통화 시간
	if s.metrics != nil {
		s.metrics.RecordRoomEnded()
		s.metrics.RecordCallDuration(callDuration)
	}

	s.logger.Info("룸 종료 완료",
		zap.String("room_id", roomID.String()),
		zap.String("ended_by", userID.String()),
		zap.Duration("call_duration", callDuration),
	)

	return nil
}

// generateLiveKitToken은 LiveKit 액세스 토큰을 생성합니다.
func (s *roomService) generateLiveKitToken(roomName, userID, userName string) (string, error) {
	if s.lkConfig.APIKey == "" || s.lkConfig.APISecret == "" {
		return "", response.ErrLiveKitNotConfigured
	}

	at := auth.NewAccessToken(s.lkConfig.APIKey, s.lkConfig.APISecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.AddGrant(grant).
		SetIdentity(userID).
		SetName(userName).
		SetValidFor(24 * time.Hour)

	return at.ToJWT()
}
