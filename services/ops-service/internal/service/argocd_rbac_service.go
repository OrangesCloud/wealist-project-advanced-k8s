package service

import (
	"context"
	"encoding/json"
	"ops-service/internal/client"
	"ops-service/internal/domain"
	"ops-service/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ArgoCDRBACService handles ArgoCD RBAC operations
type ArgoCDRBACService struct {
	k8sClient     *client.K8sClient
	argoCDClient  *client.ArgoCDClient
	auditLogRepo  *repository.AuditLogRepository
	argoCDNS      string
	logger        *zap.Logger
}

// ArgoCDRBACServiceConfig holds configuration for ArgoCDRBACService
type ArgoCDRBACServiceConfig struct {
	K8sClient     *client.K8sClient
	ArgoCDClient  *client.ArgoCDClient
	AuditLogRepo  *repository.AuditLogRepository
	ArgoCDNS      string // ArgoCD namespace (default: "argocd")
	Logger        *zap.Logger
}

// NewArgoCDRBACService creates a new ArgoCD RBAC service
func NewArgoCDRBACService(cfg ArgoCDRBACServiceConfig) *ArgoCDRBACService {
	ns := cfg.ArgoCDNS
	if ns == "" {
		ns = "argocd"
	}

	return &ArgoCDRBACService{
		k8sClient:    cfg.K8sClient,
		argoCDClient: cfg.ArgoCDClient,
		auditLogRepo: cfg.AuditLogRepo,
		argoCDNS:     ns,
		logger:       cfg.Logger,
	}
}

// RBACInfo holds RBAC information
type RBACInfo struct {
	Namespace    string   `json:"namespace"`
	AdminUsers   []string `json:"adminUsers"`
	PolicyCSV    string   `json:"policyCSV"`
	DefaultRole  string   `json:"defaultRole"`
	Scopes       string   `json:"scopes"`
}

// GetRBAC returns current ArgoCD RBAC configuration
func (s *ArgoCDRBACService) GetRBAC(ctx context.Context) (*RBACInfo, error) {
	config, err := s.k8sClient.GetArgoCDRBACConfigMap(ctx, s.argoCDNS)
	if err != nil {
		s.logger.Error("Failed to get ArgoCD RBAC config", zap.Error(err))
		return nil, err
	}

	return &RBACInfo{
		Namespace:   s.argoCDNS,
		AdminUsers:  config.AdminUsers,
		PolicyCSV:   config.PolicyCSV,
		DefaultRole: config.Policy,
		Scopes:      config.Scopes,
	}, nil
}

// AddAdmin adds an admin user to ArgoCD RBAC
func (s *ArgoCDRBACService) AddAdmin(ctx context.Context, email string, performedBy uuid.UUID) error {
	// Add admin to RBAC
	if err := s.k8sClient.AddArgoCDAdmin(ctx, s.argoCDNS, email); err != nil {
		s.logger.Error("Failed to add ArgoCD admin",
			zap.String("email", email),
			zap.Error(err))
		return err
	}

	// Log the action
	details, _ := json.Marshal(map[string]interface{}{"email": email, "namespace": s.argoCDNS})
	if err := s.auditLogRepo.Create(&domain.AuditLog{
		UserID:       performedBy,
		Action:       "argocd_admin_added",
		ResourceType: domain.ResourceArgoCD,
		ResourceID:   email,
		Details:      string(details),
	}); err != nil {
		s.logger.Warn("Failed to create audit log", zap.Error(err))
	}

	s.logger.Info("Added ArgoCD admin",
		zap.String("email", email),
		zap.String("performedBy", performedBy.String()))

	return nil
}

// RemoveAdmin removes an admin user from ArgoCD RBAC
func (s *ArgoCDRBACService) RemoveAdmin(ctx context.Context, email string, performedBy uuid.UUID) error {
	// Remove admin from RBAC
	if err := s.k8sClient.RemoveArgoCDAdmin(ctx, s.argoCDNS, email); err != nil {
		s.logger.Error("Failed to remove ArgoCD admin",
			zap.String("email", email),
			zap.Error(err))
		return err
	}

	// Log the action
	details, _ := json.Marshal(map[string]interface{}{"email": email, "namespace": s.argoCDNS})
	if err := s.auditLogRepo.Create(&domain.AuditLog{
		UserID:       performedBy,
		Action:       "argocd_admin_removed",
		ResourceType: domain.ResourceArgoCD,
		ResourceID:   email,
		Details:      string(details),
	}); err != nil {
		s.logger.Warn("Failed to create audit log", zap.Error(err))
	}

	s.logger.Info("Removed ArgoCD admin",
		zap.String("email", email),
		zap.String("performedBy", performedBy.String()))

	return nil
}

// GetApplications returns ArgoCD applications for monitoring
func (s *ArgoCDRBACService) GetApplications(ctx context.Context) ([]client.Application, error) {
	if s.argoCDClient == nil {
		s.logger.Warn("ArgoCD client not configured")
		return nil, nil
	}

	apps, err := s.argoCDClient.GetApplications()
	if err != nil {
		s.logger.Error("Failed to get ArgoCD applications", zap.Error(err))
		return nil, err
	}

	return apps, nil
}

// SyncApplication syncs an ArgoCD application
func (s *ArgoCDRBACService) SyncApplication(ctx context.Context, name string, performedBy uuid.UUID) error {
	if s.argoCDClient == nil {
		s.logger.Warn("ArgoCD client not configured")
		return nil
	}

	if err := s.argoCDClient.SyncApplication(name); err != nil {
		s.logger.Error("Failed to sync ArgoCD application",
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	// Log the action
	details, _ := json.Marshal(map[string]interface{}{"application": name})
	if err := s.auditLogRepo.Create(&domain.AuditLog{
		UserID:       performedBy,
		Action:       "argocd_app_synced",
		ResourceType: domain.ResourceArgoCD,
		ResourceID:   name,
		Details:      string(details),
	}); err != nil {
		s.logger.Warn("Failed to create audit log", zap.Error(err))
	}

	s.logger.Info("Synced ArgoCD application",
		zap.String("name", name),
		zap.String("performedBy", performedBy.String()))

	return nil
}
