package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func TestBoardHandler_GetBoardsByProject(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name           string
		projectID      string
		queryParams    string
		mockService    func(*MockBoardService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "성공: Project의 Board 목록 조회",
			projectID: projectID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardsByProjectFunc = func(ctx context.Context, id uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					return []*dto.BoardResponse{
						{
							ID:        uuid.New(),
							ProjectID: id,
							Title:     "Board 1",
							CustomFields: map[string]interface{}{
								"stage": "in_progress",
							},
						},
						{
							ID:        uuid.New(),
							ProjectID: id,
							Title:     "Board 2",
							CustomFields: map[string]interface{}{
								"stage": "pending",
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "성공: CustomFields 필터링 (단일 필드)",
			projectID:   projectID.String(),
			queryParams: `?customFields={"stage":"in_progress"}`,
			mockService: func(m *MockBoardService) {
				m.GetBoardsByProjectFunc = func(ctx context.Context, id uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					// Verify filters are passed correctly
					if filters == nil || filters.CustomFields == nil {
						t.Error("Expected filters to be present")
					}
					if filters != nil && filters.CustomFields != nil {
						if filters.CustomFields["stage"] != "in_progress" {
							t.Errorf("Expected stage filter 'in_progress', got '%v'", filters.CustomFields["stage"])
						}
					}
					return []*dto.BoardResponse{
						{
							ID:        uuid.New(),
							ProjectID: id,
							Title:     "Board 1",
							CustomFields: map[string]interface{}{
								"stage": "in_progress",
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "성공: CustomFields 필터링 (여러 필드)",
			projectID:   projectID.String(),
			queryParams: `?customFields={"stage":"in_progress","role":"developer"}`,
			mockService: func(m *MockBoardService) {
				m.GetBoardsByProjectFunc = func(ctx context.Context, id uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					// Verify multiple filters
					if filters == nil || filters.CustomFields == nil {
						t.Error("Expected filters to be present")
					}
					if filters != nil && filters.CustomFields != nil {
						if filters.CustomFields["stage"] != "in_progress" {
							t.Errorf("Expected stage filter 'in_progress', got '%v'", filters.CustomFields["stage"])
						}
						if filters.CustomFields["role"] != "developer" {
							t.Errorf("Expected role filter 'developer', got '%v'", filters.CustomFields["role"])
						}
					}
					return []*dto.BoardResponse{
						{
							ID:        uuid.New(),
							ProjectID: id,
							Title:     "Board 1",
							CustomFields: map[string]interface{}{
								"stage": "in_progress",
								"role":  "developer",
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			projectID:      "invalid-uuid",
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "실패: Project가 존재하지 않음",
			projectID: projectID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardsByProjectFunc = func(ctx context.Context, id uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Project not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "실패: 잘못된 CustomFields JSON 형식",
			projectID:      projectID.String(),
			queryParams:    `?customFields=invalid-json`,
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				errorData, ok := resp.Error.(map[string]interface{})
				if !ok {
					t.Fatal("Error field is not a map")
				}

				if errorData["code"] != response.ErrCodeValidation {
					t.Errorf("Expected error code '%s', got '%s'", response.ErrCodeValidation, errorData["code"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockBoardService{}
			tt.mockService(mockService)
			handler := NewBoardHandler(mockService)

			router := setupTestRouter()
			router.GET("/api/boards/project/:projectId", handler.GetBoardsByProject)

			url := "/api/boards/project/" + tt.projectID
			if tt.queryParams != "" {
				url += tt.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetBoardsByProject() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestBoardHandler_UpdateBoard(t *testing.T) {
	boardID := uuid.New()
	newTitle := "Updated Title"
	newCustomFields := map[string]interface{}{
		"stage":      "review",
		"importance": "normal",
	}

	tests := []struct {
		name           string
		boardID        string
		requestBody    interface{}
		mockService    func(*MockBoardService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "성공: Board 업데이트",
			boardID: boardID.String(),
			requestBody: dto.UpdateBoardRequest{
				Title: &newTitle,
			},
			mockService: func(m *MockBoardService) {
				m.UpdateBoardFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
					return &dto.BoardResponse{
						ID:    id,
						Title: *req.Title,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "성공: CustomFields 업데이트",
			boardID: boardID.String(),
			requestBody: dto.UpdateBoardRequest{
				CustomFields: &newCustomFields,
			},
			mockService: func(m *MockBoardService) {
				m.UpdateBoardFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
					return &dto.BoardResponse{
						ID:           id,
						Title:        "Test Board",
						CustomFields: *req.CustomFields,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp response.SuccessResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				dataBytes, _ := json.Marshal(resp.Data)
				var board dto.BoardResponse
				if err := json.Unmarshal(dataBytes, &board); err != nil {
					t.Fatalf("Failed to unmarshal data: %v", err)
				}

				if board.CustomFields == nil {
					t.Fatal("Expected CustomFields to be present")
				}
				if board.CustomFields["stage"] != "review" {
					t.Errorf("Expected stage 'review', got '%v'", board.CustomFields["stage"])
				}
			},
		},
		{
			name:           "실패: 잘못된 UUID",
			boardID:        "invalid-uuid",
			requestBody:    dto.UpdateBoardRequest{},
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID.String(),
			requestBody: dto.UpdateBoardRequest{
				Title: &newTitle,
			},
			mockService: func(m *MockBoardService) {
				m.UpdateBoardFunc = func(ctx context.Context, id uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockService := &MockBoardService{}
			tt.mockService(mockService)
			handler := NewBoardHandler(mockService)

			router := setupTestRouter()
			router.PUT("/api/boards/:boardId", handler.UpdateBoard)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/boards/"+tt.boardID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateBoard() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
