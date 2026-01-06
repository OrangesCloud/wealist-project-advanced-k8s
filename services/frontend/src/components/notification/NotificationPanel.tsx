// src/components/notification/NotificationPanel.tsx

import React from 'react';
import { X, Bell, Check, CheckCheck, Trash2, ClipboardList, MessageCircle, Users, FolderKanban } from 'lucide-react';
import { useTheme } from '../../contexts/ThemeContext';
import type { Notification } from '../../types/notification';
import { getNotificationMessage } from '../../api/notificationService';

interface NotificationPanelProps {
  isOpen: boolean;
  onClose: () => void;
  notifications: Notification[];
  unreadCount: number;
  isLoading: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
  onMarkAsRead: (notificationId: string) => void;
  onMarkAllAsRead: () => void;
  onDelete: (notificationId: string) => void;
  onNotificationClick?: (notification: Notification) => void;
}

const getNotificationIcon = (type: Notification['type']) => {
  if (type === 'PARTICIPANT_ADDED') return Users;
  if (type.startsWith('BOARD_')) return FolderKanban;
  if (type.startsWith('TASK_')) return ClipboardList;
  if (type.startsWith('COMMENT_')) return MessageCircle;
  if (type.startsWith('WORKSPACE_')) return Users;
  if (type.startsWith('PROJECT_')) return FolderKanban;
  return Bell;
};

const formatTimeAgo = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return '방금 전';
  if (diffMins < 60) return `${diffMins}분 전`;
  if (diffHours < 24) return `${diffHours}시간 전`;
  if (diffDays < 7) return `${diffDays}일 전`;
  return date.toLocaleDateString('ko-KR');
};

export const NotificationPanel: React.FC<NotificationPanelProps> = ({
  isOpen,
  onClose,
  notifications,
  unreadCount,
  isLoading,
  hasMore,
  onLoadMore,
  onMarkAsRead,
  onMarkAllAsRead,
  onDelete,
  onNotificationClick,
}) => {
  const { theme } = useTheme();

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50" onClick={onClose}>
      {/* 배경 오버레이 */}
      <div className="absolute inset-0 bg-black/20" />

      {/* 패널 */}
      <div
        className={`absolute left-20 top-0 h-full w-80 ${theme.colors.card} shadow-2xl border-r ${theme.colors.border} flex flex-col`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* 헤더 */}
        <div className={`flex items-center justify-between p-4 border-b ${theme.colors.border}`}>
          <div className="flex items-center gap-2">
            <Bell className="w-5 h-5 text-blue-500" />
            <h2 className="font-semibold text-gray-800">알림</h2>
            {unreadCount > 0 && (
              <span className="px-2 py-0.5 bg-red-500 text-white text-xs rounded-full font-bold">
                {unreadCount}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {unreadCount > 0 && (
              <button
                onClick={onMarkAllAsRead}
                className="p-1.5 hover:bg-gray-100 rounded-lg transition text-gray-500"
                title="모두 읽음 처리"
              >
                <CheckCheck className="w-4 h-4" />
              </button>
            )}
            <button
              onClick={onClose}
              className="p-1.5 hover:bg-gray-100 rounded-lg transition text-gray-500"
              title="닫기"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 알림 목록 */}
        <div className="flex-1 overflow-y-auto">
          {notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-gray-400">
              <Bell className="w-12 h-12 mb-3 opacity-30" />
              <p className="text-sm">알림이 없습니다</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-100">
              {notifications.map((notification) => {
                const IconComponent = getNotificationIcon(notification.type);

                return (
                  <div
                    key={notification.id}
                    className={`p-4 hover:bg-gray-50 transition cursor-pointer group ${
                      !notification.isRead ? 'bg-blue-50/50' : ''
                    }`}
                    onClick={() => {
                      if (!notification.isRead) {
                        onMarkAsRead(notification.id);
                      }
                      if (onNotificationClick) {
                        onNotificationClick(notification);
                        onClose();
                      }
                    }}
                  >
                    <div className="flex gap-3">
                      {/* 아이콘 */}
                      <div
                        className={`w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${
                          notification.isRead ? 'bg-gray-100' : 'bg-blue-100'
                        }`}
                      >
                        <IconComponent
                          className={`w-5 h-5 ${
                            notification.isRead ? 'text-gray-500' : 'text-blue-500'
                          }`}
                        />
                      </div>

                      {/* 내용 */}
                      <div className="flex-1 min-w-0">
                        <p
                          className={`text-sm ${
                            notification.isRead ? 'text-gray-600' : 'text-gray-800 font-medium'
                          }`}
                        >
                          {getNotificationMessage(notification)}
                        </p>
                        <p className="text-xs text-gray-400 mt-1">
                          {formatTimeAgo(notification.createdAt)}
                        </p>
                      </div>

                      {/* 액션 버튼 */}
                      <div className="flex items-start gap-1 opacity-0 group-hover:opacity-100 transition">
                        {!notification.isRead && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              onMarkAsRead(notification.id);
                            }}
                            className="p-1 hover:bg-gray-200 rounded transition"
                            title="읽음 처리"
                          >
                            <Check className="w-4 h-4 text-gray-500" />
                          </button>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onDelete(notification.id);
                          }}
                          className="p-1 hover:bg-red-100 rounded transition"
                          title="삭제"
                        >
                          <Trash2 className="w-4 h-4 text-gray-400 hover:text-red-500" />
                        </button>
                      </div>
                    </div>

                    {/* 읽지 않음 표시 */}
                    {!notification.isRead && (
                      <div className="absolute right-4 top-1/2 -translate-y-1/2">
                        <div className="w-2 h-2 bg-blue-500 rounded-full" />
                      </div>
                    )}
                  </div>
                );
              })}

              {/* 더 보기 버튼 */}
              {hasMore && (
                <button
                  onClick={onLoadMore}
                  disabled={isLoading}
                  className="w-full py-3 text-sm text-blue-500 hover:bg-gray-50 transition disabled:opacity-50"
                >
                  {isLoading ? '로딩 중...' : '더 보기'}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default NotificationPanel;
