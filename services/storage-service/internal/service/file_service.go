package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"storage-service/internal/client"
	"storage-service/internal/domain"
	"storage-service/internal/metrics"
	"storage-service/internal/repository"
	"storage-service/internal/response"
)

// Allowed file extensions
var (
	allowedImageExts    = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".ico"}
	allowedDocumentExts = []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf", ".odt", ".ods", ".odp"}
	allowedArchiveExts  = []string{".zip", ".rar", ".7z", ".tar", ".gz"}
	allowedVideoExts    = []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv"}
	allowedAudioExts    = []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".wma"}
)

// MaxFileSize is the maximum allowed file size (100MB)
const MaxFileSize = 100 * 1024 * 1024

// FileService handles file business logic
// 파일 업로드, 다운로드, 삭제 등의 비즈니스 로직을 처리합니다.
// 메트릭과 로깅을 통해 모니터링을 지원합니다.
type FileService struct {
	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository
	s3Client   *client.S3Client
	logger     *zap.Logger
	metrics    *metrics.Metrics // 메트릭 수집을 위한 필드
}

// NewFileService creates a new FileService
// metrics 파라미터가 nil인 경우에도 안전하게 동작합니다.
func NewFileService(
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	s3Client *client.S3Client,
	logger *zap.Logger,
	m *metrics.Metrics,
) *FileService {
	return &FileService{
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		s3Client:   s3Client,
		logger:     logger,
		metrics:    m,
	}
}

// isAllowedExtension checks if file extension is allowed
func isAllowedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	allAllowed := append(allowedImageExts, allowedDocumentExts...)
	allAllowed = append(allAllowed, allowedArchiveExts...)
	allAllowed = append(allAllowed, allowedVideoExts...)
	allAllowed = append(allAllowed, allowedAudioExts...)

	for _, allowed := range allAllowed {
		if ext == allowed {
			return true
		}
	}
	return false
}

// GenerateUploadURL generates a presigned URL for file upload
func (s *FileService) GenerateUploadURL(ctx context.Context, req domain.GenerateUploadURLRequest, userID uuid.UUID) (*domain.GenerateUploadURLResponse, error) {
	// Validate file size
	if req.FileSize > MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed (%d MB)", MaxFileSize/(1024*1024))
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if !isAllowedExtension(ext) {
		return nil, fmt.Errorf("file type not allowed: %s", ext)
	}

	// 폴더 검증 (폴더 ID가 제공된 경우)
	if req.FolderID != nil {
		folder, err := s.folderRepo.FindByID(ctx, *req.FolderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.NewNotFoundError("folder not found", req.FolderID.String())
			}
			return nil, fmt.Errorf("failed to find folder: %w", err)
		}
		// 다른 워크스페이스의 폴더 차단
		if folder.WorkspaceID != req.WorkspaceID {
			return nil, response.NewForbiddenError("folder belongs to different workspace", "")
		}
	}

	// Generate presigned URL
	uploadURL, fileKey, err := s.s3Client.GeneratePresignedURL(ctx, req.WorkspaceID.String(), req.FileName, req.ContentType)
	if err != nil {
		s.logger.Error("Failed to generate presigned URL", zap.Error(err))
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Create file record with UPLOADING status
	file := &domain.File{
		ID:           uuid.New(),
		WorkspaceID:  req.WorkspaceID,
		ProjectID:    req.ProjectID,
		FolderID:     req.FolderID,
		Name:         req.FileName,
		OriginalName: req.FileName,
		FileKey:      fileKey,
		FileSize:     req.FileSize,
		ContentType:  req.ContentType,
		Status:       domain.FileStatusUploading,
		Version:      1,
		UploadedBy:   userID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		s.logger.Error("Failed to create file record", zap.Error(err))
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	s.logger.Info("Upload URL generated",
		zap.String("fileId", file.ID.String()),
		zap.String("fileName", req.FileName),
		zap.String("userId", userID.String()),
	)

	return &domain.GenerateUploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   fileKey,
		FileID:    file.ID,
		ExpiresAt: expiresAt,
	}, nil
}

// ConfirmUpload confirms that file upload is complete
// 파일 업로드 확정: 상태를 UPLOADING에서 ACTIVE로 변경
func (s *FileService) ConfirmUpload(ctx context.Context, fileID, userID uuid.UUID) (*domain.File, error) {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("file not found", fileID.String())
		}
		return nil, fmt.Errorf("failed to find file: %w", err)
	}

	// 업로드한 사용자만 확정 가능
	if file.UploadedBy != userID {
		return nil, response.NewForbiddenError("not authorized to confirm this upload", "")
	}

	// UPLOADING 상태에서만 확정 가능
	if file.Status != domain.FileStatusUploading {
		return nil, response.NewConflictError("file is not in uploading state", string(file.Status))
	}

	// Generate unique name if necessary
	uniqueName, err := s.fileRepo.GenerateUniqueName(ctx, file.WorkspaceID, file.FolderID, file.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique name: %w", err)
	}
	file.Name = uniqueName

	file.Status = domain.FileStatusActive
	file.UpdatedAt = time.Now()

	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to update file status: %w", err)
	}

	// 메트릭 기록: 파일 업로드 성공
	if s.metrics != nil {
		s.metrics.RecordFileUpload()
	}

	s.logger.Info("File upload confirmed",
		zap.String("fileId", file.ID.String()),
		zap.String("userId", userID.String()),
		zap.Int64("fileSize", file.FileSize),
	)

	return file, nil
}

// GetFile gets a file by ID
// 파일 ID로 파일 조회
func (s *FileService) GetFile(ctx context.Context, fileID uuid.UUID) (*domain.File, error) {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("file not found", fileID.String())
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return file, nil
}

// GetFileURL returns the public URL for a file
func (s *FileService) GetFileURL(fileKey string) string {
	return s.s3Client.GetFileURL(fileKey)
}

// GetWorkspaceFiles gets all files in a workspace with pagination
func (s *FileService) GetWorkspaceFiles(ctx context.Context, workspaceID uuid.UUID, page, pageSize int) (*domain.FileListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	files, total, err := s.fileRepo.FindByWorkspaceID(ctx, workspaceID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	var fileResponses []domain.FileResponse
	for _, file := range files {
		fileResponses = append(fileResponses, file.ToResponse(s.GetFileURL(file.FileKey)))
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.FileListResponse{
		Files:      fileResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetFolderFiles gets all files in a folder
func (s *FileService) GetFolderFiles(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID) ([]domain.File, error) {
	return s.fileRepo.FindByFolderID(ctx, workspaceID, folderID)
}

// UpdateFile updates a file
// 파일 수정: 이름 변경, 폴더 이동
func (s *FileService) UpdateFile(ctx context.Context, fileID uuid.UUID, req domain.UpdateFileRequest, userID uuid.UUID) (*domain.File, error) {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError("file not found", fileID.String())
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// 이름 변경
	if req.Name != nil && *req.Name != file.Name {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, response.NewValidationError("file name cannot be empty", "")
		}

		// 확장자 유지
		oldExt := filepath.Ext(file.Name)
		newExt := filepath.Ext(name)
		if newExt == "" {
			name = name + oldExt
		}

		// 중복 이름 확인
		exists, err := s.fileRepo.ExistsByNameInFolder(ctx, file.WorkspaceID, file.FolderID, name)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate name: %w", err)
		}
		if exists {
			return nil, response.NewAlreadyExistsError("file with this name already exists", name)
		}

		file.Name = name
	}

	// 폴더 이동
	if req.FolderID != nil {
		if *req.FolderID == uuid.Nil {
			file.FolderID = nil
		} else {
			folder, err := s.folderRepo.FindByID(ctx, *req.FolderID)
			if err != nil {
				return nil, response.NewNotFoundError("destination folder not found", req.FolderID.String())
			}
			// 다른 워크스페이스로 이동 불가
			if folder.WorkspaceID != file.WorkspaceID {
				return nil, response.NewForbiddenError("cannot move file to different workspace", "")
			}
			file.FolderID = &folder.ID
		}
	}

	file.UpdatedAt = time.Now()

	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to update file: %w", err)
	}

	s.logger.Info("File updated",
		zap.String("fileId", file.ID.String()),
		zap.String("userId", userID.String()),
	)

	return file, nil
}

// DeleteFile soft deletes a file (move to trash)
// 파일 삭제 (휴지통으로 이동): 소프트 삭제 성공 시 메트릭을 기록합니다.
func (s *FileService) DeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("file not found", fileID.String())
		}
		s.logger.Error("Failed to find file for deletion",
			zap.String("fileId", fileID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get file: %w", err)
	}

	if err := s.fileRepo.SoftDelete(ctx, fileID); err != nil {
		s.logger.Error("Failed to soft delete file",
			zap.String("fileId", fileID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 메트릭 기록: 파일 삭제 성공
	if s.metrics != nil {
		s.metrics.RecordFileDelete()
	}

	s.logger.Info("File deleted (moved to trash)",
		zap.String("fileId", file.ID.String()),
		zap.String("userId", userID.String()),
		zap.String("fileName", file.Name),
	)

	return nil
}

// RestoreFile restores a deleted file
// 삭제된 파일 복원
func (s *FileService) RestoreFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.fileRepo.FindByIDWithDeleted(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("file not found", fileID.String())
		}
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 삭제 상태 확인
	if file.DeletedAt == nil && file.Status != domain.FileStatusDeleted {
		return response.NewConflictError("file is not deleted", fileID.String())
	}

	if err := s.fileRepo.Restore(ctx, fileID); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	s.logger.Info("File restored",
		zap.String("fileId", file.ID.String()),
		zap.String("userId", userID.String()),
	)

	return nil
}

// PermanentDeleteFile permanently deletes a file
// 파일 영구 삭제: S3와 DB에서 완전 삭제
func (s *FileService) PermanentDeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.fileRepo.FindByIDWithDeleted(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError("file not found", fileID.String())
		}
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Delete from S3
	if err := s.s3Client.DeleteFile(ctx, file.FileKey); err != nil {
		s.logger.Error("Failed to delete file from S3", zap.Error(err))
		// Continue anyway to delete database record
	}

	if err := s.fileRepo.PermanentDelete(ctx, fileID); err != nil {
		return fmt.Errorf("failed to permanently delete file: %w", err)
	}

	s.logger.Info("File permanently deleted",
		zap.String("fileId", file.ID.String()),
		zap.String("userId", userID.String()),
	)

	return nil
}

// GetTrashFiles gets all deleted files in a workspace
func (s *FileService) GetTrashFiles(ctx context.Context, workspaceID uuid.UUID) ([]domain.File, error) {
	return s.fileRepo.FindDeleted(ctx, workspaceID)
}

// SearchFiles searches files by name
func (s *FileService) SearchFiles(ctx context.Context, workspaceID uuid.UUID, query string, page, pageSize int) (*domain.FileListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	files, total, err := s.fileRepo.Search(ctx, workspaceID, query, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	var fileResponses []domain.FileResponse
	for _, file := range files {
		fileResponses = append(fileResponses, file.ToResponse(s.GetFileURL(file.FileKey)))
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.FileListResponse{
		Files:      fileResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetStorageUsage gets storage usage for a workspace
func (s *FileService) GetStorageUsage(ctx context.Context, workspaceID uuid.UUID) (int64, int64, error) {
	totalSize, err := s.fileRepo.SumSizeByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get storage usage: %w", err)
	}

	fileCount, err := s.fileRepo.CountByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count files: %w", err)
	}

	return totalSize, fileCount, nil
}

// CleanupOrphanedUploads cleans up files stuck in uploading state
func (s *FileService) CleanupOrphanedUploads(ctx context.Context) error {
	files, err := s.fileRepo.FindUploadingFiles(ctx, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to find orphaned uploads: %w", err)
	}

	for _, file := range files {
		// Delete from S3
		if err := s.s3Client.DeleteFile(ctx, file.FileKey); err != nil {
			s.logger.Error("Failed to delete orphaned file from S3",
				zap.Error(err),
				zap.String("fileKey", file.FileKey),
			)
		}

		// Delete database record
		if err := s.fileRepo.PermanentDelete(ctx, file.ID); err != nil {
			s.logger.Error("Failed to delete orphaned file record",
				zap.Error(err),
				zap.String("fileId", file.ID.String()),
			)
		}
	}

	if len(files) > 0 {
		s.logger.Info("Cleaned up orphaned uploads", zap.Int("count", len(files)))
	}

	return nil
}

// GenerateDownloadURL generates a presigned URL for file download
// 다운로드 URL 생성: 다운로드 URL 생성 시 메트릭을 기록합니다.
func (s *FileService) GenerateDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", response.NewNotFoundError("file not found", fileID.String())
		}
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	url, err := s.s3Client.GenerateDownloadURL(ctx, file.FileKey, file.OriginalName)
	if err != nil {
		s.logger.Error("Failed to generate download URL",
			zap.String("fileId", fileID.String()),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	// 메트릭 기록: 파일 다운로드 요청
	if s.metrics != nil {
		s.metrics.RecordFileDownload()
	}

	s.logger.Info("Download URL generated",
		zap.String("fileId", fileID.String()),
		zap.String("fileName", file.Name),
	)

	return url, nil
}
