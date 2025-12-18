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

// MockBoardService is a mock implementation of BoardService
type MockBoardService struct {
	CreateBoardFunc        func(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error)
	GetBoardFunc           func(ctx context.Context, boardID uuid.UUID) (*dto.BoardDetailResponse, error)
	GetBoardsByProjectFunc func(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error)
	UpdateBoardFunc        func(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error)
	DeleteBoardFunc        func(ctx context.Context, boardID uuid.UUID) error
}

func (m *MockBoardService) CreateBoard(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error) {
	if m.CreateBoardFunc != nil {
		return m.CreateBoardFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockBoardService) GetBoard(ctx context.Context, boardID uuid.UUID) (*dto.BoardDetailResponse, error) {
	if m.GetBoardFunc != nil {
		return m.GetBoardFunc(ctx, boardID)
	}
	return nil, nil
}

func (m *MockBoardService) GetBoardsByProject(ctx context.Context, projectID uuid.UUID, filters *dto.BoardFilters) ([]*dto.BoardResponse, error) {
	if m.GetBoardsByProjectFunc != nil {
		return m.GetBoardsByProjectFunc(ctx, projectID, filters)
	}
	return nil, nil
}

func (m *MockBoardService) UpdateBoard(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
	if m.UpdateBoardFunc != nil {
		return m.UpdateBoardFunc(ctx, boardID, req)
	}
	return nil, nil
}

func (m *MockBoardService) DeleteBoard(ctx context.Context, boardID uuid.UUID) error {
	if m.DeleteBoardFunc != nil {
		return m.DeleteBoardFunc(ctx, boardID)
	}
	return nil
}

func TestBoardHandler_CreateBoard(t *testing.T) {
	projectID := uuid.New()
	boardID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    func(*MockBoardService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "성공: Board 생성",
			requestBody: dto.CreateBoardRequest{
				ProjectID: projectID,
				Title:     "Test Board",
				Content:   "Test Content",
				CustomFields: map[string]interface{}{
					"stage":      "in_progress",
					"importance": "urgent",
					"role":       "developer",
				},
			},
			mockService: func(m *MockBoardService) {
				m.CreateBoardFunc = func(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error) {
					return &dto.BoardResponse{
						ID:           boardID,
						ProjectID:    req.ProjectID,
						Title:        req.Title,
						Content:      req.Content,
						CustomFields: req.CustomFields,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
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
				if board.CustomFields["stage"] != "in_progress" {
					t.Errorf("Expected stage 'in_progress', got '%v'", board.CustomFields["stage"])
				}
			},
		},
		{
			name:           "실패: 잘못된 요청 본문",
			requestBody:    "invalid json",
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "실패: Project가 존재하지 않음",
			requestBody: dto.CreateBoardRequest{
				ProjectID: projectID,
				Title:     "Test Board",
				Content:   "Test Content",
				CustomFields: map[string]interface{}{
					"stage": "in_progress",
				},
			},
			mockService: func(m *MockBoardService) {
				m.CreateBoardFunc = func(ctx context.Context, req *dto.CreateBoardRequest) (*dto.BoardResponse, error) {
					return nil, response.NewAppError(response.ErrCodeNotFound, "Project not found", "")
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
			router.POST("/api/boards", handler.CreateBoard)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("CreateBoard() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestBoardHandler_GetBoard(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name           string
		boardID        string
		mockService    func(*MockBoardService)
		expectedStatus int
	}{
		{
			name:    "성공: Board 조회",
			boardID: boardID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardFunc = func(ctx context.Context, id uuid.UUID) (*dto.BoardDetailResponse, error) {
					return &dto.BoardDetailResponse{
						BoardResponse: dto.BoardResponse{
							ID:    id,
							Title: "Test Board",
						},
						Participants: []dto.ParticipantResponse{},
						Comments:     []dto.CommentResponse{},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			boardID:        "invalid-uuid",
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardFunc = func(ctx context.Context, id uuid.UUID) (*dto.BoardDetailResponse, error) {
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
			router.GET("/api/boards/:boardId", handler.GetBoard)

			req := httptest.NewRequest(http.MethodGet, "/api/boards/"+tt.boardID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("GetBoard() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestBoardHandler_DeleteBoard(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name           string
		boardID        string
		mockService    func(*MockBoardService)
		expectedStatus int
	}{
		{
			name:    "성공: Board 삭제",
			boardID: boardID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardFunc = func(ctx context.Context, id uuid.UUID) (*dto.BoardDetailResponse, error) {
					return &dto.BoardDetailResponse{
						BoardResponse: dto.BoardResponse{
							ID:        boardID,
							ProjectID: uuid.New(),
							Title:     "Test Board",
						},
					}, nil
				}
				m.DeleteBoardFunc = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "실패: 잘못된 UUID",
			boardID:        "invalid-uuid",
			mockService:    func(m *MockBoardService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID.String(),
			mockService: func(m *MockBoardService) {
				m.GetBoardFunc = func(ctx context.Context, id uuid.UUID) (*dto.BoardDetailResponse, error) {
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
			router.DELETE("/api/boards/:boardId", handler.DeleteBoard)

			req := httptest.NewRequest(http.MethodDelete, "/api/boards/"+tt.boardID, nil)
			w := httptest.NewRecorder()

			// When
			router.ServeHTTP(w, req)

			// Then
			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteBoard() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
