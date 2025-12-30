package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"

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

// log returns a trace-context aware logger
func (s *UserService) log(ctx context.Context) *zap.Logger {
	return commnotel.WithTraceContext(ctx, s.logger)
}

// CreateUser creates a new user
// 이메일이 이미 존재하면 AlreadyExistsError를 반환합니다.
func (s *UserService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("CreateUser service started", zap.String("user.email", req.Email))

	// 이미 존재하는 사용자인지 확인
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		log.Debug("CreateUser user already exists", zap.String("user.email", req.Email))
		return nil, response.NewAlreadyExistsError("User with this email already exists", req.Email)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("CreateUser failed to check existing user", zap.Error(err))
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
		log.Error("CreateUser failed to create user", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: 사용자 생성 성공
	if s.metrics != nil {
		s.metrics.RecordUserCreated()
	}

	log.Info("User created", zap.String("enduser.id", user.ID.String()))
	return user, nil
}

// GetUser gets a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("GetUser service started", zap.String("enduser.id", id.String()))

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		log.Debug("GetUser user not found", zap.String("enduser.id", id.String()))
		return nil, err
	}

	log.Debug("GetUser completed", zap.String("enduser.id", id.String()))
	return user, nil
}

// GetUserByEmail gets a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("GetUserByEmail service started", zap.String("user.email", email))
	return s.userRepo.FindByEmail(email)
}

// GetUserByGoogleID gets a user by Google ID
func (s *UserService) GetUserByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("GetUserByGoogleID service started")
	return s.userRepo.FindByGoogleID(googleID)
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("UpdateUser service started", zap.String("enduser.id", id.String()))

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		log.Debug("UpdateUser user not found", zap.String("enduser.id", id.String()))
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
		log.Error("UpdateUser failed to update user", zap.Error(err))
		return nil, err
	}

	log.Info("User updated", zap.String("enduser.id", user.ID.String()))
	return user, nil
}

// DeleteUser soft deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	log := s.log(ctx)
	log.Debug("DeleteUser service started", zap.String("enduser.id", id.String()))

	if err := s.userRepo.SoftDelete(id); err != nil {
		log.Error("DeleteUser failed to delete user", zap.Error(err))
		return err
	}

	log.Info("User deleted", zap.String("enduser.id", id.String()))
	return nil
}

// RestoreUser restores a soft deleted user
func (s *UserService) RestoreUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("RestoreUser service started", zap.String("enduser.id", id.String()))

	if err := s.userRepo.Restore(id); err != nil {
		log.Error("RestoreUser failed to restore user", zap.Error(err))
		return nil, err
	}

	log.Info("User restored", zap.String("enduser.id", id.String()))
	return s.userRepo.FindByID(id)
}

// UserExists checks if a user exists
func (s *UserService) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	log := s.log(ctx)
	log.Debug("UserExists service started", zap.String("enduser.id", id.String()))
	return s.userRepo.Exists(id)
}

// FindOrCreateUser finds or creates a user (for OAuth)
func (s *UserService) FindOrCreateUser(ctx context.Context, email string, googleID *string) (*domain.User, error) {
	log := s.log(ctx)
	log.Debug("FindOrCreateUser service started", zap.String("user.email", email))

	// Try to find by Google ID first
	if googleID != nil {
		user, err := s.userRepo.FindByGoogleID(*googleID)
		if err == nil {
			log.Debug("FindOrCreateUser found by Google ID", zap.String("enduser.id", user.ID.String()))
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
				log.Error("FindOrCreateUser failed to update user with Google ID", zap.Error(err))
			}
		}
		log.Debug("FindOrCreateUser found by email", zap.String("enduser.id", user.ID.String()))
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new user
	log.Debug("FindOrCreateUser creating new user", zap.String("user.email", email))
	return s.CreateUser(ctx, domain.CreateUserRequest{
		Email:    email,
		GoogleID: googleID,
		Provider: "google",
	})
}

// FindOrCreateOAuthUser finds or creates a user for OAuth login (called by auth-service)
func (s *UserService) FindOrCreateOAuthUser(ctx context.Context, email, name, provider string) (*domain.User, error) {
	log := s.log(ctx)
	log.Info("OAuth login attempt",
		zap.String("user.email", email),
		zap.String("oauth.provider", provider))

	// Try to find by email
	user, err := s.userRepo.FindByEmail(email)
	if err == nil {
		// Update name if it was empty (for existing users without name)
		if user.Name == "" && name != "" {
			user.Name = name
			user.UpdatedAt = time.Now()
			if err := s.userRepo.Update(user); err != nil {
				log.Error("FindOrCreateOAuthUser failed to update user name", zap.Error(err))
			}
		}
		log.Info("Existing user found for OAuth", zap.String("enduser.id", user.ID.String()))
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("FindOrCreateOAuthUser failed to find user by email", zap.Error(err))
		return nil, err
	}

	// Create new user
	log.Debug("FindOrCreateOAuthUser creating new OAuth user", zap.String("user.email", email))
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
		log.Error("FindOrCreateOAuthUser failed to create OAuth user", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: OAuth 사용자 생성 성공
	if s.metrics != nil {
		s.metrics.RecordUserCreated()
	}

	log.Info("OAuth user created",
		zap.String("enduser.id", newUser.ID.String()),
		zap.String("user.email", email))
	return newUser, nil
}
