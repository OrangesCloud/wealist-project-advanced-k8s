package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ops-service/internal/domain"
)

// PortalUserRepository handles portal user database operations
type PortalUserRepository struct {
	db *gorm.DB
}

// NewPortalUserRepository creates a new portal user repository
func NewPortalUserRepository(db *gorm.DB) *PortalUserRepository {
	return &PortalUserRepository{db: db}
}

// Create creates a new portal user
func (r *PortalUserRepository) Create(user *domain.PortalUser) error {
	return r.db.Create(user).Error
}

// GetByID gets a portal user by ID
func (r *PortalUserRepository) GetByID(id uuid.UUID) (*domain.PortalUser, error) {
	var user domain.PortalUser
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail gets a portal user by email
func (r *PortalUserRepository) GetByEmail(email string) (*domain.PortalUser, error) {
	var user domain.PortalUser
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAll gets all portal users
func (r *PortalUserRepository) GetAll() ([]domain.PortalUser, error) {
	var users []domain.PortalUser
	if err := r.db.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Update updates a portal user
func (r *PortalUserRepository) Update(user *domain.PortalUser) error {
	return r.db.Save(user).Error
}

// Delete soft deletes a portal user
func (r *PortalUserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.PortalUser{}, id).Error
}

// CountByRole counts users by role
func (r *PortalUserRepository) CountByRole(role domain.Role) (int64, error) {
	var count int64
	if err := r.db.Model(&domain.PortalUser{}).Where("role = ? AND is_active = ?", role, true).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByEmail checks if a user exists by email
func (r *PortalUserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.PortalUser{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
