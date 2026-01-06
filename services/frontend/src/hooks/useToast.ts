// src/hooks/useToast.ts

import { useState, useCallback } from 'react';
import type { Notification } from '../types/notification';

export interface Toast {
  id: string;
  notification: Notification;
  visible: boolean;
}

interface UseToastReturn {
  toasts: Toast[];
  showToast: (notification: Notification) => void;
  hideToast: (id: string) => void;
  clearAllToasts: () => void;
}

const TOAST_DURATION = 5000; // 5초 후 자동 숨김

export const useToast = (): UseToastReturn => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((notification: Notification) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    // 새 토스트 추가 (visible: true)
    setToasts((prev) => [...prev, { id, notification, visible: true }]);

    // 5초 후 visible: false로 전환 (애니메이션 시작)
    setTimeout(() => {
      setToasts((prev) =>
        prev.map((t) => (t.id === id ? { ...t, visible: false } : t))
      );
    }, TOAST_DURATION);

    // 애니메이션 완료 후 제거 (300ms)
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, TOAST_DURATION + 300);
  }, []);

  const hideToast = useCallback((id: string) => {
    // visible: false로 전환 (애니메이션 시작)
    setToasts((prev) =>
      prev.map((t) => (t.id === id ? { ...t, visible: false } : t))
    );

    // 애니메이션 완료 후 제거
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 300);
  }, []);

  const clearAllToasts = useCallback(() => {
    setToasts([]);
  }, []);

  return {
    toasts,
    showToast,
    hideToast,
    clearAllToasts,
  };
};
