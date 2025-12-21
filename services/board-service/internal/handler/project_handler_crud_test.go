package handler

import (
	"bytes"
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

func TestProjectHandler_UpdateProject(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		requestBody    interface{}
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:      "성공: Project 수정",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name:        stringPtr("Updated Project"),
				Description: stringPtr("Updated Description"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
					return &dto.ProjectResponse{
						ID:          pID,
						WorkspaceID: uuid.New(),
						OwnerID:     uID,
						Name:        *req.Name,
						Description: *req.Description,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			requestBody:    dto.UpdateProjectRequest{},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "실패: 잘못된 요청 본문",
			projectID:      projectID.String(),
			requestBody:    "invalid json",
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "실패: OWNER가 아님",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name: stringPtr("Updated Project"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
					return nil, response.NewForbiddenError("Only project owner can update project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "실패: Project가 존재하지 않음",
			projectID: projectID.String(),
			requestBody: dto.UpdateProjectRequest{
				Name: stringPtr("Updated Project"),
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.UpdateProjectFunc = func(ctx context.Context, pID, uID uuid.UUID, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
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
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.PUT("/api/projects/:projectId", handler.UpdateProject)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/projects/"+tt.projectID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name:       "성공: Project 삭제",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return nil
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
			name:       "실패: OWNER가 아님",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return response.NewForbiddenError("Only project owner can delete project", "")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "실패: Project가 존재하지 않음",
			projectID:  projectID.String(),
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.DeleteProjectFunc = func(ctx context.Context, pID, uID uuid.UUID) error {
					return response.NewNotFoundError("Project not found", "")
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
					c.Set("requestId", uuid.New().String())
					c.Next()
				})
			}

			router.DELETE("/api/projects/:projectId", handler.DeleteProject)

			req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+tt.projectID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestProjectHandler_SearchProjects(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		queryParams    map[string]string
		setContext     bool
		mockService    func(*MockProjectService)
		expectedStatus int
	}{
		{
			name: "성공: Project 검색",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
				"query":       "test",
				"page":        "1",
				"limit":       "10",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.SearchProjectsFunc = func(ctx context.Context, wID, uID uuid.UUID, query string, page, limit int, t string) (*dto.PaginatedProjectsResponse, error) {
					return &dto.PaginatedProjectsResponse{
						Projects: []dto.ProjectResponse{
							{
								ID:          uuid.New(),
								WorkspaceID: wID,
								Name:        "Test Project",
							},
						},
						Page:  page,
						Limit: limit,
						Total: 1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "실패: workspaceId 누락",
			queryParams: map[string]string{
				"query": "test",
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: query 누락",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: 잘못된 workspaceId",
			queryParams: map[string]string{
				"workspaceId": "invalid-uuid",
				"query":       "test",
			},
			setContext:     true,
			mockService:    func(m *MockProjectService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: Workspace 멤버가 아님",
			queryParams: map[string]string{
				"workspaceId": workspaceID.String(),
				"query":       "test",
			},
			setContext: true,
			mockService: func(m *MockProjectService) {
				m.SearchProjectsFunc = func(ctx context.Context, wID, uID uuid.UUID, query string, page, limit int, t string) (*dto.PaginatedProjectsResponse, error) {
					return nil, response.NewForbiddenError("You are not a member of this workspace", "")
				}
			},
			expectedStatus: http.StatusForbidden,
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

			router.GET("/api/projects/search", handler.SearchProjects)

			url := "/api/projects/search?"
			for k, v := range tt.queryParams {
				url += k + "=" + v + "&"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("SearchProjects() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
