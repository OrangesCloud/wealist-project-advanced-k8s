// src/api/notificationService.ts

import { notiServiceClient, getNotificationSSEUrl } from './apiConfig';
import type { Notification, PaginatedNotifications, UnreadCount } from '../types/notification';

/**
 * 알림 목록 조회
 * [API] GET /api/notifications
 */
export const getNotifications = async (
  workspaceId: string,
  page: number = 1,
  limit: number = 20,
  unreadOnly: boolean = false,
): Promise<PaginatedNotifications> => {
  const response = await notiServiceClient.get('/api/notifications', {
    params: { page, limit, unreadOnly },
    headers: { 'X-Workspace-Id': workspaceId },
  });
  return response.data;
};

/**
 * 읽지 않은 알림 수 조회
 * [API] GET /api/notifications/unread-count
 */
export const getUnreadCount = async (workspaceId: string): Promise<UnreadCount> => {
  const response = await notiServiceClient.get('/api/notifications/unread-count', {
    headers: { 'X-Workspace-Id': workspaceId },
  });
  return response.data;
};

/**
 * 알림 읽음 처리
 * [API] PATCH /api/notifications/:id/read
 */
export const markAsRead = async (notificationId: string): Promise<Notification> => {
  const response = await notiServiceClient.patch(`/api/notifications/${notificationId}/read`);
  return response.data;
};

/**
 * 모든 알림 읽음 처리
 * [API] POST /api/notifications/read-all
 */
export const markAllAsRead = async (workspaceId: string): Promise<void> => {
  await notiServiceClient.post(
    '/api/notifications/read-all',
    {},
    { headers: { 'X-Workspace-Id': workspaceId } },
  );
};

/**
 * 알림 삭제
 * [API] DELETE /api/notifications/:id
 */
export const deleteNotification = async (notificationId: string): Promise<void> => {
  await notiServiceClient.delete(`/api/notifications/${notificationId}`);
};

/**
 * SSE 스트림 URL 생성 (apiConfig에서 중앙 관리)
 */
export const getSSEStreamUrl = (): string => {
  return getNotificationSSEUrl();
};

/**
 * 알림 타입에 따른 메시지 생성
 */
export const getNotificationMessage = (notification: Notification): string => {
  const resourceName = notification.resourceName || '항목';
  const projectName = (notification.metadata?.projectName as string) || '';
  const projectPrefix = projectName ? `[${projectName}] ` : '';

  switch (notification.type) {
    case 'TASK_ASSIGNED':
      return `${projectPrefix}"${resourceName}" 작업에 할당되었습니다.`;
    case 'TASK_UNASSIGNED':
      return `${projectPrefix}"${resourceName}" 작업 할당이 해제되었습니다.`;
    case 'TASK_MENTIONED':
      return `${projectPrefix}"${resourceName}" 작업에서 언급되었습니다.`;
    case 'TASK_DUE_SOON':
      return `${projectPrefix}"${resourceName}" 작업 마감이 임박했습니다.`;
    case 'TASK_OVERDUE':
      return `${projectPrefix}"${resourceName}" 작업이 마감되었습니다.`;
    case 'TASK_STATUS_CHANGED':
      return `${projectPrefix}"${resourceName}" 작업 상태가 변경되었습니다.`;
    case 'PARTICIPANT_ADDED':
      return `${projectPrefix}"${resourceName}" 작업에 참여자로 추가되었습니다.`;
    case 'COMMENT_ADDED':
      return `${projectPrefix}"${resourceName}"에 새 댓글이 추가되었습니다.`;
    case 'COMMENT_MENTIONED':
      return `${projectPrefix}댓글에서 언급되었습니다.`;
    case 'WORKSPACE_INVITED':
      return `"${resourceName}" 워크스페이스에 초대되었습니다.`;
    case 'WORKSPACE_ROLE_CHANGED':
      return `"${resourceName}" 워크스페이스 역할이 변경되었습니다.`;
    case 'WORKSPACE_REMOVED':
      return `"${resourceName}" 워크스페이스에서 제외되었습니다.`;
    case 'PROJECT_INVITED':
      return `"${resourceName}" 프로젝트에 초대되었습니다.`;
    case 'PROJECT_ROLE_CHANGED':
      return `"${resourceName}" 프로젝트 역할이 변경되었습니다.`;
    case 'PROJECT_REMOVED':
      return `"${resourceName}" 프로젝트에서 제외되었습니다.`;
    // Board (Kanban) notifications
    case 'BOARD_ASSIGNED':
      return `${projectPrefix}"${resourceName}" 카드에 담당자로 지정되었습니다.`;
    case 'BOARD_UNASSIGNED':
      return `${projectPrefix}"${resourceName}" 카드 담당이 해제되었습니다.`;
    case 'BOARD_PARTICIPANT_ADDED':
      return `${projectPrefix}"${resourceName}" 카드에 참여자로 추가되었습니다.`;
    case 'BOARD_UPDATED':
      return `${projectPrefix}"${resourceName}" 카드가 수정되었습니다.`;
    case 'BOARD_STATUS_CHANGED':
      return `${projectPrefix}"${resourceName}" 카드 상태가 변경되었습니다.`;
    case 'BOARD_COMMENT_ADDED':
      return `${projectPrefix}"${resourceName}" 카드에 새 댓글이 추가되었습니다.`;
    case 'BOARD_DUE_SOON':
      return `${projectPrefix}"${resourceName}" 카드 마감이 임박했습니다.`;
    case 'BOARD_OVERDUE':
      return `${projectPrefix}"${resourceName}" 카드가 마감일을 초과했습니다.`;
    default:
      return '새 알림이 있습니다.';
  }
};

/**
 * 알림 타입에 따른 아이콘 이름 반환
 */
export const getNotificationIcon = (
  type: Notification['type'],
): 'task' | 'comment' | 'workspace' | 'project' | 'board' => {
  if (type.startsWith('BOARD_')) return 'board';
  if (type.startsWith('TASK_')) return 'task';
  if (type.startsWith('COMMENT_')) return 'comment';
  if (type.startsWith('WORKSPACE_')) return 'workspace';
  if (type.startsWith('PROJECT_')) return 'project';
  return 'task';
};
