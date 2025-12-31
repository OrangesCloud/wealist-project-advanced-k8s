package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"project-board-api/internal/metrics"
)

// Shared metrics instance for all tests to avoid duplicate registration
var testClientMetrics *metrics.Metrics

func init() {
	testClientMetrics = metrics.NewTestMetrics()
}

func TestUserClient_ValidateWorkspaceMember(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	token := "test-token"

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantValid      bool
		wantErr        bool
	}{
		{
			name: "성공: 유효한 멤버 (Valid 필드)",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(WorkspaceValidationResponse{
					WorkspaceID: workspaceID,
					UserID:      userID,
					Valid:       true,
				})
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "성공: 유효한 멤버 (IsValid 필드)",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(WorkspaceValidationResponse{
					WorkspaceID: workspaceID,
					UserID:      userID,
					IsValid:     true,
				})
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "성공: 유효하지 않은 멤버",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(WorkspaceValidationResponse{
					WorkspaceID: workspaceID,
					UserID:      userID,
					Valid:       false,
					IsValid:     false,
				})
			},
			wantValid: false,
			wantErr:   false,
		},
		{
			name: "실패: 404 에러",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "workspace not found"}`))
			},
			wantValid: false,
			wantErr:   true,
		},
		{
			name: "실패: 500 에러",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := zap.NewNop()
			client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

			// When
			valid, err := client.ValidateWorkspaceMember(context.Background(), workspaceID, userID, token)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateWorkspaceMember() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWorkspaceMember() unexpected error = %v", err)
				}
			}

			if valid != tt.wantValid {
				t.Errorf("ValidateWorkspaceMember() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestUserClient_GetUserProfile(t *testing.T) {
	userID := uuid.New()
	token := "test-token"

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantProfile    *UserProfile
		wantErr        bool
	}{
		{
			name: "성공: 사용자 프로필 조회",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(UserProfile{
					UserID:   userID,
					Email:    "test@example.com",
					Provider: "google",
				})
			},
			wantProfile: &UserProfile{
				UserID:   userID,
				Email:    "test@example.com",
				Provider: "google",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 404 에러 시 빈 프로필 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "user not found"}`))
			},
			wantProfile: &UserProfile{
				UserID: userID,
				Email:  "",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 500 에러 시 빈 프로필 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			wantProfile: &UserProfile{
				UserID: userID,
				Email:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := zap.NewNop()
			client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

			// When
			profile, err := client.GetUserProfile(context.Background(), userID, token)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetUserProfile() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("GetUserProfile() unexpected error = %v", err)
				}
			}

			if profile == nil {
				t.Fatal("GetUserProfile() returned nil profile")
				return
			}

			if profile.UserID != tt.wantProfile.UserID {
				t.Errorf("GetUserProfile() UserID = %v, want %v", profile.UserID, tt.wantProfile.UserID)
			}

			if profile.Email != tt.wantProfile.Email {
				t.Errorf("GetUserProfile() Email = %v, want %v", profile.Email, tt.wantProfile.Email)
			}
		})
	}
}

func TestUserClient_GetWorkspaceProfile(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	profileID := uuid.New()
	token := "test-token"

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantProfile    *WorkspaceProfile
		wantErr        bool
	}{
		{
			name: "성공: 워크스페이스 프로필 조회",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(WorkspaceProfile{
					ProfileID:       profileID,
					WorkspaceID:     workspaceID,
					UserID:          userID,
					NickName:        "Test User",
					Email:           "test@example.com",
					ProfileImageURL: "https://example.com/image.jpg",
				})
			},
			wantProfile: &WorkspaceProfile{
				ProfileID:       profileID,
				WorkspaceID:     workspaceID,
				UserID:          userID,
				NickName:        "Test User",
				Email:           "test@example.com",
				ProfileImageURL: "https://example.com/image.jpg",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 404 에러 시 빈 프로필 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "profile not found"}`))
			},
			wantProfile: &WorkspaceProfile{
				WorkspaceID: workspaceID,
				UserID:      userID,
				NickName:    "",
				Email:       "",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 500 에러 시 빈 프로필 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			wantProfile: &WorkspaceProfile{
				WorkspaceID: workspaceID,
				UserID:      userID,
				NickName:    "",
				Email:       "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := zap.NewNop()
			client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

			// When
			profile, err := client.GetWorkspaceProfile(context.Background(), workspaceID, userID, token)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetWorkspaceProfile() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("GetWorkspaceProfile() unexpected error = %v", err)
				}
			}

			if profile == nil {
				t.Fatal("GetWorkspaceProfile() returned nil profile")
				return
			}

			if profile.WorkspaceID != tt.wantProfile.WorkspaceID {
				t.Errorf("GetWorkspaceProfile() WorkspaceID = %v, want %v", profile.WorkspaceID, tt.wantProfile.WorkspaceID)
			}

			if profile.UserID != tt.wantProfile.UserID {
				t.Errorf("GetWorkspaceProfile() UserID = %v, want %v", profile.UserID, tt.wantProfile.UserID)
			}

			if profile.NickName != tt.wantProfile.NickName {
				t.Errorf("GetWorkspaceProfile() NickName = %v, want %v", profile.NickName, tt.wantProfile.NickName)
			}
		})
	}
}

func TestUserClient_GetWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	ownerID := uuid.New()
	token := "test-token"

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantWorkspace  *Workspace
		wantErr        bool
	}{
		{
			name: "성공: 워크스페이스 조회",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(Workspace{
					ID:          workspaceID,
					Name:        "Test Workspace",
					Description: "Test Description",
					OwnerID:     ownerID,
					OwnerName:   "Owner Name",
					OwnerEmail:  "owner@example.com",
					CreatedAt:   "2024-01-01T00:00:00Z",
					UpdatedAt:   "2024-01-01T00:00:00Z",
				})
			},
			wantWorkspace: &Workspace{
				ID:          workspaceID,
				Name:        "Test Workspace",
				Description: "Test Description",
				OwnerID:     ownerID,
				OwnerName:   "Owner Name",
				OwnerEmail:  "owner@example.com",
				CreatedAt:   "2024-01-01T00:00:00Z",
				UpdatedAt:   "2024-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 404 에러 시 빈 워크스페이스 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "workspace not found"}`))
			},
			wantWorkspace: &Workspace{
				ID:   workspaceID,
				Name: "",
			},
			wantErr: false,
		},
		{
			name: "Graceful degradation: 500 에러 시 빈 워크스페이스 반환",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			wantWorkspace: &Workspace{
				ID:   workspaceID,
				Name: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := zap.NewNop()
			client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

			// When
			workspace, err := client.GetWorkspace(context.Background(), workspaceID, token)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetWorkspace() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("GetWorkspace() unexpected error = %v", err)
				}
			}

			if workspace == nil {
				t.Fatal("GetWorkspace() returned nil workspace")
				return
			}

			if workspace.ID != tt.wantWorkspace.ID {
				t.Errorf("GetWorkspace() ID = %v, want %v", workspace.ID, tt.wantWorkspace.ID)
			}

			if workspace.Name != tt.wantWorkspace.Name {
				t.Errorf("GetWorkspace() Name = %v, want %v", workspace.Name, tt.wantWorkspace.Name)
			}
		})
	}
}

