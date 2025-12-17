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

func TestProjectMemberService_GetMembers(t *testing.T) {
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
		wantCount   int
	}{
		{
			name: "성공: 멤버 목록 조회",
			mockRepo: func(m *MockProjectRepository) {
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return true, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.FindMembersByProjectIDFunc = func(ctx context.Context, pID uuid.UUID) ([]*domain.ProjectMember, error) {
					return []*domain.ProjectMember{
						{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    userID,
							RoleName:  domain.ProjectRoleOwner,
							JoinedAt:  time.Now(),
						},
						{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uuid.New(),
							RoleName:  domain.ProjectRoleMember,
							JoinedAt:  time.Now(),
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
			name: "실패: 프로젝트 멤버가 아님",
			mockRepo: func(m *MockProjectRepository) {
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return false, nil
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name: "실패: 프로젝트를 찾을 수 없음",
			mockRepo: func(m *MockProjectRepository) {
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return true, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			mockClient:  func(m *MockUserClient) {},
			wantErr:     true,
			wantErrCode: response.ErrCodeNotFound,
		},
		{
			name: "실패: 멤버 조회 DB 에러",
			mockRepo: func(m *MockProjectRepository) {
				m.IsProjectMemberFunc = func(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
					return true, nil
				}
				m.FindByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel:   domain.BaseModel{ID: projectID},
						WorkspaceID: workspaceID,
					}, nil
				}
				m.FindMembersByProjectIDFunc = func(ctx context.Context, pID uuid.UUID) ([]*domain.ProjectMember, error) {
					return nil, errors.New("database error")
				}
			},
			mockClient:  func(m *MockUserClient) {},
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

			service := NewProjectMemberService(mockRepo, mockClient)
			got, err := service.GetMembers(context.Background(), projectID, userID, token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetMembers() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("GetMembers() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetMembers() unexpected error = %v", err)
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("GetMembers() count = %v, want %v", len(got), tt.wantCount)
				}
			}
		})
	}
}

func TestProjectMemberService_RemoveMember(t *testing.T) {
	projectID := uuid.New()
	requesterID := uuid.New()
	memberID := uuid.New()

	tests := []struct {
		name        string
		memberID    uuid.UUID
		mockRepo    func(*MockProjectRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name:     "성공: 멤버 삭제 (OWNER가 요청)",
			memberID: memberID,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					if uID == requesterID {
						return &domain.ProjectMember{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							RoleName:  domain.ProjectRoleOwner,
						}, nil
					}
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember,
					}, nil
				}
				m.RemoveMemberFunc = func(ctx context.Context, memberID uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "성공: 멤버 삭제 (ADMIN이 요청)",
			memberID: memberID,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					if uID == requesterID {
						return &domain.ProjectMember{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							RoleName:  domain.ProjectRoleAdmin,
						}, nil
					}
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember,
					}, nil
				}
				m.RemoveMemberFunc = func(ctx context.Context, memberID uuid.UUID) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "실패: 요청자가 멤버가 아님",
			memberID: memberID,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:     "실패: 권한 없음 (일반 멤버)",
			memberID: memberID,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember,
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name:     "실패: OWNER 삭제 시도",
			memberID: memberID,
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					if uID == requesterID {
						return &domain.ProjectMember{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							RoleName:  domain.ProjectRoleOwner,
						}, nil
					}
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner, // Target is also owner
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name:     "실패: 자기 자신 삭제 시도",
			memberID: requesterID, // Same as requester
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockProjectRepository{}
			tt.mockRepo(mockRepo)

			service := NewProjectMemberService(mockRepo, &MockUserClient{})
			err := service.RemoveMember(context.Background(), projectID, requesterID, tt.memberID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveMember() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("RemoveMember() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("RemoveMember() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestProjectMemberService_UpdateMemberRole(t *testing.T) {
	projectID := uuid.New()
	requesterID := uuid.New()
	memberID := uuid.New()

	tests := []struct {
		name        string
		role        string
		mockRepo    func(*MockProjectRepository)
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "성공: 역할을 ADMIN으로 변경",
			role: "ADMIN",
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					if uID == requesterID {
						return &domain.ProjectMember{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							RoleName:  domain.ProjectRoleOwner,
						}, nil
					}
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleMember,
					}, nil
				}
				m.UpdateMemberRoleFunc = func(ctx context.Context, memberID uuid.UUID, role domain.ProjectRole) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "실패: 요청자가 OWNER가 아님",
			role: "ADMIN",
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleAdmin, // Not owner
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeForbidden,
		},
		{
			name: "실패: 잘못된 역할",
			role: "INVALID_ROLE",
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner,
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
		{
			name: "실패: OWNER 역할 변경 시도",
			role: "MEMBER",
			mockRepo: func(m *MockProjectRepository) {
				m.FindMemberByProjectAndUserFunc = func(ctx context.Context, pID, uID uuid.UUID) (*domain.ProjectMember, error) {
					if uID == requesterID {
						return &domain.ProjectMember{
							ID: uuid.New(),
							ProjectID: pID,
							UserID:    uID,
							RoleName:  domain.ProjectRoleOwner,
						}, nil
					}
					return &domain.ProjectMember{
						ID: uuid.New(),
						ProjectID: pID,
						UserID:    uID,
						RoleName:  domain.ProjectRoleOwner, // Target is owner
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: response.ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockProjectRepository{}
			tt.mockRepo(mockRepo)

			service := NewProjectMemberService(mockRepo, &MockUserClient{})
			_, err := service.UpdateMemberRole(context.Background(), projectID, requesterID, memberID, tt.role)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateMemberRole() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if appErr, ok := err.(*response.AppError); ok {
					if appErr.Code != tt.wantErrCode {
						t.Errorf("UpdateMemberRole() error code = %v, want %v", appErr.Code, tt.wantErrCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateMemberRole() unexpected error = %v", err)
				}
			}
		})
	}
}
