// src/api/userService.ts

import {
  CreateWorkspaceRequest,
  UpdateProfileRequest,
  UpdateWorkspaceSettingsRequest,
  UserProfileResponse,
  WorkspaceResponse,
  WorkspaceMemberResponse,
  WorkspaceMemberRole,
  WorkspaceSettingsResponse,
  JoinRequestResponse,
  InviteUserRequest,
  UserWorkspaceResponse,
  SaveAttachmentRequest,
  // 💡 [수정] Attachment ID 기반 요청 DTO를 명시적으로 사용
  UpdateProfileImageRequest,
  PresignedUrlRequest,
  PresignedUrlResponse,
  AttachmentResponse,
} from '../types/user';
import { userRepoClient } from './apiConfig';
import { AxiosResponse } from 'axios';

// ========================================
// Workspace API Functions (워크스페이스 전체 관리)
// ========================================

/**
 * 워크스페이스 목록 조회 (현재 사용자가 속한 모든 워크스페이스)
 * [API] GET /api/workspaces/all
 */
export const getMyWorkspaces = async (): Promise<UserWorkspaceResponse[]> => {
  const response: AxiosResponse<UserWorkspaceResponse[]> = await userRepoClient.get(
    '/api/workspaces/all',
  );
  console.log(response.data);
  return response.data;
};

/**
 * 퍼블릭 워크스페이스 목록 조회 (GET /api/workspaces/public/{workspaceName})
 * @param workspaceName 검색/필터링할 워크스페이스 이름
 * @returns WorkspaceResponse 배열을 담은 Promise
 */
export const getPublicWorkspaces = async (workspaceName: string): Promise<WorkspaceResponse[]> => {
  const response: AxiosResponse<WorkspaceResponse[]> = await userRepoClient.get(
    `/api/workspaces/public/${workspaceName}`,
  );
  return response.data;
};

/**
 * 특정 워크스페이스 조회
 * [API] GET /api/workspaces/{workspaceId}
 * * Response: { data: WorkspaceResponse }
 */
export const getWorkspace = async (workspaceId: string): Promise<WorkspaceResponse> => {
  const response: AxiosResponse<{ data: WorkspaceResponse }> = await userRepoClient.get(
    `/api/workspaces/${workspaceId}`,
  );
  return response.data.data; // data 필드 추출
};

/**
 * 워크스페이스 생성
 * [API] POST /api/workspaces/create
 * * Response: WorkspaceResponse
 */
export const createWorkspace = async (data: CreateWorkspaceRequest): Promise<WorkspaceResponse> => {
  const response: AxiosResponse<WorkspaceResponse> = await userRepoClient.post(
    '/api/workspaces/create',
    data,
  );
  return response.data;
};

/**
 * 워크스페이스 수정
 * [API] PUT /api/workspaces/{workspaceId}
 * * Response: { data: WorkspaceResponse }
 */
export const updateWorkspace = async (
  workspaceId: string,
  data: { workspaceName?: string; workspaceDescription?: string },
): Promise<WorkspaceResponse> => {
  const response: AxiosResponse<{ data: WorkspaceResponse }> = await userRepoClient.put(
    `/api/workspaces/${workspaceId}`,
    data,
  );
  return response.data.data; // data 필드 추출
};

/**
 * 워크스페이스 삭제 (소프트 삭제)
 * [API] DELETE /api/workspaces/{workspaceId}
 */
export const deleteWorkspace = async (workspaceId: string): Promise<void> => {
  await userRepoClient.delete(`/api/workspaces/${workspaceId}`);
};

/**
 * 워크스페이스 설정 조회
 * [API] GET /api/workspaces/{workspaceId}/settings
 */
export const getWorkspaceSettings = async (
  workspaceId: string,
): Promise<WorkspaceSettingsResponse> => {
  const response: AxiosResponse<WorkspaceSettingsResponse> = await userRepoClient.get(
    `/api/workspaces/${workspaceId}/settings`,
  );
  return response.data;
};

/**
 * 워크스페이스 설정 수정
 * [API] PUT /api/workspaces/{workspaceId}/settings
 */
export const updateWorkspaceSettings = async (
  workspaceId: string,
  data: UpdateWorkspaceSettingsRequest,
): Promise<WorkspaceSettingsResponse> => {
  const response: AxiosResponse<WorkspaceSettingsResponse> = await userRepoClient.put(
    `/api/workspaces/${workspaceId}/settings`,
    data,
  );
  return response.data;
};

// ========================================
// Member & Join Request API Functions
// ========================================

/**
 * 워크스페이스 회원 목록 조회
 * [API] GET /api/workspaces/{workspaceId}/members
 */
export const getWorkspaceMembers = async (
  workspaceId: string,
): Promise<WorkspaceMemberResponse[]> => {
  const response: AxiosResponse<WorkspaceMemberResponse[]> = await userRepoClient.get(
    `/api/workspaces/${workspaceId}/members`,
  );
  return response.data;
};

/**
 * 승인 대기 회원 목록 조회
 * [API] GET /api/workspaces/{workspaceId}/pendingMembers
 */
export const getPendingMembers = async (workspaceId: string): Promise<JoinRequestResponse[]> => {
  const response: AxiosResponse<JoinRequestResponse[]> = await userRepoClient.get(
    `/api/workspaces/${workspaceId}/pendingMembers`,
  );
  return response.data;
};

/**
 * 가입 신청 승인
 * [API] POST /api/workspaces/{workspaceId}/members/{userId}/approve
 */
export const approveMember = async (workspaceId: string, userId: string): Promise<void> => {
  await userRepoClient.post(`/api/workspaces/${workspaceId}/members/${userId}/approve`, {});
};

/**
 * 가입 신청 거절
 * [API] POST /api/workspaces/{workspaceId}/members/{userId}/reject
 */
export const rejectMember = async (workspaceId: string, userId: string): Promise<void> => {
  await userRepoClient.post(`/api/workspaces/${workspaceId}/members/${userId}/reject`, {});
};

/**
 * 워크스페이스에 사용자 초대 (이메일 기준)
 * [API] POST /api/workspaces/{workspaceId}/members/invite
 * * Response: { data: WorkspaceMemberResponse }
 */
export const inviteUser = async (
  workspaceId: string,
  email: string,
): Promise<WorkspaceMemberResponse> => {
  const data: InviteUserRequest = { email };

  const response: AxiosResponse<{ data: WorkspaceMemberResponse }> = await userRepoClient.post(
    `/api/workspaces/${workspaceId}/members/invite`,
    data,
  );
  return response.data.data; // data 필드 추출
};

/**
 * 멤버 역할 변경
 * [API] PUT /api/workspaces/{workspaceId}/members/{memberId}/role
 * * Response: { data: WorkspaceMemberResponse }
 */
export const updateMemberRole = async (
  workspaceId: string,
  memberId: string,
  roleName: WorkspaceMemberRole, // DTO 타입을 사용
): Promise<WorkspaceMemberResponse> => {
  const data = { roleName };

  const response: AxiosResponse<{ data: WorkspaceMemberResponse }> = await userRepoClient.put(
    `/api/workspaces/${workspaceId}/members/${memberId}/role`,
    data,
  );
  return response.data.data; // data 필드 추출
};

/**
 * 멤버 제거
 * [API] DELETE /api/workspaces/{workspaceId}/members/{memberId}
 */
export const removeMember = async (workspaceId: string, memberId: string): Promise<void> => {
  await userRepoClient.delete(`/api/workspaces/${workspaceId}/members/${memberId}`);
};

/**
 * 가입 신청 목록 조회 (status 필터 가능)
 * [API] GET /api/workspaces/{workspaceId}/joinRequests
 * * Response: { data: JoinRequestResponse[] }
 */
export const getJoinRequests = async (
  workspaceId: string,
  status?: string, // 'PENDING', 'APPROVED', 'REJECTED'
): Promise<JoinRequestResponse[]> => {
  const response: AxiosResponse<{ data: JoinRequestResponse[] }> = await userRepoClient.get(
    `/api/workspaces/${workspaceId}/joinRequests`,
    { params: { status } },
  );
  return response.data.data; // data 필드 추출
};

/**
 * 워크스페이스 가입 신청
 * [API] POST /api/workspaces/join-requests
 * * Response: JoinRequestResponse (API 스펙에 따라 data 필드가 없을 수 있음)
 */
export const createJoinRequest = async (workspaceId: string): Promise<JoinRequestResponse> => {
  const data = { workspaceId };
  const response: AxiosResponse<JoinRequestResponse> = await userRepoClient.post(
    '/api/workspaces/join-requests',
    data,
  );
  console.log(response.data);
  return response.data;
};

// ========================================
// UserProfile API Functions
// ========================================

/**
 * 내 프로필 조회 (워크스페이스별 프로필)
 * [API] GET /api/profiles/me
 * @param workspaceId - 워크스페이스 ID (X-Workspace-Id 헤더로 전송)
 * * Response: { data: UserProfileResponse }
 */
export const getMyProfile = async (workspaceId: string): Promise<UserProfileResponse> => {
  const response: AxiosResponse<UserProfileResponse> = await userRepoClient.get('/api/profiles/me', {
    headers: {
      'X-Workspace-Id': workspaceId,
    },
  });
  return response.data;
};

/**
 * 내 모든 프로필 조회 (기본 프로필 + 워크스페이스별 프로필)
 * [API] GET /api/profiles/all/me
 * * Response: UserProfileResponse[] (API 스펙에 따라 data 필드가 없을 수 있음)
 */
export const getAllMyProfiles = async (): Promise<UserProfileResponse[]> => {
  const response: AxiosResponse<UserProfileResponse[]> = await userRepoClient.get(
    '/api/profiles/all/me',
  );
  console.log(response.data);
  return response.data;
};

/**
 * 내 프로필 정보 통합 업데이트 (닉네임/이메일 등)
 * [API] PUT /api/profiles/me
 * * Header: X-Workspace-Id required
 * * Response: { data: UserProfileResponse }
 */
export const updateMyProfile = async (data: UpdateProfileRequest): Promise<UserProfileResponse> => {
  const response: AxiosResponse<{ data: UserProfileResponse }> = await userRepoClient.put(
    '/api/profiles/me',
    data,
    {
      headers: {
        'X-Workspace-Id': data.workspaceId,
      },
    },
  );
  return response.data.data; // data 필드 추출
};

// ========================================
// New API Functions (기타)
// ========================================

/**
 * 기본 워크스페이스 설정
 * [API] POST /api/workspaces/default
 */
export const setDefaultWorkspace = async (workspaceId: string): Promise<void> => {
  const data = { workspaceId };
  await userRepoClient.post('/api/workspaces/default', data);
};

// ========================================
// Profile Image Upload API Functions
// ========================================

/**
 * 프로필 이미지 업로드를 위한 Presigned URL 생성
 * [API] POST /api/profiles/me/image/presigned-url
 */
export const generateProfilePresignedUrl = async (
  data: PresignedUrlRequest,
): Promise<PresignedUrlResponse> => {
  const response: AxiosResponse<PresignedUrlResponse> = await userRepoClient.post(
    '/api/profiles/me/image/presigned-url',
    data,
  );
  return response.data;
};

/**
 * 프로필 이미지 첨부파일 메타데이터 저장
 * [API] POST /api/profiles/me/image/attachment
 * 💡 [수정] 응답 타입이 AttachmentResponse임을 명시
 */
export const saveProfileAttachmentMetadata = async (
  data: SaveAttachmentRequest,
): Promise<AttachmentResponse> => {
  const response: AxiosResponse<AttachmentResponse> = await userRepoClient.post(
    '/api/profiles/me/image/attachment',
    data,
  );
  return response.data;
};

/**
 * 💡 [추가/수정] 프로필 이미지 업데이트 (Attachment ID 기반)
 * [API] PUT /api/profiles/me/image
 * * DTO: { workspaceId, attachmentId }
 * * Header: X-Workspace-Id required
 */
export const updateProfileImage = async (
  workspaceId: string,
  attachmentId: string,
): Promise<UserProfileResponse> => {
  const data: UpdateProfileImageRequest = {
    workspaceId,
    attachmentId,
  };
  const response: AxiosResponse<UserProfileResponse> = await userRepoClient.put(
    '/api/profiles/me/image',
    data,
    {
      headers: {
        'X-Workspace-Id': workspaceId,
      },
    },
  );
  return response.data;
};

/**
 * 💡 [수정] 프로필 이미지 업로드 헬퍼 함수 (전체 플로우 중 Attachment 저장까지)
 * 1. Presigned URL 생성
 * 2. S3에 직접 업로드
 * 3. 메타데이터 저장 -> AttachmentResponse 반환
 * * @returns AttachmentResponse Attachment ID가 포함된 객체
 */
export const uploadProfileImage = async (
  file: File,
  workspaceId: string,
): Promise<AttachmentResponse> => {
  try {
    // 1. Presigned URL 요청
    const presignedData: PresignedUrlRequest = {
      workspaceId,
      fileName: file.name,
      fileSize: file.size,
      contentType: file.type,
    };

    const { uploadUrl, fileKey } = await generateProfilePresignedUrl(presignedData);

    // 2. S3에 파일 업로드
    const uploadResponse = await fetch(uploadUrl, {
      method: 'PUT',
      headers: {
        'Content-Type': file.type,
      },
      body: file,
    });

    if (!uploadResponse.ok) {
      const errorText = await uploadResponse.text();
      console.error('S3 업로드 실패:', uploadResponse.status, errorText);
      throw new Error(`S3 업로드 실패: ${uploadResponse.status}`);
    }

    // 3. 첨부파일 메타데이터 저장 및 Attachment ID 반환
    const attachmentData: SaveAttachmentRequest = {
      fileKey,
      fileName: file.name,
      fileSize: file.size,
      contentType: file.type,
    };

    // AttachmentResponse 반환 (ID 포함)
    return await saveProfileAttachmentMetadata(attachmentData);
  } catch (error) {
    console.error('프로필 이미지 업로드 실패:', error);
    throw error;
  }
};

// 🚨 [제거] 기존 updateProfileImageByKey는 updateProfileImage로 대체됨.
// export const updateProfileImageByKey = async (
//   data: UpdateProfileImageByKeyRequest,
// ): Promise<UserProfileResponse> => {
//   // ... (로직 제거)
// };
