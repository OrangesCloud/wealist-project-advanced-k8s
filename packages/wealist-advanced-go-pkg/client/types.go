// Package client provides common HTTP client types for service-to-service communication.
package client

import "github.com/google/uuid"

// WorkspaceValidationResponse represents the response from workspace validation endpoint.
// This is the standard response format from user-service /workspaces/{id}/validate-member/{userId}.
type WorkspaceValidationResponse struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	UserID      uuid.UUID `json:"userId"`
	Valid       bool      `json:"valid"`
	IsValid     bool      `json:"isValid"`
	IsMember    bool      `json:"isMember"` // User Service returns this field
}

// IsWorkspaceMember returns true if any of the validation fields indicates membership.
func (r *WorkspaceValidationResponse) IsWorkspaceMember() bool {
	return r.Valid || r.IsValid || r.IsMember
}

// UserProfile represents basic user profile information.
type UserProfile struct {
	UserID   uuid.UUID `json:"userId"`
	Email    string    `json:"email"`
	Provider string    `json:"provider"`
}

// WorkspaceProfile represents workspace-specific user profile.
type WorkspaceProfile struct {
	ProfileID       uuid.UUID `json:"profileId"`
	WorkspaceID     uuid.UUID `json:"workspaceId"`
	UserID          uuid.UUID `json:"userId"`
	NickName        string    `json:"nickName"`
	Email           string    `json:"email"`
	ProfileImageURL string    `json:"profileImageUrl"`
}

// Workspace represents workspace information.
type Workspace struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"ownerId"`
	OwnerName   string    `json:"ownerName"`
	OwnerEmail  string    `json:"ownerEmail"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
}

// TokenValidationResponse represents the response from auth-service token validation.
type TokenValidationResponse struct {
	UserID  string `json:"userId"`
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}
