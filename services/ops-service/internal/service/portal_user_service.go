package service

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ops-service/internal/domain"
	"ops-service/internal/repository"
	"ops-service/internal/response"
)

// PortalUserService handles portal user business logic
type PortalUserService struct {
	userRepo *repository.PortalUserRepository
	auditSvc *AuditLogService
	logger   *zap.Logger
}

// NewPortalUserService creates a new portal user service
func NewPortalUserService(
	userRepo *repository.PortalUserRepository,
	auditSvc *AuditLogService,
	logger *zap.Logger,
) *PortalUserService {
	return &PortalUserService{
		userRepo: userRepo,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

// GetByEmail gets a portal user by email (implements PortalUserGetter interface)
func (s *PortalUserService) GetByEmail(email string) (*domain.PortalUser, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetOrCreateFromOAuth gets or creates a portal user from OAuth login
func (s *PortalUserService) GetOrCreateFromOAuth(email, name, picture string) (*domain.PortalUser, bool, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err == nil {
		// Update last login
		now := time.Now()
		user.LastLoginAt = &now
		if name != "" && user.Name == "" {
			user.Name = name
		}
		if picture != "" {
			user.Picture = picture
		}
		if err := s.userRepo.Update(user); err != nil {
			s.logger.Warn("Failed to update user last login", zap.Error(err))
		}
		return user, false, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	// Create new user with viewer role by default
	now := time.Now()
	user = &domain.PortalUser{
		Email:       email,
		Name:        name,
		Picture:     picture,
		Role:        domain.RoleViewer,
		IsActive:    true,
		LastLoginAt: &now,
	}

	// First user becomes admin
	count, err := s.userRepo.CountByRole(domain.RoleAdmin)
	if err == nil && count == 0 {
		user.Role = domain.RoleAdmin
		s.logger.Info("First portal user, granting admin role", zap.String("email", email))
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, false, err
	}

	s.logger.Info("Created new portal user",
		zap.String("email", email),
		zap.String("role", string(user.Role)),
	)

	return user, true, nil
}

// GetByID gets a portal user by ID
func (s *PortalUserService) GetByID(id uuid.UUID) (*domain.PortalUser, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetAll gets all portal users
func (s *PortalUserService) GetAll() ([]domain.PortalUser, error) {
	return s.userRepo.GetAll()
}

// InviteUser invites a new user to the portal
func (s *PortalUserService) InviteUser(inviterID uuid.UUID, inviterEmail string, req domain.InviteUserRequest) (*domain.PortalUser, error) {
	if !req.Role.IsValid() {
		return nil, response.ErrInvalidRole
	}

	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, response.ErrUserAlreadyExists
	}

	user := &domain.PortalUser{
		Email:     req.Email,
		Name:      "",
		Role:      req.Role,
		IsActive:  true,
		InvitedBy: &inviterID,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Log audit
	s.auditSvc.Log(inviterID, inviterEmail, domain.ActionCreate, domain.ResourceUser, user.ID.String(), "Invited user: "+req.Email)

	s.logger.Info("User invited",
		zap.String("email", req.Email),
		zap.String("role", string(req.Role)),
		zap.String("invitedBy", inviterEmail),
	)

	return user, nil
}

// UpdateRole updates a user's role
func (s *PortalUserService) UpdateRole(currentUserID uuid.UUID, currentUserEmail string, targetUserID uuid.UUID, role domain.Role) error {
	if currentUserID == targetUserID {
		return response.ErrSelfModification
	}

	if !role.IsValid() {
		return response.ErrInvalidRole
	}

	user, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.ErrUserNotFound
		}
		return err
	}

	// If demoting from admin, check if it's the last admin
	if user.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		count, err := s.userRepo.CountByRole(domain.RoleAdmin)
		if err != nil {
			return err
		}
		if count <= 1 {
			return response.ErrLastAdmin
		}
	}

	oldRole := user.Role
	user.Role = role

	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// Log audit
	s.auditSvc.Log(currentUserID, currentUserEmail, domain.ActionUpdate, domain.ResourceUser, user.ID.String(),
		"Changed role from "+string(oldRole)+" to "+string(role))

	s.logger.Info("User role updated",
		zap.String("email", user.Email),
		zap.String("oldRole", string(oldRole)),
		zap.String("newRole", string(role)),
		zap.String("updatedBy", currentUserEmail),
	)

	return nil
}

// DeactivateUser deactivates a user
func (s *PortalUserService) DeactivateUser(currentUserID uuid.UUID, currentUserEmail string, targetUserID uuid.UUID) error {
	if currentUserID == targetUserID {
		return response.ErrSelfModification
	}

	user, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.ErrUserNotFound
		}
		return err
	}

	// Check if it's the last admin
	if user.Role == domain.RoleAdmin {
		count, err := s.userRepo.CountByRole(domain.RoleAdmin)
		if err != nil {
			return err
		}
		if count <= 1 {
			return response.ErrLastAdmin
		}
	}

	user.IsActive = false

	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// Log audit
	s.auditSvc.Log(currentUserID, currentUserEmail, domain.ActionUpdate, domain.ResourceUser, user.ID.String(), "Deactivated user")

	s.logger.Info("User deactivated",
		zap.String("email", user.Email),
		zap.String("deactivatedBy", currentUserEmail),
	)

	return nil
}

// ReactivateUser reactivates a user
func (s *PortalUserService) ReactivateUser(currentUserID uuid.UUID, currentUserEmail string, targetUserID uuid.UUID) error {
	user, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.ErrUserNotFound
		}
		return err
	}

	user.IsActive = true

	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// Log audit
	s.auditSvc.Log(currentUserID, currentUserEmail, domain.ActionUpdate, domain.ResourceUser, user.ID.String(), "Reactivated user")

	s.logger.Info("User reactivated",
		zap.String("email", user.Email),
		zap.String("reactivatedBy", currentUserEmail),
	)

	return nil
}
