package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ops-service/internal/domain"
)

// AppConfigRepository handles app config database operations
type AppConfigRepository struct {
	db *gorm.DB
}

// NewAppConfigRepository creates a new app config repository
func NewAppConfigRepository(db *gorm.DB) *AppConfigRepository {
	return &AppConfigRepository{db: db}
}

// Create creates a new app config
func (r *AppConfigRepository) Create(config *domain.AppConfig) error {
	return r.db.Create(config).Error
}

// GetByID gets an app config by ID
func (r *AppConfigRepository) GetByID(id uuid.UUID) (*domain.AppConfig, error) {
	var config domain.AppConfig
	if err := r.db.Where("id = ?", id).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByKey gets an app config by key
func (r *AppConfigRepository) GetByKey(key string) (*domain.AppConfig, error) {
	var config domain.AppConfig
	if err := r.db.Where("key = ?", key).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetAll gets all app configs
func (r *AppConfigRepository) GetAll() ([]domain.AppConfig, error) {
	var configs []domain.AppConfig
	if err := r.db.Order("key").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetActive gets all active app configs
func (r *AppConfigRepository) GetActive() ([]domain.AppConfig, error) {
	var configs []domain.AppConfig
	if err := r.db.Where("is_active = ?", true).Order("key").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// Update updates an app config
func (r *AppConfigRepository) Update(config *domain.AppConfig) error {
	return r.db.Save(config).Error
}

// Delete soft deletes an app config
func (r *AppConfigRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.AppConfig{}, id).Error
}

// ExistsByKey checks if a config exists by key
func (r *AppConfigRepository) ExistsByKey(key string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.AppConfig{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
