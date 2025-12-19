package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"user-service/internal/domain"
	"user-service/internal/metrics"
	"user-service/internal/repository"
	"user-service/internal/response"
)

// UserService handles user business logic
// 사용자 생성, 조회, 수정, 삭제 등의 비즈니스 로직을 처리합니다.
// 메트릭과 로깅을 통해 모니터링을 지원합니다.
type UserService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
	metrics  *metrics.Metrics // 메트릭 수집을 위한 필드
}

// NewUserService creates a new UserService
// metrics 파라미터가 nil인 경우에도 안전하게 동작합니다.
func NewUserService(userRepo *repository.UserRepository, logger *zap.Logger, m *metrics.Metrics) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
		metrics:  m,
	}
}

// CreateUser creates a new user
// 이메일이 이미 존재하면 AlreadyExistsError를 반환합니다.
func (s *UserService) CreateUser(req domain.CreateUserRequest) (*domain.User, error) {
	// 이미 존재하는 사용자인지 확인
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, response.NewAlreadyExistsError("User with this email already exists", req.Email)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.NewInternalError("Failed to check existing user", err.Error())
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
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: 사용자 생성 성공
	if s.metrics != nil {
		s.metrics.RecordUserCreated()
	}

	s.logger.Info("User created", zap.String("userId", user.ID.String()))
	return user, nil
}

// GetUser gets a user by ID
func (s *UserService) GetUser(id uuid.UUID) (*domain.User, error) {
	return s.userRepo.FindByID(id)
}

// GetUserByEmail gets a user by email
func (s *UserService) GetUserByEmail(email string) (*domain.User, error) {
	return s.userRepo.FindByEmail(email)
}

// GetUserByGoogleID gets a user by Google ID
func (s *UserService) GetUserByGoogleID(googleID string) (*domain.User, error) {
	return s.userRepo.FindByGoogleID(googleID)
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(id uuid.UUID, req domain.UpdateUserRequest) (*domain.User, error) {
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
		s.logger.Error("Failed to update user", zap.Error(err))
		return nil, err
	}

	s.logger.Info("User updated", zap.String("userId", user.ID.String()))
	return user, nil
}

// DeleteUser soft deletes a user
func (s *UserService) DeleteUser(id uuid.UUID) error {
	if err := s.userRepo.SoftDelete(id); err != nil {
		s.logger.Error("Failed to delete user", zap.Error(err))
		return err
	}
	s.logger.Info("User deleted", zap.String("userId", id.String()))
	return nil
}

// RestoreUser restores a soft deleted user
func (s *UserService) RestoreUser(id uuid.UUID) (*domain.User, error) {
	if err := s.userRepo.Restore(id); err != nil {
		s.logger.Error("Failed to restore user", zap.Error(err))
		return nil, err
	}
	s.logger.Info("User restored", zap.String("userId", id.String()))
	return s.userRepo.FindByID(id)
}

// UserExists checks if a user exists
func (s *UserService) UserExists(id uuid.UUID) (bool, error) {
	return s.userRepo.Exists(id)
}

// FindOrCreateUser finds or creates a user (for OAuth)
func (s *UserService) FindOrCreateUser(email string, googleID *string) (*domain.User, error) {
	// Try to find by Google ID first
	if googleID != nil {
		user, err := s.userRepo.FindByGoogleID(*googleID)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Try to find by email
	user, err := s.userRepo.FindByEmail(email)
	if err == nil {
		// Update Google ID if not set
		if googleID != nil && user.GoogleID == nil {
			user.GoogleID = googleID
			user.UpdatedAt = time.Now()
			if err := s.userRepo.Update(user); err != nil {
				s.logger.Error("Failed to update user with Google ID", zap.Error(err))
			}
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new user
	return s.CreateUser(domain.CreateUserRequest{
		Email:    email,
		GoogleID: googleID,
		Provider: "google",
	})
}

// FindOrCreateOAuthUser finds or creates a user for OAuth login (called by auth-service)
func (s *UserService) FindOrCreateOAuthUser(email, name, provider string) (*domain.User, error) {
	s.logger.Info("OAuth login attempt", zap.String("email", email), zap.String("provider", provider))

	// Try to find by email
	user, err := s.userRepo.FindByEmail(email)
	if err == nil {
		// Update name if it was empty (for existing users without name)
		if user.Name == "" && name != "" {
			user.Name = name
			user.UpdatedAt = time.Now()
			if err := s.userRepo.Update(user); err != nil {
				s.logger.Error("Failed to update user name", zap.Error(err))
			}
		}
		s.logger.Info("Existing user found for OAuth", zap.String("userId", user.ID.String()))
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("Failed to find user by email", zap.Error(err))
		return nil, err
	}

	// Create new user
	newUser := &domain.User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		Provider:  provider,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(newUser); err != nil {
		s.logger.Error("Failed to create OAuth user", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: OAuth 사용자 생성 성공
	if s.metrics != nil {
		s.metrics.RecordUserCreated()
	}

	s.logger.Info("OAuth user created", zap.String("userId", newUser.ID.String()), zap.String("email", email))
	return newUser, nil
}
