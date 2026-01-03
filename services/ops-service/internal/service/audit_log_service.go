package service

import (
	"github.com/google/uuid"
	"go.uber.org/zap"

	"ops-service/internal/domain"
	"ops-service/internal/repository"
)

// AuditLogService handles audit log business logic
type AuditLogService struct {
	repo   *repository.AuditLogRepository
	logger *zap.Logger
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService(repo *repository.AuditLogRepository, logger *zap.Logger) *AuditLogService {
	return &AuditLogService{
		repo:   repo,
		logger: logger,
	}
}

// Log creates an audit log entry
func (s *AuditLogService) Log(userID uuid.UUID, userEmail string, action domain.ActionType, resourceType domain.ResourceType, resourceID, details string) {
	log := &domain.AuditLog{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
	}

	if err := s.repo.Create(log); err != nil {
		s.logger.Error("Failed to create audit log",
			zap.Error(err),
			zap.String("action", string(action)),
			zap.String("resourceType", string(resourceType)),
			zap.String("resourceID", resourceID),
		)
	}
}

// LogWithContext creates an audit log entry with IP and user agent
func (s *AuditLogService) LogWithContext(userID uuid.UUID, userEmail string, action domain.ActionType, resourceType domain.ResourceType, resourceID, details, ipAddress, userAgent string) {
	log := &domain.AuditLog{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.repo.Create(log); err != nil {
		s.logger.Error("Failed to create audit log",
			zap.Error(err),
			zap.String("action", string(action)),
			zap.String("resourceType", string(resourceType)),
			zap.String("resourceID", resourceID),
		)
	}
}

// List lists audit logs with filtering and pagination
func (s *AuditLogService) List(opts repository.ListOptions) ([]domain.AuditLog, int64, error) {
	return s.repo.List(opts)
}

// GetByID gets an audit log by ID
func (s *AuditLogService) GetByID(id uuid.UUID) (*domain.AuditLog, error) {
	return s.repo.GetByID(id)
}
