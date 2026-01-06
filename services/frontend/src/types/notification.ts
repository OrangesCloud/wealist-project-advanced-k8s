// src/types/notification.ts

export type NotificationType =
  | 'TASK_ASSIGNED'
  | 'TASK_UNASSIGNED'
  | 'TASK_MENTIONED'
  | 'TASK_DUE_SOON'
  | 'TASK_OVERDUE'
  | 'TASK_STATUS_CHANGED'
  | 'COMMENT_ADDED'
  | 'COMMENT_MENTIONED'
  | 'WORKSPACE_INVITED'
  | 'WORKSPACE_ROLE_CHANGED'
  | 'WORKSPACE_REMOVED'
  | 'PROJECT_INVITED'
  | 'PROJECT_ROLE_CHANGED'
  | 'PROJECT_REMOVED'
  | 'PARTICIPANT_ADDED'
  // Board (Kanban) notification types
  | 'BOARD_ASSIGNED'
  | 'BOARD_UNASSIGNED'
  | 'BOARD_PARTICIPANT_ADDED'
  | 'BOARD_UPDATED'
  | 'BOARD_STATUS_CHANGED'
  | 'BOARD_COMMENT_ADDED'
  | 'BOARD_DUE_SOON'
  | 'BOARD_OVERDUE';

export type ResourceType = 'task' | 'comment' | 'workspace' | 'project' | 'board';

export interface Notification {
  id: string;
  type: NotificationType;
  actorId: string;
  targetUserId: string;
  workspaceId: string;
  resourceType: ResourceType;
  resourceId: string;
  resourceName?: string;
  metadata?: Record<string, unknown>;
  isRead: boolean;
  readAt?: string;
  createdAt: string;
}

export interface PaginatedNotifications {
  notifications: Notification[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

export interface UnreadCount {
  count: number;
  workspaceId: string;
}

export interface NotificationEvent {
  type: NotificationType;
  actorId: string;
  targetUserId: string;
  workspaceId: string;
  resourceType: ResourceType;
  resourceId: string;
  resourceName?: string;
  metadata?: Record<string, unknown>;
}

// SSE 이벤트 타입
export interface SSENotificationEvent {
  type: 'notification' | 'connected' | 'pong';
  data?: Notification;
}
