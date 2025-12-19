package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"storage-service/internal/client"
	"storage-service/internal/domain"
	"storage-service/internal/repository"
	"storage-service/internal/response"
)

// AccessService handles all access control logic for storage resources
type AccessService interface {
	// Workspace level
	ValidateWorkspaceAccess(ctx context.Context, workspaceID, userID uuid.UUID, token string) error

	// Project level
	ValidateProjectAccess(ctx context.Context, projectID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error
	GetProjectPermission(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectPermission, error)

	// File level - checks project permission if file belongs to a project
	ValidateFileAccess(ctx context.Context, fileID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error

	// Folder level - checks project permission if folder belongs to a project
	ValidateFolderAccess(ctx context.Context, folderID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error

	// Bulk validation for workspace + optional project
	ValidateResourceAccess(ctx context.Context, workspaceID uuid.UUID, projectID *uuid.UUID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error
}

type accessService struct {
	projectRepo repository.ProjectRepository
	fileRepo    *repository.FileRepository
	folderRepo  *repository.FolderRepository
	userClient  client.UserClient
	logger      *zap.Logger
}

// NewAccessService creates a new access service
func NewAccessService(
	projectRepo repository.ProjectRepository,
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	userClient client.UserClient,
	logger *zap.Logger,
) AccessService {
	return &accessService{
		projectRepo: projectRepo,
		fileRepo:    fileRepo,
		folderRepo:  folderRepo,
		userClient:  userClient,
		logger:      logger,
	}
}

// ValidateWorkspaceAccess validates that a user is a member of a workspace
func (s *accessService) ValidateWorkspaceAccess(ctx context.Context, workspaceID, userID uuid.UUID, token string) error {
	if s.userClient == nil {
		s.logger.Warn("User client not configured, skipping workspace validation")
		return nil
	}

	isValid, err := s.userClient.ValidateWorkspaceMember(ctx, workspaceID, userID, token)
	if err != nil {
		s.logger.Error("Failed to validate workspace member",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		return response.ErrNotWorkspaceMember
	}

	if !isValid {
		return response.ErrNotWorkspaceMember
	}

	return nil
}

// ValidateProjectAccess validates that a user has the required permission for a project
func (s *accessService) ValidateProjectAccess(ctx context.Context, projectID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// First validate workspace membership
	if err := s.ValidateWorkspaceAccess(ctx, project.WorkspaceID, userID, token); err != nil {
		return err
	}

	// Get user's permission for this project
	perm, err := s.projectRepo.GetUserPermission(ctx, projectID, userID)
	if err != nil {
		return response.ErrAccessDenied
	}

	// Check if permission is sufficient
	switch requiredPermission {
	case domain.ProjectPermissionViewer:
		if !perm.CanView() {
			return response.ErrInsufficientPermission
		}
	case domain.ProjectPermissionEditor:
		if !perm.CanEdit() {
			return response.ErrInsufficientPermission
		}
	case domain.ProjectPermissionOwner:
		if !perm.CanManage() {
			return response.ErrInsufficientPermission
		}
	}

	return nil
}

// GetProjectPermission gets the user's permission for a project
func (s *accessService) GetProjectPermission(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectPermission, error) {
	return s.projectRepo.GetUserPermission(ctx, projectID, userID)
}

// ValidateFileAccess validates access to a file
func (s *accessService) ValidateFileAccess(ctx context.Context, fileID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceAccess(ctx, file.WorkspaceID, userID, token); err != nil {
		return err
	}

	// If file belongs to a project, validate project permission
	if file.ProjectID != nil {
		return s.ValidateProjectAccess(ctx, *file.ProjectID, userID, token, requiredPermission)
	}

	// File is at workspace level - workspace membership is sufficient for viewing
	// For editing, we could add additional checks here if needed
	return nil
}

// ValidateFolderAccess validates access to a folder
func (s *accessService) ValidateFolderAccess(ctx context.Context, folderID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error {
	folder, err := s.folderRepo.FindByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceAccess(ctx, folder.WorkspaceID, userID, token); err != nil {
		return err
	}

	// If folder belongs to a project, validate project permission
	if folder.ProjectID != nil {
		return s.ValidateProjectAccess(ctx, *folder.ProjectID, userID, token, requiredPermission)
	}

	// Folder is at workspace level - workspace membership is sufficient for viewing
	return nil
}

// ValidateResourceAccess validates access to a workspace and optionally a project
func (s *accessService) ValidateResourceAccess(ctx context.Context, workspaceID uuid.UUID, projectID *uuid.UUID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) error {
	// Always validate workspace membership
	if err := s.ValidateWorkspaceAccess(ctx, workspaceID, userID, token); err != nil {
		return err
	}

	// If project is specified, validate project permission
	if projectID != nil {
		// Verify project belongs to this workspace
		project, err := s.projectRepo.GetByID(ctx, *projectID)
		if err != nil {
			return err
		}
		// 프로젝트가 해당 워크스페이스에 속하는지 확인
		if project.WorkspaceID != workspaceID {
			return response.NewForbiddenError("project does not belong to this workspace", "")
		}

		return s.ValidateProjectAccess(ctx, *projectID, userID, token, requiredPermission)
	}

	return nil
}
