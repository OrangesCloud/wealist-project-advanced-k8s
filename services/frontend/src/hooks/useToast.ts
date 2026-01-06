// src/hooks/useToast.ts

import { useState, useCallback } from 'react';
import type { Notification } from '../types/notification';

export interface Toast {
  id: string;
  notification: Notification;
  visible: boolean;
}

interface UseToastOptions {
  maxToasts?: number;
  autoHideDelay?: number;
}

interface UseToastReturn {
  toasts: Toast[];
  addToast: (notification: Notification) => void;
  removeToast: (id: string) => void;
  clearAllToasts: () => void;
}

export const useToast = ({
  maxToasts = 5,
  autoHideDelay = 5000,
}: UseToastOptions = {}): UseToastReturn => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const addToast = useCallback(
    (notification: Notification) => {
      const id = `toast-${notification.id}-${Date.now()}`;

      setToasts((prev) => {
        // Remove oldest if we've reached max
        const updated = prev.length >= maxToasts ? prev.slice(1) : prev;
        return [...updated, { id, notification, visible: true }];
      });

      // Auto hide after delay
      setTimeout(() => {
        setToasts((prev) =>
          prev.map((toast) => (toast.id === id ? { ...toast, visible: false } : toast)),
        );

        // Remove from DOM after animation
        setTimeout(() => {
          removeToast(id);
        }, 300);
      }, autoHideDelay);
    },
    [maxToasts, autoHideDelay, removeToast],
  );

  const clearAllToasts = useCallback(() => {
    setToasts([]);
  }, []);

  return {
    toasts,
    addToast,
    removeToast,
    clearAllToasts,
  };
};
