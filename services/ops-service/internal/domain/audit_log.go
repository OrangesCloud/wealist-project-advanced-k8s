package domain

import (
	"time"

	"github.com/google/uuid"
)

// ActionType represents the type of action performed
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionLogin  ActionType = "login"
	ActionLogout ActionType = "logout"
)

// ResourceType represents the type of resource affected
type ResourceType string

const (
	ResourceUser        ResourceType = "portal_user"
	ResourceArgoCD      ResourceType = "argocd_rbac"
	ResourceFeatureFlag ResourceType = "feature_flag"
	ResourceAppConfig   ResourceType = "app_config"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID    `gorm:"type:uuid;not null;index" json:"userId"`
	UserEmail    string       `gorm:"not null" json:"userEmail"`
	Action       ActionType   `gorm:"type:varchar(20);not null" json:"action"`
	ResourceType ResourceType `gorm:"type:varchar(50);not null" json:"resourceType"`
	ResourceID   string       `gorm:"not null" json:"resourceId"`
	Details      string       `gorm:"type:text" json:"details,omitempty"`
	IPAddress    string       `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`
	UserAgent    string       `gorm:"type:text" json:"userAgent,omitempty"`
	CreatedAt    time.Time    `gorm:"autoCreateTime;index" json:"createdAt"`
}

// TableName returns the table name for GORM
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogResponse is the response DTO for audit log
type AuditLogResponse struct {
	ID           uuid.UUID    `json:"id"`
	UserID       uuid.UUID    `json:"userId"`
	UserEmail    string       `json:"userEmail"`
	Action       ActionType   `json:"action"`
	ResourceType ResourceType `json:"resourceType"`
	ResourceID   string       `json:"resourceId"`
	Details      string       `json:"details,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
}

// ToResponse converts AuditLog to AuditLogResponse
func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		UserEmail:    a.UserEmail,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		Details:      a.Details,
		CreatedAt:    a.CreatedAt,
	}
}
