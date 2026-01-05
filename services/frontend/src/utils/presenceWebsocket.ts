// src/utils/presenceWebSocket.ts
// 🔥 Global Presence WebSocket - 앱 접속 시 온라인 상태 등록

import { getPresenceWebSocketUrl, refreshAccessToken } from '../api/apiConfig';

let presenceWs: WebSocket | null = null;
let pingInterval: number | null = null;
let isConnecting = false;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;
const reconnectDelay = 5000;

export const connectPresenceWebSocket = (onStatusChange?: (data: any) => void) => {
  // 이미 연결 중이면 무시
  if (isConnecting) {
    console.log('⚠️ [Presence WS] 이미 연결 중입니다.');
    return;
  }

  // 기존 연결 정리
  if (presenceWs) {
    if (presenceWs.readyState === WebSocket.OPEN || presenceWs.readyState === WebSocket.CONNECTING) {
      console.log('🔌 [Presence WS] 기존 연결 종료 중...');
      presenceWs.close();
    }
    presenceWs = null;
  }

  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }

  const connect = () => {
    const token = localStorage.getItem('accessToken');
    if (!token) {
      console.log('⚠️ [Presence WS] 토큰 없음 - 연결 건너뜀');
      isConnecting = false;
      return;
    }

    const wsUrl = getPresenceWebSocketUrl(token);
    console.log('🟢 [Presence WS] 연결 시도:', wsUrl);

    isConnecting = true;
    presenceWs = new WebSocket(wsUrl);

    presenceWs.onopen = () => {
      console.log('✅ [Presence WS] 온라인 상태 등록 성공!');
      isConnecting = false;
      reconnectAttempts = 0;

      // Heartbeat (연결 유지)
      pingInterval = window.setInterval(() => {
        if (presenceWs && presenceWs.readyState === WebSocket.OPEN) {
          try {
            presenceWs.send(JSON.stringify({ type: 'heartbeat' }));
          } catch (error) {
            console.error('❌ [Presence WS] Heartbeat 전송 실패:', error);
          }
        }
      }, 30000);
    };

    presenceWs.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        console.log('📨 [Presence WS] 상태 업데이트:', data);
        onStatusChange?.(data);
      } catch (error) {
        // 무시 (pong 등)
      }
    };

    presenceWs.onerror = (e) => {
      console.error('❌ [Presence WS] 에러:', e);
      isConnecting = false;
    };

    presenceWs.onclose = async (event) => {
      console.log(`🔌 [Presence WS] 연결 닫힘: ${event.code}`);
      isConnecting = false;

      if (pingInterval) {
        clearInterval(pingInterval);
        pingInterval = null;
      }

      // 재연결 (정상 종료가 아닌 경우)
      if (event.code !== 1000 && reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        console.log(`🔄 [Presence WS] 재연결 시도 ${reconnectAttempts}/${maxReconnectAttempts}...`);

        // 🔥 재연결 전 토큰 갱신 시도
        try {
          console.log('🔄 [Presence WS] 토큰 갱신 시도...');
          await refreshAccessToken();
          console.log('✅ [Presence WS] 토큰 갱신 성공');
        } catch (error) {
          console.error('❌ [Presence WS] 토큰 갱신 실패, 재연결 중단');
          return; // 토큰 갱신 실패 시 재연결하지 않음 (로그아웃 처리됨)
        }

        setTimeout(connect, reconnectDelay);
      }
    };
  };

  connect();
};

export const disconnectPresenceWebSocket = () => {
  console.log('🔌 [Presence WS] 연결 해제');

  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }

  if (presenceWs) {
    if (presenceWs.readyState === WebSocket.OPEN) {
      presenceWs.close(1000, 'User logout');
    } else if (presenceWs.readyState === WebSocket.CONNECTING) {
      presenceWs.close();
    }
    presenceWs = null;
  }

  isConnecting = false;
  reconnectAttempts = 0;
};

export const isPresenceConnected = (): boolean => {
  return presenceWs !== null && presenceWs.readyState === WebSocket.OPEN;
};
