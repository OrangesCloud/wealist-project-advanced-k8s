package repository

import (
	"chat-service/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) Create(chat *domain.Chat) error {
	return r.db.Create(chat).Error
}

func (r *ChatRepository) GetByID(id uuid.UUID) (*domain.Chat, error) {
	var chat domain.Chat
	err := r.db.Preload("Participants", "is_active = ?", true).
		First(&chat, "id = ? AND deleted_at IS NULL", id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *ChatRepository) GetUserChats(userID uuid.UUID) ([]domain.ChatWithUnread, error) {
	var chats []domain.Chat

	err := r.db.
		Joins("JOIN chat_participants ON chats.id = chat_participants.chat_id").
		Where("chat_participants.user_id = ? AND chat_participants.is_active = ? AND chats.deleted_at IS NULL", userID, true).
		Preload("Participants", "is_active = ?", true).
		Order("chats.updated_at DESC").
		Find(&chats).Error

	if err != nil {
		return nil, err
	}

	// Calculate unread count for each chat
	result := make([]domain.ChatWithUnread, len(chats))
	for i, chat := range chats {
		result[i].Chat = chat

		// Get participant's last read time
		var participant domain.ChatParticipant
		r.db.Where("chat_id = ? AND user_id = ? AND is_active = ?", chat.ID, userID, true).First(&participant)

		// Count unread messages
		query := r.db.Model(&domain.Message{}).
			Where("chat_id = ? AND deleted_at IS NULL AND user_id != ?", chat.ID, userID)

		if participant.LastReadAt != nil {
			query = query.Where("created_at > ?", participant.LastReadAt)
		}

		query.Count(&result[i].UnreadCount)
	}

	return result, nil
}

func (r *ChatRepository) GetWorkspaceChats(workspaceID uuid.UUID) ([]domain.Chat, error) {
	var chats []domain.Chat
	err := r.db.
		Where("workspace_id = ? AND deleted_at IS NULL", workspaceID).
		Preload("Participants", "is_active = ?", true).
		Order("updated_at DESC").
		Find(&chats).Error
	return chats, err
}

func (r *ChatRepository) SoftDelete(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&domain.Chat{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error
}

func (r *ChatRepository) UpdateTimestamp(id uuid.UUID) error {
	return r.db.Model(&domain.Chat{}).
		Where("id = ?", id).
		Update("updated_at", time.Now()).Error
}

func (r *ChatRepository) AddParticipant(participant *domain.ChatParticipant) error {
	// Upsert: reactivate if exists, create if not
	return r.db.Exec(`
		INSERT INTO chat_participants (id, chat_id, user_id, joined_at, is_active)
		VALUES (?, ?, ?, ?, true)
		ON CONFLICT (chat_id, user_id) WHERE is_active = true
		DO UPDATE SET is_active = true, joined_at = EXCLUDED.joined_at
	`, uuid.New(), participant.ChatID, participant.UserID, time.Now()).Error
}

func (r *ChatRepository) AddParticipants(chatID uuid.UUID, userIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			err := tx.Exec(`
				INSERT INTO chat_participants (id, chat_id, user_id, joined_at, is_active)
				VALUES (?, ?, ?, ?, true)
				ON CONFLICT (chat_id, user_id) WHERE is_active = true
				DO UPDATE SET is_active = true
			`, uuid.New(), chatID, userID, time.Now()).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ChatRepository) RemoveParticipant(chatID, userID uuid.UUID) error {
	return r.db.Model(&domain.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Update("is_active", false).Error
}

func (r *ChatRepository) IsUserInChat(chatID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ? AND is_active = ?", chatID, userID, true).
		Count(&count).Error
	return count > 0, err
}

func (r *ChatRepository) UpdateLastReadAt(chatID, userID uuid.UUID) error {
	return r.db.Model(&domain.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ? AND is_active = ?", chatID, userID, true).
		Update("last_read_at", time.Now()).Error
}

// CountAll은 전체 채팅방 수를 반환합니다.
// 메트릭용으로 사용됩니다.
func (r *ChatRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Chat{}).
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}
