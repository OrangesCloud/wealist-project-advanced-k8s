// src/types/board.ts

// =======================================================
// Common Types
// =======================================================

/**
 * @summary 공통 응답 래퍼 (response.SuccessResponse)
 */
export interface SuccessResponse<T = any> {
  data: T;
  requestId: string;
}

/**
 * @summary 공통 에러 응답 (response.ErrorResponse)
 */
export interface ErrorResponse {
  error: any;
  requestId: string;
}

// =======================================================
// Attachment Types (대폭 수정됨)
// =======================================================

/**
 * @summary 첨부파일 응답 DTO (internal_handler.AttachmentResponse)
 * [API: GET /api/boards/{boardId}/attachments 등]
 */
export interface AttachmentResponse {
  attachmentId: string; // 첨부파일 ID (format: uuid)
  entityId?: string; // 연관된 엔티티 ID (Board, Project, Comment)
  entityType?: string; // BOARD, PROJECT, COMMENT
  fileName: string; // 원본 파일명
  fileUrl: string; // S3 URL
  contentType: string; // MIME type
  fileSize: number; // Byte 단위
  uploadedBy?: string; // 업로더 ID
  uploadedAt: string; // 업로드 일시
  status?: string; // 파일 상태
  expiresAt?: string; // 만료 일시 (임시 파일인 경우)
}

/**
 * @summary Presigned URL 요청 DTO (internal_handler.PresignedURLRequest)
 * [API: POST /api/attachments/presigned-url]
 */
export interface PresignedURLRequest {
  workspaceId: string;
  fileName: string;
  fileSize: number;
  contentType: string;
  entityType: 'BOARD' | 'PROJECT' | 'COMMENT'; // 대문자 사용
}

/**
 * @summary Presigned URL 응답 DTO (internal_handler.PresignedURLResponse)
 */
export interface PresignedURLResponse {
  attachmentId: string;
  uploadUrl: string; // S3에 PUT 요청을 보낼 URL
  fileKey: string; // 업로드 후 서버에 저장할 Key
  expiresIn: number; // 유효 시간 (초)
}

/**
 * @summary 첨부파일 메타데이터 저장 요청 DTO (internal_handler.SaveAttachmentMetadataRequest)
 * [API: POST /api/attachments]
 * S3 업로드 성공 후 DB에 메타데이터를 저장할 때 사용
 */
export interface SaveAttachmentMetadataRequest {
  entityType: 'BOARD' | 'PROJECT' | 'COMMENT';
  fileKey: string;
  fileName: string;
  fileSize: number;
  contentType: string;
}

// =======================================================
// Board Types
// =======================================================

/**
 * @summary 보드 응답 DTO (dto.BoardResponse)
 * [API: GET /api/boards/{boardId}, POST /api/boards, PUT /api/boards/{boardId}]
 */
export interface BoardResponse {
  boardId: string;
  projectId: string;
  title: string;
  content: string;
  assigneeId: string;
  authorId: string;
  startDate?: string; // Swagger에 존재하므로 추가
  dueDate?: string;
  customFields: Record<string, any>;
  createdAt: string;
  updatedAt: string;
  participantIds?: string[]; // Swagger: participantIds
  attachments: AttachmentResponse[]; // 💡 [변경] 단일 URL -> 배열 객체
}

/**
 * @summary 보드 상세 응답 DTO (dto.BoardDetailResponse)
 * [API: GET /api/boards/{boardId}]
 */
export interface BoardDetailResponse extends BoardResponse {
  participants: ParticipantResponse[];
  comments: CommentResponse[];
  // attachments는 상속받은 BoardResponse에 이미 포함됨
}

/**
 * @summary 보드 생성 요청 (dto.CreateBoardRequest)
 * [API: POST /api/boards]
 */
export interface CreateBoardRequest {
  projectId: string;
  title: string;
  content?: string;
  assigneeId?: string;
  startDate?: string;
  dueDate?: string;
  customFields?: Record<string, any>;
  participants?: string[]; // 생성 시 참여자 ID 배열 (최대 50개)
  attachmentIds?: string[]; // 💡 [변경] fileUrl -> 업로드 완료된 attachment ID 배열
}

/**
 * @summary 보드 수정 요청 (dto.UpdateBoardRequest)
 * [API: PUT /api/boards/{boardId}]
 */
export interface UpdateBoardRequest {
  title?: string;
  content?: string;
  assigneeId?: string;
  startDate?: string;
  dueDate?: string;
  customFields?: Record<string, any>;
  attachmentIds?: string[]; // 💡 [변경] 추가할 attachment ID 배열
}

/**
 * @summary 보드 이동 요청 (dto.MoveBoardRequest)
 * [API: PUT /api/boards/{boardId}/move]
 */
export interface MoveBoardRequest {
  projectId: string;
  groupByFieldName: string; // 예: 'stage'
  newFieldValue: string; // 예: 'in_progress'
}

/**
 * @summary 보드 필터 (dto.BoardFilters)
 */
export interface BoardFilters {
  customFields?: Record<string, any>;
}

/**
 * @summary 페이징된 보드 목록 (dto.PaginatedBoardsResponse)
 */
export interface PaginatedBoardsResponse {
  boards: BoardResponse[];
  total: number;
  page: number;
  limit: number;
}

/**
 * @summary 보드 필드 수정 요청 (dto.UpdateBoardFieldRequest)
 */
export interface UpdateBoardFieldRequest {
  fieldId: 'stage' | 'importance' | 'role';
  value: string;
}

// =======================================================
// Project Types
// =======================================================

/**
 * @summary 프로젝트 응답 DTO (dto.ProjectResponse)
 * [API: GET /api/projects/{projectId}, POST /api/projects]
 */
export interface ProjectResponse {
  projectId: string;
  workspaceId: string;
  ownerId: string;
  ownerEmail?: string;
  ownerName?: string;
  name: string;
  description?: string;
  startDate?: string;
  dueDate?: string;
  isPublic: boolean;
  attachments: AttachmentResponse[]; // ✅ 추가: 첨부파일 배열
  createdAt: string;
  updatedAt: string;
}
/**
 * @summary 프로젝트 기본 정보 (dto.ProjectBasicInfo)
 */
export interface ProjectBasicInfo {
  projectId: string;
  workspaceId: string;
  workspaceName: string;
  workspaceEmail: string;
  name: string;
  description: string;
  ownerId: string;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
  // Swagger BasicInfo에는 startDate/dueDate 명시 확인 필요하나 보통 포함됨
  startDate?: string;
  dueDate?: string;
}

/**
 * @summary 프로젝트 생성 요청 (dto.CreateProjectRequest)
 * [API: POST /api/projects]
 */
export interface CreateProjectRequest {
  workspaceId: string;
  name: string;
  description?: string;
  startDate?: string;
  dueDate?: string;
  attachmentIds?: string[]; // 💡 [변경] fileUrl/Name 제거 -> attachmentIds
}

/**
 * @summary 프로젝트 수정 요청 (dto.UpdateProjectRequest)
 * [API: PUT /api/projects/{projectId}]
 */
export interface UpdateProjectRequest {
  name?: string;
  description?: string;
  startDate?: string;
  dueDate?: string;
  attachmentIds?: string[]; // 💡 [변경]
}

/**
 * @summary 페이징된 프로젝트 목록 (dto.PaginatedProjectsResponse)
 * [API: GET /api/projects/search]
 */
export interface PaginatedProjectsResponse {
  projects: ProjectResponse[];
  total: number;
  page: number;
  limit: number;
}

// =======================================================
// Project Init Settings Types
// =======================================================

/**
 * @summary 필드 타입 정보 (dto.FieldTypeInfo)
 */
export interface FieldTypeInfo {
  typeId: string;
  typeName: string;
  description: string;
}

/**
 * @summary 필드 옵션 (dto.FieldOption)
 */
export interface FieldOption {
  optionId: string;
  optionValue: string;
  optionLabel: string;
  color?: string;
  fieldId?: string; // Swagger DTO에 있음
  description?: string;
  displayOrder?: number;
}

/**
 * @summary 옵션이 포함된 필드 응답 (dto.FieldWithOptionsResponse)
 */
export interface FieldWithOptionsResponse {
  fieldId: string;
  fieldName: string;
  fieldType: string;
  description: string;
  isRequired: boolean;
  options: FieldOption[];
}

/**
 * @summary 프로젝트 초기 설정 응답 (dto.ProjectInitSettingsResponse)
 * [API: GET /api/projects/{projectId}/init-settings]
 */
export interface ProjectInitSettingsResponse {
  project: ProjectBasicInfo;
  fields: FieldWithOptionsResponse[];
  fieldTypes: FieldTypeInfo[];
  defaultViewId: string;
}

// =======================================================
// Field Option Types
// =======================================================

/**
 * @summary 필드 옵션 응답 (dto.FieldOptionResponse)
 * [API: GET /api/field-options 등]
 */
export interface FieldOptionResponse {
  optionId: string;
  fieldType: string;
  value: string;
  label: string;
  color: string;
  displayOrder: number;
  isSystemDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * @summary 필드 옵션 생성 요청 (dto.CreateFieldOptionRequest)
 * [API: POST /api/field-options]
 */
export interface CreateFieldOptionRequest {
  fieldType: 'stage' | 'role' | 'importance';
  value: string;
  label: string;
  color: string;
  displayOrder?: number;
}

/**
 * @summary 필드 옵션 수정 요청 (dto.UpdateFieldOptionRequest)
 * [API: PATCH /api/field-options/{optionId}]
 */
export interface UpdateFieldOptionRequest {
  label?: string;
  color?: string;
  displayOrder?: number;
}

// =======================================================
// Project Member Types
// =======================================================

/**
 * @summary 프로젝트 멤버 응답 (dto.ProjectMemberResponse)
 * [API: GET /api/projects/{projectId}/members]
 */
export interface ProjectMemberResponse {
  memberId: string;
  projectId: string;
  userId: string;
  userName: string;
  userEmail: string;
  roleName: string;
  joinedAt: string;
}

/**
 * @summary 멤버 역할 변경 요청 (dto.UpdateProjectMemberRoleRequest)
 * [API: PUT /api/projects/{projectId}/members/{memberId}/role]
 */
export interface UpdateProjectMemberRoleRequest {
  roleName: 'OWNER' | 'ADMIN' | 'MEMBER';
}

// =======================================================
// Project Join Request Types
// =======================================================

/**
 * @summary 프로젝트 가입 요청 응답 (dto.ProjectJoinRequestResponse)
 * [API: GET /api/projects/{projectId}/join-requests 등]
 */
export interface ProjectJoinRequestResponse {
  requestId: string;
  projectId: string;
  userId: string;
  userName: string;
  userEmail: string;
  status: string;
  requestedAt: string;
  updatedAt: string;
}

/**
 * @summary 프로젝트 가입 요청 생성 (dto.CreateProjectJoinRequestRequest)
 * [API: POST /api/join-requests]
 */
export interface CreateProjectJoinRequestRequest {
  projectId: string;
}

/**
 * @summary 프로젝트 가입 요청 상태 변경 (dto.UpdateProjectJoinRequestRequest)
 * [API: PUT /api/join-requests/{joinRequestId}]
 */
export interface UpdateProjectJoinRequestRequest {
  status: 'APPROVED' | 'REJECTED';
}

// =======================================================
// Comment Types
// =======================================================

/**
 * @summary 댓글 응답 DTO (dto.CommentResponse)
 * [API: GET /api/comments/board/{boardId}]
 */
export interface CommentResponse {
  commentId: string;
  boardId: string;
  userId: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  attachments: AttachmentResponse[]; // 💡 [추가] 댓글도 첨부파일 배열 포함
}

/**
 * @summary 댓글 생성 요청 (dto.CreateCommentRequest)
 * [API: POST /api/comments]
 */
export interface CreateCommentRequest {
  boardId: string;
  content: string;
  attachmentIds?: string[]; // 💡 [추가] 첨부파일 연결 지원
}

/**
 * @summary 댓글 수정 요청 (dto.UpdateCommentRequest)
 * [API: PUT /api/comments/{commentId}]
 */
export interface UpdateCommentRequest {
  content: string; // boardId는 request body에서 제외됨 (path param으로 식별되거나 로직상 불필요할 수 있음, swagger 확인)
  attachmentIds?: string[]; // 💡 [추가]
}

// =======================================================
// Participant Types
// =======================================================

/**
 * @summary 참여자 응답 DTO (dto.ParticipantResponse)
 * [API: GET /api/participants/board/{boardId}]
 */
export interface ParticipantResponse {
  participantId: string; // Participant ID (format: uuid)
  boardId: string;
  userId: string;
  createdAt: string;
}

/**
 * @summary 참여자 추가 결과 (dto.ParticipantResult)
 */
export interface ParticipantResult {
  userId: string;
  success: boolean;
  error?: string;
}

/**
 * @summary 참여자 대량 추가 응답 (dto.AddParticipantsResponse)
 * [API: POST /api/participants]
 */
export interface AddParticipantsResponse {
  results: ParticipantResult[];
  totalRequested: number;
  totalSuccess: number;
  totalFailed: number;
}

/**
 * @summary 참여자 추가 요청 (dto.AddParticipantsRequest)
 * [API: POST /api/participants] - 💡 Bulk Insert로 변경됨
 */
export interface AddParticipantsRequest {
  boardId: string;
  userIds: string[]; // 단일 userId -> userIds 배열로 변경
}

// =======================================================
// Frontend Utility Types
// =======================================================

export type Priority = 'HIGH' | 'MEDIUM' | 'LOW' | '';
export type TLayout = 'table' | 'board' | undefined;
export type TView = 'stage' | 'role' | 'importance' | undefined;

export interface Column {
  stageId: string;
  title: string;
  color?: string;
  boards: BoardResponse[];
}

export interface ViewState {
  currentView?: TView;
  searchQuery?: string;
  filterOption?: string;
  currentLayout?: TLayout;
  showCompleted?: boolean;
  showDeleted?: boolean;
  sortColumn?: 'title' | 'stage' | 'role' | 'importance' | 'assignee' | 'dueDate' | null;
  sortDirection?: 'asc' | 'desc';
}

export interface FieldOptionsLookup {
  [fieldId: string]: FieldOption[] | undefined;
}

export interface IEditCustomFields {
  name: string;
  fieldType:
    | 'text'
    | 'number'
    | 'single_select'
    | 'multi_select'
    | 'date'
    | 'single_user'
    | 'multi_user';
  options?: FieldOption[];
  value?: string | number | null;
}
