// src/types/storage.ts

// =======================================================
// Storage Types (Google Drive-like Storage)
// =======================================================

/**
 * 파일 상태
 */
export type FileStatus = 'UPLOADING' | 'ACTIVE' | 'DELETED';

/**
 * 공유 타입
 */
export type ShareType = 'FILE' | 'FOLDER';

/**
 * 권한 레벨 (공유용)
 */
export type PermissionLevel = 'VIEWER' | 'COMMENTER' | 'EDITOR';

/**
 * 프로젝트 권한 레벨
 */
export type ProjectPermission = 'OWNER' | 'EDITOR' | 'VIEWER';

/**
 * 뷰 모드
 */
export type ViewMode = 'grid' | 'list';

/**
 * 정렬 기준
 */
export type SortBy = 'name' | 'modifiedAt' | 'size' | 'type';

/**
 * 정렬 방향
 */
export type SortDirection = 'asc' | 'desc';

// =======================================================
// Project Types
// =======================================================

/**
 * 프로젝트 응답 DTO
 */
export interface StorageProject {
  id: string;
  workspaceId: string;
  name: string;
  description?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 프로젝트 멤버 응답 DTO
 */
export interface ProjectMember {
  id: string;
  projectId: string;
  userId: string;
  userName?: string;
  userEmail?: string;
  permission: ProjectPermission;
  createdAt: string;
  updatedAt: string;
}

/**
 * 프로젝트 생성 요청
 */
export interface CreateProjectRequest {
  workspaceId: string;
  name: string;
  description?: string;
}

/**
 * 프로젝트 수정 요청
 */
export interface UpdateProjectRequest {
  name?: string;
  description?: string;
}

/**
 * 프로젝트 멤버 추가 요청
 */
export interface AddProjectMemberRequest {
  userId: string;
  permission: ProjectPermission;
}

/**
 * 프로젝트 멤버 수정 요청
 */
export interface UpdateProjectMemberRequest {
  permission: ProjectPermission;
}

/**
 * 프로젝트 통계
 */
export interface ProjectStats {
  projectId: string;
  fileCount: number;
  folderCount: number;
  totalSize: number;
  memberCount: number;
}

// =======================================================
// Folder Types
// =======================================================

/**
 * 폴더 응답 DTO
 */
export interface StorageFolder {
  id: string;
  workspaceId: string;
  projectId?: string;
  parentId?: string;
  name: string;
  path: string;
  color?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  isDeleted: boolean;
  children?: StorageFolder[];
  files?: StorageFile[];
  fileCount?: number;
  folderCount?: number;
  totalSize?: number;
}

/**
 * 폴더 생성 요청
 */
export interface CreateFolderRequest {
  workspaceId: string;
  projectId?: string;
  parentId?: string;
  name: string;
  color?: string;
}

/**
 * 폴더 수정 요청
 */
export interface UpdateFolderRequest {
  name?: string;
  color?: string;
  parentId?: string;
}

// =======================================================
// File Types
// =======================================================

/**
 * 파일 응답 DTO
 */
export interface StorageFile {
  id: string;
  workspaceId: string;
  projectId?: string;
  folderId?: string;
  name: string;
  originalName: string;
  fileUrl: string;
  fileSize: number;
  contentType: string;
  status: FileStatus;
  version: number;
  uploadedBy: string;
  createdAt: string;
  updatedAt: string;
  isDeleted: boolean;
  isImage: boolean;
  isDocument: boolean;
  extension: string;
}

/**
 * 파일 목록 응답
 */
export interface FileListResponse {
  files: StorageFile[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/**
 * 업로드 URL 생성 요청
 */
export interface GenerateUploadURLRequest {
  workspaceId: string;
  projectId?: string;
  folderId?: string;
  fileName: string;
  contentType: string;
  fileSize: number;
}

/**
 * 업로드 URL 생성 응답
 */
export interface GenerateUploadURLResponse {
  uploadUrl: string;
  fileKey: string;
  fileId: string;
  expiresAt: string;
}

/**
 * 업로드 확인 요청
 */
export interface ConfirmUploadRequest {
  fileId: string;
}

/**
 * 파일 수정 요청
 */
export interface UpdateFileRequest {
  name?: string;
  folderId?: string;
}

// =======================================================
// Share Types
// =======================================================

/**
 * 공유 응답 DTO
 */
export interface StorageShare {
  id: string;
  entityType: ShareType;
  entityId: string;
  entityName: string;
  sharedWithId?: string;
  sharedWithName?: string;
  sharedById: string;
  sharedByName: string;
  permission: PermissionLevel;
  shareLink?: string;
  shareUrl?: string;
  linkExpiresAt?: string;
  isPublic: boolean;
  isExpired: boolean;
  includeChildren?: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * 공유 생성 요청
 */
export interface CreateShareRequest {
  entityType: ShareType;
  entityId: string;
  sharedWithId?: string;
  permission: PermissionLevel;
  isPublic?: boolean;
  expiresInDays?: number;
  includeChildren?: boolean;
}

/**
 * 공유 수정 요청
 */
export interface UpdateShareRequest {
  permission?: PermissionLevel;
  expiresInDays?: number;
}

/**
 * 나에게 공유된 항목
 */
export interface SharedItem {
  entityType: ShareType;
  entityId: string;
  entityName: string;
  permission: PermissionLevel;
  sharedById: string;
  sharedByName: string;
  sharedAt: string;
}

// =======================================================
// Storage Usage Types
// =======================================================

/**
 * 스토리지 사용량 응답
 */
export interface StorageUsage {
  totalSize: number;
  totalSizeMB: number;
  totalSizeGB: number;
  fileCount: number;
  workspaceId: string;
}

// =======================================================
// Folder Contents Response
// =======================================================

/**
 * 폴더 내용 응답 (폴더 + 파일 목록)
 */
export interface FolderContentsResponse {
  id: string;
  workspaceId: string;
  parentId?: string;
  name: string;
  path: string;
  color?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  isDeleted: boolean;
  children: StorageFolder[];
  files: StorageFile[];
  fileCount: number;
  folderCount: number;
  totalSize: number;
}

// =======================================================
// UI State Types
// =======================================================

/**
 * 선택된 항목
 */
export interface SelectedItem {
  type: 'file' | 'folder';
  id: string;
  name: string;
  data: StorageFile | StorageFolder;
}

/**
 * 브레드크럼 항목
 */
export interface BreadcrumbItem {
  id: string | null;
  name: string;
  path: string;
}

/**
 * 컨텍스트 메뉴 위치
 */
export interface ContextMenuPosition {
  x: number;
  y: number;
}

/**
 * 스토리지 뷰 상태
 */
export interface StorageViewState {
  viewMode: ViewMode;
  sortBy: SortBy;
  sortDirection: SortDirection;
  showHidden: boolean;
  currentProjectId: string | null;
  currentProjectPermission: ProjectPermission | null;
  currentFolderId: string | null;
  selectedItems: SelectedItem[];
  searchQuery: string;
}

// =======================================================
// Utility Types
// =======================================================

/**
 * 아이템 타입 (파일 또는 폴더)
 */
export type StorageItem = (StorageFile & { itemType: 'file' }) | (StorageFolder & { itemType: 'folder' });

/**
 * 파일 타입 카테고리
 */
export type FileCategory = 'image' | 'document' | 'video' | 'audio' | 'archive' | 'other';

/**
 * 파일 확장자별 카테고리 매핑
 */
export const FILE_CATEGORY_MAP: Record<string, FileCategory> = {
  // Images
  '.jpg': 'image',
  '.jpeg': 'image',
  '.png': 'image',
  '.gif': 'image',
  '.webp': 'image',
  '.svg': 'image',
  '.bmp': 'image',
  '.ico': 'image',
  // Documents
  '.pdf': 'document',
  '.doc': 'document',
  '.docx': 'document',
  '.xls': 'document',
  '.xlsx': 'document',
  '.ppt': 'document',
  '.pptx': 'document',
  '.txt': 'document',
  '.rtf': 'document',
  '.odt': 'document',
  // Videos
  '.mp4': 'video',
  '.avi': 'video',
  '.mov': 'video',
  '.wmv': 'video',
  '.flv': 'video',
  '.webm': 'video',
  '.mkv': 'video',
  // Audio
  '.mp3': 'audio',
  '.wav': 'audio',
  '.ogg': 'audio',
  '.flac': 'audio',
  '.aac': 'audio',
  '.wma': 'audio',
  // Archives
  '.zip': 'archive',
  '.rar': 'archive',
  '.7z': 'archive',
  '.tar': 'archive',
  '.gz': 'archive',
};

/**
 * 파일 크기 포맷팅
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

/**
 * 파일 카테고리 가져오기
 */
export const getFileCategory = (extension: string): FileCategory => {
  return FILE_CATEGORY_MAP[extension.toLowerCase()] || 'other';
};
