package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func TestProjectHandler_GetProjectInitSettings(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:       "성공: 초기 설정 조회",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
					return &dto.ProjectInitSettingsResponse{
						Project: dto.ProjectBasicInfo{
							ProjectID:   pID,
							WorkspaceID: uuid.New(),
							Name:        "Test Project",
						},
						Fields:     []dto.FieldWithOptionsResponse{},
						FieldTypes: []dto.FieldTypeInfo{},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: Context에 user_id 없음",
			projectID:      projectID.String(),
			setContext:     false,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "실패: 프로젝트 멤버가 아님",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
					return nil, response.NewForbiddenError("You are not a member of this project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "실패: Project가 존재하지 않음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
					return nil, response.NewNotFoundError("Project not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockProjectService{}
			tt.mockService(mockService)
			handler := NewProjectHandler(mockService)

			router := setupTestRouter()

			if tt.setContext {
				router.Use(func(c *gin.Context) {
					c.Set("user_id", userID)
					c.Set("jwtToken", token)
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.GET("/api/projects/:projectId/init-settings", handler.GetProjectInitSettings)

			req := httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.projectID+"/init-settings", nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetProjectInitSettings() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["requestId"]; !ok {
					t.Error("GetProjectInitSettings() response missing requestId field")
				}
			}
		})
	}
}

func TestProjectHandler_GetProjectInitSettings_WithFieldOptions(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	t.Run("성공: 새 프로젝트의 초기 설정에 12개의 field options 반환", func(t *testing.T) {
		// Given
		mockService := &MockProjectService{}
		mockService.GetProjectInitSettingsFunc = func(ctx context.Context, pID, uID uuid.UUID, t string) (*dto.ProjectInitSettingsResponse, error) {
			// Stage options (4개)
			stageOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "대기", OptionValue: "pending", Color: "#F59E0B", DisplayOrder: 1, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "진행중", OptionValue: "in_progress", Color: "#3B82F6", DisplayOrder: 2, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "검토", OptionValue: "review", Color: "#8B5CF6", DisplayOrder: 3, FieldID: "stage"},
				{OptionID: uuid.New().String(), OptionLabel: "완료", OptionValue: "approved", Color: "#10B981", DisplayOrder: 4, FieldID: "stage"},
			}

			// Importance options (4개)
			importanceOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "긴급", OptionValue: "urgent", Color: "#EF4444", DisplayOrder: 1, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "높음", OptionValue: "high", Color: "#F97316", DisplayOrder: 2, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "보통", OptionValue: "normal", Color: "#10B981", DisplayOrder: 3, FieldID: "importance"},
				{OptionID: uuid.New().String(), OptionLabel: "낮음", OptionValue: "low", Color: "#6B7280", DisplayOrder: 4, FieldID: "importance"},
			}

			// Role options (4개)
			roleOptions := []dto.FieldOption{
				{OptionID: uuid.New().String(), OptionLabel: "개발자", OptionValue: "developer", Color: "#8B5CF6", DisplayOrder: 1, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "기획자", OptionValue: "planner", Color: "#EC4899", DisplayOrder: 2, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "디자이너", OptionValue: "designer", Color: "#F59E0B", DisplayOrder: 3, FieldID: "role"},
				{OptionID: uuid.New().String(), OptionLabel: "QA", OptionValue: "qa", Color: "#06B6D4", DisplayOrder: 4, FieldID: "role"},
			}

			return &dto.ProjectInitSettingsResponse{
				Project: dto.ProjectBasicInfo{
					ProjectID:   pID,
					WorkspaceID: uuid.New(),
					Name:        "Test Project",
				},
				Fields: []dto.FieldWithOptionsResponse{
					{
						FieldID:     "stage",
						FieldName:   "Stage",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Current stage of the board",
						Options:     stageOptions,
					},
					{
						FieldID:     "importance",
						FieldName:   "Importance",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Priority level of the board",
						Options:     importanceOptions,
					},
					{
						FieldID:     "role",
						FieldName:   "Role",
						FieldType:   "select",
						IsRequired:  true,
						Description: "Role responsible for the board",
						Options:     roleOptions,
					},
				},
				FieldTypes: []dto.FieldTypeInfo{},
			}, nil
		}

		handler := NewProjectHandler(mockService)
		router := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("jwtToken", token)
			c.Set("requestId", uuid.New().String())
			c.Next()
		})
		router.GET("/api/projects/:projectId/init-settings", handler.GetProjectInitSettings)

		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+projectID.String()+"/init-settings", nil)
		w := httptest.NewRecorder()

		// When
		router.ServeHTTP(w, req)

		// Then
		if w.Code != http.StatusOK {
			t.Errorf("GetProjectInitSettings() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				Project dto.ProjectBasicInfo           `json:"project"`
				Fields  []dto.FieldWithOptionsResponse `json:"fields"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify 3 fields are returned
		if len(response.Data.Fields) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(response.Data.Fields))
		}

		// Count total options across all fields
		totalOptions := 0
		for _, field := range response.Data.Fields {
			totalOptions += len(field.Options)
		}

		// Verify 12 total field options are returned
		if totalOptions != 12 {
			t.Errorf("Expected 12 total field options, got %d", totalOptions)
		}

		// Verify stage field has 4 options
		var stageField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "stage" {
				stageField = &response.Data.Fields[i]
				break
			}
		}
		if stageField == nil {
			t.Error("Stage field not found")
		} else if len(stageField.Options) != 4 {
			t.Errorf("Expected stage field to have 4 options, got %d", len(stageField.Options))
		}

		// Verify importance field has 4 options
		var importanceField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "importance" {
				importanceField = &response.Data.Fields[i]
				break
			}
		}
		if importanceField == nil {
			t.Error("Importance field not found")
		} else if len(importanceField.Options) != 4 {
			t.Errorf("Expected importance field to have 4 options, got %d", len(importanceField.Options))
		}

		// Verify role field has 4 options
		var roleField *dto.FieldWithOptionsResponse
		for i := range response.Data.Fields {
			if response.Data.Fields[i].FieldID == "role" {
				roleField = &response.Data.Fields[i]
				break
			}
		}
		if roleField == nil {
			t.Error("Role field not found")
		} else if len(roleField.Options) != 4 {
			t.Errorf("Expected role field to have 4 options, got %d", len(roleField.Options))
		}
	})
}
