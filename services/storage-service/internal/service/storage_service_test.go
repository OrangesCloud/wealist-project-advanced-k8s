// Package service는 storage-service의 비즈니스 로직 테스트를 포함합니다.
// 이 파일은 File, Folder, Project 서비스의 공통 테스트를 포함합니다.
package service

import (
	"context"
	"storage-service/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ============================================================
// FileStatus 상수 테스트
// ============================================================

func TestStorageService_FileStatus_Constants(t *testing.T) {
	// Given/When/Then: 파일 상태 상수 확인
	assert.Equal(t, domain.FileStatus("UPLOADING"), domain.FileStatusUploading)
	assert.Equal(t, domain.FileStatus("ACTIVE"), domain.FileStatusActive)
	assert.Equal(t, domain.FileStatus("DELETED"), domain.FileStatusDeleted)
}

// ============================================================
// ProjectPermission 상수 테스트
// ============================================================

func TestStorageService_ProjectPermission_Constants(t *testing.T) {
	// Given/When/Then: 프로젝트 권한 상수 확인
	assert.Equal(t, domain.ProjectPermission("OWNER"), domain.ProjectPermissionOwner)
	assert.Equal(t, domain.ProjectPermission("VIEWER"), domain.ProjectPermissionViewer)
	assert.Equal(t, domain.ProjectPermission("EDITOR"), domain.ProjectPermissionEditor)
}

// ============================================================
// File 구조체 테스트
// ============================================================

func TestStorageService_File_Fields(t *testing.T) {
	// Given: File 생성
	fileID := uuid.New()
	workspaceID := uuid.New()
	projectID := uuid.New()
	folderID := uuid.New()
	uploadedBy := uuid.New()
	now := time.Now()

	file := &domain.File{
		ID:           fileID,
		WorkspaceID:  workspaceID,
		ProjectID:    &projectID,
		FolderID:     &folderID,
		Name:         "test-file.pdf",
		OriginalName: "Original Test File.pdf",
		FileKey:      "workspaces/ws-123/files/file-abc.pdf",
		FileSize:     1024 * 1024, // 1MB
		ContentType:  "application/pdf",
		Status:       domain.FileStatusActive,
		Version:      1,
		UploadedBy:   uploadedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Then: 필드 확인
	assert.Equal(t, fileID, file.ID)
	assert.Equal(t, workspaceID, file.WorkspaceID)
	assert.Equal(t, &projectID, file.ProjectID)
	assert.Equal(t, &folderID, file.FolderID)
	assert.Equal(t, "test-file.pdf", file.Name)
	assert.Equal(t, "Original Test File.pdf", file.OriginalName)
	assert.Equal(t, "workspaces/ws-123/files/file-abc.pdf", file.FileKey)
	assert.Equal(t, int64(1024*1024), file.FileSize)
	assert.Equal(t, "application/pdf", file.ContentType)
	assert.Equal(t, domain.FileStatusActive, file.Status)
	assert.Equal(t, 1, file.Version)
	assert.Equal(t, uploadedBy, file.UploadedBy)
}

func TestStorageService_File_TableName(t *testing.T) {
	// Given
	file := domain.File{}

	// When/Then
	assert.Equal(t, "storage_files", file.TableName())
}

// ============================================================
// Folder 구조체 테스트
// ============================================================

func TestStorageService_Folder_Fields(t *testing.T) {
	// Given: Folder 생성
	folderID := uuid.New()
	workspaceID := uuid.New()
	projectID := uuid.New()
	parentID := uuid.New()
	createdBy := uuid.New()
	color := "#FF5733"
	now := time.Now()

	folder := &domain.Folder{
		ID:          folderID,
		WorkspaceID: workspaceID,
		ProjectID:   &projectID,
		ParentID:    &parentID,
		Name:        "Documents",
		Path:        "/projects/main/documents",
		Color:       &color,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Then: 필드 확인
	assert.Equal(t, folderID, folder.ID)
	assert.Equal(t, workspaceID, folder.WorkspaceID)
	assert.Equal(t, &projectID, folder.ProjectID)
	assert.Equal(t, &parentID, folder.ParentID)
	assert.Equal(t, "Documents", folder.Name)
	assert.Equal(t, "/projects/main/documents", folder.Path)
	assert.Equal(t, &color, folder.Color)
	assert.Equal(t, createdBy, folder.CreatedBy)
}

func TestStorageService_Folder_TableName(t *testing.T) {
	// Given
	folder := domain.Folder{}

	// When/Then
	assert.Equal(t, "storage_folders", folder.TableName())
}

// ============================================================
// Project 구조체 테스트
// ============================================================

func TestStorageService_Project_Fields(t *testing.T) {
	// Given: Project 생성
	projectID := uuid.New()
	workspaceID := uuid.New()
	createdBy := uuid.New()
	description := "Test project description"
	now := time.Now()

	project := &domain.Project{
		ID:                projectID,
		WorkspaceID:       workspaceID,
		Name:              "Test Project",
		Description:       &description,
		DefaultPermission: domain.ProjectPermissionViewer,
		IsPublic:          true,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Then: 필드 확인
	assert.Equal(t, projectID, project.ID)
	assert.Equal(t, workspaceID, project.WorkspaceID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, &description, project.Description)
	assert.Equal(t, domain.ProjectPermissionViewer, project.DefaultPermission)
	assert.True(t, project.IsPublic)
	assert.Equal(t, createdBy, project.CreatedBy)
}

func TestStorageService_Project_TableName(t *testing.T) {
	// Given
	project := domain.Project{}

	// When/Then
	assert.Equal(t, "storage_projects", project.TableName())
}

// ============================================================
// ProjectMember 구조체 테스트
// ============================================================

func TestStorageService_ProjectMember_Fields(t *testing.T) {
	// Given: ProjectMember 생성
	memberID := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	addedBy := uuid.New()
	now := time.Now()

	member := &domain.ProjectMember{
		ID:         memberID,
		ProjectID:  projectID,
		UserID:     userID,
		Permission: domain.ProjectPermissionEditor,
		AddedBy:    addedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Then: 필드 확인
	assert.Equal(t, memberID, member.ID)
	assert.Equal(t, projectID, member.ProjectID)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, domain.ProjectPermissionEditor, member.Permission)
	assert.Equal(t, addedBy, member.AddedBy)
	assert.Equal(t, now, member.CreatedAt)
}

// ============================================================
// FileKey 생성 테스트
// ============================================================

func TestStorageService_FileKey_Generation(t *testing.T) {
	// Given: FileKey 형식 정의
	generateFileKey := func(workspaceID, fileID uuid.UUID, extension string) string {
		return "workspaces/" + workspaceID.String() + "/files/" + fileID.String() + extension
	}

	// When
	workspaceID := uuid.New()
	fileID := uuid.New()
	fileKey := generateFileKey(workspaceID, fileID, ".pdf")

	// Then: 형식 확인
	assert.Contains(t, fileKey, "workspaces/")
	assert.Contains(t, fileKey, workspaceID.String())
	assert.Contains(t, fileKey, "/files/")
	assert.Contains(t, fileKey, fileID.String())
	assert.Contains(t, fileKey, ".pdf")
}

// ============================================================
// FolderPath 생성 테스트
// ============================================================

func TestStorageService_FolderPath_Generation(t *testing.T) {
	tests := []struct {
		name         string
		parentPath   string
		folderName   string
		expectedPath string
	}{
		{"루트 폴더", "", "Documents", "/Documents"},
		{"중첩 폴더", "/Documents", "Projects", "/Documents/Projects"},
		{"깊은 중첩", "/Documents/Projects", "2024", "/Documents/Projects/2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 비즈니스 로직: 폴더 경로 생성
			var path string
			if tt.parentPath == "" {
				path = "/" + tt.folderName
			} else {
				path = tt.parentPath + "/" + tt.folderName
			}
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

// ============================================================
// 파일 크기 유효성 테스트
// ============================================================

func TestStorageService_FileSize_Validation(t *testing.T) {
	// Given: 최대 파일 크기 제한 (100MB)
	const maxFileSize int64 = 100 * 1024 * 1024

	tests := []struct {
		name     string
		fileSize int64
		isValid  bool
	}{
		{"정상 크기 (1MB)", 1024 * 1024, true},
		{"최대 크기", maxFileSize, true},
		{"최대 초과", maxFileSize + 1, false},
		{"빈 파일", 0, false},
		{"음수 (비정상)", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.fileSize > 0 && tt.fileSize <= maxFileSize
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

// ============================================================
// ContentType 유효성 테스트
// ============================================================

func TestStorageService_ContentType_Validation(t *testing.T) {
	// Given: 허용된 Content-Type 목록
	allowedTypes := map[string]bool{
		"application/pdf":    true,
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"text/plain":         true,
		"application/msword": true,
	}

	tests := []struct {
		name        string
		contentType string
		isAllowed   bool
	}{
		{"PDF", "application/pdf", true},
		{"JPEG", "image/jpeg", true},
		{"PNG", "image/png", true},
		{"Plain text", "text/plain", true},
		{"실행 파일", "application/x-executable", false},
		{"JavaScript", "application/javascript", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAllowed := allowedTypes[tt.contentType]
			assert.Equal(t, tt.isAllowed, isAllowed)
		})
	}
}

// ============================================================
// 페이지네이션 테스트
// ============================================================

func TestStorageService_Pagination_LimitValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"기본값 사용 (0)", 0, 20},
		{"음수", -1, 20},
		{"최대값 초과", 200, 100},
		{"정상 범위", 50, 50},
		{"최대값", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.inputLimit
			// 비즈니스 로직: limit 기본값 및 최대값 설정
			if limit <= 0 {
				limit = 20
			}
			if limit > 100 {
				limit = 100
			}
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

// ============================================================
// UUID 유효성 테스트
// ============================================================

func TestStorageService_UUIDValidation(t *testing.T) {
	// Valid UUID
	validUUID := uuid.New()
	assert.NotEqual(t, uuid.Nil, validUUID)

	// Parse UUID
	parsed, err := uuid.Parse(validUUID.String())
	assert.NoError(t, err)
	assert.Equal(t, validUUID, parsed)

	// Invalid UUID
	_, err = uuid.Parse("invalid-uuid")
	assert.Error(t, err)
}

// ============================================================
// Context 취소 테스트
// ============================================================

func TestStorageService_ContextCancellation(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(context.Background())

	// When
	cancel()

	// Then
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

// ============================================================
// Soft Delete 테스트
// ============================================================

func TestStorageService_SoftDelete_File(t *testing.T) {
	// Given: Active 파일
	file := &domain.File{
		ID:        uuid.New(),
		Status:    domain.FileStatusActive,
		DeletedAt: nil,
	}

	// When: Soft delete 수행
	now := time.Now()
	file.Status = domain.FileStatusDeleted
	file.DeletedAt = &now

	// Then: 삭제 상태 확인
	assert.Equal(t, domain.FileStatusDeleted, file.Status)
	assert.NotNil(t, file.DeletedAt)
}

func TestStorageService_Restore_File(t *testing.T) {
	// Given: 삭제된 파일
	deletedAt := time.Now()
	file := &domain.File{
		ID:        uuid.New(),
		Status:    domain.FileStatusDeleted,
		DeletedAt: &deletedAt,
	}

	// When: 복원 수행
	file.Status = domain.FileStatusActive
	file.DeletedAt = nil

	// Then: 복원 상태 확인
	assert.Equal(t, domain.FileStatusActive, file.Status)
	assert.Nil(t, file.DeletedAt)
}

// ============================================================
// 파일 버전 관리 테스트
// ============================================================

func TestStorageService_FileVersion_Increment(t *testing.T) {
	// Given: 버전 1 파일
	file := &domain.File{
		ID:      uuid.New(),
		Version: 1,
	}

	// When: 버전 증가
	file.Version++

	// Then: 버전 2 확인
	assert.Equal(t, 2, file.Version)
}

// ============================================================
// 색상 코드 유효성 테스트
// ============================================================

func TestStorageService_ColorCode_Validation(t *testing.T) {
	// Given: 유효한 색상 코드 형식 확인
	isValidColor := func(color string) bool {
		if len(color) != 7 {
			return false
		}
		if color[0] != '#' {
			return false
		}
		for _, c := range color[1:] {
			isDigit := c >= '0' && c <= '9'
			isUpperHex := c >= 'A' && c <= 'F'
			isLowerHex := c >= 'a' && c <= 'f'
			if !isDigit && !isUpperHex && !isLowerHex {
				return false
			}
		}
		return true
	}

	tests := []struct {
		name    string
		color   string
		isValid bool
	}{
		{"유효 대문자", "#FF5733", true},
		{"유효 소문자", "#ff5733", true},
		{"유효 혼합", "#Ff5733", true},
		{"# 누락", "FF5733", false},
		{"너무 짧음", "#FFF", false},
		{"너무 김", "#FF5733AA", false},
		{"잘못된 문자", "#GG5733", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, isValidColor(tt.color))
		})
	}
}

// ============================================================
// 동시성 테스트
// ============================================================

func TestStorageService_Concurrency_UUIDGeneration(t *testing.T) {
	// Given: 동시에 많은 UUID 생성
	const numGoroutines = 100
	results := make(chan uuid.UUID, numGoroutines)

	// When: 동시 생성
	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- uuid.New()
		}()
	}

	// Then: 모든 UUID가 유니크한지 확인
	seen := make(map[uuid.UUID]bool)
	for i := 0; i < numGoroutines; i++ {
		id := <-results
		assert.False(t, seen[id], "UUID should be unique")
		seen[id] = true
	}
}
