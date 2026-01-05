package domain

import (
	"time"

	"github.com/google/uuid"
)

// PortalUser represents an admin portal user
type PortalUser struct {
	BaseModel
	Email       string     `gorm:"uniqueIndex;not null" json:"email"`
	Name        string     `gorm:"not null" json:"name"`
	Picture     string     `json:"picture,omitempty"`
	Role        Role       `gorm:"type:varchar(20);not null;default:'viewer'" json:"role"`
	IsActive    bool       `gorm:"default:true" json:"isActive"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	InvitedBy   *uuid.UUID `gorm:"type:uuid" json:"invitedBy,omitempty"`
}

// TableName returns the table name for GORM
func (PortalUser) TableName() string {
	return "portal_users"
}

// PortalUserResponse is the response DTO for portal user
type PortalUserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Picture     string     `json:"picture,omitempty"`
	Role        Role       `json:"role"`
	IsActive    bool       `json:"isActive"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ToResponse converts PortalUser to PortalUserResponse
func (u *PortalUser) ToResponse() PortalUserResponse {
	return PortalUserResponse{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Picture:     u.Picture,
		Role:        u.Role,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

// InviteUserRequest is the request DTO for inviting a user
type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  Role   `json:"role" binding:"required"`
}

// UpdateUserRoleRequest is the request DTO for updating a user's role
type UpdateUserRoleRequest struct {
	Role Role `json:"role" binding:"required"`
}
