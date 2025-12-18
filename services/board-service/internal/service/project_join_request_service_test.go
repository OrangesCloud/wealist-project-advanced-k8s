package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/response"
)

func TestProjectJoinRequestService_CreateJoinRequest(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	workspaceID := uuid.New()
	token := "test-token"

	tests := []struct {
		name        string
		mockRepo    func(*MockProjectRepository)
		mockClient  func(*MockUserClient)
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: 가입 요청 생성",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return false, nil
				}
				m.FindPendingByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateJoinRequestFunc = func(ctx context.Context, request *domain.ProjectJoinRequest) error {
					request.ID = uuid.New()
					return nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return true, nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: 프로젝트를 찾을 수 없음",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name: "실패: 워크스페이스 멤버 검증 에러",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return false, errors.New("validation error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name: "실패: 워크스페이스 멤버가 아님",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return false, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name: "실패: 이미 프로젝트 멤버임",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return true, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return true, nil
				}
			},
			wantErr:     true,
			wantErrCode: "ALREADY_MEMBER",
		},
		{
			name: "실패: 대기 중인 요청이 이미 존재",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return false, nil
				}
				m.FindPendingByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						Status:    domain.JoinRequestPending,
					}, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return true, nil
				}
			},
			wantErr:     true,
			wantErrCode: "PENDING_REQUEST_EXISTS",
		},
		{
			name: "실패: 가입 요청 생성 DB 에러",
			mockRepo: func(m *MockProjectRepository) {
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return false, nil
				}
				m.FindPendingByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return nil, gorm.ErrRecordNotFound
				}
				m.CreateJoinRequestFunc = func(ctx context.Context, request *domain.ProjectJoinRequest) error {
					return errors.New("database error")
				}
			},
			mockClient: func(m *MockUserClient) {
				m.ValidateWorkspaceMemberFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (bool, error) {
					return true, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockProjectRepository{}
			mockClient := &MockUserClient{}
			tt.mockRepo(mockRepo)
			tt.mockClient(mockClient)

			service := NewProjectJoinRequestService(mockRepo, mockClient)
			got, err := service.CreateJoinRequest(context.Background(), projectID, userID, token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateJoinRequest() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("CreateJoinRequest() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateJoinRequest() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("CreateJoinRequest() returned nil response")
					return
				}
				if got.ProjectID != projectID {
					t.Errorf("CreateJoinRequest() ProjectID = %v, want %v", got.ProjectID, projectID)
				}
				if got.UserID != userID {
					t.Errorf("CreateJoinRequest() UserID = %v, want %v", got.UserID, userID)
				}
			}
		})
	}
}

func TestProjectJoinRequestService_GetJoinRequests(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	workspaceID := uuid.New()
	token := "test-token"
	pendingStatus := "PENDING"

	tests := []struct {
		name        string
		status      *string
		mockRepo    func(*MockProjectRepository)
		mockClient  func(*MockUserClient)
		wantErr     bool
		wantErrCode string
		wantCount   int
	}{
		{
			name:   "성공: 상태 필터 없이 전체 조회",
			status: nil,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.FindJoinRequestsByProjectIDFunc = func(ctx context.Context, pID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
					return []*domain.ProjectJoinRequest{
						{
							ID:          uuid.New(),
							ProjectID:   pID,
							UserID:      uuid.New(),
							Status:      domain.JoinRequestPending,
							RequestedAt: time.Now(),
						},
						{
							ID:          uuid.New(),
							ProjectID:   pID,
							UserID:      uuid.New(),
							Status:      domain.JoinRequestApproved,
							RequestedAt: time.Now(),
						},
					}, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.GetWorkspaceProfileFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (*client.WorkspaceProfile, error) {
					return &client.WorkspaceProfile{
						WorkspaceID: wID,
						UserID:      uID,
						NickName:    "Test User",
						Email:       "test@example.com",
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:   "성공: PENDING 상태만 조회",
			status: &pendingStatus,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleAdmin,
					}, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.FindJoinRequestsByProjectIDFunc = func(ctx context.Context, pID uuid.UUID, status *domain.ProjectJoinRequestStatus) ([]*domain.ProjectJoinRequest, error) {
					return []*domain.ProjectJoinRequest{
						{
							ID:          uuid.New(),
							ProjectID:   pID,
							UserID:      uuid.New(),
							Status:      domain.JoinRequestPending,
							RequestedAt: time.Now(),
						},
					}, nil
				}
			},
			mockClient: func(m *MockUserClient) {
				m.GetWorkspaceProfileFunc = func(ctx context.Context, wID, uID uuid.UUID, t string) (*client.WorkspaceProfile, error) {
					return &client.WorkspaceProfile{
						WorkspaceID: wID,
						UserID:      uID,
						NickName:    "Test User",
						Email:       "test@example.com",
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:   "실패: 프로젝트 멤버가 아님",
			status: nil,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:   "실패: OWNER/ADMIN이 아님",
			status: nil,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember,
					}, nil
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:   "실패: 프로젝트를 찾을 수 없음",
			status: nil,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockProjectRepository{}
			mockClient := &MockUserClient{}
			tt.mockRepo(mockRepo)
			tt.mockClient(mockClient)

			service := NewProjectJoinRequestService(mockRepo, mockClient)
			got, err := service.GetJoinRequests(context.Background(), projectID, userID, tt.status, token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetJoinRequests() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetJoinRequests() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetJoinRequests() unexpected error = %v", err)
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetJoinRequests() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}

func TestProjectJoinRequestService_UpdateJoinRequest(t *testing.T) {
	requestID := uuid.New()
	projectID := uuid.New()
	requesterID := uuid.New()
	applicantID := uuid.New()
	token := "test-token"

	tests := []struct {
		name        string
		status      string
		mockRepo    func(*MockProjectRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:   "성공: 승인 (멤버 추가됨)",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
				m.UpdateJoinRequestStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
					return nil
				}
				m.AddMemberFunc = func(ctx context.Context, member *domain.ProjectMember) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:   "성공: 거절",
			status: "REJECTED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleAdmin,
					}, nil
				}
				m.UpdateJoinRequestStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "실패: 잘못된 상태값",
			status:      "INVALID_STATUS",
			mockRepo:    func(m *MockProjectRepository) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name:   "실패: 요청을 찾을 수 없음",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name:   "실패: 이미 처리된 요청",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestApproved, // Already processed
						RequestedAt: time.Now(),
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name:   "실패: 프로젝트 멤버가 아님",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:   "실패: OWNER/ADMIN이 아님",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember, // Not owner/admin
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:   "실패: 상태 업데이트 DB 에러",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
				m.UpdateJoinRequestStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
		{
			name:   "실패: 멤버 추가 DB 에러",
			status: "APPROVED",
			mockRepo: func(m *MockProjectRepository) {
				m.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.JoinRequestPending,
						RequestedAt: time.Now(),
					}, nil
				}
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID:        uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
				m.UpdateJoinRequestStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.ProjectJoinRequestStatus) error {
					return nil
				}
				m.AddMemberFunc = func(ctx context.Context, member *domain.ProjectMember) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockProjectRepository{}
			tt.mockRepo(mockRepo)

			// For successful cases, need to set up the final fetch
			if !tt.wantErr {
				originalFunc := mockRepo.FindJoinRequestByIDFunc
				callCount := 0
				mockRepo.FindJoinRequestByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.ProjectJoinRequest, error) {
					callCount++
					if callCount == 1 {
						return originalFunc(ctx, id)
					}
					// Return updated request for second call
					return &domain.ProjectJoinRequest{
						ID:          requestID,
						ProjectID:   projectID,
						UserID:      applicantID,
						Status:      domain.ProjectJoinRequestStatus(tt.status),
						RequestedAt: time.Now(),
						UpdatedAt:   time.Now(),
					}, nil
				}
			}

			service := NewProjectJoinRequestService(mockRepo, &MockUserClient{})
			got, err := service.UpdateJoinRequest(context.Background(), requestID, requesterID, tt.status, token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateJoinRequest() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("UpdateJoinRequest() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateJoinRequest() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Error("UpdateJoinRequest() returned nil response")
					return
				}
				if got.Status != tt.status {
					t.Errorf("UpdateJoinRequest() Status = %v, want %v", got.Status, tt.status)
				}
			}
		})
	}
}
