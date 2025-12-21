package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"storage-service/internal/client"
	"storage-service/internal/domain"
	"storage-service/internal/repository"
	"storage-service/internal/response"
)

// ProjectService defines the interface for project operations
type ProjectService interface {
	// Project CRUD
	CreateProject(ctx context.Context, req domain.CreateProjectRequest, userID uuid.UUID, token string) (*domain.ProjectResponse, error)
	GetProject(ctx context.Context, projectID, userID uuid.UUID, token string) (*domain.ProjectResponse, error)
	GetWorkspaceProjects(ctx context.Context, workspaceID, userID uuid.UUID, token string, page, pageSize int) (*domain.ProjectListResponse, error)
	UpdateProject(ctx context.Context, projectID uuid.UUID, req domain.UpdateProjectRequest, userID uuid.UUID, token string) (*domain.ProjectResponse, error)
	DeleteProject(ctx context.Context, projectID, userID uuid.UUID, token string) error

	// Project Members
	AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddProjectMemberRequest, userID uuid.UUID, token string) (*domain.ProjectMemberResponse, error)
	GetMembers(ctx context.Context, projectID, userID uuid.UUID, token string) ([]domain.ProjectMemberResponse, error)
	UpdateMember(ctx context.Context, projectID, memberUserID uuid.UUID, req domain.UpdateProjectMemberRequest, userID uuid.UUID, token string) (*domain.ProjectMemberResponse, error)
	RemoveMember(ctx context.Context, projectID, memberUserID, userID uuid.UUID, token string) error

	// Access Control
	CheckAccess(ctx context.Context, projectID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) (*domain.ProjectAccessCheckResult, error)
	GetUserPermission(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectPermission, error)

	// Workspace validation
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) error
}

type projectService struct {
	projectRepo repository.ProjectRepository
	userClient  client.UserClient
	logger      *zap.Logger
}

// NewProjectService creates a new project service
func NewProjectService(projectRepo repository.ProjectRepository, userClient client.UserClient, logger *zap.Logger) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		userClient:  userClient,
		logger:      logger,
	}
}

// ValidateWorkspaceMember validates that a user is a member of a workspace
func (s *projectService) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) error {
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
		// On error, deny access for security
		return response.ErrNotWorkspaceMember
	}

	if !isValid {
		return response.ErrNotWorkspaceMember
	}

	return nil
}

// CreateProject creates a new project
func (s *projectService) CreateProject(ctx context.Context, req domain.CreateProjectRequest, userID uuid.UUID, token string) (*domain.ProjectResponse, error) {
	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, req.WorkspaceID, userID, token); err != nil {
		return nil, err
	}

	// Set defaults
	defaultPermission := domain.ProjectPermissionViewer
	if req.DefaultPermission != nil && req.DefaultPermission.IsValid() {
		defaultPermission = *req.DefaultPermission
	}

	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	project := &domain.Project{
		WorkspaceID:       req.WorkspaceID,
		Name:              req.Name,
		Description:       req.Description,
		DefaultPermission: defaultPermission,
		IsPublic:          isPublic,
		CreatedBy:         userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		return nil, err
	}

	// Add creator as owner
	member := &domain.ProjectMember{
		ProjectID:  project.ID,
		UserID:     userID,
		Permission: domain.ProjectPermissionOwner,
		AddedBy:    userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.projectRepo.AddMember(ctx, member); err != nil {
		s.logger.Error("Failed to add creator as owner", zap.Error(err))
		// Don't fail project creation, but log the error
	}

	response := project.ToResponse()
	ownerPerm := domain.ProjectPermissionOwner
	response.MyPermission = &ownerPerm
	response.MemberCount = 1

	return &response, nil
}

// GetProject retrieves a project by ID
func (s *projectService) GetProject(ctx context.Context, projectID, userID uuid.UUID, token string) (*domain.ProjectResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate workspace membership first
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return nil, err
	}

	// Check project access
	perm, err := s.projectRepo.GetUserPermission(ctx, projectID, userID)
	if err != nil {
		return nil, response.ErrAccessDenied
	}

	response := project.ToResponse()
	response.MyPermission = perm

	// Get stats
	fileCount, folderCount, totalSize, err := s.projectRepo.GetProjectStats(ctx, projectID)
	if err == nil {
		response.FileCount = fileCount
		response.FolderCount = folderCount
		response.TotalSize = totalSize
	}

	// Get member count
	members, err := s.projectRepo.GetMembers(ctx, projectID)
	if err == nil {
		response.MemberCount = int64(len(members))
	}

	return &response, nil
}

// GetWorkspaceProjects retrieves projects in a workspace that user has access to
func (s *projectService) GetWorkspaceProjects(ctx context.Context, workspaceID, userID uuid.UUID, token string, page, pageSize int) (*domain.ProjectListResponse, error) {
	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, workspaceID, userID, token); err != nil {
		return nil, err
	}

	// Get projects user has access to
	projects, err := s.projectRepo.GetUserProjects(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	total := int64(len(projects))
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(projects) {
		start = len(projects)
	}
	if end > len(projects) {
		end = len(projects)
	}
	pagedProjects := projects[start:end]

	// Convert to responses
	responses := make([]domain.ProjectResponse, len(pagedProjects))
	for i, p := range pagedProjects {
		responses[i] = p.ToResponse()

		// Get user's permission
		perm, _ := s.projectRepo.GetUserPermission(ctx, p.ID, userID)
		responses[i].MyPermission = perm

		// Get stats
		fileCount, folderCount, totalSize, err := s.projectRepo.GetProjectStats(ctx, p.ID)
		if err == nil {
			responses[i].FileCount = fileCount
			responses[i].FolderCount = folderCount
			responses[i].TotalSize = totalSize
		}
	}

	return &domain.ProjectListResponse{
		Projects:   responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateProject updates a project
func (s *projectService) UpdateProject(ctx context.Context, projectID uuid.UUID, req domain.UpdateProjectRequest, userID uuid.UUID, token string) (*domain.ProjectResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return nil, err
	}

	// Check permission (must be owner or editor)
	result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionEditor)
	if err != nil || !result.HasAccess {
		return nil, response.ErrInsufficientPermission
	}

	// Update fields
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.DefaultPermission != nil {
		if !req.DefaultPermission.IsValid() {
			return nil, response.ErrInvalidPermission
		}
		project.DefaultPermission = *req.DefaultPermission
	}
	if req.IsPublic != nil {
		project.IsPublic = *req.IsPublic
	}
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	response := project.ToResponse()
	response.MyPermission = result.Permission

	return &response, nil
}

// DeleteProject deletes a project (soft delete)
func (s *projectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID, token string) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return err
	}

	// Check permission (must be owner)
	result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionOwner)
	if err != nil || !result.HasAccess {
		return response.ErrInsufficientPermission
	}

	return s.projectRepo.Delete(ctx, projectID)
}

// AddMember adds a member to a project
func (s *projectService) AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddProjectMemberRequest, userID uuid.UUID, token string) (*domain.ProjectMemberResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate workspace membership for both users
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return nil, err
	}
	// 대상 사용자의 워크스페이스 멤버십 검증
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, req.UserID, token); err != nil {
		return nil, response.NewForbiddenError("target user is not a workspace member", req.UserID.String())
	}

	// Check permission (must be owner or editor)
	result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionEditor)
	if err != nil || !result.HasAccess {
		return nil, response.ErrInsufficientPermission
	}

	// Only owners can add other owners
	if req.Permission == domain.ProjectPermissionOwner && !result.IsOwner {
		return nil, response.ErrInsufficientPermission
	}

	if !req.Permission.IsValid() {
		return nil, response.ErrInvalidPermission
	}

	member := &domain.ProjectMember{
		ProjectID:  projectID,
		UserID:     req.UserID,
		Permission: req.Permission,
		AddedBy:    userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.projectRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	response := member.ToResponse()
	return &response, nil
}

// GetMembers retrieves all members of a project
func (s *projectService) GetMembers(ctx context.Context, projectID, userID uuid.UUID, token string) ([]domain.ProjectMemberResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return nil, err
	}

	// Check access (must have at least view permission)
	result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionViewer)
	if err != nil || !result.HasAccess {
		return nil, response.ErrAccessDenied
	}

	members, err := s.projectRepo.GetMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	responses := make([]domain.ProjectMemberResponse, len(members))
	for i, m := range members {
		responses[i] = m.ToResponse()
	}

	return responses, nil
}

// UpdateMember updates a project member's permission
func (s *projectService) UpdateMember(ctx context.Context, projectID, memberUserID uuid.UUID, req domain.UpdateProjectMemberRequest, userID uuid.UUID, token string) (*domain.ProjectMemberResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return nil, err
	}

	// Cannot change own role
	if memberUserID == userID {
		return nil, response.ErrCannotChangeOwnRole
	}

	// Check permission (must be owner)
	result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionOwner)
	if err != nil || !result.HasAccess {
		return nil, response.ErrInsufficientPermission
	}

	if !req.Permission.IsValid() {
		return nil, response.ErrInvalidPermission
	}

	member, err := s.projectRepo.GetMember(ctx, projectID, memberUserID)
	if err != nil {
		return nil, err
	}

	member.Permission = req.Permission
	member.UpdatedAt = time.Now()

	if err := s.projectRepo.UpdateMember(ctx, member); err != nil {
		return nil, err
	}

	response := member.ToResponse()
	return &response, nil
}

// RemoveMember removes a member from a project
func (s *projectService) RemoveMember(ctx context.Context, projectID, memberUserID, userID uuid.UUID, token string) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Validate workspace membership
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		return err
	}

	// Check if trying to remove owner
	member, err := s.projectRepo.GetMember(ctx, projectID, memberUserID)
	if err != nil {
		return err
	}
	if member.Permission == domain.ProjectPermissionOwner {
		// Check if there are other owners
		members, _ := s.projectRepo.GetMembers(ctx, projectID)
		ownerCount := 0
		for _, m := range members {
			if m.Permission == domain.ProjectPermissionOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return response.ErrCannotRemoveOwner
		}
	}

	// Check permission (must be owner, or removing self)
	if memberUserID != userID {
		result, err := s.CheckAccess(ctx, projectID, userID, token, domain.ProjectPermissionOwner)
		if err != nil || !result.HasAccess {
			return response.ErrInsufficientPermission
		}
	}

	return s.projectRepo.RemoveMember(ctx, projectID, memberUserID)
}

// CheckAccess checks if a user has sufficient access to a project
func (s *projectService) CheckAccess(ctx context.Context, projectID, userID uuid.UUID, token string, requiredPermission domain.ProjectPermission) (*domain.ProjectAccessCheckResult, error) {
	result := &domain.ProjectAccessCheckResult{
		HasAccess: false,
		IsOwner:   false,
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		result.Reason = "project not found"
		return result, err
	}

	// Validate workspace membership first
	if err := s.ValidateWorkspaceMember(ctx, project.WorkspaceID, userID, token); err != nil {
		result.Reason = "not a workspace member"
		return result, err
	}

	// Get user's permission
	perm, err := s.projectRepo.GetUserPermission(ctx, projectID, userID)
	if err != nil {
		result.Reason = "no access to project"
		return result, nil
	}

	result.Permission = perm
	result.IsOwner = *perm == domain.ProjectPermissionOwner

	// Check if permission is sufficient
	switch requiredPermission {
	case domain.ProjectPermissionViewer:
		result.HasAccess = perm.CanView()
	case domain.ProjectPermissionEditor:
		result.HasAccess = perm.CanEdit()
	case domain.ProjectPermissionOwner:
		result.HasAccess = perm.CanManage()
	}

	if !result.HasAccess {
		result.Reason = "insufficient permission"
	}

	return result, nil
}

// GetUserPermission gets the user's permission for a project
func (s *projectService) GetUserPermission(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectPermission, error) {
	return s.projectRepo.GetUserPermission(ctx, projectID, userID)
}
