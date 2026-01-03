// utils/websocket.ts

import { getBoardWebSocketUrl, refreshAccessToken } from '../api/apiConfig';

let ws: WebSocket | null = null;
let pingInterval: number | null = null;
let isConnecting = false; // 🔥 연결 중 플래그 추가

export const WS_BOARD_MTH = [
  'BOARD_CREATED',
  'BOARD_UPDATED',
  'BOARD_MOVED',
  'BOARD_DELETED',
] as const;

export type WSBoardMethod = (typeof WS_BOARD_MTH)[number];

export const connectWebSocket = (projectId: string, onMessage: (data: any) => void) => {
  // 🔥 이미 연결 중이면 무시
  if (isConnecting) {
    console.log('⚠️ [WS] 이미 연결 중입니다.');
    return;
  }

  // 🔥 기존 연결 정리
  if (ws) {
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      console.log('🔌 [WS] 기존 연결 종료 중...');
      ws.close();
    }
    ws = null;
  }

  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }

  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  const reconnectDelay = 3000;

  const connect = () => {
    const token = localStorage.getItem('accessToken');
    if (!token) {
      console.error('❌ No access token');
      isConnecting = false;
      return;
    }

    const wsUrl = getBoardWebSocketUrl(projectId, token);
    console.log('🔌 [WS] 연결 시도:', wsUrl);

    isConnecting = true; // 🔥 연결 시작
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('✅ WebSocket 연결 성공!');
      isConnecting = false; // 🔥 연결 완료
      reconnectAttempts = 0;

      // 🔥 Ping 시작
      pingInterval = window.setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          try {
            ws.send(JSON.stringify({ type: 'ping' }));
            console.log('🏓 [WS] Ping 전송');
          } catch (error) {
            console.error('❌ [WS] Ping 전송 실패:', error);
          }
        }
      }, 30000);
    };

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);

        if (data.type === 'pong') {
          console.log('🏓 [WS] Pong 수신');
          return;
        }

        console.log('📨 [WS] 메시지 수신:', data);
        onMessage(data);
      } catch (error) {
        console.error('❌ [WS] 메시지 파싱 실패:', error);
      }
    };

    ws.onerror = (e) => {
      console.error('❌ [WS] 에러:', e);
      isConnecting = false; // 🔥 에러 시 플래그 해제
    };

    ws.onclose = async (event) => {
      console.log(`🔌 [WS] 연결 닫힘: ${event.code} ${event.reason}`);
      isConnecting = false; // 🔥 연결 종료

      // Ping 정리
      if (pingInterval) {
        clearInterval(pingInterval);
        pingInterval = null;
      }

      // 🔥 정상 종료(1000)가 아니면 재연결
      if (event.code !== 1000 && reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        console.log(`🔄 [WS] 재연결 시도 ${reconnectAttempts}/${maxReconnectAttempts}...`);

        // 🔥 재연결 전 토큰 갱신 시도
        try {
          console.log('🔄 [WS] 토큰 갱신 시도...');
          await refreshAccessToken();
          console.log('✅ [WS] 토큰 갱신 성공');
        } catch (error) {
          console.error('❌ [WS] 토큰 갱신 실패, 재연결 중단');
          return; // 토큰 갱신 실패 시 재연결하지 않음 (로그아웃 처리됨)
        }

        setTimeout(connect, reconnectDelay);
      } else if (reconnectAttempts >= maxReconnectAttempts) {
        console.error('❌ [WS] 최대 재연결 시도 초과');
      }
    };
  };

  connect();
};

export const disconnectWebSocket = () => {
  console.log('🔌 [WS] 연결 해제 시도');

  // 🔥 Ping 정리
  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }

  // 🔥 WebSocket 정리
  if (ws) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.close(1000, 'User disconnected');
      console.log('✅ [WS] 정상 종료');
    } else if (ws.readyState === WebSocket.CONNECTING) {
      ws.close();
      console.log('⚠️ [WS] 연결 중 강제 종료');
    }
    ws = null;
  }

  isConnecting = false;
};
