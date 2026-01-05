package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ops-service/internal/domain"
)

// AuditLogRepository handles audit log database operations
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(log *domain.AuditLog) error {
	return r.db.Create(log).Error
}

// GetByID gets an audit log by ID
func (r *AuditLogRepository) GetByID(id uuid.UUID) (*domain.AuditLog, error) {
	var log domain.AuditLog
	if err := r.db.Where("id = ?", id).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// ListOptions holds options for listing audit logs
type ListOptions struct {
	UserID       *uuid.UUID
	ResourceType *domain.ResourceType
	Action       *domain.ActionType
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	Limit        int
}

// List lists audit logs with filtering and pagination
func (r *AuditLogRepository) List(opts ListOptions) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64

	query := r.db.Model(&domain.AuditLog{})

	if opts.UserID != nil {
		query = query.Where("user_id = ?", *opts.UserID)
	}
	if opts.ResourceType != nil {
		query = query.Where("resource_type = ?", *opts.ResourceType)
	}
	if opts.Action != nil {
		query = query.Where("action = ?", *opts.Action)
	}
	if opts.StartTime != nil {
		query = query.Where("created_at >= ?", *opts.StartTime)
	}
	if opts.EndTime != nil {
		query = query.Where("created_at <= ?", *opts.EndTime)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 {
		opts.Limit = 20
	}
	offset := (opts.Page - 1) * opts.Limit

	if err := query.Order("created_at DESC").Offset(offset).Limit(opts.Limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
