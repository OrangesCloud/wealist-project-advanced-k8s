// Package repository provides data access layer for video-service.
//
// This package implements the RoomRepository interface for managing
// video rooms, participants, call history, and transcripts using GORM.
package repository

import (
	"video-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoomRepository defines the interface for room data persistence operations.
// It provides methods for room CRUD, participant management, call history,
// and transcript storage.
type RoomRepository interface {
	// Room CRUD operations
	Create(room *domain.Room) error
	GetByID(id uuid.UUID) (*domain.Room, error)
	GetByWorkspaceID(workspaceID uuid.UUID) ([]domain.Room, error)
	GetActiveByWorkspaceID(workspaceID uuid.UUID) ([]domain.Room, error)
	Update(room *domain.Room) error
	Delete(id uuid.UUID) error

	// Participant management
	AddParticipant(participant *domain.RoomParticipant) error
	RemoveParticipant(roomID, userID uuid.UUID) error
	UpdateParticipant(participant *domain.RoomParticipant) error
	GetParticipant(roomID, userID uuid.UUID) (*domain.RoomParticipant, error)
	GetActiveParticipants(roomID uuid.UUID) ([]domain.RoomParticipant, error)
	GetAllParticipants(roomID uuid.UUID) ([]domain.RoomParticipant, error)
	CountActiveParticipants(roomID uuid.UUID) (int64, error)

	// Call history operations
	CreateCallHistory(history *domain.CallHistory) error
	GetCallHistoryByWorkspace(workspaceID uuid.UUID, limit, offset int) ([]domain.CallHistory, int64, error)
	GetCallHistoryByUser(userID uuid.UUID, limit, offset int) ([]domain.CallHistory, int64, error)
	GetCallHistoryByID(id uuid.UUID) (*domain.CallHistory, error)

	// Transcript operations
	SaveTranscript(transcript *domain.CallTranscript) error
	GetTranscriptByCallHistoryID(callHistoryID uuid.UUID) (*domain.CallTranscript, error)
	GetTranscriptByRoomID(roomID uuid.UUID) (*domain.CallTranscript, error)
}

// roomRepository implements RoomRepository using GORM.
type roomRepository struct {
	db *gorm.DB
}

// NewRoomRepository creates a new RoomRepository with the given GORM database.
func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(room *domain.Room) error {
	return r.db.Create(room).Error
}

func (r *roomRepository) GetByID(id uuid.UUID) (*domain.Room, error) {
	var room domain.Room
	if err := r.db.Preload("Participants", "is_active = ?", true).First(&room, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) GetByWorkspaceID(workspaceID uuid.UUID) ([]domain.Room, error) {
	var rooms []domain.Room
	if err := r.db.Preload("Participants", "is_active = ?", true).
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepository) GetActiveByWorkspaceID(workspaceID uuid.UUID) ([]domain.Room, error) {
	var rooms []domain.Room
	if err := r.db.Preload("Participants", "is_active = ?", true).
		Where("workspace_id = ? AND is_active = ?", workspaceID, true).
		Order("created_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepository) Update(room *domain.Room) error {
	return r.db.Save(room).Error
}

func (r *roomRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Room{}, "id = ?", id).Error
}

func (r *roomRepository) AddParticipant(participant *domain.RoomParticipant) error {
	return r.db.Create(participant).Error
}

func (r *roomRepository) RemoveParticipant(roomID, userID uuid.UUID) error {
	return r.db.Model(&domain.RoomParticipant{}).
		Where("room_id = ? AND user_id = ? AND is_active = ?", roomID, userID, true).
		Updates(map[string]interface{}{
			"is_active": false,
			"left_at":   gorm.Expr("NOW()"),
		}).Error
}

func (r *roomRepository) UpdateParticipant(participant *domain.RoomParticipant) error {
	return r.db.Save(participant).Error
}

func (r *roomRepository) GetParticipant(roomID, userID uuid.UUID) (*domain.RoomParticipant, error) {
	var participant domain.RoomParticipant
	if err := r.db.Where("room_id = ? AND user_id = ? AND is_active = ?", roomID, userID, true).
		First(&participant).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *roomRepository) GetActiveParticipants(roomID uuid.UUID) ([]domain.RoomParticipant, error) {
	var participants []domain.RoomParticipant
	if err := r.db.Where("room_id = ? AND is_active = ?", roomID, true).
		Find(&participants).Error; err != nil {
		return nil, err
	}
	return participants, nil
}

func (r *roomRepository) CountActiveParticipants(roomID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.Model(&domain.RoomParticipant{}).
		Where("room_id = ? AND is_active = ?", roomID, true).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *roomRepository) GetAllParticipants(roomID uuid.UUID) ([]domain.RoomParticipant, error) {
	var participants []domain.RoomParticipant
	if err := r.db.Where("room_id = ?", roomID).
		Order("joined_at ASC").
		Find(&participants).Error; err != nil {
		return nil, err
	}
	return participants, nil
}

// Call History methods

func (r *roomRepository) CreateCallHistory(history *domain.CallHistory) error {
	return r.db.Create(history).Error
}

func (r *roomRepository) GetCallHistoryByWorkspace(workspaceID uuid.UUID, limit, offset int) ([]domain.CallHistory, int64, error) {
	var histories []domain.CallHistory
	var total int64

	// Count total
	if err := r.db.Model(&domain.CallHistory{}).
		Where("workspace_id = ?", workspaceID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with participants
	if err := r.db.Preload("Participants").
		Where("workspace_id = ?", workspaceID).
		Order("ended_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *roomRepository) GetCallHistoryByUser(userID uuid.UUID, limit, offset int) ([]domain.CallHistory, int64, error) {
	var histories []domain.CallHistory
	var total int64

	// Subquery to find call history IDs where user participated
	subQuery := r.db.Model(&domain.CallHistoryParticipant{}).
		Select("call_history_id").
		Where("user_id = ?", userID)

	// Count total
	if err := r.db.Model(&domain.CallHistory{}).
		Where("id IN (?)", subQuery).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with participants
	if err := r.db.Preload("Participants").
		Where("id IN (?)", subQuery).
		Order("ended_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *roomRepository) GetCallHistoryByID(id uuid.UUID) (*domain.CallHistory, error) {
	var history domain.CallHistory
	if err := r.db.Preload("Participants").First(&history, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

// Transcript methods

func (r *roomRepository) SaveTranscript(transcript *domain.CallTranscript) error {
	// Upsert - update if exists, create if not
	return r.db.Save(transcript).Error
}

func (r *roomRepository) GetTranscriptByCallHistoryID(callHistoryID uuid.UUID) (*domain.CallTranscript, error) {
	var transcript domain.CallTranscript
	if err := r.db.Where("call_history_id = ?", callHistoryID).First(&transcript).Error; err != nil {
		return nil, err
	}
	return &transcript, nil
}

func (r *roomRepository) GetTranscriptByRoomID(roomID uuid.UUID) (*domain.CallTranscript, error) {
	var transcript domain.CallTranscript
	if err := r.db.Where("room_id = ?", roomID).First(&transcript).Error; err != nil {
		return nil, err
	}
	return &transcript, nil
}
