package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"user-service/internal/domain"
)

// WorkspaceMemberRepository handles workspace member data access
type WorkspaceMemberRepository struct {
	db *gorm.DB
}

// NewWorkspaceMemberRepository creates a new WorkspaceMemberRepository
func NewWorkspaceMemberRepository(db *gorm.DB) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

// Create creates a new workspace member
func (r *WorkspaceMemberRepository) Create(member *domain.WorkspaceMember) error {
	return r.db.Create(member).Error
}

// FindByID finds a workspace member by ID
func (r *WorkspaceMemberRepository) FindByID(id uuid.UUID) (*domain.WorkspaceMember, error) {
	var member domain.WorkspaceMember
	err := r.db.Preload("User").Where("id = ? AND is_active = true", id).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// FindByWorkspaceAndUser finds a member by workspace and user
func (r *WorkspaceMemberRepository) FindByWorkspaceAndUser(workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	var member domain.WorkspaceMember
	err := r.db.Where("workspace_id = ? AND user_id = ? AND is_active = true", workspaceID, userID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// FindByWorkspace finds all members of a workspace
func (r *WorkspaceMemberRepository) FindByWorkspace(workspaceID uuid.UUID) ([]domain.WorkspaceMember, error) {
	var members []domain.WorkspaceMember
	err := r.db.Preload("User").Where("workspace_id = ? AND is_active = true", workspaceID).Find(&members).Error
	return members, err
}

// FindByUser finds all workspace memberships for a user (excludes soft-deleted workspaces)
func (r *WorkspaceMemberRepository) FindByUser(userID uuid.UUID) ([]domain.WorkspaceMember, error) {
	var members []domain.WorkspaceMember
	err := r.db.
		Joins("JOIN workspaces ON workspaces.id = workspace_members.workspace_id AND workspaces.deleted_at IS NULL").
		Preload("Workspace", "deleted_at IS NULL").
		Where("workspace_members.user_id = ? AND workspace_members.is_active = true", userID).
		Find(&members).Error
	return members, err
}

// FindDefaultWorkspace finds user's default workspace (excludes soft-deleted workspaces)
func (r *WorkspaceMemberRepository) FindDefaultWorkspace(userID uuid.UUID) (*domain.WorkspaceMember, error) {
	var member domain.WorkspaceMember
	err := r.db.
		Joins("JOIN workspaces ON workspaces.id = workspace_members.workspace_id AND workspaces.deleted_at IS NULL").
		Preload("Workspace", "deleted_at IS NULL").
		Where("workspace_members.user_id = ? AND workspace_members.is_default = true AND workspace_members.is_active = true", userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// Update updates a workspace member
func (r *WorkspaceMemberRepository) Update(member *domain.WorkspaceMember) error {
	return r.db.Save(member).Error
}

// Delete removes a workspace member
func (r *WorkspaceMemberRepository) Delete(id uuid.UUID) error {
	return r.db.Model(&domain.WorkspaceMember{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// SetDefault sets a workspace as default for user
func (r *WorkspaceMemberRepository) SetDefault(userID, workspaceID uuid.UUID) error {
	// First, unset all defaults for user
	if err := r.db.Model(&domain.WorkspaceMember{}).
		Where("user_id = ?", userID).
		Update("is_default", false).Error; err != nil {
		return err
	}
	// Then set the new default
	return r.db.Model(&domain.WorkspaceMember{}).
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		Update("is_default", true).Error
}

// IsMember checks if user is a member of workspace
func (r *WorkspaceMemberRepository) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ? AND is_active = true", workspaceID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetRole gets user's role in workspace
func (r *WorkspaceMemberRepository) GetRole(workspaceID, userID uuid.UUID) (domain.RoleName, error) {
	var member domain.WorkspaceMember
	err := r.db.Where("workspace_id = ? AND user_id = ? AND is_active = true", workspaceID, userID).First(&member).Error
	if err != nil {
		return "", err
	}
	return member.RoleName, nil
}

// CountByRole counts members with a specific role in a workspace
func (r *WorkspaceMemberRepository) CountByRole(workspaceID uuid.UUID, roleName domain.RoleName) (int64, error) {
	var count int64
	err := r.db.Model(&domain.WorkspaceMember{}).
		Where("workspace_id = ? AND role_name = ? AND is_active = true", workspaceID, roleName).
		Count(&count).Error
	return count, err
}
