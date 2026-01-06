package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	// Task events
	NotificationTypeTaskAssigned      NotificationType = "TASK_ASSIGNED"
	NotificationTypeTaskUnassigned    NotificationType = "TASK_UNASSIGNED"
	NotificationTypeTaskMentioned     NotificationType = "TASK_MENTIONED"
	NotificationTypeTaskDueSoon       NotificationType = "TASK_DUE_SOON"
	NotificationTypeTaskOverdue       NotificationType = "TASK_OVERDUE"
	NotificationTypeTaskStatusChanged NotificationType = "TASK_STATUS_CHANGED"

	// Comment events
	NotificationTypeCommentAdded     NotificationType = "COMMENT_ADDED"
	NotificationTypeCommentMentioned NotificationType = "COMMENT_MENTIONED"

	// Workspace events
	NotificationTypeWorkspaceInvited     NotificationType = "WORKSPACE_INVITED"
	NotificationTypeWorkspaceRoleChanged NotificationType = "WORKSPACE_ROLE_CHANGED"
	NotificationTypeWorkspaceRemoved     NotificationType = "WORKSPACE_REMOVED"

	// Project events
	NotificationTypeProjectInvited     NotificationType = "PROJECT_INVITED"
	NotificationTypeProjectRoleChanged NotificationType = "PROJECT_ROLE_CHANGED"
	NotificationTypeProjectRemoved     NotificationType = "PROJECT_REMOVED"

	// Board (Kanban) events
	NotificationTypeBoardAssigned        NotificationType = "BOARD_ASSIGNED"
	NotificationTypeBoardUnassigned      NotificationType = "BOARD_UNASSIGNED"
	NotificationTypeBoardParticipantAdded NotificationType = "BOARD_PARTICIPANT_ADDED"
	NotificationTypeBoardUpdated         NotificationType = "BOARD_UPDATED"
	NotificationTypeBoardStatusChanged   NotificationType = "BOARD_STATUS_CHANGED"
	NotificationTypeBoardCommentAdded    NotificationType = "BOARD_COMMENT_ADDED"
	NotificationTypeBoardDueSoon         NotificationType = "BOARD_DUE_SOON"
	NotificationTypeBoardOverdue         NotificationType = "BOARD_OVERDUE"
)

// ResourceType defines the type of resource
type ResourceType string

const (
	ResourceTypeTask      ResourceType = "task"
	ResourceTypeComment   ResourceType = "comment"
	ResourceTypeWorkspace ResourceType = "workspace"
	ResourceTypeProject   ResourceType = "project"
	ResourceTypeBoard     ResourceType = "board"
)

// Notification represents a notification entity
type Notification struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Type         NotificationType       `gorm:"type:varchar(50);not null" json:"type"`
	ActorID      uuid.UUID              `gorm:"type:uuid;not null;index" json:"actorId"`
	TargetUserID uuid.UUID              `gorm:"type:uuid;not null" json:"targetUserId"`
	WorkspaceID  uuid.UUID              `gorm:"type:uuid;not null" json:"workspaceId"`
	ResourceType ResourceType           `gorm:"type:varchar(30);not null" json:"resourceType"`
	ResourceID   uuid.UUID              `gorm:"type:uuid;not null" json:"resourceId"`
	ResourceName *string                `gorm:"type:varchar(255)" json:"resourceName,omitempty"`
	Metadata     map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	IsRead       bool                   `gorm:"default:false" json:"isRead"`
	ReadAt       *time.Time             `gorm:"type:timestamptz" json:"readAt,omitempty"`
	CreatedAt    time.Time              `gorm:"type:timestamptz;default:now();not null" json:"createdAt"`
}

func (Notification) TableName() string {
	return "notifications"
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	WorkspaceID *uuid.UUID `gorm:"type:uuid;index" json:"workspaceId,omitempty"`
	Type        string     `gorm:"type:varchar(50);not null" json:"type"`
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;default:now();not null" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;default:now();not null" json:"updatedAt"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

// NotificationEvent represents an incoming notification event
type NotificationEvent struct {
	Type         NotificationType       `json:"type" binding:"required"`
	ActorID      uuid.UUID              `json:"actorId" binding:"required"`
	TargetUserID uuid.UUID              `json:"targetUserId" binding:"required"`
	WorkspaceID  uuid.UUID              `json:"workspaceId" binding:"required"`
	ResourceType ResourceType           `json:"resourceType" binding:"required"`
	ResourceID   uuid.UUID              `json:"resourceId" binding:"required"`
	ResourceName *string                `json:"resourceName,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	OccurredAt   *time.Time             `json:"occurredAt,omitempty"`
}

// PaginatedNotifications represents paginated notification response
type PaginatedNotifications struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
	HasMore       bool           `json:"hasMore"`
}

// UnreadCount represents unread count response
type UnreadCount struct {
	Count       int64     `json:"count"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
}
