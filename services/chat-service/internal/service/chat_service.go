// Package service는 chat-service의 비즈니스 로직을 구현합니다.
//
// 이 패키지는 채팅방 생성, 메시지 전송, 참가자 관리 등의 핵심 기능을 제공합니다.
// 메트릭 수집 및 로깅이 통합되어 있습니다.
package service

import (
	"chat-service/internal/client"
	"chat-service/internal/domain"
	"chat-service/internal/metrics"
	"chat-service/internal/repository"
	"chat-service/internal/response"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ChatService는 채팅 관련 비즈니스 로직을 처리합니다.
type ChatService struct {
	chatRepo    *repository.ChatRepository
	messageRepo *repository.MessageRepository
	userClient  client.UserClient
	redis       *redis.Client
	logger      *zap.Logger
	metrics     *metrics.Metrics
}

// NewChatService는 새 ChatService를 생성합니다.
func NewChatService(
	chatRepo *repository.ChatRepository,
	messageRepo *repository.MessageRepository,
	userClient client.UserClient,
	redis *redis.Client,
	logger *zap.Logger,
	m *metrics.Metrics,
) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		userClient:  userClient,
		redis:       redis,
		logger:      logger,
		metrics:     m,
	}
}

// ============================================================
// 비즈니스 검증 헬퍼 메서드
// ============================================================

// validateWorkspaceMember는 사용자가 워크스페이스 멤버인지 검증합니다.
func (s *ChatService) validateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) error {
	if s.userClient == nil {
		s.logger.Warn("UserClient가 설정되지 않음, 워크스페이스 검증 건너뜀")
		return nil
	}

	isMember, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
	if err != nil {
		s.logger.Error("워크스페이스 멤버십 검증 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return response.ErrNotWorkspaceMember
	}

	if !isMember {
		return response.ErrNotWorkspaceMember
	}

	return nil
}

// validateChatParticipant는 사용자가 채팅 참가자인지 검증합니다.
func (s *ChatService) validateChatParticipant(chatID, userID uuid.UUID) error {
	isParticipant, err := s.chatRepo.IsUserInChat(chatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return response.ErrNotChatParticipant
	}
	return nil
}

// validateChatCreator는 사용자가 채팅방 생성자인지 검증합니다.
func (s *ChatService) validateChatCreator(chat *domain.Chat, userID uuid.UUID) error {
	if chat.CreatedBy != userID {
		return response.ErrNotChatCreator
	}
	return nil
}

// CreateChat은 새 채팅방을 생성합니다.
// 워크스페이스 멤버십 검증 후 생성자를 자동으로 참가자 목록에 추가합니다.
func (s *ChatService) CreateChat(ctx context.Context, req *domain.CreateChatRequest, createdBy uuid.UUID) (*domain.Chat, error) {
	chat := &domain.Chat{
		ID:          uuid.New(),
		WorkspaceID: req.WorkspaceID,
		ProjectID:   req.ProjectID,
		ChatType:    req.ChatType,
		ChatName:    req.ChatName,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.chatRepo.Create(chat); err != nil {
		s.logger.Error("채팅방 생성 실패",
			zap.String("workspace_id", req.WorkspaceID.String()),
			zap.String("created_by", createdBy.String()),
			zap.Error(err))
		return nil, err
	}

	// 생성자를 참가자 목록에 추가
	participantIDs := append([]uuid.UUID{createdBy}, req.Participants...)
	uniqueIDs := make(map[uuid.UUID]bool)
	for _, id := range participantIDs {
		uniqueIDs[id] = true
	}

	var uniqueParticipants []uuid.UUID
	for id := range uniqueIDs {
		uniqueParticipants = append(uniqueParticipants, id)
	}

	if err := s.chatRepo.AddParticipants(chat.ID, uniqueParticipants); err != nil {
		s.logger.Error("채팅방 참가자 추가 실패",
			zap.String("chat_id", chat.ID.String()),
			zap.Error(err))
		return nil, err
	}

	// 📊 메트릭: 채팅방 수 업데이트
	if s.metrics != nil {
		count, _ := s.chatRepo.CountAll()
		s.metrics.SetChatsTotal(count)
	}

	s.logger.Info("채팅방 생성 완료",
		zap.String("chat_id", chat.ID.String()),
		zap.String("workspace_id", req.WorkspaceID.String()),
		zap.String("chat_type", string(req.ChatType)),
		zap.String("created_by", createdBy.String()),
		zap.Int("participant_count", len(uniqueParticipants)))

	// 참가자 정보와 함께 리로드
	return s.chatRepo.GetByID(chat.ID)
}

// GetChatByID는 ID로 채팅방을 조회합니다.
func (s *ChatService) GetChatByID(ctx context.Context, chatID uuid.UUID) (*domain.Chat, error) {
	return s.chatRepo.GetByID(chatID)
}

// GetUserChats는 사용자가 참여 중인 채팅방 목록을 조회합니다.
func (s *ChatService) GetUserChats(ctx context.Context, userID uuid.UUID) ([]domain.ChatWithUnread, error) {
	return s.chatRepo.GetUserChats(userID)
}

// GetWorkspaceChats는 워크스페이스의 채팅방 목록을 조회합니다.
func (s *ChatService) GetWorkspaceChats(ctx context.Context, workspaceID uuid.UUID) ([]domain.Chat, error) {
	return s.chatRepo.GetWorkspaceChats(workspaceID)
}

// DeleteChat은 채팅방을 소프트 삭제합니다.
// 채팅방 생성자만 삭제할 수 있습니다.
func (s *ChatService) DeleteChat(ctx context.Context, chatID, userID uuid.UUID) error {
	// 📋 채팅방 존재 및 생성자 검증
	chat, err := s.chatRepo.GetByID(chatID)
	if err != nil {
		s.logger.Warn("채팅방 조회 실패",
			zap.String("chat_id", chatID.String()),
			zap.Error(err))
		return response.ErrChatNotFound
	}

	if err := s.validateChatCreator(chat, userID); err != nil {
		s.logger.Warn("채팅방 삭제 권한 없음",
			zap.String("chat_id", chatID.String()),
			zap.String("user_id", userID.String()),
			zap.String("creator_id", chat.CreatedBy.String()))
		return err
	}

	if err := s.chatRepo.SoftDelete(chatID); err != nil {
		s.logger.Error("채팅방 삭제 실패",
			zap.String("chat_id", chatID.String()),
			zap.Error(err))
		return err
	}

	// 📊 메트릭: 채팅방 수 업데이트
	if s.metrics != nil {
		count, _ := s.chatRepo.CountAll()
		s.metrics.SetChatsTotal(count)
	}

	s.logger.Info("채팅방 삭제 완료",
		zap.String("chat_id", chatID.String()),
		zap.String("deleted_by", userID.String()))

	return nil
}

// AddParticipants는 채팅방에 참가자를 추가합니다.
func (s *ChatService) AddParticipants(ctx context.Context, chatID uuid.UUID, userIDs []uuid.UUID) error {
	if err := s.chatRepo.AddParticipants(chatID, userIDs); err != nil {
		s.logger.Error("참가자 추가 실패",
			zap.String("chat_id", chatID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("참가자 추가 완료",
		zap.String("chat_id", chatID.String()),
		zap.Int("added_count", len(userIDs)))

	return nil
}

// RemoveParticipant는 채팅방에서 참가자를 제거합니다.
// 본인이 나가거나, 채팅방 생성자가 다른 참가자를 제거할 수 있습니다.
func (s *ChatService) RemoveParticipant(ctx context.Context, chatID, targetUserID, requesterID uuid.UUID) error {
	// 📋 채팅방 존재 확인
	chat, err := s.chatRepo.GetByID(chatID)
	if err != nil {
		s.logger.Warn("채팅방 조회 실패",
			zap.String("chat_id", chatID.String()),
			zap.Error(err))
		return response.ErrChatNotFound
	}

	// 📋 권한 검증: 본인 제거 또는 생성자의 타인 제거만 허용
	isSelfRemoval := targetUserID == requesterID
	isCreator := chat.CreatedBy == requesterID

	if !isSelfRemoval && !isCreator {
		s.logger.Warn("참가자 제거 권한 없음",
			zap.String("chat_id", chatID.String()),
			zap.String("target_user_id", targetUserID.String()),
			zap.String("requester_id", requesterID.String()))
		return response.ErrNotChatCreator
	}

	// 📋 생성자는 본인을 제거할 수 없음 (채팅방 삭제 필요)
	if chat.CreatedBy == targetUserID {
		s.logger.Warn("생성자 본인 제거 시도",
			zap.String("chat_id", chatID.String()),
			zap.String("creator_id", targetUserID.String()))
		return response.NewForbiddenError("creator cannot leave chat", "use delete chat instead")
	}

	if err := s.chatRepo.RemoveParticipant(chatID, targetUserID); err != nil {
		s.logger.Error("참가자 제거 실패",
			zap.String("chat_id", chatID.String()),
			zap.String("user_id", targetUserID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("참가자 제거 완료",
		zap.String("chat_id", chatID.String()),
		zap.String("removed_user_id", targetUserID.String()),
		zap.String("removed_by", requesterID.String()))

	return nil
}

// IsUserInChat은 사용자가 채팅방에 참여 중인지 확인합니다.
func (s *ChatService) IsUserInChat(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	return s.chatRepo.IsUserInChat(chatID, userID)
}

// SendMessage는 채팅방에 메시지를 전송합니다.
// 참가자 검증 후 메시지 생성 및 Redis를 통해 실시간 브로드캐스트합니다.
func (s *ChatService) SendMessage(ctx context.Context, chatID, userID uuid.UUID, req *domain.SendMessageRequest) (*domain.Message, error) {
	// 📋 참가자 검증: 채팅방 참가자만 메시지 전송 가능
	if err := s.validateChatParticipant(chatID, userID); err != nil {
		s.logger.Warn("채팅 참가자 검증 실패",
			zap.String("chat_id", chatID.String()),
			zap.String("user_id", userID.String()))
		return nil, err
	}

	// 📋 메시지 검증: 텍스트 메시지는 내용이 비어있으면 안됨
	messageType := domain.MessageTypeText
	if req.MessageType != "" {
		messageType = req.MessageType
	}

	if messageType == domain.MessageTypeText && strings.TrimSpace(req.Content) == "" {
		return nil, response.ErrEmptyMessage
	}

	message := &domain.Message{
		ID:          uuid.New(),
		ChatID:      chatID,
		UserID:      userID,
		Content:     req.Content,
		MessageType: messageType,
		FileURL:     req.FileURL,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.messageRepo.Create(message); err != nil {
		s.logger.Error("메시지 생성 실패",
			zap.String("chat_id", chatID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, err
	}

	// 채팅방 타임스탬프 업데이트
	s.chatRepo.UpdateTimestamp(chatID)

	// 📊 메트릭: 메시지 전송 카운트 증가
	if s.metrics != nil {
		s.metrics.RecordMessageSent()
		count, _ := s.messageRepo.CountAll()
		s.metrics.SetMessagesTotal(count)
	}

	s.logger.Debug("메시지 전송 완료",
		zap.String("message_id", message.ID.String()),
		zap.String("chat_id", chatID.String()),
		zap.String("user_id", userID.String()),
		zap.String("message_type", string(messageType)))

	// Redis를 통해 WebSocket 브로드캐스트
	s.publishMessage(ctx, chatID, message)

	return message, nil
}

// GetMessages는 채팅방의 메시지 목록을 조회합니다.
// 커서 기반 페이지네이션을 지원합니다.
func (s *ChatService) GetMessages(ctx context.Context, chatID uuid.UUID, limit int, before *uuid.UUID) ([]domain.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.messageRepo.GetByChatID(chatID, limit, before)
}

// DeleteMessage는 메시지를 소프트 삭제합니다.
// 메시지 작성자만 삭제할 수 있습니다.
func (s *ChatService) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	// 📋 메시지 존재 및 소유자 검증
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		s.logger.Warn("메시지 조회 실패",
			zap.String("message_id", messageID.String()),
			zap.Error(err))
		return response.ErrMessageNotFound
	}

	if message.UserID != userID {
		s.logger.Warn("메시지 삭제 권한 없음",
			zap.String("message_id", messageID.String()),
			zap.String("user_id", userID.String()),
			zap.String("owner_id", message.UserID.String()))
		return response.ErrNotMessageOwner
	}

	if err := s.messageRepo.SoftDelete(messageID); err != nil {
		s.logger.Error("메시지 삭제 실패",
			zap.String("message_id", messageID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("메시지 삭제 완료",
		zap.String("message_id", messageID.String()),
		zap.String("deleted_by", userID.String()))

	return nil
}

// MarkMessagesAsRead는 메시지들을 읽음으로 표시합니다.
func (s *ChatService) MarkMessagesAsRead(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) error {
	if err := s.messageRepo.MarkMultipleAsRead(messageIDs, userID); err != nil {
		s.logger.Error("메시지 읽음 표시 실패",
			zap.String("user_id", userID.String()),
			zap.Int("message_count", len(messageIDs)),
			zap.Error(err))
		return err
	}

	// 📊 메트릭: 메시지 읽음 카운트 증가
	if s.metrics != nil {
		for range messageIDs {
			s.metrics.RecordMessageRead()
		}
	}

	s.logger.Debug("메시지 읽음 표시 완료",
		zap.String("user_id", userID.String()),
		zap.Int("message_count", len(messageIDs)))

	return nil
}

// UpdateLastReadAt은 채팅방의 마지막 읽은 시간을 업데이트합니다.
func (s *ChatService) UpdateLastReadAt(ctx context.Context, chatID, userID uuid.UUID) error {
	return s.chatRepo.UpdateLastReadAt(chatID, userID)
}

// GetUnreadCount는 채팅방의 안 읽은 메시지 수를 반환합니다.
func (s *ChatService) GetUnreadCount(ctx context.Context, chatID, userID uuid.UUID) (int64, error) {
	chat, err := s.chatRepo.GetByID(chatID)
	if err != nil {
		return 0, err
	}

	var lastReadAt *time.Time
	for _, p := range chat.Participants {
		if p.UserID == userID {
			lastReadAt = p.LastReadAt
			break
		}
	}

	return s.messageRepo.GetUnreadCount(chatID, userID, lastReadAt)
}

// publishMessage는 Redis를 통해 메시지를 브로드캐스트합니다.
func (s *ChatService) publishMessage(ctx context.Context, chatID uuid.UUID, message *domain.Message) {
	if s.redis == nil {
		return
	}

	channel := fmt.Sprintf("chat:%s", chatID.String())
	data, err := json.Marshal(map[string]interface{}{
		"type":    "MESSAGE_RECEIVED",
		"message": message,
	})
	if err != nil {
		s.logger.Error("메시지 직렬화 실패",
			zap.String("chat_id", chatID.String()),
			zap.Error(err))
		return
	}

	if err := s.redis.Publish(ctx, channel, data).Err(); err != nil {
		s.logger.Error("Redis 메시지 발행 실패",
			zap.String("channel", channel),
			zap.Error(err))
	}
}
