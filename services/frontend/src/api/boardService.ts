// src/apis/board/index.ts

import { boardServiceClient } from './apiConfig'; // 프로젝트 설정에 맞게 경로 확인
import axios, { AxiosResponse } from 'axios';

import {
  SuccessResponse,
  BoardResponse,
  BoardDetailResponse,
  CreateBoardRequest,
  UpdateBoardRequest,
  MoveBoardRequest, // 추가
  BoardFilters,
  ProjectResponse,
  CreateProjectRequest,
  UpdateProjectRequest,
  PaginatedProjectsResponse,
  ProjectInitSettingsResponse,
  ProjectMemberResponse,
  UpdateProjectMemberRoleRequest,
  ProjectJoinRequestResponse,
  CreateProjectJoinRequestRequest,
  UpdateProjectJoinRequestRequest,
  CommentResponse,
  CreateCommentRequest,
  UpdateCommentRequest,
  ParticipantResponse,
  AddParticipantsRequest, // 변경 (AddParticipantRequest -> AddParticipantsRequest)
  AddParticipantsResponse, // 추가
  FieldOptionResponse,
  CreateFieldOptionRequest,
  UpdateFieldOptionRequest,
  AttachmentResponse,
  PresignedURLRequest, // 추가
  PresignedURLResponse, // 추가
  SaveAttachmentMetadataRequest, // 추가
} from '../types/board';

/**
 * ========================================
 * 목업 모드 설정
 * ========================================
 */
const USE_MOCK_DATA = false;

// ============================================================================
// 💡 [신규] 프로젝트 초기 데이터 로드 API
// ============================================================================

export const getProjectInitSettings = async (
  projectId: string,
): Promise<ProjectInitSettingsResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectInitSettingsResponse>> =
      await boardServiceClient.get(`/projects/${projectId}/init-settings`);
    return response.data.data;
  } catch (error) {
    console.error('getProjectInitSettings error:', error);
    throw error;
  }
};

// ============================================================================
// 프로젝트 관련 API
// ============================================================================

export const getProjects = async (workspaceId: string): Promise<ProjectResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectResponse[]>> =
      await boardServiceClient.get(`/projects/workspace/${workspaceId}`);
    return response.data.data || [];
  } catch (error) {
    console.error('getProjects error:', error);
    throw error;
  }
};

export const getDefaultProject = async (workspaceId: string): Promise<ProjectResponse> => {
  if (USE_MOCK_DATA) {
    return {
      projectId: 'mock-default-project',
      workspaceId: workspaceId,
      name: 'Default Project',
      description: 'Default mock project',
      ownerId: 'mock-owner',
      ownerName: 'Mock Owner',
      ownerEmail: 'owner@example.com',
      isPublic: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      attachments: [],
    };
  }

  try {
    const response: AxiosResponse<SuccessResponse<ProjectResponse>> = await boardServiceClient.get(
      `/projects/workspace/${workspaceId}/default`,
    );
    return response.data.data;
  } catch (error) {
    console.error('getDefaultProject error:', error);
    throw error;
  }
};

export const getProject = async (projectId: string): Promise<ProjectResponse> => {
  if (USE_MOCK_DATA) {
    return {
      projectId: projectId,
      workspaceId: 'mock-workspace-id',
      name: 'Mock Project',
      description: 'Mock project description',
      ownerId: 'mock-owner-id',
      ownerName: 'Mock Owner',
      ownerEmail: 'owner@example.com',
      isPublic: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      attachments: [],
    };
  }

  try {
    const response: AxiosResponse<SuccessResponse<ProjectResponse>> = await boardServiceClient.get(
      `/projects/${projectId}`,
    );
    return response.data.data;
  } catch (error) {
    console.error('getProject error:', error);
    throw error;
  }
};

export const createProject = async (data: CreateProjectRequest): Promise<ProjectResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectResponse>> = await boardServiceClient.post(
      '/projects',
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('createProject error:', error);
    throw error;
  }
};

export const updateProject = async (
  projectId: string,
  data: UpdateProjectRequest,
): Promise<ProjectResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectResponse>> = await boardServiceClient.put(
      `/projects/${projectId}`,
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('updateProject error:', error);
    throw error;
  }
};

export const deleteProject = async (projectId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/projects/${projectId}`);
  } catch (error) {
    console.error('deleteProject error:', error);
    throw error;
  }
};

export const searchProjects = async (
  workspaceId: string,
  query: string,
  page: number = 1,
  limit: number = 10,
): Promise<PaginatedProjectsResponse> => {
  if (USE_MOCK_DATA) {
    return {
      projects: [],
      total: 0,
      page: page,
      limit: limit,
    };
  }

  try {
    const response: AxiosResponse<SuccessResponse<PaginatedProjectsResponse>> =
      await boardServiceClient.get('/projects/search', {
        params: { workspaceId, query, page, limit },
      });
    return response.data.data;
  } catch (error) {
    console.error('searchProjects error:', error);
    throw error;
  }
};

// ============================================================================
// 프로젝트 멤버 관련 API
// ============================================================================

export const getProjectMembers = async (projectId: string): Promise<ProjectMemberResponse[]> => {
  if (USE_MOCK_DATA) {
    return [
      {
        memberId: 'mock-member-1',
        projectId: projectId,
        userId: 'mock-user-1',
        userName: 'Mock User',
        userEmail: 'user@example.com',
        roleName: 'MEMBER',
        joinedAt: new Date().toISOString(),
      },
    ];
  }

  try {
    const response: AxiosResponse<SuccessResponse<ProjectMemberResponse[]>> =
      await boardServiceClient.get(`/projects/${projectId}/members`);
    return response.data.data || [];
  } catch (error) {
    console.error('getProjectMembers error:', error);
    throw error;
  }
};

export const updateProjectMemberRole = async (
  projectId: string,
  memberId: string,
  data: UpdateProjectMemberRoleRequest,
): Promise<ProjectMemberResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectMemberResponse>> =
      await boardServiceClient.put(`/projects/${projectId}/members/${memberId}/role`, data);
    return response.data.data;
  } catch (error) {
    console.error('updateProjectMemberRole error:', error);
    throw error;
  }
};

export const removeProjectMember = async (projectId: string, memberId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/projects/${projectId}/members/${memberId}`);
  } catch (error) {
    console.error('removeProjectMember error:', error);
    throw error;
  }
};

// ============================================================================
// 🔥 프로젝트 온라인 사용자 조회 API (WebSocket 연결 기반)
// ============================================================================

/**
 * 프로젝트에 WebSocket으로 연결된 온라인 사용자 목록을 조회합니다.
 * [API] GET /api/projects/{projectId}/online-users
 */
export const getProjectOnlineUsers = async (projectId: string): Promise<string[]> => {
  try {
    const response: AxiosResponse<{ onlineUsers: string[]; count: number }> =
      await boardServiceClient.get(`/projects/${projectId}/online-users`);
    return response.data?.onlineUsers || [];
  } catch (error) {
    console.error('getProjectOnlineUsers error:', error);
    return []; // 에러 시 빈 배열 반환
  }
};

// ============================================================================
// 프로젝트 가입 요청 관련 API
// ============================================================================

export const getProjectJoinRequests = async (
  projectId: string,
  status?: string,
): Promise<ProjectJoinRequestResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectJoinRequestResponse[]>> =
      await boardServiceClient.get(`/projects/${projectId}/join-requests`, {
        params: { status },
      });
    return response.data.data || [];
  } catch (error) {
    console.error('getProjectJoinRequests error:', error);
    throw error;
  }
};

export const createProjectJoinRequest = async (
  data: CreateProjectJoinRequestRequest,
): Promise<ProjectJoinRequestResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectJoinRequestResponse>> =
      await boardServiceClient.post('/join-requests', data);
    return response.data.data;
  } catch (error) {
    console.error('createProjectJoinRequest error:', error);
    throw error;
  }
};

export const updateProjectJoinRequest = async (
  joinRequestId: string,
  data: UpdateProjectJoinRequestRequest,
): Promise<ProjectJoinRequestResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<ProjectJoinRequestResponse>> =
      await boardServiceClient.put(`/join-requests/${joinRequestId}`, data);
    return response.data.data;
  } catch (error) {
    console.error('updateProjectJoinRequest error:', error);
    throw error;
  }
};

// ============================================================================
// 보드 관련 API
// ============================================================================

export const getBoards = async (
  projectId: string,
  filters?: BoardFilters,
): Promise<BoardResponse[]> => {
  if (USE_MOCK_DATA) {
    return [
      {
        boardId: 'mock-board-1',
        projectId: projectId,
        title: 'Mock Board',
        content: 'Mock content',
        customFields: {
          stage: 'in_progress',
          role: 'developer',
          importance: 'normal',
        },
        authorId: 'mock-author',
        assigneeId: 'mock-assignee',
        dueDate: new Date().toISOString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        attachments: [],
      },
    ];
  }

  try {
    const params: any = { projectId };
    if (filters?.customFields) {
      params.customFields = JSON.stringify(filters.customFields);
    }

    const response: AxiosResponse<SuccessResponse<BoardResponse[]>> = await boardServiceClient.get(
      '/boards',
      { params },
    );
    return response.data.data || [];
  } catch (error) {
    console.error('getBoards error:', error);
    throw error;
  }
};

export const getBoardsByProject = async (
  projectId: string,
  filters?: BoardFilters,
): Promise<BoardResponse[]> => {
  // getBoards와 동일한 로직 또는 별도 엔드포인트 사용
  try {
    const params: any = {};
    if (filters?.customFields) {
      params.customFields = JSON.stringify(filters.customFields);
    }

    const response: AxiosResponse<SuccessResponse<BoardResponse[]>> = await boardServiceClient.get(
      `/boards/project/${projectId}`,
      { params },
    );
    return response.data.data || [];
  } catch (error) {
    console.error('getBoardsByProject error:', error);
    throw error;
  }
};

export const getBoard = async (boardId: string): Promise<BoardDetailResponse> => {
  if (USE_MOCK_DATA) {
    return {
      boardId: boardId,
      projectId: 'mock-project',
      title: 'Mock Detail Board',
      content: 'Content',
      customFields: {},
      authorId: 'author',
      assigneeId: 'assignee',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      attachments: [],
      participants: [],
      comments: [],
    };
  }

  try {
    const response: AxiosResponse<SuccessResponse<BoardDetailResponse>> =
      await boardServiceClient.get(`/boards/${boardId}`);
    return response.data.data;
  } catch (error) {
    console.error('getBoard error:', error);
    throw error;
  }
};

export const createBoard = async (data: CreateBoardRequest): Promise<BoardResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<BoardResponse>> = await boardServiceClient.post(
      '/boards',
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('createBoard error:', error);
    throw error;
  }
};

export const updateBoard = async (
  boardId: string,
  data: UpdateBoardRequest,
): Promise<BoardResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<BoardResponse>> = await boardServiceClient.put(
      `/boards/${boardId}`,
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('updateBoard error:', error);
    throw error;
  }
};

export const deleteBoard = async (boardId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/boards/${boardId}`);
  } catch (error) {
    console.error('deleteBoard error:', error);
    throw error;
  }
};

// ============================================================================
// 보드 이동 API (WebSocket 실시간 동기화용)
// ============================================================================

export const moveBoard = async (boardId: string, data: MoveBoardRequest): Promise<void> => {
  try {
    await boardServiceClient.put(`/boards/${boardId}/move`, data);
  } catch (error) {
    console.error('moveBoard error:', error);
    throw error;
  }
};

// ============================================================================
// 필드 옵션 관련 API
// ============================================================================

export const getFieldOptions = async (
  fieldType: 'stage' | 'role' | 'importance',
): Promise<FieldOptionResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<FieldOptionResponse[]>> =
      await boardServiceClient.get('/field-options', {
        params: { fieldType },
      });
    return response.data.data || [];
  } catch (error) {
    console.error('getFieldOptions error:', error);
    throw error;
  }
};

export const createFieldOption = async (
  data: CreateFieldOptionRequest,
): Promise<FieldOptionResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<FieldOptionResponse>> =
      await boardServiceClient.post('/field-options', data);
    return response.data.data;
  } catch (error) {
    console.error('createFieldOption error:', error);
    throw error;
  }
};

export const updateFieldOption = async (
  optionId: string,
  data: UpdateFieldOptionRequest,
): Promise<FieldOptionResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<FieldOptionResponse>> =
      await boardServiceClient.patch(`/field-options/${optionId}`, data);
    return response.data.data;
  } catch (error) {
    console.error('updateFieldOption error:', error);
    throw error;
  }
};

export const deleteFieldOption = async (optionId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/field-options/${optionId}`);
  } catch (error) {
    console.error('deleteFieldOption error:', error);
    throw error;
  }
};

// ============================================================================
// 댓글 관련 API
// ============================================================================

export const getComments = async (boardId: string): Promise<CommentResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<CommentResponse[]>> =
      await boardServiceClient.get('/comments', {
        params: { boardId },
      });
    return response.data.data || [];
  } catch (error) {
    console.error('getComments error:', error);
    throw error;
  }
};

export const getCommentsByBoard = async (boardId: string): Promise<CommentResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<CommentResponse[]>> =
      await boardServiceClient.get(`/comments/board/${boardId}`);
    return response.data.data || [];
  } catch (error) {
    console.error('getCommentsByBoard error:', error);
    throw error;
  }
};

export const createComment = async (data: CreateCommentRequest): Promise<CommentResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<CommentResponse>> = await boardServiceClient.post(
      '/comments',
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('createComment error:', error);
    throw error;
  }
};

export const updateComment = async (
  commentId: string,
  data: UpdateCommentRequest,
): Promise<CommentResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<CommentResponse>> = await boardServiceClient.put(
      `/comments/${commentId}`,
      data,
    );
    return response.data.data;
  } catch (error) {
    console.error('updateComment error:', error);
    throw error;
  }
};

export const deleteComment = async (commentId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/comments/${commentId}`);
  } catch (error) {
    console.error('deleteComment error:', error);
    throw error;
  }
};

// ============================================================================
// 참여자(Participant) 관련 API
// ============================================================================

export const getParticipants = async (boardId: string): Promise<ParticipantResponse[]> => {
  try {
    const response: AxiosResponse<SuccessResponse<ParticipantResponse[]>> =
      await boardServiceClient.get(`/participants/board/${boardId}`);
    return response.data.data || [];
  } catch (error) {
    console.error('getParticipants error:', error);
    throw error;
  }
};

/**
 * 보드에 참여자를 추가합니다 (Bulk).
 * [API] POST /api/participants
 */
export const addParticipants = async (
  data: AddParticipantsRequest,
): Promise<AddParticipantsResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<AddParticipantsResponse>> =
      await boardServiceClient.post('/participants', data);
    return response.data.data;
  } catch (error) {
    console.error('addParticipants error:', error);
    throw error;
  }
};

export const removeParticipant = async (boardId: string, userId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/participants/board/${boardId}/user/${userId}`);
  } catch (error) {
    console.error('removeParticipant error:', error);
    throw error;
  }
};

// ============================================================================
// 💡 [신규/수정] 첨부파일(Attachment) 관련 API - Presigned URL 방식 적용
// ============================================================================

/**
 * 특정 보드의 첨부파일 목록을 조회합니다.
 * [API] GET /api/boards/{boardId}/attachments
 */
export const getAttachments = async (boardId: string): Promise<AttachmentResponse[]> => {
  if (USE_MOCK_DATA) {
    return [
      {
        attachmentId: 'mock-file-1',
        entityId: boardId,
        entityType: 'BOARD',
        fileName: 'mock.pdf',
        fileUrl: 'http://mock.url/file.pdf',
        contentType: 'application/pdf',
        fileSize: 1024,
        uploadedBy: 'user-1',
        uploadedAt: new Date().toISOString(),
        status: 'UPLOADED',
      },
    ];
  }

  try {
    const response: AxiosResponse<SuccessResponse<AttachmentResponse[]>> =
      await boardServiceClient.get(`/boards/${boardId}/attachments`);
    return response.data.data || [];
  } catch (error) {
    console.error('getAttachments error:', error);
    throw error;
  }
};

/**
 * 1단계: Presigned URL을 요청합니다.
 * [API] POST /api/attachments/presigned-url
 */
export const requestPresignedUrl = async (
  data: PresignedURLRequest,
): Promise<PresignedURLResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<PresignedURLResponse>> =
      await boardServiceClient.post('/attachments/presigned-url', data);
    return response.data.data;
  } catch (error) {
    console.error('requestPresignedUrl error:', error);
    throw error;
  }
};

/**
 * 2단계: S3(또는 스토리지)에 파일을 직접 업로드합니다.
 * (서버 API를 통하지 않고 AWS 등으로 직접 전송하므로 boardServiceClient를 쓰지 않고 순수 axios 사용 권장)
 */
export const uploadFileToS3 = async (uploadUrl: string, file: File): Promise<void> => {
  try {
    // 중요: Presigned URL로 PUT 요청 시 Content-Type이 요청 시점과 일치해야 함
    await axios.put(uploadUrl, file, {
      headers: {
        'Content-Type': file.type,
      },
    });
  } catch (error) {
    console.error('uploadFileToS3 error:', error);
    throw error;
  }
};

/**
 * 3단계: 업로드 완료 후 메타데이터를 서버에 저장합니다.
 * [API] POST /api/attachments
 */
export const saveAttachmentMetadata = async (
  data: SaveAttachmentMetadataRequest,
): Promise<AttachmentResponse> => {
  try {
    const response: AxiosResponse<SuccessResponse<AttachmentResponse & { id?: string }>> =
      await boardServiceClient.post('/attachments', data);
    const rawData = response.data.data;
    // API returns 'id' but frontend expects 'attachmentId' - map accordingly
    return {
      ...rawData,
      attachmentId: rawData.attachmentId || rawData.id || '',
    };
  } catch (error) {
    console.error('saveAttachmentMetadata error:', error);
    throw error;
  }
};

/**
 * [유틸리티 함수] 전체 업로드 프로세스를 한 번에 수행합니다.
 * 1. Presigned URL 획득 -> 2. S3 업로드 -> 3. 메타데이터 저장
 */
export const uploadAttachment = async (
  file: File,
  entityType: 'BOARD' | 'PROJECT' | 'COMMENT',
  workspaceId: string,
): Promise<AttachmentResponse> => {
  try {
    // 1. Presigned URL 요청
    const presignedData = await requestPresignedUrl({
      workspaceId,
      fileName: file.name,
      fileSize: file.size,
      contentType: file.type,
      entityType,
    });

    // 2. S3 업로드
    await uploadFileToS3(presignedData.uploadUrl, file);

    // 3. 메타데이터 저장
    const savedAttachment = await saveAttachmentMetadata({
      entityType,
      fileKey: presignedData.fileKey,
      fileName: file.name,
      fileSize: file.size,
      contentType: file.type,
    });

    return savedAttachment;
  } catch (error) {
    console.error('Full uploadAttachment process error:', error);
    throw error;
  }
};

/**
 * 첨부파일을 다운로드합니다 (바이너리 Blob).
 * [API] GET /api/attachments/{id}/download (엔드포인트는 서버 구현에 따라 다를 수 있음)
 */
export const downloadAttachment = async (attachmentId: string, fileName: string): Promise<void> => {
  try {
    // 주의: 서버 엔드포인트가 /attachments/{id}/download 인지 확인 필요
    const response = await boardServiceClient.get(`/attachments/${attachmentId}/download`, {
      responseType: 'blob',
    });

    const url = window.URL.createObjectURL(new Blob([response.data]));
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', fileName);
    document.body.appendChild(link);
    link.click();

    link.remove();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    console.error('downloadAttachment error:', error);
    throw error;
  }
};

/**
 * 첨부파일을 삭제합니다.
 * [API] DELETE /api/attachments/{id}
 */
export const deleteAttachment = async (attachmentId: string): Promise<void> => {
  try {
    await boardServiceClient.delete(`/attachments/${attachmentId}`);
  } catch (error) {
    console.error('deleteAttachment error:', error);
    throw error;
  }
};
