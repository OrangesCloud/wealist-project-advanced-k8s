// src/components/notification/NotificationToast.tsx

import React from 'react';
import { X, Bell, ClipboardList, MessageCircle, Users, FolderKanban } from 'lucide-react';
import { useTheme } from '../../contexts/ThemeContext';
import type { Notification } from '../../types/notification';
import type { Toast } from '../../hooks/useToast';
import { getNotificationMessage } from '../../api/notificationService';

interface NotificationToastProps {
  toasts: Toast[];
  onClose: (id: string) => void;
  onClick?: (notification: Notification) => void;
}

const getNotificationIcon = (type: Notification['type']) => {
  if (type === 'PARTICIPANT_ADDED') return Users;
  if (type.startsWith('TASK_')) return ClipboardList;
  if (type.startsWith('COMMENT_')) return MessageCircle;
  if (type.startsWith('WORKSPACE_')) return Users;
  if (type.startsWith('PROJECT_')) return FolderKanban;
  return Bell;
};

const getNotificationColor = (type: Notification['type']): string => {
  if (type === 'TASK_ASSIGNED' || type === 'PARTICIPANT_ADDED') return 'bg-blue-500';
  if (type.startsWith('COMMENT_')) return 'bg-green-500';
  if (type.startsWith('TASK_DUE') || type.startsWith('TASK_OVERDUE')) return 'bg-orange-500';
  if (type.startsWith('WORKSPACE_') || type.startsWith('PROJECT_')) return 'bg-purple-500';
  return 'bg-gray-500';
};

export const NotificationToast: React.FC<NotificationToastProps> = ({
  toasts,
  onClose,
  onClick,
}) => {
  const { theme } = useTheme();

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => {
        const Icon = getNotificationIcon(toast.notification.type);
        const message = getNotificationMessage(toast.notification);
        const colorClass = getNotificationColor(toast.notification.type);

        return (
          <div
            key={toast.id}
            className={`
              ${theme.colors.card}
              border ${theme.colors.border}
              rounded-lg shadow-lg
              transform transition-all duration-300 ease-in-out
              ${toast.visible ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0'}
              cursor-pointer hover:shadow-xl
            `}
            onClick={() => onClick?.(toast.notification)}
          >
            <div className="flex items-start gap-3 p-4">
              {/* Icon */}
              <div className={`${colorClass} rounded-full p-2 flex-shrink-0`}>
                <Icon className="w-4 h-4 text-white" />
              </div>

              {/* Content */}
              <div className="flex-1 min-w-0">
                <p className={`text-sm font-medium ${theme.colors.text}`}>
                  {toast.notification.resourceName || 'Notification'}
                </p>
                <p className={`text-xs ${theme.colors.secondary} mt-1 line-clamp-2`}>{message}</p>
              </div>

              {/* Close button */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onClose(toast.id);
                }}
                className={`
                  flex-shrink-0 p-1 rounded-full
                  ${theme.colors.muted}
                  hover:bg-gray-200 dark:hover:bg-gray-600
                  transition-colors
                `}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
};
