package service

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"user-service/internal/domain"
	"user-service/internal/metrics"
	"user-service/internal/repository"
	"user-service/internal/response"
)

// ProfileService handles user profile business logic
// 사용자 프로필 생성, 조회, 수정, 삭제 등의 비즈니스 로직을 처리합니다.
// 메트릭과 로깅을 통해 모니터링을 지원합니다.
type ProfileService struct {
	profileRepo *repository.UserProfileRepository
	memberRepo  *repository.WorkspaceMemberRepository
	userRepo    *repository.UserRepository
	logger      *zap.Logger
	metrics     *metrics.Metrics // 메트릭 수집을 위한 필드
}

// NewProfileService creates a new ProfileService
// metrics 파라미터가 nil인 경우에도 안전하게 동작합니다.
func NewProfileService(
	profileRepo *repository.UserProfileRepository,
	memberRepo *repository.WorkspaceMemberRepository,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
	m *metrics.Metrics,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		logger:      logger,
		metrics:     m,
	}
}

// CreateProfile creates a new user profile
func (s *ProfileService) CreateProfile(userID uuid.UUID, req domain.CreateProfileRequest) (*domain.UserProfile, error) {
	// Check if user is a member of the workspace
	isMember, err := s.memberRepo.IsMember(req.WorkspaceID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, response.NewForbiddenError("User is not a member of this workspace", "")
	}

	// Check if profile already exists
	existing, _ := s.profileRepo.FindByUserAndWorkspace(userID, req.WorkspaceID)
	if existing != nil {
		return nil, response.NewAlreadyExistsError("Profile already exists for this workspace", "")
	}

	profile := &domain.UserProfile{
		ID:              uuid.New(),
		UserID:          userID,
		WorkspaceID:     req.WorkspaceID,
		NickName:        req.NickName,
		Email:           req.Email,
		ProfileImageURL: req.ProfileImageURL,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.profileRepo.Create(profile); err != nil {
		s.logger.Error("Failed to create profile", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: 프로필 생성 성공
	if s.metrics != nil {
		s.metrics.RecordProfileCreated()
	}

	s.logger.Info("Profile created", zap.String("profileId", profile.ID.String()))
	return profile, nil
}

// GetMyProfile gets user's profile for a workspace
func (s *ProfileService) GetMyProfile(userID, workspaceID uuid.UUID) (*domain.UserProfile, error) {
	return s.profileRepo.FindByUserAndWorkspace(userID, workspaceID)
}

// GetAllMyProfiles gets all profiles for a user
func (s *ProfileService) GetAllMyProfiles(userID uuid.UUID) ([]domain.UserProfile, error) {
	return s.profileRepo.FindByUser(userID)
}

// GetUserProfile gets another user's profile in a workspace
func (s *ProfileService) GetUserProfile(viewerID, targetUserID, workspaceID uuid.UUID) (*domain.UserProfile, error) {
	// Check if viewer is a member of the workspace
	isMember, err := s.memberRepo.IsMember(workspaceID, viewerID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, response.NewForbiddenError("Viewer is not a member of this workspace", "")
	}

	return s.profileRepo.FindByUserAndWorkspace(targetUserID, workspaceID)
}

// UpdateProfile updates a user profile
func (s *ProfileService) UpdateProfile(userID, workspaceID uuid.UUID, req domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	profile, err := s.profileRepo.FindByUserAndWorkspace(userID, workspaceID)
	if err != nil {
		return nil, err
	}

	if req.NickName != nil {
		profile.NickName = *req.NickName
	}
	if req.ProfileImageURL != nil {
		profile.ProfileImageURL = req.ProfileImageURL
	}
	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(profile); err != nil {
		s.logger.Error("Failed to update profile", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Profile updated", zap.String("profileId", profile.ID.String()))
	return profile, nil
}

// DeleteProfile deletes a user profile
func (s *ProfileService) DeleteProfile(userID, workspaceID uuid.UUID) error {
	if err := s.profileRepo.DeleteByUserAndWorkspace(userID, workspaceID); err != nil {
		s.logger.Error("Failed to delete profile", zap.Error(err))
		return err
	}
	s.logger.Info("Profile deleted", zap.String("userId", userID.String()), zap.String("workspaceId", workspaceID.String()))
	return nil
}

// UpdateProfileImage updates user's profile image
func (s *ProfileService) UpdateProfileImage(userID, workspaceID uuid.UUID, imageURL string) (*domain.UserProfile, error) {
	profile, err := s.profileRepo.FindByUserAndWorkspace(userID, workspaceID)
	if err != nil {
		return nil, err
	}

	profile.ProfileImageURL = &imageURL
	profile.UpdatedAt = time.Now()

	if err := s.profileRepo.Update(profile); err != nil {
		s.logger.Error("Failed to update profile image", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Profile image updated", zap.String("profileId", profile.ID.String()))
	return profile, nil
}

// GetOrCreateProfile gets or creates a user profile for a workspace
func (s *ProfileService) GetOrCreateProfile(userID, workspaceID uuid.UUID, defaultNickName string) (*domain.UserProfile, error) {
	// Handle default/nil workspace ID - find user's actual workspace
	nilUUID := uuid.UUID{}
	if workspaceID == nilUUID {
		s.logger.Info("Default workspace ID detected, finding user's actual workspace",
			zap.String("userId", userID.String()))

		// Try to find user's default workspace first
		defaultMember, err := s.memberRepo.FindDefaultWorkspace(userID)
		if err == nil && defaultMember != nil {
			workspaceID = defaultMember.WorkspaceID
			s.logger.Info("Using user's default workspace",
				zap.String("workspaceId", workspaceID.String()))
		} else {
			// Fallback: get any workspace the user belongs to
			members, err := s.memberRepo.FindByUser(userID)
			if err != nil || len(members) == 0 {
				s.logger.Error("User has no workspaces", zap.String("userId", userID.String()))
				return nil, response.NewNotFoundError("User has no workspaces", userID.String())
			}
			workspaceID = members[0].WorkspaceID
			s.logger.Info("Using first available workspace",
				zap.String("workspaceId", workspaceID.String()))
		}
	}

	// Try to find existing profile
	profile, err := s.profileRepo.FindByUserAndWorkspace(userID, workspaceID)
	if err == nil && profile != nil {
		return profile, nil
	}

	// Profile not found, create a new one
	s.logger.Info("Profile not found, creating new one",
		zap.String("userId", userID.String()),
		zap.String("workspaceId", workspaceID.String()))

	// Get user's email from user table
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		s.logger.Error("Failed to find user", zap.Error(err))
		return nil, err
	}

	newProfile := &domain.UserProfile{
		ID:          uuid.New(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		NickName:    defaultNickName,
		Email:       user.Email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.profileRepo.Create(newProfile); err != nil {
		s.logger.Error("Failed to create profile", zap.Error(err))
		return nil, err
	}

	// 메트릭 기록: 프로필 자동 생성 성공
	if s.metrics != nil {
		s.metrics.RecordProfileCreated()
	}

	s.logger.Info("Profile created", zap.String("profileId", newProfile.ID.String()))
	return newProfile, nil
}
