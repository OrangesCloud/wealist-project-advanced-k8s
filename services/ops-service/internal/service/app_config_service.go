package service

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ops-service/internal/domain"
	"ops-service/internal/repository"
	"ops-service/internal/response"
)

// AppConfigService handles app config business logic
type AppConfigService struct {
	repo     *repository.AppConfigRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
}

// NewAppConfigService creates a new app config service
func NewAppConfigService(
	repo *repository.AppConfigRepository,
	auditSvc *AuditLogService,
	logger *zap.Logger,
) *AppConfigService {
	return &AppConfigService{
		repo:     repo,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

// GetByKey gets an app config by key
func (s *AppConfigService) GetByKey(key string) (*domain.AppConfig, error) {
	config, err := s.repo.GetByKey(key)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.ErrConfigNotFound
		}
		return nil, err
	}
	return config, nil
}

// GetByID gets an app config by ID
func (s *AppConfigService) GetByID(id uuid.UUID) (*domain.AppConfig, error) {
	config, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.ErrConfigNotFound
		}
		return nil, err
	}
	return config, nil
}

// GetAll gets all app configs
func (s *AppConfigService) GetAll() ([]domain.AppConfig, error) {
	return s.repo.GetAll()
}

// GetActive gets all active app configs
func (s *AppConfigService) GetActive() ([]domain.AppConfig, error) {
	return s.repo.GetActive()
}

// Create creates a new app config
func (s *AppConfigService) Create(userID uuid.UUID, userEmail string, req domain.CreateAppConfigRequest) (*domain.AppConfig, error) {
	exists, err := s.repo.ExistsByKey(req.Key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, response.ErrConfigExists
	}

	config := &domain.AppConfig{
		Key:         req.Key,
		Value:       req.Value,
		Description: req.Description,
		IsActive:    true,
		UpdatedBy:   userID,
	}

	if err := s.repo.Create(config); err != nil {
		return nil, err
	}

	// Log audit
	s.auditSvc.Log(userID, userEmail, domain.ActionCreate, domain.ResourceAppConfig, config.ID.String(), "Created config: "+req.Key)

	s.logger.Info("App config created",
		zap.String("key", req.Key),
		zap.String("createdBy", userEmail),
	)

	return config, nil
}

// Update updates an app config
func (s *AppConfigService) Update(userID uuid.UUID, userEmail string, id uuid.UUID, req domain.UpdateAppConfigRequest) (*domain.AppConfig, error) {
	config, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.ErrConfigNotFound
		}
		return nil, err
	}

	oldValue := config.Value
	config.Value = req.Value
	config.Description = req.Description
	config.UpdatedBy = userID

	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}

	if err := s.repo.Update(config); err != nil {
		return nil, err
	}

	// Log audit
	details := "Updated config: " + config.Key
	if oldValue != req.Value {
		details += " (value changed)"
	}
	s.auditSvc.Log(userID, userEmail, domain.ActionUpdate, domain.ResourceAppConfig, config.ID.String(), details)

	s.logger.Info("App config updated",
		zap.String("key", config.Key),
		zap.String("updatedBy", userEmail),
	)

	return config, nil
}

// Delete deletes an app config
func (s *AppConfigService) Delete(userID uuid.UUID, userEmail string, id uuid.UUID) error {
	config, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.ErrConfigNotFound
		}
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Log audit
	s.auditSvc.Log(userID, userEmail, domain.ActionDelete, domain.ResourceAppConfig, id.String(), "Deleted config: "+config.Key)

	s.logger.Info("App config deleted",
		zap.String("key", config.Key),
		zap.String("deletedBy", userEmail),
	)

	return nil
}
