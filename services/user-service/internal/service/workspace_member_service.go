// Package service는 user-service의 비즈니스 로직을 구현합니다.
//
// 이 파일은 워크스페이스 멤버 관리 및 참여 요청 관련 비즈니스 로직을 포함합니다.
// 워크스페이스 CRUD는 workspace_service.go에서 처리합니다.
package service

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"user-service/internal/domain"
	"user-service/internal/response"
)

// ============================================================
// 멤버 조회 메서드
// ============================================================

// GetMembers는 워크스페이스의 멤버 목록을 조회합니다.
func (s *WorkspaceService) GetMembers(workspaceID uuid.UUID) ([]domain.WorkspaceMember, error) {
	return s.memberRepo.FindByWorkspace(workspaceID)
}

// GetMembersWithProfiles는 프로필 정보를 포함한 멤버 목록을 조회합니다.
// 닉네임과 프로필 이미지 URL을 포함합니다.
func (s *WorkspaceService) GetMembersWithProfiles(workspaceID uuid.UUID) ([]domain.WorkspaceMemberResponse, error) {
	// 멤버 목록 조회
	members, err := s.memberRepo.FindByWorkspace(workspaceID)
	if err != nil {
		s.logger.Error("멤버 목록 조회 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, err
	}

	// 프로필 목록 조회
	profiles, err := s.profileRepo.FindByWorkspace(workspaceID)
	if err != nil {
		s.logger.Warn("프로필 목록 조회 실패, 기본 응답 반환",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
	}

	// UserID -> Profile 매핑 생성
	profileMap := make(map[uuid.UUID]*domain.UserProfile)
	for i := range profiles {
		profileMap[profiles[i].UserID] = &profiles[i]
	}

	// 응답 생성
	responses := make([]domain.WorkspaceMemberResponse, len(members))
	for i, member := range members {
		resp := member.ToResponse()

		// 프로필 정보 추가
		if profile, ok := profileMap[member.UserID]; ok {
			resp.NickName = profile.NickName
			if profile.ProfileImageURL != nil {
				resp.ProfileImageUrl = *profile.ProfileImageURL
			}
		}

		responses[i] = resp
	}

	s.logger.Debug("멤버 목록 조회 완료",
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("count", len(responses)))

	return responses, nil
}

// ============================================================
// 멤버 관리 메서드
// ============================================================

// InviteMember는 사용자를 워크스페이스에 초대합니다.
// 초대 권한 확인 후 이메일로 사용자를 찾아 멤버로 추가합니다.
func (s *WorkspaceService) InviteMember(workspaceID, inviterID uuid.UUID, req domain.InviteMemberRequest) (*domain.WorkspaceMember, error) {
	// 워크스페이스 조회
	workspace, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return nil, response.NewNotFoundError("Workspace not found", workspaceID.String())
	}

	// 초대 권한 확인
	if workspace.OnlyOwnerCanInvite && workspace.OwnerID != inviterID {
		// 소유자만 초대 가능 설정인 경우
		s.logger.Warn("초대 권한 없음 - 소유자만 초대 가능",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("inviter_id", inviterID.String()))
		return nil, response.NewForbiddenError("Only owner can invite members to this workspace", "")
	}

	// 초대자의 역할 확인 (MEMBER는 초대 불가)
	inviterRole, err := s.memberRepo.GetRole(workspaceID, inviterID)
	if err != nil {
		s.logger.Error("초대자 역할 조회 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("inviter_id", inviterID.String()),
			zap.Error(err))
		return nil, response.NewInternalError("Failed to verify invite permission", err.Error())
	}
	if inviterRole == domain.RoleMember {
		return nil, response.NewForbiddenError("Members cannot invite others", "")
	}

	// 이메일로 사용자 조회
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		s.logger.Warn("초대할 사용자를 찾을 수 없음",
			zap.String("email", req.Email),
			zap.Error(err))
		return nil, response.NewNotFoundError("User not found with the provided email", req.Email)
	}

	// 이미 멤버인지 확인
	isMember, _ := s.memberRepo.IsMember(workspaceID, user.ID)
	if isMember {
		return nil, response.NewAlreadyExistsError("User is already a member of this workspace", "")
	}

	// 역할 결정 (기본값: MEMBER)
	roleName := domain.RoleMember
	if req.RoleName != "" {
		roleName = req.RoleName
	}
	// OWNER 역할은 부여 불가
	if roleName == domain.RoleOwner {
		return nil, response.NewForbiddenError("Cannot assign owner role through invitation", "")
	}

	// 멤버 생성
	member := &domain.WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      user.ID,
		RoleName:    roleName,
		IsDefault:   false,
		IsActive:    true,
		JoinedAt:    time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.memberRepo.Create(member); err != nil {
		s.logger.Error("멤버 생성 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
		return nil, err
	}

	// 프로필 생성
	profile := &domain.UserProfile{
		ID:          uuid.New(),
		UserID:      user.ID,
		WorkspaceID: workspaceID,
		NickName:    user.Name,
		Email:       user.Email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if profile.NickName == "" {
		profile.NickName = user.Email
	}
	if err := s.profileRepo.Create(profile); err != nil {
		s.logger.Error("프로필 생성 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
	}

	// User 정보 포함
	member.User = user

	s.logger.Info("멤버 초대 완료",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("role", string(roleName)),
		zap.String("invited_by", inviterID.String()))

	return member, nil
}

// UpdateMemberRole은 멤버의 역할을 업데이트합니다.
// 소유자 또는 관리자만 역할을 변경할 수 있습니다.
func (s *WorkspaceService) UpdateMemberRole(workspaceID, memberID, updaterID uuid.UUID, req domain.UpdateMemberRoleRequest) (*domain.WorkspaceMember, error) {
	// 워크스페이스 조회
	workspace, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return nil, response.NewNotFoundError("Workspace not found", workspaceID.String())
	}

	// 업데이터의 역할 확인
	updaterRole, err := s.memberRepo.GetRole(workspaceID, updaterID)
	if err != nil {
		return nil, response.NewForbiddenError("Permission denied", "failed to verify role")
	}

	// MEMBER는 역할 변경 불가
	if updaterRole == domain.RoleMember {
		return nil, response.NewForbiddenError("Members cannot change roles", "")
	}

	// 대상 멤버 조회
	member, err := s.memberRepo.FindByID(memberID)
	if err != nil {
		return nil, response.NewNotFoundError("Member not found", memberID.String())
	}

	// 소유자의 역할은 변경 불가
	if member.UserID == workspace.OwnerID {
		return nil, response.NewForbiddenError("Cannot change owner's role", "")
	}

	// OWNER 역할 부여 불가
	if req.RoleName == domain.RoleOwner {
		return nil, response.NewForbiddenError("Cannot assign owner role", "")
	}

	// ADMIN이 다른 ADMIN의 역할을 변경하려 하면 거부
	if updaterRole == domain.RoleAdmin && member.RoleName == domain.RoleAdmin {
		return nil, response.NewForbiddenError("Admins cannot change other admins' roles", "")
	}

	// 역할 업데이트
	member.RoleName = req.RoleName
	member.UpdatedAt = time.Now()

	if err := s.memberRepo.Update(member); err != nil {
		s.logger.Error("멤버 역할 업데이트 실패",
			zap.String("member_id", memberID.String()),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("멤버 역할 업데이트 완료",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("member_id", memberID.String()),
		zap.String("new_role", string(req.RoleName)),
		zap.String("updated_by", updaterID.String()))

	return member, nil
}

// RemoveMember는 워크스페이스에서 멤버를 제거합니다.
// 소유자 또는 관리자만 멤버를 제거할 수 있습니다.
func (s *WorkspaceService) RemoveMember(workspaceID, memberID, removerID uuid.UUID) error {
	// 워크스페이스 조회
	workspace, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return response.NewNotFoundError("Workspace not found", workspaceID.String())
	}

	// 제거자의 역할 확인
	removerRole, err := s.memberRepo.GetRole(workspaceID, removerID)
	if err != nil {
		return response.NewForbiddenError("Permission denied", "failed to verify role")
	}

	// MEMBER는 다른 멤버 제거 불가 (자기 탈퇴만 가능)
	member, err := s.memberRepo.FindByID(memberID)
	if err != nil {
		return response.NewNotFoundError("Member not found", memberID.String())
	}

	// 자기 탈퇴가 아닌 경우 권한 확인
	if member.UserID != removerID {
		if removerRole == domain.RoleMember {
			return response.NewForbiddenError("Members cannot remove others", "")
		}
		// ADMIN이 다른 ADMIN 제거 시도 시 거부
		if removerRole == domain.RoleAdmin && member.RoleName == domain.RoleAdmin {
			return response.NewForbiddenError("Admins cannot remove other admins", "")
		}
	}

	// 소유자는 제거 불가
	if member.UserID == workspace.OwnerID {
		return response.NewForbiddenError("Cannot remove workspace owner", "")
	}

	// 멤버 제거 (soft delete)
	if err := s.memberRepo.Delete(memberID); err != nil {
		s.logger.Error("멤버 제거 실패",
			zap.String("member_id", memberID.String()),
			zap.Error(err))
		return err
	}

	// 프로필도 함께 삭제
	if err := s.profileRepo.DeleteByUserAndWorkspace(member.UserID, workspaceID); err != nil {
		s.logger.Warn("프로필 삭제 실패",
			zap.String("user_id", member.UserID.String()),
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
	}

	s.logger.Info("멤버 제거 완료",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("member_id", memberID.String()),
		zap.String("removed_by", removerID.String()))

	return nil
}

// ============================================================
// 멤버 검증 메서드
// ============================================================

// IsMember는 사용자가 워크스페이스의 멤버인지 확인합니다.
func (s *WorkspaceService) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	return s.memberRepo.IsMember(workspaceID, userID)
}

// ValidateMemberAccess는 사용자가 워크스페이스에 접근 권한이 있는지 확인합니다.
// 멤버이거나 공개 워크스페이스인 경우 접근 가능합니다.
func (s *WorkspaceService) ValidateMemberAccess(workspaceID, userID uuid.UUID) (bool, error) {
	// 먼저 멤버인지 확인
	isMember, err := s.memberRepo.IsMember(workspaceID, userID)
	if err != nil {
		s.logger.Error("멤버 확인 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return false, err
	}
	if isMember {
		return true, nil
	}

	// 멤버가 아니면 공개 워크스페이스인지 확인
	workspace, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return false, err
	}

	// 공개 워크스페이스면 접근 가능 (읽기만)
	return workspace.IsPublic, nil
}

// ============================================================
// 참여 요청 메서드
// ============================================================

// CreateJoinRequest는 워크스페이스 참여 요청을 생성합니다.
// 이미 멤버이거나 대기 중인 요청이 있으면 실패합니다.
func (s *WorkspaceService) CreateJoinRequest(workspaceID, userID uuid.UUID) (*domain.WorkspaceJoinRequest, error) {
	// 워크스페이스 존재 확인
	workspace, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		return nil, response.NewNotFoundError("Workspace not found", workspaceID.String())
	}

	// 비공개 워크스페이스는 참여 요청 불가
	if !workspace.IsPublic {
		return nil, response.NewForbiddenError("Cannot request to join private workspace", "")
	}

	// 이미 멤버인지 확인
	isMember, _ := s.memberRepo.IsMember(workspaceID, userID)
	if isMember {
		return nil, response.NewAlreadyExistsError("Already a member of this workspace", "")
	}

	// 대기 중인 요청이 있는지 확인
	hasPending, _ := s.joinReqRepo.HasPendingRequest(workspaceID, userID)
	if hasPending {
		return nil, response.NewAlreadyExistsError("Already have a pending request", "")
	}

	// 승인 필요 없는 경우 바로 멤버로 추가
	if !workspace.NeedApproved {
		member := &domain.WorkspaceMember{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			UserID:      userID,
			RoleName:    domain.RoleMember,
			IsDefault:   false,
			IsActive:    true,
			JoinedAt:    time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.memberRepo.Create(member); err != nil {
			s.logger.Error("자동 참여 실패",
				zap.String("workspace_id", workspaceID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err))
			return nil, err
		}

		// 프로필 생성
		user, _ := s.userRepo.FindByID(userID)
		if user != nil {
			profile := &domain.UserProfile{
				ID:          uuid.New(),
				UserID:      userID,
				WorkspaceID: workspaceID,
				NickName:    user.Name,
				Email:       user.Email,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if profile.NickName == "" {
				profile.NickName = user.Email
			}
			_ = s.profileRepo.Create(profile)
		}

		s.logger.Info("자동 참여 완료 (승인 불필요)",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()))

		// 자동 승인된 요청으로 반환
		return &domain.WorkspaceJoinRequest{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			UserID:      userID,
			Status:      domain.JoinStatusApproved,
			RequestedAt: time.Now(),
			UpdatedAt:   time.Now(),
			Workspace:   workspace,
		}, nil
	}

	// 참여 요청 생성
	request := &domain.WorkspaceJoinRequest{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Status:      domain.JoinStatusPending,
		RequestedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.joinReqRepo.Create(request); err != nil {
		s.logger.Error("참여 요청 생성 실패",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, err
	}

	// 워크스페이스 정보 포함
	request.Workspace = workspace

	s.logger.Info("참여 요청 생성 완료",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()))

	return request, nil
}

// GetJoinRequests는 워크스페이스의 참여 요청 목록을 조회합니다.
// 소유자 또는 관리자만 조회할 수 있습니다.
func (s *WorkspaceService) GetJoinRequests(workspaceID, requesterID uuid.UUID) ([]domain.WorkspaceJoinRequest, error) {
	// 요청자의 역할 확인
	role, err := s.memberRepo.GetRole(workspaceID, requesterID)
	if err != nil {
		return nil, response.NewForbiddenError("Permission denied", "failed to verify role")
	}

	// MEMBER는 조회 불가
	if role == domain.RoleMember {
		return nil, response.NewForbiddenError("Members cannot view join requests", "")
	}

	return s.joinReqRepo.FindPendingByWorkspace(workspaceID)
}

// GetPendingJoinRequests는 대기 중인 참여 요청만 조회합니다.
func (s *WorkspaceService) GetPendingJoinRequests(workspaceID uuid.UUID) ([]domain.WorkspaceJoinRequest, error) {
	return s.joinReqRepo.FindPendingByWorkspace(workspaceID)
}

// ProcessJoinRequest는 참여 요청을 처리합니다 (승인/거부).
// 소유자 또는 관리자만 처리할 수 있습니다.
func (s *WorkspaceService) ProcessJoinRequest(workspaceID, requestID, processorID uuid.UUID, req domain.ProcessJoinRequestRequest) (*domain.WorkspaceJoinRequest, error) {
	// 처리자의 역할 확인
	role, err := s.memberRepo.GetRole(workspaceID, processorID)
	if err != nil {
		return nil, response.NewForbiddenError("Permission denied", "failed to verify role")
	}

	// MEMBER는 처리 불가
	if role == domain.RoleMember {
		return nil, response.NewForbiddenError("Members cannot process join requests", "")
	}

	// 요청 조회
	request, err := s.joinReqRepo.FindByID(requestID)
	if err != nil {
		return nil, response.NewNotFoundError("Join request not found", requestID.String())
	}

	// 이미 처리된 요청인지 확인
	if request.Status != domain.JoinStatusPending {
		return nil, response.NewConflictError("Request already processed", "")
	}

	// 워크스페이스 ID 일치 확인
	if request.WorkspaceID != workspaceID {
		return nil, response.NewBadRequestError("Request does not belong to this workspace", "")
	}

	// 상태 업데이트
	request.Status = req.Status
	request.UpdatedAt = time.Now()

	if err := s.joinReqRepo.Update(request); err != nil {
		s.logger.Error("참여 요청 처리 실패",
			zap.String("request_id", requestID.String()),
			zap.Error(err))
		return nil, err
	}

	// 승인된 경우 멤버로 추가
	if req.Status == domain.JoinStatusApproved {
		member := &domain.WorkspaceMember{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			UserID:      request.UserID,
			RoleName:    domain.RoleMember,
			IsDefault:   false,
			IsActive:    true,
			JoinedAt:    time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.memberRepo.Create(member); err != nil {
			s.logger.Error("멤버 생성 실패 (승인 후)",
				zap.String("workspace_id", workspaceID.String()),
				zap.String("user_id", request.UserID.String()),
				zap.Error(err))
			return nil, err
		}

		// 프로필 생성
		user, _ := s.userRepo.FindByID(request.UserID)
		if user != nil {
			profile := &domain.UserProfile{
				ID:          uuid.New(),
				UserID:      request.UserID,
				WorkspaceID: workspaceID,
				NickName:    user.Name,
				Email:       user.Email,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if profile.NickName == "" {
				profile.NickName = user.Email
			}
			_ = s.profileRepo.Create(profile)
		}

		s.logger.Info("참여 요청 승인 완료",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", request.UserID.String()),
			zap.String("processed_by", processorID.String()))
	} else {
		s.logger.Info("참여 요청 거부됨",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", request.UserID.String()),
			zap.String("processed_by", processorID.String()))
	}

	return request, nil
}
