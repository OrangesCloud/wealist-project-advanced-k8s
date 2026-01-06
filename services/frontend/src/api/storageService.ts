// src/api/storageService.ts

import { storageServiceClient } from './apiConfig';
import axios, { AxiosResponse } from 'axios';
import type {
  StorageFolder,
  StorageFile,
  StorageShare,
  StorageUsage,
  StorageProject,
  ProjectMember,
  ProjectPermission,
  ProjectStats,
  CreateFolderRequest,
  UpdateFolderRequest,
  CreateProjectRequest,
  UpdateProjectRequest,
  AddProjectMemberRequest,
  UpdateProjectMemberRequest,
  FileListResponse,
  GenerateUploadURLRequest,
  GenerateUploadURLResponse,
  ConfirmUploadRequest,
  UpdateFileRequest,
  CreateShareRequest,
  UpdateShareRequest,
  SharedItem,
  FolderContentsResponse,
} from '../types/storage';

// ============================================================================
// API Response Wrapper (Backend에서 사용하는 공통 응답 형식)
// ============================================================================

interface SuccessResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

// ============================================================================
// Project API
// ============================================================================

/**
 * 프로젝트 생성
 * [API] POST /storage/projects
 */
export const createProject = async (data: CreateProjectRequest): Promise<StorageProject> => {
  const response: AxiosResponse<SuccessResponse<StorageProject>> = await storageServiceClient.post(
    '/storage/projects',
    data,
  );
  return response.data.data;
};

/**
 * 프로젝트 조회
 * [API] GET /storage/projects/{projectId}
 */
export const getProject = async (projectId: string): Promise<StorageProject> => {
  const response: AxiosResponse<SuccessResponse<StorageProject>> = await storageServiceClient.get(
    `/storage/projects/${projectId}`,
  );
  return response.data.data;
};

/**
 * 워크스페이스의 프로젝트 목록 조회
 * [API] GET /storage/workspaces/{workspaceId}/projects
 */
export const getWorkspaceProjects = async (workspaceId: string): Promise<StorageProject[]> => {
  const response: AxiosResponse<SuccessResponse<StorageProject[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/projects`,
  );
  return response.data.data || [];
};

/**
 * 프로젝트 수정
 * [API] PUT /storage/projects/{projectId}
 */
export const updateProject = async (
  projectId: string,
  data: UpdateProjectRequest,
): Promise<StorageProject> => {
  const response: AxiosResponse<SuccessResponse<StorageProject>> = await storageServiceClient.put(
    `/storage/projects/${projectId}`,
    data,
  );
  return response.data.data;
};

/**
 * 프로젝트 삭제
 * [API] DELETE /storage/projects/{projectId}
 */
export const deleteProject = async (projectId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/projects/${projectId}`);
};

/**
 * 프로젝트 멤버 추가
 * [API] POST /storage/projects/{projectId}/members
 */
export const addProjectMember = async (
  projectId: string,
  data: AddProjectMemberRequest,
): Promise<ProjectMember> => {
  const response: AxiosResponse<SuccessResponse<ProjectMember>> = await storageServiceClient.post(
    `/storage/projects/${projectId}/members`,
    data,
  );
  return response.data.data;
};

/**
 * 프로젝트 멤버 목록 조회
 * [API] GET /storage/projects/{projectId}/members
 */
export const getProjectMembers = async (projectId: string): Promise<ProjectMember[]> => {
  const response: AxiosResponse<SuccessResponse<ProjectMember[]>> = await storageServiceClient.get(
    `/storage/projects/${projectId}/members`,
  );
  return response.data.data || [];
};

/**
 * 프로젝트 멤버 권한 수정
 * [API] PUT /storage/projects/{projectId}/members/{memberId}
 */
export const updateProjectMember = async (
  projectId: string,
  memberId: string,
  data: UpdateProjectMemberRequest,
): Promise<ProjectMember> => {
  const response: AxiosResponse<SuccessResponse<ProjectMember>> = await storageServiceClient.put(
    `/storage/projects/${projectId}/members/${memberId}`,
    data,
  );
  return response.data.data;
};

/**
 * 프로젝트 멤버 제거
 * [API] DELETE /storage/projects/{projectId}/members/{memberId}
 */
export const removeProjectMember = async (projectId: string, memberId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/projects/${projectId}/members/${memberId}`);
};

/**
 * 현재 사용자의 프로젝트 권한 조회
 * [API] GET /storage/projects/{projectId}/my-permission
 */
export const getMyProjectPermission = async (projectId: string): Promise<ProjectPermission | null> => {
  try {
    const response: AxiosResponse<SuccessResponse<{ permission: ProjectPermission }>> =
      await storageServiceClient.get(`/storage/projects/${projectId}/my-permission`);
    return response.data.data?.permission || null;
  } catch {
    return null;
  }
};

/**
 * 프로젝트 통계 조회
 * [API] GET /storage/projects/{projectId}/stats
 */
export const getProjectStats = async (projectId: string): Promise<ProjectStats> => {
  const response: AxiosResponse<SuccessResponse<ProjectStats>> = await storageServiceClient.get(
    `/storage/projects/${projectId}/stats`,
  );
  return response.data.data;
};

// ============================================================================
// Folder API
// ============================================================================

/**
 * 폴더 생성
 * [API] POST /storage/folders
 */
export const createFolder = async (data: CreateFolderRequest): Promise<StorageFolder> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder>> = await storageServiceClient.post(
    '/storage/folders',
    data,
  );
  return response.data.data;
};

/**
 * 폴더 조회
 * [API] GET /storage/folders/{folderId}
 */
export const getFolder = async (folderId: string): Promise<StorageFolder> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder>> = await storageServiceClient.get(
    `/storage/folders/${folderId}`,
  );
  return response.data.data;
};

/**
 * 폴더 내용 조회 (서브폴더 + 파일)
 * [API] GET /storage/folders/contents?workspaceId=...&folderId=...
 */
export const getFolderContents = async (
  workspaceId: string,
  folderId: string,
): Promise<FolderContentsResponse> => {
  const response: AxiosResponse<SuccessResponse<FolderContentsResponse>> =
    await storageServiceClient.get('/storage/folders/contents', {
      params: { workspaceId, folderId },
    });
  // API 응답이 null/undefined인 경우 빈 children/files 배열을 가진 기본 객체 반환
  return response.data.data ?? ({
    id: '',
    workspaceId,
    name: '',
    path: '/',
    createdBy: '',
    createdAt: '',
    updatedAt: '',
    isDeleted: false,
    children: [],
    files: [],
    fileCount: 0,
    folderCount: 0,
    totalSize: 0,
  } as FolderContentsResponse);
};

/**
 * 워크스페이스 루트 폴더 목록 조회
 * [API] GET /storage/workspaces/{workspaceId}/folders
 */
export const getRootFolders = async (workspaceId: string): Promise<StorageFolder[]> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/folders`,
  );
  return response.data.data || [];
};

/**
 * 워크스페이스 루트 내용 조회 (폴더 + 파일)
 * [API] GET /storage/folders/contents?workspaceId=...
 */
export const getRootContents = async (workspaceId: string): Promise<FolderContentsResponse> => {
  const response: AxiosResponse<SuccessResponse<FolderContentsResponse>> =
    await storageServiceClient.get('/storage/folders/contents', {
      params: { workspaceId },
    });
  // API 응답이 null/undefined인 경우 빈 children/files 배열을 가진 기본 객체 반환
  return response.data.data ?? ({
    id: '',
    workspaceId,
    name: '',
    path: '/',
    createdBy: '',
    createdAt: '',
    updatedAt: '',
    isDeleted: false,
    children: [],
    files: [],
    fileCount: 0,
    folderCount: 0,
    totalSize: 0,
  } as FolderContentsResponse);
};

/**
 * 폴더 트리 조회
 * [API] GET /storage/workspaces/{workspaceId}/folders
 */
export const getFolderTree = async (workspaceId: string): Promise<StorageFolder[]> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/folders`,
  );
  return response.data.data || [];
};

/**
 * 폴더 수정
 * [API] PUT /storage/folders/{folderId}
 */
export const updateFolder = async (
  folderId: string,
  data: UpdateFolderRequest,
): Promise<StorageFolder> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder>> = await storageServiceClient.put(
    `/storage/folders/${folderId}`,
    data,
  );
  return response.data.data;
};

/**
 * 폴더 삭제 (휴지통으로 이동)
 * [API] DELETE /storage/folders/{folderId}
 */
export const deleteFolder = async (folderId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/folders/${folderId}`);
};

/**
 * 폴더 영구 삭제
 * [API] DELETE /storage/folders/{folderId}/permanent
 */
export const deleteFolderPermanent = async (folderId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/folders/${folderId}/permanent`);
};

/**
 * 폴더 복원
 * [API] POST /storage/folders/{folderId}/restore
 */
export const restoreFolder = async (folderId: string): Promise<StorageFolder> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder>> = await storageServiceClient.post(
    `/storage/folders/${folderId}/restore`,
  );
  return response.data.data;
};

// ============================================================================
// File API
// ============================================================================

/**
 * 업로드 URL 생성
 * [API] POST /storage/files/upload-url
 */
export const generateUploadURL = async (
  data: GenerateUploadURLRequest,
): Promise<GenerateUploadURLResponse> => {
  const response: AxiosResponse<SuccessResponse<GenerateUploadURLResponse>> =
    await storageServiceClient.post('/storage/files/upload-url', data);
  return response.data.data;
};

/**
 * S3에 직접 파일 업로드
 */
export const uploadFileToS3 = async (uploadUrl: string, file: File): Promise<void> => {
  await axios.put(uploadUrl, file, {
    headers: {
      'Content-Type': file.type,
    },
  });
};

/**
 * 업로드 확인
 * [API] POST /storage/files/confirm
 */
export const confirmUpload = async (data: ConfirmUploadRequest): Promise<StorageFile> => {
  const response: AxiosResponse<SuccessResponse<StorageFile>> = await storageServiceClient.post(
    '/storage/files/confirm',
    data,
  );
  return response.data.data;
};

/**
 * 전체 파일 업로드 프로세스
 * 1. Upload URL 생성 -> 2. S3 업로드 -> 3. 업로드 확인
 */
export const uploadFile = async (
  file: File,
  workspaceId: string,
  folderId?: string,
  onProgress?: (progress: number) => void,
): Promise<StorageFile> => {
  // 1. Upload URL 생성
  const uploadUrlResponse = await generateUploadURL({
    workspaceId,
    folderId,
    fileName: file.name,
    contentType: file.type,
    fileSize: file.size,
  });

  // 2. S3에 직접 업로드
  await axios.put(uploadUrlResponse.uploadUrl, file, {
    headers: {
      'Content-Type': file.type,
    },
    onUploadProgress: (progressEvent) => {
      if (onProgress && progressEvent.total) {
        const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
        onProgress(progress);
      }
    },
  });

  // 3. 업로드 확인
  const confirmedFile = await confirmUpload({
    fileId: uploadUrlResponse.fileId,
  });

  return confirmedFile;
};

/**
 * 파일 조회
 * [API] GET /storage/files/{fileId}
 */
export const getFile = async (fileId: string): Promise<StorageFile> => {
  const response: AxiosResponse<SuccessResponse<StorageFile>> = await storageServiceClient.get(
    `/storage/files/${fileId}`,
  );
  return response.data.data;
};

/**
 * 폴더 내 파일 목록 조회
 * [API] GET /storage/files/folder/{folderId}
 */
export const getFilesByFolder = async (
  folderId: string,
  page: number = 1,
  pageSize: number = 50,
): Promise<FileListResponse> => {
  const response: AxiosResponse<SuccessResponse<FileListResponse>> = await storageServiceClient.get(
    `/storage/files/folder/${folderId}`,
    {
      params: { page, pageSize },
    },
  );
  return response.data.data;
};

/**
 * 워크스페이스 루트 파일 목록 조회
 * [API] GET /storage/files/workspace/{workspaceId}/root
 */
export const getRootFiles = async (
  workspaceId: string,
  page: number = 1,
  pageSize: number = 50,
): Promise<FileListResponse> => {
  const response: AxiosResponse<SuccessResponse<FileListResponse>> = await storageServiceClient.get(
    `/storage/files/workspace/${workspaceId}/root`,
    {
      params: { page, pageSize },
    },
  );
  return response.data.data;
};

/**
 * 파일 수정
 * [API] PUT /storage/files/{fileId}
 */
export const updateFile = async (fileId: string, data: UpdateFileRequest): Promise<StorageFile> => {
  const response: AxiosResponse<SuccessResponse<StorageFile>> = await storageServiceClient.put(
    `/storage/files/${fileId}`,
    data,
  );
  return response.data.data;
};

/**
 * 파일 삭제 (휴지통으로 이동)
 * [API] DELETE /storage/files/{fileId}
 */
export const deleteFile = async (fileId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/files/${fileId}`);
};

/**
 * 파일 영구 삭제
 * [API] DELETE /storage/files/{fileId}/permanent
 */
export const deleteFilePermanent = async (fileId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/files/${fileId}/permanent`);
};

/**
 * 파일 복원
 * [API] POST /storage/files/{fileId}/restore
 */
export const restoreFile = async (fileId: string): Promise<StorageFile> => {
  const response: AxiosResponse<SuccessResponse<StorageFile>> = await storageServiceClient.post(
    `/storage/files/${fileId}/restore`,
  );
  return response.data.data;
};

/**
 * 파일 다운로드 URL 생성
 * [API] GET /storage/files/{fileId}/download
 */
export const getDownloadURL = async (fileId: string): Promise<string> => {
  const response: AxiosResponse<SuccessResponse<{ downloadUrl: string }>> =
    await storageServiceClient.get(`/storage/files/${fileId}/download`);
  return response.data.data.downloadUrl;
};

/**
 * 파일 다운로드 (브라우저)
 */
export const downloadFile = async (fileId: string, fileName: string): Promise<void> => {
  const downloadUrl = await getDownloadURL(fileId);
  const link = document.createElement('a');
  link.href = downloadUrl;
  link.setAttribute('download', fileName);
  document.body.appendChild(link);
  link.click();
  link.remove();
};

// ============================================================================
// Share API
// ============================================================================

/**
 * 공유 생성
 * [API] POST /storage/shares
 */
export const createShare = async (data: CreateShareRequest): Promise<StorageShare> => {
  const response: AxiosResponse<SuccessResponse<StorageShare>> = await storageServiceClient.post(
    '/storage/shares',
    data,
  );
  return response.data.data;
};

/**
 * 공유 조회
 * [API] GET /storage/shares/{shareId}
 */
export const getShare = async (shareId: string): Promise<StorageShare> => {
  const response: AxiosResponse<SuccessResponse<StorageShare>> = await storageServiceClient.get(
    `/storage/shares/${shareId}`,
  );
  return response.data.data;
};

/**
 * 엔티티의 공유 목록 조회
 * [API] GET /storage/shares/entity/{entityType}/{entityId}
 */
export const getSharesByEntity = async (
  entityType: 'FILE' | 'FOLDER',
  entityId: string,
): Promise<StorageShare[]> => {
  const response: AxiosResponse<SuccessResponse<StorageShare[]>> = await storageServiceClient.get(
    `/storage/shares/entity/${entityType}/${entityId}`,
  );
  return response.data.data || [];
};

/**
 * 나에게 공유된 항목 목록
 * [API] GET /storage/shared-with-me
 */
export const getSharedWithMe = async (): Promise<SharedItem[]> => {
  const response: AxiosResponse<SuccessResponse<SharedItem[]>> = await storageServiceClient.get(
    '/storage/shared-with-me',
  );
  return response.data.data || [];
};

/**
 * 공유 수정
 * [API] PUT /storage/shares/{shareId}
 */
export const updateShare = async (
  shareId: string,
  data: UpdateShareRequest,
): Promise<StorageShare> => {
  const response: AxiosResponse<SuccessResponse<StorageShare>> = await storageServiceClient.put(
    `/storage/shares/${shareId}`,
    data,
  );
  return response.data.data;
};

/**
 * 공유 삭제
 * [API] DELETE /storage/shares/{shareId}
 */
export const deleteShare = async (shareId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/shares/${shareId}`);
};

/**
 * 공유 링크로 접근
 * [API] GET /public/storage/share/{shareLink}
 */
export const accessShareLink = async (
  shareLink: string,
): Promise<{
  share: StorageShare;
  file?: StorageFile;
  folder?: StorageFolder;
}> => {
  const response: AxiosResponse<
    SuccessResponse<{
      share: StorageShare;
      file?: StorageFile;
      folder?: StorageFolder;
    }>
  > = await storageServiceClient.get(`/public/storage/share/${shareLink}`);
  return response.data.data;
};

// ============================================================================
// Storage Usage API
// ============================================================================

/**
 * 워크스페이스 스토리지 사용량 조회
 * [API] GET /storage/workspaces/{workspaceId}/usage
 */
export const getStorageUsage = async (workspaceId: string): Promise<StorageUsage> => {
  const response: AxiosResponse<SuccessResponse<StorageUsage>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/usage`,
  );
  return response.data.data;
};

// ============================================================================
// Trash API (휴지통)
// ============================================================================

/**
 * 휴지통 목록 조회 (삭제된 폴더)
 * [API] GET /storage/workspaces/{workspaceId}/trash/folders
 */
export const getTrashFolders = async (workspaceId: string): Promise<StorageFolder[]> => {
  const response: AxiosResponse<SuccessResponse<StorageFolder[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/trash/folders`,
  );
  return response.data.data || [];
};

/**
 * 휴지통 목록 조회 (삭제된 파일)
 * [API] GET /storage/workspaces/{workspaceId}/trash/files
 */
export const getTrashFiles = async (workspaceId: string): Promise<StorageFile[]> => {
  const response: AxiosResponse<SuccessResponse<StorageFile[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/trash/files`,
  );
  return response.data.data || [];
};

/**
 * 휴지통 비우기
 * [API] DELETE /storage/trash/workspace/{workspaceId}/empty
 */
export const emptyTrash = async (workspaceId: string): Promise<void> => {
  await storageServiceClient.delete(`/storage/trash/workspace/${workspaceId}/empty`);
};

// ============================================================================
// Search API
// ============================================================================

/**
 * 파일/폴더 검색
 * [API] GET /storage/workspaces/{workspaceId}/files/search
 */
export const searchStorage = async (
  workspaceId: string,
  query: string,
  type?: 'file' | 'folder' | 'all',
): Promise<{ files: StorageFile[]; folders: StorageFolder[] }> => {
  const response: AxiosResponse<
    SuccessResponse<{ files: StorageFile[]; folders: StorageFolder[] }>
  > = await storageServiceClient.get(`/storage/workspaces/${workspaceId}/files/search`, {
    params: { query, type: type || 'all' },
  });
  return response.data.data;
};

// ============================================================================
// Recent Files API
// ============================================================================

/**
 * 최근 파일 목록 조회
 * [API] GET /storage/workspaces/{workspaceId}/files
 * Note: Backend에서 최근 파일 정렬을 지원하지 않으므로 전체 파일 목록에서 최근 순으로 정렬
 */
export const getRecentFiles = async (
  workspaceId: string,
  limit: number = 20,
): Promise<StorageFile[]> => {
  const response: AxiosResponse<SuccessResponse<StorageFile[]>> = await storageServiceClient.get(
    `/storage/workspaces/${workspaceId}/files`,
  );
  const files = response.data.data || [];
  // 최근 수정된 순으로 정렬 후 limit만큼 반환
  return files
    .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
    .slice(0, limit);
};

// ============================================================================
// Starred Files/Folders API (즐겨찾기)
// Note: Backend에서 아직 즐겨찾기 기능을 지원하지 않으므로 stub 구현
// ============================================================================

/**
 * 즐겨찾기 추가
 * @todo Backend에 즐겨찾기 API 구현 필요
 */
export const addToStarred = async (): // entityType: 'FILE' | 'FOLDER',
// entityId: string,
Promise<void> => {
  console.warn('Starred feature not yet implemented in backend');
  // TODO: await storageServiceClient.post('/storage/starred', { entityType, entityId });
};

/**
 * 즐겨찾기 제거
 * @todo Backend에 즐겨찾기 API 구현 필요
 */
export const removeFromStarred = async (): // entityType: 'FILE' | 'FOLDER',
// entityId: string
Promise<void> => {
  console.warn('Starred feature not yet implemented in backend');
  // TODO: await storageServiceClient.delete(`/storage/starred/${entityType}/${entityId}`);
};

/**
 * 즐겨찾기 목록 조회
 * @todo Backend에 즐겨찾기 API 구현 필요
 */
export const getStarredItems = async (): // workspaceId: string
Promise<{ files: StorageFile[]; folders: StorageFolder[] }> => {
  console.warn('Starred feature not yet implemented in backend');
  // 빈 배열 반환 (Backend 미구현)
  return { files: [], folders: [] };
};
