// src/hooks/useBrowserNotification.ts

import { useState, useEffect, useCallback } from 'react';

interface UseBrowserNotificationReturn {
  permission: NotificationPermission;
  isSupported: boolean;
  requestPermission: () => Promise<NotificationPermission>;
  showNotification: (title: string, options?: NotificationOptions) => Notification | null;
}

export const useBrowserNotification = (): UseBrowserNotificationReturn => {
  const [permission, setPermission] = useState<NotificationPermission>('default');
  const isSupported = typeof window !== 'undefined' && 'Notification' in window;

  useEffect(() => {
    if (isSupported) {
      setPermission(Notification.permission);
    }
  }, [isSupported]);

  const requestPermission = useCallback(async (): Promise<NotificationPermission> => {
    if (!isSupported) {
      console.warn('[BrowserNotification] Notifications not supported');
      return 'denied';
    }

    try {
      const result = await Notification.requestPermission();
      setPermission(result);
      console.log('[BrowserNotification] Permission result:', result);
      return result;
    } catch (err) {
      console.error('[BrowserNotification] Permission request failed:', err);
      return 'denied';
    }
  }, [isSupported]);

  const showNotification = useCallback(
    (title: string, options?: NotificationOptions): Notification | null => {
      if (!isSupported) {
        console.warn('[BrowserNotification] Notifications not supported');
        return null;
      }

      if (permission !== 'granted') {
        console.warn('[BrowserNotification] Permission not granted');
        return null;
      }

      // Don't show browser notification if the window is focused
      if (document.hasFocus()) {
        console.log('[BrowserNotification] Window focused, skipping browser notification');
        return null;
      }

      try {
        const notification = new Notification(title, {
          icon: '/favicon.ico',
          badge: '/favicon.ico',
          ...options,
        });

        // Auto close after 5 seconds
        setTimeout(() => {
          notification.close();
        }, 5000);

        return notification;
      } catch (err) {
        console.error('[BrowserNotification] Failed to show notification:', err);
        return null;
      }
    },
    [isSupported, permission],
  );

  return {
    permission,
    isSupported,
    requestPermission,
    showNotification,
  };
};
