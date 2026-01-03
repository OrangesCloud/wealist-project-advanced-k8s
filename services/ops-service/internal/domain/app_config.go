package domain

import (
	"time"

	"github.com/google/uuid"
)

// AppConfig represents an application configuration (Remote Config)
type AppConfig struct {
	BaseModel
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	UpdatedBy   uuid.UUID `gorm:"type:uuid" json:"updatedBy"`
}

// TableName returns the table name for GORM
func (AppConfig) TableName() string {
	return "app_configs"
}

// AppConfigResponse is the response DTO for app config
type AppConfigResponse struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"isActive"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ToResponse converts AppConfig to AppConfigResponse
func (c *AppConfig) ToResponse() AppConfigResponse {
	return AppConfigResponse{
		ID:          c.ID,
		Key:         c.Key,
		Value:       c.Value,
		Description: c.Description,
		IsActive:    c.IsActive,
		UpdatedAt:   c.UpdatedAt,
	}
}

// CreateAppConfigRequest is the request DTO for creating an app config
type CreateAppConfigRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
}

// UpdateAppConfigRequest is the request DTO for updating an app config
type UpdateAppConfigRequest struct {
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
	IsActive    *bool  `json:"isActive"`
}
