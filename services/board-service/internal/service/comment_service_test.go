package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// Mock implementations are in mock_test.go

func TestCommentService_CreateComment(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name        string
		req         *dto.CreateCommentRequest
		mockBoard   func(*MockBoardRepository)
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: 정상적인 Comment 생성",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
					comment.ID = uuid.New()
					comment.CreatedAt = time.Now()
					comment.UpdatedAt = time.Now()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: Board가 존재하지 않음",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockComment: func(m *MockCommentRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name: "실패: Comment 생성 중 DB 에러",
			req: &dto.CreateCommentRequest{
				BoardID: boardID,
				Content: "Test Comment",
			},
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.CreateFunc = func(ctx context.Context, comment *domain.Comment) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockCommentRepo := &MockCommentRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			userID := uuid.New()
			got, err := service.CreateComment(context.Background(), userID, tt.req)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateComment() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("CreateComment() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateComment() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("CreateComment() returned nil response")
					return
				}
				if got.Content != tt.req.Content {
					t.Errorf("CreateComment() Content = %v, want %v", got.Content, tt.req.Content)
				}
			}
		})
	}
}

func TestCommentService_GetComments(t *testing.T) {
	boardID := uuid.New()

	tests := []struct {
		name        string
		boardID     uuid.UUID
		mockBoard   func(*MockBoardRepository)
		mockComment func(*MockCommentRepository)
		wantErr     bool
		wantErrCode string
		wantCount   int
	}{
		{
			name:    "성공: Comment 목록 조회",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Comment, error) {
					return []*domain.Comment{
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
							Content:   "Comment 1",
						},
						{
							BaseModel: domain.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
							BoardID:   boardID,
							UserID:    uuid.New(),
							Content:   "Comment 2",
						},
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "성공: 빈 Comment 목록",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return &domain.Board{}, nil
				}
			},
			mockComment: func(m *MockCommentRepository) {
				m.FindByBoardIDFunc = func(ctx context.Context, bID uuid.UUID) ([]*domain.Comment, error) {
					return []*domain.Comment{}, nil
				}
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:    "실패: Board가 존재하지 않음",
			boardID: boardID,
			mockBoard: func(m *MockBoardRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockComment: func(m *MockCommentRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockBoardRepo := &MockBoardRepository{}
			mockCommentRepo := &MockCommentRepository{}
			tt.mockBoard(mockBoardRepo)
			tt.mockComment(mockCommentRepo)

			logger, _ := zap.NewDevelopment()
			service := NewCommentService(mockCommentRepo, mockBoardRepo, &MockAttachmentRepository{}, nil, logger)

			// When
			got, err := service.GetComments(context.Background(), tt.boardID)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetComments() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetComments() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetComments() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("GetComments() returned nil response")
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetComments() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}
