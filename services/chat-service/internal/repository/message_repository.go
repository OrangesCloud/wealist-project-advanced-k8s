package repository

import (
	"chat-service/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(message *domain.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepository) GetByID(id uuid.UUID) (*domain.Message, error) {
	var message domain.Message
	err := r.db.First(&message, "id = ? AND deleted_at IS NULL", id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepository) GetByChatID(chatID uuid.UUID, limit int, before *uuid.UUID) ([]domain.Message, error) {
	var messages []domain.Message

	query := r.db.Where("chat_id = ? AND deleted_at IS NULL", chatID)

	if before != nil {
		var beforeMsg domain.Message
		if err := r.db.First(&beforeMsg, "id = ?", before).Error; err == nil {
			query = query.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, err
}

func (r *MessageRepository) SoftDelete(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&domain.Message{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error
}

func (r *MessageRepository) MarkAsRead(messageID, userID uuid.UUID) error {
	read := &domain.MessageRead{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		ReadAt:    time.Now(),
	}

	return r.db.Clauses().Create(read).Error
}

func (r *MessageRepository) MarkMultipleAsRead(messageIDs []uuid.UUID, userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, msgID := range messageIDs {
			read := &domain.MessageRead{
				ID:        uuid.New(),
				MessageID: msgID,
				UserID:    userID,
				ReadAt:    time.Now(),
			}
			// Ignore conflict (already read)
			tx.Exec(`
				INSERT INTO message_reads (id, message_id, user_id, read_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT (message_id, user_id) DO NOTHING
			`, read.ID, read.MessageID, read.UserID, read.ReadAt)
		}
		return nil
	})
}

func (r *MessageRepository) GetUnreadCount(chatID, userID uuid.UUID, lastReadAt *time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&domain.Message{}).
		Where("chat_id = ? AND deleted_at IS NULL AND user_id != ?", chatID, userID)

	if lastReadAt != nil {
		query = query.Where("created_at > ?", lastReadAt)
	}

	err := query.Count(&count).Error
	return count, err
}

// CountAll은 전체 메시지 수를 반환합니다.
// 메트릭용으로 사용됩니다.
func (r *MessageRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Message{}).
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}
