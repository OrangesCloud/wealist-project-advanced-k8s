// Package service는 video-service의 비즈니스 로직을 구현합니다.
//
// 이 파일은 roomService의 참가자 관리, 통화 기록, 트랜스크립트 관련 메서드를 포함합니다.
package service

import (
	"context"
	"fmt"
	"time"
	"video-service/internal/domain"
	"video-service/internal/response"

	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	"go.uber.org/zap"
)

// JoinRoom은 사용자를 룸에 추가하고 LiveKit 토큰을 반환합니다.
// 이미 참가한 사용자는 재참가로 처리됩니다.
func (s *roomService) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, userName string, token string) (*domain.JoinRoomResponse, error) {
	room, err := s.roomRepo.GetByID(roomID)
	if err != nil {
		return nil, response.ErrRoomNotFound
	}

	if !room.IsActive {
		return nil, response.ErrRoomNotActive
	}

	// 워크스페이스 멤버십 검증
	if s.userClient != nil {
		isMember, err := s.userClient.ValidateWorkspaceMember(ctx, room.WorkspaceID, userID, token)
		if err != nil {
			s.logger.Error("워크스페이스 멤버십 검증 실패", zap.Error(err))
			return nil, fmt.Errorf("failed to validate workspace membership: %w", err)
		}
		if !isMember {
			return nil, response.ErrNotWorkspaceMember
		}
	}

	// 이미 참가한 사용자인지 확인 (비활성 상태 포함)
	existingParticipant, _ := s.roomRepo.GetParticipant(roomID, userID)
	if existingParticipant != nil {
		// 사용자가 재참가하는 경우
		s.logger.Info("사용자 룸 재참가",
			zap.String("room_id", roomID.String()),
			zap.String("user_id", userID.String()),
		)
		existingParticipant.IsActive = true
		existingParticipant.LeftAt = nil    // 나간 시간 초기화
		existingParticipant.Name = userName // 이름 업데이트

		if err := s.roomRepo.UpdateParticipant(existingParticipant); err != nil {
			return nil, fmt.Errorf("failed to update participant: %w", err)
		}
	} else {
		// 새 참가자 - 룸 용량 확인
		count, err := s.roomRepo.CountActiveParticipants(roomID)
		if err != nil {
			return nil, fmt.Errorf("failed to count participants: %w", err)
		}
		if count >= int64(room.MaxParticipants) {
			return nil, response.ErrRoomFull
		}

		// 참가자 추가
		participant := &domain.RoomParticipant{
			RoomID:   roomID,
			UserID:   userID,
			IsActive: true,
			Name:     userName,
		}
		if err := s.roomRepo.AddParticipant(participant); err != nil {
			return nil, fmt.Errorf("failed to add participant: %w", err)
		}

		// 메트릭 기록: 참가자 추가 성공
		if s.metrics != nil {
			s.metrics.RecordParticipantJoined()
		}

		s.logger.Info("참가자 추가 완료",
			zap.String("room_id", roomID.String()),
			zap.String("user_id", userID.String()),
			zap.String("user_name", userName),
		)
	}

	// LiveKit 토큰 생성
	lkToken, err := s.generateLiveKitToken(room.ID.String(), userID.String(), userName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 룸 데이터 갱신
	room, _ = s.roomRepo.GetByID(roomID)

	return &domain.JoinRoomResponse{
		Room:  room.ToResponse(),
		Token: lkToken,
		WSUrl: s.lkConfig.WSUrl,
	}, nil
}

// LeaveRoom은 사용자를 룸에서 제거합니다.
// 생성자가 나가면 룸이 자동으로 종료됩니다.
func (s *roomService) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	// 룸 존재 여부 확인 및 생성자 확인
	room, err := s.roomRepo.GetByID(roomID)
	if err != nil {
		return response.ErrRoomNotFound
	}

	// 생성자가 나가면 룸 종료
	if room.CreatorID == userID {
		s.logger.Info("생성자 퇴장으로 룸 종료",
			zap.String("room_id", roomID.String()),
		)
		return s.EndRoom(ctx, roomID, userID)
	}

	// 일반 참가자 확인
	_, err = s.roomRepo.GetParticipant(roomID, userID)
	if err != nil {
		return response.ErrNotInRoom
	}

	if err := s.roomRepo.RemoveParticipant(roomID, userID); err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	// 메트릭 기록: 참가자 퇴장 성공
	if s.metrics != nil {
		s.metrics.RecordParticipantLeft()
	}

	s.logger.Info("참가자 퇴장 완료",
		zap.String("room_id", roomID.String()),
		zap.String("user_id", userID.String()),
	)

	// 룸이 비어있으면 종료 (보조 체크)
	count, _ := s.roomRepo.CountActiveParticipants(roomID)
	if count == 0 {
		room.IsActive = false
		_ = s.roomRepo.Update(room)

		// 통화 기록 생성
		s.createCallHistory(room)

		// LiveKit에서 룸 삭제
		if s.lkClient != nil {
			_, _ = s.lkClient.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
				Room: room.ID.String(),
			})
		}

		s.logger.Info("빈 룸 자동 종료",
			zap.String("room_id", roomID.String()),
		)
	}

	return nil
}

// GetParticipants는 룸의 활성 참가자 목록을 반환합니다.
func (s *roomService) GetParticipants(ctx context.Context, roomID uuid.UUID) ([]domain.ParticipantResponse, error) {
	participants, err := s.roomRepo.GetActiveParticipants(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	responses := make([]domain.ParticipantResponse, len(participants))
	for i, p := range participants {
		responses[i] = domain.ParticipantResponse{
			ID:       p.ID.String(),
			UserID:   p.UserID.String(),
			Name:     p.Name,
			JoinedAt: p.JoinedAt,
			LeftAt:   p.LeftAt,
			IsActive: p.IsActive,
		}
	}

	return responses, nil
}

// createCallHistory는 룸 종료 시 통화 기록을 생성합니다.
func (s *roomService) createCallHistory(room *domain.Room) {
	// 모든 참가자 조회 (이미 나간 참가자 포함)
	participants, err := s.roomRepo.GetAllParticipants(room.ID)
	if err != nil {
		s.logger.Error("통화 기록용 참가자 조회 실패", zap.Error(err))
		return
	}

	if len(participants) == 0 {
		s.logger.Debug("참가자 없음, 통화 기록 생성 건너뜀")
		return
	}

	// 통화 시간 계산
	endedAt := time.Now()
	startedAt := room.CreatedAt
	durationSeconds := int(endedAt.Sub(startedAt).Seconds())

	// 중복 제거 로직: UserID별로 그룹화
	type UserStats struct {
		JoinedAt time.Time
		LeftAt   time.Time
	}
	userStatsMap := make(map[uuid.UUID]UserStats)

	for _, p := range participants {
		leftAt := endedAt
		if p.LeftAt != nil {
			leftAt = *p.LeftAt
		}

		stats, exists := userStatsMap[p.UserID]
		if !exists {
			userStatsMap[p.UserID] = UserStats{
				JoinedAt: p.JoinedAt,
				LeftAt:   leftAt,
			}
		} else {
			// 이미 존재하면 가장 빠른 입장, 가장 늦은 퇴장 시간 사용
			if p.JoinedAt.Before(stats.JoinedAt) {
				stats.JoinedAt = p.JoinedAt
			}
			if leftAt.After(stats.LeftAt) {
				stats.LeftAt = leftAt
			}
			userStatsMap[p.UserID] = stats
		}
	}

	// 통화 기록 생성
	history := &domain.CallHistory{
		RoomID:            room.ID,
		RoomName:          room.Name,
		WorkspaceID:       room.WorkspaceID,
		CreatorID:         room.CreatorID,
		StartedAt:         startedAt,
		EndedAt:           endedAt,
		DurationSeconds:   durationSeconds,
		MaxParticipants:   room.MaxParticipants,
		TotalParticipants: len(userStatsMap),
	}

	// 맵에서 CallHistoryParticipant 목록 생성
	historyParticipants := make([]domain.CallHistoryParticipant, 0, len(userStatsMap))
	for uid, stats := range userStatsMap {
		participantDuration := int(stats.LeftAt.Sub(stats.JoinedAt).Seconds())

		historyParticipants = append(historyParticipants, domain.CallHistoryParticipant{
			UserID:          uid,
			JoinedAt:        stats.JoinedAt,
			LeftAt:          stats.LeftAt,
			DurationSeconds: participantDuration,
		})
	}

	history.Participants = historyParticipants

	if err := s.roomRepo.CreateCallHistory(history); err != nil {
		s.logger.Error("통화 기록 생성 실패", zap.Error(err))
		return
	}

	s.logger.Info("통화 기록 생성 완료",
		zap.String("room_id", room.ID.String()),
		zap.String("room_name", room.Name),
		zap.Int("duration_seconds", durationSeconds),
		zap.Int("total_participants", len(historyParticipants)),
	)
}

// GetWorkspaceCallHistory는 워크스페이스의 통화 기록을 반환합니다.
func (s *roomService) GetWorkspaceCallHistory(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID, token string, limit, offset int) ([]domain.CallHistoryResponse, int64, error) {
	// 워크스페이스 멤버십 검증
	if s.userClient != nil {
		isMember, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
		if err != nil {
			s.logger.Error("워크스페이스 멤버십 검증 실패", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to validate workspace membership: %w", err)
		}
		if !isMember {
			return nil, 0, response.ErrNotWorkspaceMember
		}
	}

	// 페이지네이션 기본값 및 제한
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	histories, total, err := s.roomRepo.GetCallHistoryByWorkspace(workspaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get call history: %w", err)
	}

	responses := make([]domain.CallHistoryResponse, len(histories))
	for i, h := range histories {
		responses[i] = h.ToResponse()
	}

	return responses, total, nil
}

// GetUserCallHistory는 사용자의 통화 기록을 반환합니다.
func (s *roomService) GetUserCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.CallHistoryResponse, int64, error) {
	// 페이지네이션 기본값 및 제한
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	histories, total, err := s.roomRepo.GetCallHistoryByUser(userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get call history: %w", err)
	}

	responses := make([]domain.CallHistoryResponse, len(histories))
	for i, h := range histories {
		responses[i] = h.ToResponse()
	}

	return responses, total, nil
}

// GetCallHistoryByID는 ID로 단일 통화 기록을 조회합니다.
func (s *roomService) GetCallHistoryByID(ctx context.Context, historyID uuid.UUID) (*domain.CallHistoryResponse, error) {
	history, err := s.roomRepo.GetCallHistoryByID(historyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get call history: %w", err)
	}
	if history == nil {
		return nil, nil
	}

	response := history.ToResponse()
	return &response, nil
}

// SaveTranscript는 룸의 트랜스크립트를 저장하거나 업데이트합니다.
func (s *roomService) SaveTranscript(ctx context.Context, roomID uuid.UUID, content string) (*domain.TranscriptResponse, error) {
	// 기존 트랜스크립트 확인
	existingTranscript, _ := s.roomRepo.GetTranscriptByRoomID(roomID)

	if existingTranscript != nil {
		// 기존 트랜스크립트 업데이트
		existingTranscript.Content = content
		if err := s.roomRepo.SaveTranscript(existingTranscript); err != nil {
			return nil, fmt.Errorf("failed to update transcript: %w", err)
		}

		s.logger.Info("트랜스크립트 업데이트 완료",
			zap.String("room_id", roomID.String()),
			zap.Int("content_length", len(content)),
		)

		response := existingTranscript.ToResponse()
		return &response, nil
	}

	// 새 트랜스크립트 생성 - roomID로 먼저 저장
	// callHistoryID는 룸이 종료될 때까지 nil
	transcript := &domain.CallTranscript{
		RoomID:  roomID,
		Content: content,
	}

	// 이미 통화 기록이 있는지 확인 (룸이 종료된 경우)
	histories, _, _ := s.roomRepo.GetCallHistoryByWorkspace(uuid.Nil, 100, 0)
	for _, h := range histories {
		if h.RoomID == roomID {
			transcript.CallHistoryID = h.ID
			break
		}
	}

	if err := s.roomRepo.SaveTranscript(transcript); err != nil {
		return nil, fmt.Errorf("failed to save transcript: %w", err)
	}

	s.logger.Info("트랜스크립트 저장 완료",
		zap.String("room_id", roomID.String()),
		zap.Int("content_length", len(content)),
	)

	response := transcript.ToResponse()
	return &response, nil
}

// GetTranscriptByCallHistoryID는 통화 기록의 트랜스크립트를 반환합니다.
func (s *roomService) GetTranscriptByCallHistoryID(ctx context.Context, callHistoryID uuid.UUID) (*domain.TranscriptResponse, error) {
	transcript, err := s.roomRepo.GetTranscriptByCallHistoryID(callHistoryID)
	if err != nil {
		// 통화 기록에서 룸 ID로 트랜스크립트 찾기 시도
		history, histErr := s.roomRepo.GetCallHistoryByID(callHistoryID)
		if histErr != nil {
			return nil, fmt.Errorf("call history not found: %w", histErr)
		}

		// 룸 ID로 트랜스크립트 찾기
		transcript, err = s.roomRepo.GetTranscriptByRoomID(history.RoomID)
		if err != nil {
			return nil, fmt.Errorf("transcript not found: %w", err)
		}
	}

	response := transcript.ToResponse()
	return &response, nil
}
