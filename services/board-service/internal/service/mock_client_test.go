package service

import (
	"context"
	"io"

	"github.com/google/uuid"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
)

// MockFieldOptionConverter is a mock implementation of FieldOptionConverter
type MockFieldOptionConverter struct {
	ConvertValuesToIDsFunc      func(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesFunc      func(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error)
	ConvertIDsToValuesBatchFunc func(ctx context.Context, boards []*domain.Board) error
}

func (m *MockFieldOptionConverter) ConvertValuesToIDs(ctx context.Context, projectID uuid.UUID, customFields map[string]interface{}) (map[string]interface{}, error) {
	if m.ConvertValuesToIDsFunc != nil {
		return m.ConvertValuesToIDsFunc(ctx, projectID, customFields)
	}
	// Default: return as-is (no conversion)
	return customFields, nil
}

func (m *MockFieldOptionConverter) ConvertIDsToValues(ctx context.Context, customFields map[string]interface{}) (map[string]interface{}, error) {
	if m.ConvertIDsToValuesFunc != nil {
		return m.ConvertIDsToValuesFunc(ctx, customFields)
	}
	// Default: return as-is (no conversion)
	return customFields, nil
}

func (m *MockFieldOptionConverter) ConvertIDsToValuesBatch(ctx context.Context, boards []*domain.Board) error {
	if m.ConvertIDsToValuesBatchFunc != nil {
		return m.ConvertIDsToValuesBatchFunc(ctx, boards)
	}
	// Default: no-op
	return nil
}

// MockUserClient is a mock implementation of UserClient
type MockUserClient struct {
	ValidateWorkspaceMemberFunc func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
	GetUserProfileFunc          func(ctx context.Context, userID uuid.UUID, token string) (*client.UserProfile, error)
	GetWorkspaceProfileFunc     func(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*client.WorkspaceProfile, error)
	GetWorkspaceFunc            func(ctx context.Context, workspaceID uuid.UUID, token string) (*client.Workspace, error)
	ValidateTokenFunc           func(ctx context.Context, token string) (uuid.UUID, error)
}

func (m *MockUserClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	if m.ValidateWorkspaceMemberFunc != nil {
		return m.ValidateWorkspaceMemberFunc(ctx, workspaceID, userID, token)
	}
	return true, nil
}

func (m *MockUserClient) GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*client.UserProfile, error) {
	if m.GetUserProfileFunc != nil {
		return m.GetUserProfileFunc(ctx, userID, token)
	}
	return &client.UserProfile{UserID: userID, Email: "test@example.com"}, nil
}

func (m *MockUserClient) GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*client.WorkspaceProfile, error) {
	if m.GetWorkspaceProfileFunc != nil {
		return m.GetWorkspaceProfileFunc(ctx, workspaceID, userID, token)
	}
	return &client.WorkspaceProfile{
		WorkspaceID: workspaceID,
		UserID:      userID,
		NickName:    "Test User",
		Email:       "test@example.com",
	}, nil
}

func (m *MockUserClient) GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*client.Workspace, error) {
	if m.GetWorkspaceFunc != nil {
		return m.GetWorkspaceFunc(ctx, workspaceID, token)
	}
	return &client.Workspace{
		ID:         workspaceID,
		Name:       "Test Workspace",
		OwnerEmail: "workspace@example.com",
	}, nil
}

func (m *MockUserClient) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, token)
	}
	return uuid.New(), nil
}

// MockS3Client is a mock implementation of S3Client
type MockS3Client struct {
	GenerateFileKeyFunc      func(entityType, workspaceID, fileExt string) (string, error)
	GeneratePresignedURLFunc func(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error)
	UploadFileFunc           func(ctx context.Context, key string, file io.Reader, contentType string) (string, error)
	DeleteFileFunc           func(ctx context.Context, key string) error
	GetFileURLFunc           func(key string) string
}

func (m *MockS3Client) GenerateFileKey(entityType, workspaceID, fileExt string) (string, error) {
	if m.GenerateFileKeyFunc != nil {
		return m.GenerateFileKeyFunc(entityType, workspaceID, fileExt)
	}
	return "mock-file-key" + fileExt, nil
}

func (m *MockS3Client) GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error) {
	if m.GeneratePresignedURLFunc != nil {
		return m.GeneratePresignedURLFunc(ctx, entityType, workspaceID, fileName, contentType)
	}
	return "https://mock-presigned-url.com", "mock-file-key", nil
}

func (m *MockS3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, key, file, contentType)
	}
	return "https://mock-s3-url.com/" + key, nil
}

func (m *MockS3Client) DeleteFile(ctx context.Context, key string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, key)
	}
	return nil
}

func (m *MockS3Client) GetFileURL(key string) string {
	if m.GetFileURLFunc != nil {
		return m.GetFileURLFunc(key)
	}
	return "https://mock-s3-url.com/" + key
}
