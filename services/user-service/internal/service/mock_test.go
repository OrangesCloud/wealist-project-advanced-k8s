package service

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"user-service/internal/domain"
)

// MockUserRepository is a mock implementation of user repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByIDIncludeDeleted(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByGoogleID(googleID string) (*domain.User, error) {
	args := m.Called(googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) SoftDelete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Restore(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Exists(id uuid.UUID) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

// MockWorkspaceRepository is a mock implementation of workspace repository
type MockWorkspaceRepository struct {
	mock.Mock
}

func (m *MockWorkspaceRepository) Create(workspace *domain.Workspace) error {
	args := m.Called(workspace)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) FindByID(id uuid.UUID) (*domain.Workspace, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workspace), args.Error(1)
}

func (m *MockWorkspaceRepository) FindByOwnerID(ownerID uuid.UUID) ([]domain.Workspace, error) {
	args := m.Called(ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Workspace), args.Error(1)
}

func (m *MockWorkspaceRepository) Update(workspace *domain.Workspace) error {
	args := m.Called(workspace)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) SearchByName(name string) ([]domain.Workspace, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Workspace), args.Error(1)
}

// MockWorkspaceMemberRepository is a mock implementation of workspace member repository
type MockWorkspaceMemberRepository struct {
	mock.Mock
}

func (m *MockWorkspaceMemberRepository) Create(member *domain.WorkspaceMember) error {
	args := m.Called(member)
	return args.Error(0)
}

func (m *MockWorkspaceMemberRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]domain.WorkspaceMember, error) {
	args := m.Called(workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberRepository) FindByUserID(userID uuid.UUID) ([]domain.WorkspaceMember, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberRepository) FindByWorkspaceAndUser(workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	args := m.Called(workspaceID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberRepository) Update(member *domain.WorkspaceMember) error {
	args := m.Called(member)
	return args.Error(0)
}

func (m *MockWorkspaceMemberRepository) Delete(workspaceID, memberID uuid.UUID) error {
	args := m.Called(workspaceID, memberID)
	return args.Error(0)
}

func (m *MockWorkspaceMemberRepository) Exists(workspaceID, userID uuid.UUID) (bool, error) {
	args := m.Called(workspaceID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkspaceMemberRepository) GetDefaultWorkspace(userID uuid.UUID) (*domain.WorkspaceMember, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberRepository) FindByID(memberID uuid.UUID) (*domain.WorkspaceMember, error) {
	args := m.Called(memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceMember), args.Error(1)
}

// MockUserProfileRepository is a mock implementation of user profile repository
type MockUserProfileRepository struct {
	mock.Mock
}

func (m *MockUserProfileRepository) Create(profile *domain.UserProfile) error {
	args := m.Called(profile)
	return args.Error(0)
}

func (m *MockUserProfileRepository) FindByID(id uuid.UUID) (*domain.UserProfile, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserProfileRepository) FindByUserAndWorkspace(userID, workspaceID uuid.UUID) (*domain.UserProfile, error) {
	args := m.Called(userID, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserProfile), args.Error(1)
}

func (m *MockUserProfileRepository) FindByUserID(userID uuid.UUID) ([]domain.UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.UserProfile), args.Error(1)
}

func (m *MockUserProfileRepository) Update(profile *domain.UserProfile) error {
	args := m.Called(profile)
	return args.Error(0)
}

func (m *MockUserProfileRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserProfileRepository) DeleteByUserAndWorkspace(userID, workspaceID uuid.UUID) error {
	args := m.Called(userID, workspaceID)
	return args.Error(0)
}

// MockJoinRequestRepository is a mock implementation of join request repository
type MockJoinRequestRepository struct {
	mock.Mock
}

func (m *MockJoinRequestRepository) Create(req *domain.WorkspaceJoinRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockJoinRequestRepository) FindByID(id uuid.UUID) (*domain.WorkspaceJoinRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceJoinRequest), args.Error(1)
}

func (m *MockJoinRequestRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]domain.WorkspaceJoinRequest, error) {
	args := m.Called(workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkspaceJoinRequest), args.Error(1)
}

func (m *MockJoinRequestRepository) Update(req *domain.WorkspaceJoinRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockJoinRequestRepository) FindPendingByUserAndWorkspace(userID, workspaceID uuid.UUID) (*domain.WorkspaceJoinRequest, error) {
	args := m.Called(userID, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceJoinRequest), args.Error(1)
}

// Helper function to create test user
func createTestUser(id uuid.UUID, email string) *domain.User {
	return &domain.User{
		ID:       id,
		Email:    email,
		Name:     "Test User",
		Provider: "google",
		IsActive: true,
	}
}

// Helper function to simulate gorm.ErrRecordNotFound
var errRecordNotFound = gorm.ErrRecordNotFound
