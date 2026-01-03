// src/utils/chatWebSocket.ts

import { getChatWebSocketUrl, refreshAccessToken } from '../api/apiConfig';

let ws: WebSocket | null = null;
let pingInterval: number | null = null;
let isConnecting = false;
let currentChatId: string | null = null; // 🔥 현재 연결된/연결 중인 chatId 추적

export const WS_CHAT_MTH = [
  'MESSAGE_RECEIVED',
  'USER_TYPING',
  'TYPING_STOP',
  'USER_JOINED',
  'USER_LEFT',
  'MESSAGE_READ',
] as const;

export type WSChatMethod = (typeof WS_CHAT_MTH)[number];

export const connectChatWebSocket = (chatId: string, onMessage: (data: any) => void) => {
  // 🔥 같은 chatId에 이미 연결되어 있으면 무시
  if (currentChatId === chatId && ws && ws.readyState === WebSocket.OPEN) {
    console.log('✅ [Chat WS] 이미 연결됨:', chatId);
    return;
  }

  // 🔥 다른 chatId로 연결 요청 시 기존 연결 강제 정리
  if (currentChatId !== chatId) {
    console.log(`🔄 [Chat WS] chatId 변경: ${currentChatId} → ${chatId}`);

    // 기존 연결 강제 정리
    if (pingInterval) {
      clearInterval(pingInterval);
      pingInterval = null;
    }

    if (ws) {
      // onclose 핸들러가 재연결하지 않도록 먼저 null로 설정
      const oldWs = ws;
      ws = null;
      isConnecting = false;

      if (oldWs.readyState === WebSocket.OPEN || oldWs.readyState === WebSocket.CONNECTING) {
        oldWs.close(1000, 'Switching chat');
      }
    }
  }

  // 🔥 이미 같은 chatId에 연결 중이면 무시
  if (isConnecting && currentChatId === chatId) {
    console.log('⚠️ [Chat WS] 같은 chatId에 연결 중입니다.');
    return;
  }

  // 🔥 현재 chatId 설정
  currentChatId = chatId;

  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  const reconnectDelay = 3000;

  const connect = () => {
    // 🔥 연결 중에 chatId가 변경되었으면 중단
    if (currentChatId !== chatId) {
      console.log('⚠️ [Chat WS] chatId 변경으로 연결 중단:', chatId);
      isConnecting = false;
      return;
    }
    const token = localStorage.getItem('accessToken');
    if (!token) {
      console.error('❌ [Chat WS] No access token');
      isConnecting = false;
      return;
    }

    const wsUrl = getChatWebSocketUrl(chatId, token);
    console.log('🔌 [Chat WS] 연결 시도:', wsUrl);

    isConnecting = true;
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('✅ [Chat WS] 연결 성공!');
      isConnecting = false;
      reconnectAttempts = 0;

      // 🔥 Ping 시작 (Redis 온라인 상태 유지)
      pingInterval = window.setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          try {
            ws.send(JSON.stringify({ type: 'ping' }));
            console.log('🏓 [Chat WS] Ping 전송');
          } catch (error) {
            console.error('❌ [Chat WS] Ping 전송 실패:', error);
          }
        }
      }, 30000); // 30초마다
    };

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);

        if (data.type === 'pong') {
          console.log('🏓 [Chat WS] Pong 수신');
          return;
        }

        console.log('📨 [Chat WS] 메시지 수신:', data);
        onMessage(data);
      } catch (error) {
        console.error('❌ [Chat WS] 메시지 파싱 실패:', error);
      }
    };

    ws.onerror = (e) => {
      console.error('❌ [Chat WS] 에러:', e);
      isConnecting = false;
    };

    ws.onclose = async (event) => {
      console.log(`🔌 [Chat WS] 연결 닫힘: ${event.code} ${event.reason}`);
      isConnecting = false;

      // Ping 정리
      if (pingInterval) {
        clearInterval(pingInterval);
        pingInterval = null;
      }

      // 🔥 chatId가 변경되었으면 재연결하지 않음
      if (currentChatId !== chatId) {
        console.log('⚠️ [Chat WS] chatId 변경됨, 재연결 스킵:', chatId, '→', currentChatId);
        return;
      }

      // 🔥 정상 종료(1000)가 아니면 재연결
      if (event.code !== 1000 && reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        console.log(`🔄 [Chat WS] 재연결 시도 ${reconnectAttempts}/${maxReconnectAttempts}...`);

        // 🔥 재연결 전 토큰 갱신 시도
        try {
          console.log('🔄 [Chat WS] 토큰 갱신 시도...');
          await refreshAccessToken();
          console.log('✅ [Chat WS] 토큰 갱신 성공');
        } catch (error) {
          console.error('❌ [Chat WS] 토큰 갱신 실패, 재연결 중단');
          return; // 토큰 갱신 실패 시 재연결하지 않음 (로그아웃 처리됨)
        }

        setTimeout(connect, reconnectDelay);
      } else if (reconnectAttempts >= maxReconnectAttempts) {
        console.error('❌ [Chat WS] 최대 재연결 시도 초과');
      }
    };
  };

  connect();
};

export const disconnectChatWebSocket = () => {
  console.log('🔌 [Chat WS] 연결 해제 시도, chatId:', currentChatId);

  // 🔥 currentChatId 초기화 (재연결 방지)
  currentChatId = null;

  // 🔥 Ping 정리
  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }

  // 🔥 WebSocket 정리
  if (ws) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.close(1000, 'User disconnected');
      console.log('✅ [Chat WS] 정상 종료');
    } else if (ws.readyState === WebSocket.CONNECTING) {
      ws.close();
      console.log('⚠️ [Chat WS] 연결 중 강제 종료');
    }
    ws = null;
  }

  isConnecting = false;
};

// 🔥 메시지 전송 헬퍼
export const sendChatMessage = (message: any) => {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    console.warn('⚠️ [Chat WS] WebSocket not connected');
    return false;
  }

  try {
    ws.send(JSON.stringify(message));
    console.log('📤 [Chat WS] 메시지 전송:', message);
    return true;
  } catch (error) {
    console.error('❌ [Chat WS] 전송 실패:', error);
    return false;
  }
};

// 🔥 WebSocket 연결 상태 확인
export const isChatWebSocketConnected = (): boolean => {
  return ws !== null && ws.readyState === WebSocket.OPEN;
};
