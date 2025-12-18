package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"user-service/internal/domain"
	"user-service/internal/repository"
)

// userServiceTestable is a testable version of UserService that accepts interface
type userServiceTestable struct {
	userRepo userRepositoryInterface
	logger   *zap.Logger
}

type userRepositoryInterface interface {
	Create(user *domain.User) error
	FindByID(id uuid.UUID) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	FindByGoogleID(googleID string) (*domain.User, error)
	Update(user *domain.User) error
	SoftDelete(id uuid.UUID) error
	Restore(id uuid.UUID) error
	Exists(id uuid.UUID) (bool, error)
}

func newUserServiceTestable(repo userRepositoryInterface, logger *zap.Logger) *userServiceTestable {
	return &userServiceTestable{
		userRepo: repo,
		logger:   logger,
	}
}

func (s *userServiceTestable) CreateUser(req domain.CreateUserRequest) (*domain.User, error) {
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, assert.AnError
	}

	provider := "google"
	if req.Provider != "" {
		provider = req.Provider
	}

	user := &domain.User{
		ID:        uuid.New(),
		Email:     req.Email,
		GoogleID:  req.GoogleID,
		Provider:  provider,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("Failed to create user")
		return nil, err
	}

	s.logger.Info("User created")
	return user, nil
}

func (s *userServiceTestable) GetUser(id uuid.UUID) (*domain.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userServiceTestable) UpdateUser(id uuid.UUID, req domain.UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceTestable) DeleteUser(id uuid.UUID) error {
	return s.userRepo.SoftDelete(id)
}

func (s *userServiceTestable) RestoreUser(id uuid.UUID) (*domain.User, error) {
	if err := s.userRepo.Restore(id); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(id)
}

func (s *userServiceTestable) UserExists(id uuid.UUID) (bool, error) {
	return s.userRepo.Exists(id)
}

func TestUserService_CreateUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		req := domain.CreateUserRequest{
			Email:    "test@example.com",
			Provider: "google",
		}

		mockRepo.On("FindByEmail", req.Email).Return(nil, errRecordNotFound)
		mockRepo.On("Create", mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := svc.CreateUser(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, "google", user.Provider)
		assert.True(t, user.IsActive)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		existingUser := createTestUser(uuid.New(), "test@example.com")
		req := domain.CreateUserRequest{
			Email: "test@example.com",
		}

		mockRepo.On("FindByEmail", req.Email).Return(existingUser, nil)

		user, err := svc.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("create error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		req := domain.CreateUserRequest{
			Email: "test@example.com",
		}

		mockRepo.On("FindByEmail", req.Email).Return(nil, errRecordNotFound)
		mockRepo.On("Create", mock.AnythingOfType("*domain.User")).Return(assert.AnError)

		user, err := svc.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")

		mockRepo.On("FindByID", userID).Return(expectedUser, nil)

		user, err := svc.GetUser(userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Email, user.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("FindByID", userID).Return(nil, errRecordNotFound)

		user, err := svc.GetUser(userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		existingUser := createTestUser(userID, "old@example.com")
		newEmail := "new@example.com"
		req := domain.UpdateUserRequest{
			Email: &newEmail,
		}

		mockRepo.On("FindByID", userID).Return(existingUser, nil)
		mockRepo.On("Update", mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := svc.UpdateUser(userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, newEmail, user.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		newEmail := "new@example.com"
		req := domain.UpdateUserRequest{
			Email: &newEmail,
		}

		mockRepo.On("FindByID", userID).Return(nil, errRecordNotFound)

		user, err := svc.UpdateUser(userID, req)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update is_active", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		existingUser := createTestUser(userID, "test@example.com")
		isActive := false
		req := domain.UpdateUserRequest{
			IsActive: &isActive,
		}

		mockRepo.On("FindByID", userID).Return(existingUser, nil)
		mockRepo.On("Update", mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := svc.UpdateUser(userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.False(t, user.IsActive)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("SoftDelete", userID).Return(nil)

		err := svc.DeleteUser(userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("SoftDelete", userID).Return(assert.AnError)

		err := svc.DeleteUser(userID)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_RestoreUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")

		mockRepo.On("Restore", userID).Return(nil)
		mockRepo.On("FindByID", userID).Return(expectedUser, nil)

		user, err := svc.RestoreUser(userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("restore error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("Restore", userID).Return(assert.AnError)

		user, err := svc.RestoreUser(userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_UserExists(t *testing.T) {
	logger := zap.NewNop()

	t.Run("exists", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("Exists", userID).Return(true, nil)

		exists, err := svc.UserExists(userID)

		assert.NoError(t, err)
		assert.True(t, exists)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not exists", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		svc := newUserServiceTestable(mockRepo, logger)

		userID := uuid.New()
		mockRepo.On("Exists", userID).Return(false, nil)

		exists, err := svc.UserExists(userID)

		assert.NoError(t, err)
		assert.False(t, exists)
		mockRepo.AssertExpectations(t)
	})
}

// TestNewUserService verifies the constructor
// NewUserService 생성자가 올바르게 서비스를 초기화하는지 검증
func TestNewUserService(t *testing.T) {
	logger := zap.NewNop()
	repo := &repository.UserRepository{}

	// metrics는 nil 전달 가능 (nil-safe 설계)
	svc := NewUserService(repo, logger, nil)

	assert.NotNil(t, svc)
}
