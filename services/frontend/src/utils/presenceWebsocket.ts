// src/utils/presenceWebSocket.ts
// 🔥 Global Presence WebSocket - 앱 접속 시 온라인 상태 등록

let presenceWs: WebSocket | null = null;
let pingInterval: number | null = null;
let isConnecting = false;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;
const reconnectDelay = 5000;

const getPresenceWebSocketUrl = (token: string): string => {
  // K8s ingress 모드 감지: window.__ENV__.API_BASE_URL === ""
  const isIngressMode = window.__ENV__?.API_BASE_URL === "";

  if (isIngressMode) {
    // K8s ingress: /svc/chat prefix 사용, 같은 origin의 WebSocket
    // local 개발환경 (non-localhost 도메인 + TLS 미설정) 감지
    const isLocalDomain = window.location.hostname.includes('local.');
    const protocol = isLocalDomain ? 'ws:' : (window.location.protocol === 'https:' ? 'wss:' : 'ws:');
    return `${protocol}//${window.location.host}/svc/chat/api/chats/ws/presence?token=${encodeURIComponent(token)}`;
  }

  const INJECTED_API_BASE_URL = window.__ENV__?.API_BASE_URL || import.meta.env.VITE_API_BASE_URL;

  if (INJECTED_API_BASE_URL) {
    const isLocalDevelopment = INJECTED_API_BASE_URL.includes('localhost');

    if (isLocalDevelopment) {
      // Docker-compose: Chat Service 직접 연결
      return `ws://localhost:8001/api/chats/ws/presence?token=${encodeURIComponent(token)}`;
    }

    // 운영: ALB를 통한 라우팅
    const protocol = INJECTED_API_BASE_URL.startsWith('https') ? 'wss:' : 'ws:';
    const host = INJECTED_API_BASE_URL.replace(/^https?:\/\//, '');
    return `${protocol}//${host}/api/chats/ws/presence?token=${encodeURIComponent(token)}`;
  }

  // Fallback
  const host = window.location.host;

  if (host.includes('localhost') || host.includes('127.0.0.1')) {
    return `ws://localhost:8001/api/chats/ws/presence?token=${encodeURIComponent(token)}`;
  }

  return `wss://api.wealist.co.kr/api/chats/ws/presence?token=${encodeURIComponent(token)}`;
};

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

    presenceWs.onclose = (event) => {
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
