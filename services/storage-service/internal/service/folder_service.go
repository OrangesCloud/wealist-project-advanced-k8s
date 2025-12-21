package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"storage-service/internal/domain"
	"storage-service/internal/repository"
	"storage-service/internal/response"
)

// FolderService handles folder business logic
type FolderService struct {
	folderRepo *repository.FolderRepository
	fileRepo   *repository.FileRepository
	logger     *zap.Logger
}

// NewFolderService creates a new FolderService
func NewFolderService(
	folderRepo *repository.FolderRepository,
	fileRepo *repository.FileRepository,
	logger *zap.Logger,
) *FolderService {
	return &FolderService{
		folderRepo: folderRepo,
		fileRepo:   fileRepo,
		logger:     logger,
	}
}

// CreateFolder creates a new folder
// 폴더 생성: 이름 검증, 부모 폴더 확인, 경로 생성
func (s *FolderService) CreateFolder(ctx context.Context, req domain.CreateFolderRequest, userID uuid.UUID) (*domain.Folder, error) {
	// 폴더명 검증
	if strings.TrimSpace(req.Name) == "" {
		return nil, response.NewValidationError("folder name cannot be empty", "")
	}

	// 경로 생성
	var path string
	if req.ParentID == nil {
		path = "/" + req.Name
	} else {
		parent, err := s.folderRepo.FindByID(ctx, *req.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.NewNotFoundError("parent folder not found", "")
			}
			return nil, fmt.Errorf("failed to find parent folder: %w", err)
		}
		// 부모 폴더가 다른 워크스페이스에 속한 경우 차단
		if parent.WorkspaceID != req.WorkspaceID {
			return nil, response.NewForbiddenError("parent folder belongs to different workspace", "")
		}
		path = parent.Path + "/" + req.Name
	}

	// Generate unique name if necessary
	uniqueName, err := s.folderRepo.GenerateUniqueName(ctx, req.WorkspaceID, req.ParentID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique name: %w", err)
	}
	if uniqueName != req.Name {
		// Update path with unique name
		if req.ParentID == nil {
			path = "/" + uniqueName
		} else {
			parent, _ := s.folderRepo.FindByID(ctx, *req.ParentID)
			path = parent.Path + "/" + uniqueName
		}
	}

	folder := &domain.Folder{
		ID:          uuid.New(),
		WorkspaceID: req.WorkspaceID,
		ProjectID:   req.ProjectID,
		ParentID:    req.ParentID,
		Name:        uniqueName,
		Path:        path,
		Color:       req.Color,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.folderRepo.Create(ctx, folder); err != nil {
		s.logger.Error("Failed to create folder",
			zap.Error(err),
			zap.String("workspaceId", req.WorkspaceID.String()),
			zap.String("name", req.Name),
		)
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	s.logger.Info("Folder created",
		zap.String("folderId", folder.ID.String()),
		zap.String("path", folder.Path),
		zap.String("userId", userID.String()),
	)

	return folder, nil
}

// GetFolder gets a folder by ID
// 폴더 ID로 폴더 조회
func (s *FolderService) GetFolder(ctx context.Context, folderID uuid.UUID) (*domain.Folder, error) {
	folder, err := s.folderRepo.FindByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("folder not found", folderID.String())
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	return folder, nil
}

// GetFolderContents gets folder with its children and files
// 폴더 내용 조회: 하위 폴더 및 파일 목록 포함
func (s *FolderService) GetFolderContents(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID) (*domain.FolderResponse, error) {
	var folderResp domain.FolderResponse

	if folderID == nil {
		// Root folder
		folderResp = domain.FolderResponse{
			ID:          uuid.Nil,
			WorkspaceID: workspaceID,
			Name:        "Root",
			Path:        "/",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	} else {
		folder, err := s.folderRepo.FindByID(ctx, *folderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.NewNotFoundError("folder not found", folderID.String())
			}
			return nil, fmt.Errorf("failed to get folder: %w", err)
		}
		folderResp = folder.ToResponse()
	}

	// 하위 폴더 조회
	childFolders, err := s.folderRepo.FindByParentID(ctx, workspaceID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get child folders: %w", err)
	}
	for _, child := range childFolders {
		childResp := child.ToResponse()
		// 각 하위 폴더의 카운트 정보
		childResp.FolderCount, _ = s.folderRepo.CountByParentID(ctx, child.ID)
		childResp.FileCount, _ = s.fileRepo.CountByFolderID(ctx, child.ID)
		childResp.TotalSize, _ = s.fileRepo.SumSizeByFolderID(ctx, child.ID)
		folderResp.Children = append(folderResp.Children, childResp)
	}

	// 폴더 내 파일 조회
	files, err := s.fileRepo.FindByFolderID(ctx, workspaceID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}
	for _, file := range files {
		folderResp.Files = append(folderResp.Files, file.ToResponse(file.FileKey))
	}

	// 카운트 설정
	folderResp.FolderCount = int64(len(childFolders))
	folderResp.FileCount = int64(len(files))

	return &folderResp, nil
}

// GetWorkspaceFolders gets all folders in a workspace
func (s *FolderService) GetWorkspaceFolders(ctx context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	return s.folderRepo.FindByWorkspaceID(ctx, workspaceID)
}

// GetRootFolders gets all root folders in a workspace
func (s *FolderService) GetRootFolders(ctx context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	return s.folderRepo.FindRootFolders(ctx, workspaceID)
}

// UpdateFolder updates a folder
// 폴더 수정: 이름 변경, 색상 변경, 이동
func (s *FolderService) UpdateFolder(ctx context.Context, folderID uuid.UUID, req domain.UpdateFolderRequest, userID uuid.UUID) (*domain.Folder, error) {
	folder, err := s.folderRepo.FindByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("folder not found", folderID.String())
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	oldPath := folder.Path

	// 이름 변경
	if req.Name != nil && *req.Name != folder.Name {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, response.NewValidationError("folder name cannot be empty", "")
		}

		// 중복 이름 확인
		exists, err := s.folderRepo.ExistsByNameInParent(ctx, folder.WorkspaceID, folder.ParentID, name)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate name: %w", err)
		}
		if exists {
			return nil, response.NewAlreadyExistsError("folder with this name already exists", name)
		}

		folder.Name = name
		// 경로 업데이트
		if folder.ParentID == nil {
			folder.Path = "/" + name
		} else {
			parent, _ := s.folderRepo.FindByID(ctx, *folder.ParentID)
			folder.Path = parent.Path + "/" + name
		}
	}

	// 색상 변경
	if req.Color != nil {
		folder.Color = req.Color
	}

	// 폴더 이동
	if req.ParentID != nil && (folder.ParentID == nil || *req.ParentID != *folder.ParentID) {
		// 새 부모 폴더 검증
		if *req.ParentID != uuid.Nil {
			newParent, err := s.folderRepo.FindByID(ctx, *req.ParentID)
			if err != nil {
				return nil, response.NewNotFoundError("new parent folder not found", req.ParentID.String())
			}
			// 다른 워크스페이스로 이동 불가
			if newParent.WorkspaceID != folder.WorkspaceID {
				return nil, response.NewForbiddenError("cannot move folder to different workspace", "")
			}
			// 자신 또는 하위 폴더로 이동 불가
			if strings.HasPrefix(newParent.Path, folder.Path+"/") || newParent.ID == folder.ID {
				return nil, response.NewBadRequestError("cannot move folder into itself or its descendants", "")
			}
			folder.ParentID = &newParent.ID
			folder.Path = newParent.Path + "/" + folder.Name
		} else {
			folder.ParentID = nil
			folder.Path = "/" + folder.Name
		}
	}

	folder.UpdatedAt = time.Now()

	if err := s.folderRepo.Update(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	// Update paths of descendants if path changed
	if oldPath != folder.Path {
		if err := s.folderRepo.UpdatePath(ctx, folder.WorkspaceID, oldPath, folder.Path); err != nil {
			s.logger.Error("Failed to update descendant paths", zap.Error(err))
		}
	}

	s.logger.Info("Folder updated",
		zap.String("folderId", folder.ID.String()),
		zap.String("userId", userID.String()),
	)

	return folder, nil
}

// DeleteFolder soft deletes a folder (move to trash)
// 폴더 삭제 (휴지통으로 이동): 하위 폴더 및 파일도 함께 삭제
func (s *FolderService) DeleteFolder(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	folder, err := s.folderRepo.FindByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("folder not found", folderID.String())
		}
		return fmt.Errorf("failed to get folder: %w", err)
	}

	// Soft delete folder and all descendants
	if err := s.folderRepo.SoftDeleteByPath(ctx, folder.WorkspaceID, folder.Path); err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	// Soft delete all files in folder and descendants
	if err := s.fileRepo.SoftDeleteByFolderID(ctx, folderID); err != nil {
		s.logger.Error("Failed to delete files in folder", zap.Error(err))
	}

	s.logger.Info("Folder deleted (moved to trash)",
		zap.String("folderId", folder.ID.String()),
		zap.String("userId", userID.String()),
	)

	return nil
}

// RestoreFolder restores a deleted folder
// 삭제된 폴더 복원
func (s *FolderService) RestoreFolder(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	folder, err := s.folderRepo.FindByIDWithDeleted(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("folder not found", folderID.String())
		}
		return fmt.Errorf("failed to get folder: %w", err)
	}

	// 삭제 상태 확인
	if folder.DeletedAt == nil {
		return response.NewConflictError("folder is not deleted", folderID.String())
	}

	if err := s.folderRepo.Restore(ctx, folderID); err != nil {
		return fmt.Errorf("failed to restore folder: %w", err)
	}

	s.logger.Info("Folder restored",
		zap.String("folderId", folder.ID.String()),
		zap.String("userId", userID.String()),
	)

	return nil
}

// PermanentDeleteFolder permanently deletes a folder
// 폴더 영구 삭제
func (s *FolderService) PermanentDeleteFolder(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	folder, err := s.folderRepo.FindByIDWithDeleted(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("folder not found", folderID.String())
		}
		return fmt.Errorf("failed to get folder: %w", err)
	}

	if err := s.folderRepo.PermanentDelete(ctx, folderID); err != nil {
		return fmt.Errorf("failed to permanently delete folder: %w", err)
	}

	s.logger.Info("Folder permanently deleted",
		zap.String("folderId", folder.ID.String()),
		zap.String("userId", userID.String()),
	)

	return nil
}

// GetTrashFolders gets all deleted folders in a workspace
func (s *FolderService) GetTrashFolders(ctx context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	return s.folderRepo.FindDeleted(ctx, workspaceID)
}
