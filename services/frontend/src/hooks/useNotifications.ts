// src/hooks/useNotifications.ts

import { useState, useEffect, useCallback, useRef } from 'react';
import type { Notification } from '../types/notification';
import {
  getNotifications,
  getUnreadCount,
  markAsRead,
  markAllAsRead,
  deleteNotification,
  getSSEStreamUrl,
} from '../api/notificationService';

interface UseNotificationsOptions {
  workspaceId: string;
  enabled?: boolean;
  onNewNotification?: (notification: Notification) => void;
}

interface UseNotificationsReturn {
  notifications: Notification[];
  unreadCount: number;
  isLoading: boolean;
  error: string | null;
  hasMore: boolean;
  isConnected: boolean;
  loadMore: () => Promise<void>;
  refresh: () => Promise<void>;
  markNotificationAsRead: (notificationId: string) => Promise<void>;
  markAllNotificationsAsRead: () => Promise<void>;
  removeNotification: (notificationId: string) => Promise<void>;
}

export const useNotifications = ({
  workspaceId,
  enabled = true,
  onNewNotification,
}: UseNotificationsOptions): UseNotificationsReturn => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [page, setPage] = useState(1);
  const [isConnected, setIsConnected] = useState(false);

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 5;

  // 알림 목록 로드
  const loadNotifications = useCallback(
    async (pageNum: number = 1, append: boolean = false) => {
      if (!workspaceId || !enabled) return;

      try {
        setIsLoading(true);
        setError(null);

        const result = await getNotifications(workspaceId, pageNum, 20);

        if (append) {
          setNotifications((prev) => [...prev, ...result.notifications]);
        } else {
          setNotifications(result.notifications);
        }

        setHasMore(result.hasMore);
        setPage(pageNum);
      } catch (err) {
        console.error('[Notifications] 알림 로드 실패:', err);
        setError('알림을 불러오는데 실패했습니다.');
      } finally {
        setIsLoading(false);
      }
    },
    [workspaceId, enabled],
  );

  // 읽지 않은 알림 수 로드
  const loadUnreadCount = useCallback(async () => {
    if (!workspaceId || !enabled) return;

    try {
      const result = await getUnreadCount(workspaceId);
      setUnreadCount(result.count);
    } catch (err) {
      console.error('[Notifications] 읽지 않은 수 조회 실패:', err);
    }
  }, [workspaceId, enabled]);

  // SSE 연결
  const connectSSE = useCallback(() => {
    if (!enabled || eventSourceRef.current) return;

    const token = localStorage.getItem('accessToken');
    if (!token) {
      console.log('[Notifications SSE] 토큰 없음 - 연결 건너뜀');
      return;
    }

    const url = getSSEStreamUrl();
    console.log('[Notifications SSE] 연결 시도:', url);

    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      console.log('[Notifications SSE] 연결됨');
      setIsConnected(true);
      reconnectAttempts.current = 0;
    };

    eventSource.addEventListener('connected', () => {
      console.log('[Notifications SSE] 서버 연결 확인');
    });

    eventSource.addEventListener('notification', (event) => {
      try {
        const notification = JSON.parse(event.data) as Notification;
        console.log('[Notifications SSE] 새 알림:', notification);

        // 새 알림을 목록 맨 앞에 추가
        setNotifications((prev) => [notification, ...prev]);
        setUnreadCount((prev) => prev + 1);

        // Trigger callback for browser notification / toast
        onNewNotification?.(notification);
      } catch (err) {
        console.error('[Notifications SSE] 알림 파싱 실패:', err);
      }
    });

    eventSource.onerror = (err) => {
      console.error('[Notifications SSE] 에러:', err);
      setIsConnected(false);

      // 연결 정리
      eventSource.close();
      eventSourceRef.current = null;

      // 재연결 시도
      if (reconnectAttempts.current < maxReconnectAttempts) {
        reconnectAttempts.current += 1;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        console.log(
          `[Notifications SSE] 재연결 시도 ${reconnectAttempts.current}/${maxReconnectAttempts} (${delay}ms 후)`,
        );

        reconnectTimeoutRef.current = setTimeout(() => {
          connectSSE();
        }, delay);
      }
    };
  }, [enabled, onNewNotification]);

  // SSE 연결 해제
  const disconnectSSE = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setIsConnected(false);
      console.log('[Notifications SSE] 연결 해제');
    }
  }, []);

  // 더 보기
  const loadMore = useCallback(async () => {
    if (!hasMore || isLoading) return;
    await loadNotifications(page + 1, true);
  }, [hasMore, isLoading, page, loadNotifications]);

  // 새로고침
  const refresh = useCallback(async () => {
    setPage(1);
    await Promise.all([loadNotifications(1, false), loadUnreadCount()]);
  }, [loadNotifications, loadUnreadCount]);

  // 읽음 처리
  const markNotificationAsRead = useCallback(async (notificationId: string) => {
    try {
      await markAsRead(notificationId);
      setNotifications((prev) =>
        prev.map((n) => (n.id === notificationId ? { ...n, isRead: true } : n)),
      );
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch (err) {
      console.error('[Notifications] 읽음 처리 실패:', err);
    }
  }, []);

  // 전체 읽음 처리
  const markAllNotificationsAsRead = useCallback(async () => {
    if (!workspaceId) return;

    try {
      await markAllAsRead(workspaceId);
      setNotifications((prev) => prev.map((n) => ({ ...n, isRead: true })));
      setUnreadCount(0);
    } catch (err) {
      console.error('[Notifications] 전체 읽음 처리 실패:', err);
    }
  }, [workspaceId]);

  // 알림 삭제
  const removeNotification = useCallback(async (notificationId: string) => {
    try {
      await deleteNotification(notificationId);
      setNotifications((prev) => {
        const notification = prev.find((n) => n.id === notificationId);
        if (notification && !notification.isRead) {
          setUnreadCount((count) => Math.max(0, count - 1));
        }
        return prev.filter((n) => n.id !== notificationId);
      });
    } catch (err) {
      console.error('[Notifications] 삭제 실패:', err);
    }
  }, []);

  // 초기 로드 및 SSE 연결
  useEffect(() => {
    if (workspaceId && enabled) {
      loadNotifications(1, false);
      loadUnreadCount();
      connectSSE();
    }

    return () => {
      disconnectSSE();
    };
  }, [workspaceId, enabled, loadNotifications, loadUnreadCount, connectSSE, disconnectSSE]);

  return {
    notifications,
    unreadCount,
    isLoading,
    error,
    hasMore,
    isConnected,
    loadMore,
    refresh,
    markNotificationAsRead,
    markAllNotificationsAsRead,
    removeNotification,
  };
};
