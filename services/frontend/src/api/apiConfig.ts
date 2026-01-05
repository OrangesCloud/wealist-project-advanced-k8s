import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios';

// Runtime config type declaration (injected via config.js)
declare global {
  interface Window {
    __ENV__?: {
      API_BASE_URL?: string;
      API_DOMAIN?: string; // Direct API domain for WebSocket/SSE (bypasses CloudFront)
    };
  }
}

// 환경 변수 가져오기 (우선순위: runtime config > build-time env)
// - K8s ingress: window.__ENV__.API_BASE_URL = "" (빈 문자열 = 상대 경로)
// - Docker-compose: import.meta.env.VITE_API_BASE_URL = "http://localhost" (포트별 접근)
// - Production: 빌드 시 주입된 URL 사용

// K8s ingress 모드 감지: 명시적으로 빈 문자열이 설정된 경우
// ⚠️ 함수로 변경하여 런타임에 체크 (config.js가 로드된 후 호출되도록)
const getIsIngressModeInternal = (): boolean => window.__ENV__?.API_BASE_URL === '';

// ingress 모드가 아닐 때만 폴백 적용
const getInjectedApiBaseUrl = (): string | undefined => {
  const isIngress = getIsIngressModeInternal();
  return isIngress ? '' : window.__ENV__?.API_BASE_URL || import.meta.env.VITE_API_BASE_URL;
};

// ============================================================================
// 💡 [핵심 수정]: Context Path를 환경에 따라 조건부로 붙입니다.
// ============================================================================

// K8s ingress용 서비스 prefix 매핑
// ingress가 /svc/{service}/* 로 라우팅하고, rewrite로 prefix 제거
// ⚠️ 프론트엔드 호출이 이미 /api를 포함하면 baseURL에서 제외
const getIngressServicePrefix = (path: string): string => {
  if (path?.includes('/api/auth')) return '/api/svc/auth';            // auth: 프론트가 /api/auth 포함
  if (path?.includes('/api/users')) return '/api/svc/user';           // user: 프론트가 /api/users 포함
  if (path?.includes('/api/workspaces')) return '/api/svc/user';      // user: 프론트가 /api/workspaces 포함
  if (path?.includes('/api/profiles')) return '/api/svc/user';        // user: 프론트가 /api/profiles 포함
  if (path?.includes('/api/boards')) return '/api/svc/board/api';      // board: 프론트가 /projects 호출
  if (path?.includes('/api/chats')) return '/api/svc/chat/api/chats'; // chat: 프론트가 /my만 호출 (basePath 필요)
  if (path?.includes('/api/notifications')) return '/api/svc/noti';   // noti: 프론트가 /api/notifications 포함
  if (path?.includes('/api/storage')) return '/api/svc/storage';      // storage: 프론트가 /api/storage 포함
  return ''; // 매칭 안 되면 prefix 없이
};

const getApiBaseUrl = (path: string): string => {
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드: 서비스 prefix만 반환 (axios가 path를 붙임)
  // 예: getApiBaseUrl('/api/users') → '/api/svc/user'
  //     axios.get('/api/workspaces/all') → '/api/svc/user/api/workspaces/all'
  if (isIngressMode) {
    const hostname = window.location.hostname;

    // ============================================================================
    // 💡 Production/Dev 환경: CloudFront /api/* behavior 사용 (same-origin)
    // CloudFront가 /api/* → api.{env}.wealist.co.kr 백엔드로 라우팅
    // ============================================================================
    if (hostname === 'wealist.co.kr' || hostname === 'www.wealist.co.kr') {
      // CloudFront behavior가 /api/* → backend로 라우팅하므로 same-origin 사용
      const prodBaseUrl = window.location.origin;
      if (path?.includes('/api/auth')) return `${prodBaseUrl}/api/svc/auth`;
      if (path?.includes('/api/users')) return `${prodBaseUrl}/api/svc/user`;
      if (path?.includes('/api/workspaces')) return `${prodBaseUrl}/api/svc/user`;
      if (path?.includes('/api/profiles')) return `${prodBaseUrl}/api/svc/user`;
      if (path?.includes('/api/boards')) return `${prodBaseUrl}/api/svc/board/api`;
      if (path?.includes('/api/chats')) return `${prodBaseUrl}/api/svc/chat/api/chats`;
      if (path?.includes('/api/notifications')) return `${prodBaseUrl}/api/svc/noti`;
      if (path?.includes('/api/storage')) return `${prodBaseUrl}/api/svc/storage`;
      return prodBaseUrl;
    }

    // Dev 환경 (dev.wealist.co.kr): CloudFront /api/* behavior 사용 (same-origin)
    if (hostname === 'dev.wealist.co.kr') {
      // CloudFront behavior가 /api/* → backend로 라우팅하므로 same-origin 사용
      const devBaseUrl = window.location.origin;
      if (path?.includes('/api/auth')) return `${devBaseUrl}/api/svc/auth`;
      if (path?.includes('/api/users')) return `${devBaseUrl}/api/svc/user`;
      if (path?.includes('/api/workspaces')) return `${devBaseUrl}/api/svc/user`;
      if (path?.includes('/api/profiles')) return `${devBaseUrl}/api/svc/user`;
      if (path?.includes('/api/boards')) return `${devBaseUrl}/api/svc/board/api`;
      if (path?.includes('/api/chats')) return `${devBaseUrl}/api/svc/chat/api/chats`;
      if (path?.includes('/api/notifications')) return `${devBaseUrl}/api/svc/noti`;
      if (path?.includes('/api/storage')) return `${devBaseUrl}/api/svc/storage`;
      return devBaseUrl;
    }

    // Kind 로컬 개발 환경 (localhost): Istio Gateway 사용
    // /svc/{service}/* 경로로 라우팅 (프론트엔드와 동일한 포트 사용)
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      // 프론트엔드가 사용하는 포트와 동일하게 사용 (CORS 문제 방지)
      const port = window.location.port || '80';
      const localBaseUrl = `http://localhost:${port}`;
      // HTTPRoute가 /svc/{service}/ → / 로 리라이트하므로,
      // 백엔드 basePath를 포함해야 올바른 경로로 도달
      // ⚠️ 단, 프론트엔드 호출이 이미 /api를 포함하면 baseURL에서 제외
      if (path?.includes('/api/auth')) return `${localBaseUrl}/api/svc/auth`;           // auth: Java, 프론트가 /api/auth 포함
      if (path?.includes('/api/users')) return `${localBaseUrl}/api/svc/user`;          // user: 프론트가 /api/users 포함
      if (path?.includes('/api/workspaces')) return `${localBaseUrl}/api/svc/user`;     // user: 프론트가 /api/workspaces 포함
      if (path?.includes('/api/profiles')) return `${localBaseUrl}/api/svc/user`;       // user: 프론트가 /api/profiles 포함
      if (path?.includes('/api/boards')) return `${localBaseUrl}/api/svc/board/api`;  // board: 프론트가 /projects 호출, 백엔드는 /api/projects
      if (path?.includes('/api/chats')) return `${localBaseUrl}/api/svc/chat/api/chats`; // chat: 프론트가 /my만 호출 (basePath 필요)
      if (path?.includes('/api/notifications')) return `${localBaseUrl}/api/svc/noti`;  // noti: 프론트가 /api/notifications 포함
      if (path?.includes('/api/storage')) return `${localBaseUrl}/api/svc/storage`;     // storage: 프론트가 /api/storage 포함
      return localBaseUrl;
    }
    return getIngressServicePrefix(path);
  }

  // 2. 환경 변수 주입 확인 (Docker-compose 등)
  if (injectedApiBaseUrl) {
    // 쉘 스크립트에서 VITE_API_BASE_URL='http://localhost'가 주입된 경우
    const isLocalDevelopment = injectedApiBaseUrl.includes('localhost');

    if (isLocalDevelopment) {
      // 🔥 로컬 개발: nginx를 통해 각 서비스로 라우팅 (포트 80)
      // nginx가 /api/* 경로를 각 백엔드 서비스로 프록시
      if (path?.includes('/api/auth')) return `${injectedApiBaseUrl}/api/auth`; // → nginx → auth-service
      if (path?.includes('/api/users')) return `${injectedApiBaseUrl}`; // → nginx → user-service
      if (path?.includes('/api/workspaces')) return `${injectedApiBaseUrl}`; // → nginx → user-service
      if (path?.includes('/api/profiles')) return `${injectedApiBaseUrl}`; // → nginx → user-service
      if (path?.includes('/api/boards')) return `${injectedApiBaseUrl}/api/boards/api`; // → nginx → board-service
      if (path?.includes('/api/chats')) return `${injectedApiBaseUrl}${path}`; // → nginx → chat-service
      if (path?.includes('/api/notifications')) return `${injectedApiBaseUrl}`; // → nginx → noti-service
      if (path?.includes('/api/storage')) return `${injectedApiBaseUrl}/api/storage/api`; // → nginx → storage-service
    }

    return `${injectedApiBaseUrl}${path}`;
  }

  // 환경 변수가 없을 경우 (Fallback, CI/CD 실패 대비)
  return `https://api.wealist.co.kr${path}`;
};

export const AUTH_SERVICE_API_URL = getApiBaseUrl('/api/auth'); // auth-service (토큰 관리)
export const USER_REPO_API_URL = getApiBaseUrl('/api/users');
export const USER_SERVICE_API_URL = getApiBaseUrl('/api/users'); // 💡 user-service base URL (프로필 API용)
export const BOARD_SERVICE_API_URL = getApiBaseUrl('/api/boards/api');
export const CHAT_SERVICE_API_URL = getApiBaseUrl('/api/chats');
export const NOTI_SERVICE_API_URL = getApiBaseUrl('/api/notifications');
export const STORAGE_SERVICE_API_URL = getApiBaseUrl('/api/storage'); // storage-service (Google Drive like)

// ============================================================================
// Axios 인스턴스 생성
// ============================================================================

/**
 * Auth Service API (Java)를 위한 Axios 인스턴스 - 토큰 관리 전용
 */
export const authServiceClient = axios.create({
  baseURL: AUTH_SERVICE_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

/**
 * User Repo API (Java)를 위한 Axios 인스턴스
 */
export const userRepoClient = axios.create({
  baseURL: USER_REPO_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

/**
 * Board Service API (Go)를 위한 Axios 인스턴스
 */
export const boardServiceClient = axios.create({
  baseURL: BOARD_SERVICE_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

/**
 * Chat Service API (Go)를 위한 Axios 인스턴스
 */
export const chatServiceClient = axios.create({
  baseURL: CHAT_SERVICE_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

/**
 * Notification Service API (Go)를 위한 Axios 인스턴스
 */
export const notiServiceClient = axios.create({
  baseURL: NOTI_SERVICE_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

/**
 * Storage Service API (Go)를 위한 Axios 인스턴스 - Google Drive like storage
 */
export const storageServiceClient = axios.create({
  baseURL: STORAGE_SERVICE_API_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

// ============================================================================
// 인증 갱신 헬퍼 함수 (기존 코드 유지)
// ============================================================================

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

const performLogout = () => {
  localStorage.removeItem('accessToken');
  localStorage.removeItem('refreshToken');
  localStorage.removeItem('nickName');
  localStorage.removeItem('userEmail');
  window.location.href = '/';
};

/**
 * 토큰 갱신 함수 (WebSocket 재연결 등에서 사용)
 * @returns 새로운 accessToken
 * @throws 갱신 실패 시 에러 (로그아웃 처리됨)
 */
export const refreshAccessToken = async (): Promise<string> => {
  const refreshToken = localStorage.getItem('refreshToken');
  if (!refreshToken) {
    console.warn('⚠️ Refresh token not found. Logging out...');
    performLogout();
    throw new Error('No refresh token available');
  }

  try {
    // auth-service의 /api/auth/refresh 엔드포인트 호출
    // K8s ingress에서 /api/svc/auth가 /로 rewrite되므로 /api/auth/refresh 전체 경로 필요
    const response = await axios.post(`${AUTH_SERVICE_API_URL}/api/auth/refresh`, {
      refreshToken,
    });

    const { accessToken, refreshToken: newRefreshToken } = response.data;

    localStorage.setItem('accessToken', accessToken);
    if (newRefreshToken) {
      localStorage.setItem('refreshToken', newRefreshToken);
    }

    return accessToken;
  } catch (error) {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('nickName');
    localStorage.removeItem('userEmail');
    window.location.href = '/';
    throw error;
  }
};

// ============================================================================
// 인터셉터 설정
// ============================================================================

const setupRequestInterceptor = (client: AxiosInstance) => {
  client.interceptors.request.use(
    (config) => {
      const accessToken = localStorage.getItem('accessToken');
      if (accessToken && !config.headers.Authorization) {
        config.headers.Authorization = `Bearer ${accessToken}`;
      }
      return config;
    },
    (error) => {
      return Promise.reject(error);
    },
  );
};

const setupUnifiedResponseInterceptor = (client: AxiosInstance) => {
  client.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      const originalRequest = error.config as InternalAxiosRequestConfig & {
        _retry?: boolean;
        retryCount?: number;
      };
      const status = error.response?.status;

      if (status === 401 && !originalRequest._retry) {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
            .then((token) => {
              originalRequest.headers.Authorization = `Bearer ${token}`;
              return client(originalRequest);
            })
            .catch((err) => {
              return Promise.reject(err);
            });
        }

        originalRequest._retry = true;
        isRefreshing = true;

        try {
          const newAccessToken = await refreshAccessToken();
          processQueue(null, newAccessToken);
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
          return client(originalRequest);
        } catch (refreshError) {
          processQueue(refreshError as Error, null);
          return Promise.reject(refreshError);
        } finally {
          isRefreshing = false;
        }
      }

      if (status && status >= 400 && status < 599) {
        return Promise.reject(error);
      }

      if (!status && error.code !== 'ERR_CANCELED') {
        originalRequest.retryCount = originalRequest.retryCount || 0;

        if (originalRequest.retryCount >= 3) {
          console.error(`[Axios Interceptor] 최대 재시도 횟수 초과: ${originalRequest.url}`);
          return Promise.reject(error);
        }

        originalRequest.retryCount += 1;
        const delay = new Promise((resolve) => setTimeout(resolve, 1000));
        console.warn(
          `[Axios Interceptor] 재시도 중 (${originalRequest.retryCount}회): ${originalRequest.url}`,
        );
        await delay;
        return client(originalRequest);
      }
    },
  );
};

// 인터셉터 적용
setupRequestInterceptor(authServiceClient);
setupRequestInterceptor(userRepoClient);
setupRequestInterceptor(boardServiceClient);
setupRequestInterceptor(chatServiceClient);
setupRequestInterceptor(notiServiceClient);
setupRequestInterceptor(storageServiceClient);

setupUnifiedResponseInterceptor(authServiceClient);
setupUnifiedResponseInterceptor(userRepoClient);
setupUnifiedResponseInterceptor(boardServiceClient);
setupUnifiedResponseInterceptor(chatServiceClient);
setupUnifiedResponseInterceptor(notiServiceClient);
setupUnifiedResponseInterceptor(storageServiceClient);

export const getAuthHeaders = (token: string) => ({
  Authorization: `Bearer ${token}`,
  Accept: 'application/json',
});

// ============================================================================
// WebSocket / SSE URL 생성 헬퍼 (중앙 집중 관리)
// ============================================================================

/**
 * 환경 감지 헬퍼 (export for external use)
 */
export const getIsIngressMode = (): boolean => getIsIngressModeInternal();
export const getIsLocalDevelopment = (): boolean => {
  const injectedApiBaseUrl = getInjectedApiBaseUrl();
  return injectedApiBaseUrl?.includes('localhost') ?? false;
};

/**
 * WebSocket 프로토콜 결정
 */
const getWebSocketProtocol = (baseUrl?: string): 'wss:' | 'ws:' => {
  if (baseUrl?.startsWith('https')) return 'wss:';
  if (window.location.protocol === 'https:') return 'wss:';
  return 'ws:';
};

/**
 * Chat WebSocket URL 생성
 * @param chatId - 채팅방 ID
 * @param token - 액세스 토큰
 */
export const getChatWebSocketUrl = (chatId: string, token: string): string => {
  const encodedToken = encodeURIComponent(token);
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드 - CloudFront를 통해 WebSocket 연결
  // CloudFront → api.dev.wealist.co.kr → ingress → chat-service
  if (isIngressMode) {
    const protocol = getWebSocketProtocol();
    return `${protocol}//${window.location.host}/api/svc/chat/api/chats/ws/${chatId}?token=${encodedToken}`;
  }

  // Kind 로컬 개발 - Istio Gateway를 통해 WebSocket 프록시
  if (injectedApiBaseUrl?.includes('localhost')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/chat/api/chats/ws/${chatId}?token=${encodedToken}`;
  }

  // 운영 환경 (ALB 라우팅)
  if (injectedApiBaseUrl) {
    const protocol = getWebSocketProtocol(injectedApiBaseUrl);
    const host = injectedApiBaseUrl.replace(/^https?:\/\//, '');
    return `${protocol}//${host}/api/chats/ws/${chatId}?token=${encodedToken}`;
  }

  // Fallback
  const host = window.location.host;
  if (host.includes('localhost') || host.includes('127.0.0.1')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/chat/api/chats/ws/${chatId}?token=${encodedToken}`;
  }

  return `wss://api.wealist.co.kr/api/chats/ws/${chatId}?token=${encodedToken}`;
};

/**
 * Presence WebSocket URL 생성 (온라인 상태)
 * @param token - 액세스 토큰
 */
export const getPresenceWebSocketUrl = (token: string): string => {
  const encodedToken = encodeURIComponent(token);
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드 - CloudFront를 통해 WebSocket 연결
  // CloudFront → api.dev.wealist.co.kr → ingress → chat-service
  if (isIngressMode) {
    const protocol = getWebSocketProtocol();
    return `${protocol}//${window.location.host}/api/svc/chat/api/chats/ws/presence?token=${encodedToken}`;
  }

  // Kind 로컬 개발 - Istio Gateway를 통해 WebSocket 프록시
  if (injectedApiBaseUrl?.includes('localhost')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/chat/api/chats/ws/presence?token=${encodedToken}`;
  }

  // 운영 환경 (ALB 라우팅)
  if (injectedApiBaseUrl) {
    const protocol = getWebSocketProtocol(injectedApiBaseUrl);
    const host = injectedApiBaseUrl.replace(/^https?:\/\//, '');
    return `${protocol}//${host}/api/chats/ws/presence?token=${encodedToken}`;
  }

  // Fallback
  const host = window.location.host;
  if (host.includes('localhost') || host.includes('127.0.0.1')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/chat/api/chats/ws/presence?token=${encodedToken}`;
  }

  return `wss://api.wealist.co.kr/api/chats/ws/presence?token=${encodedToken}`;
};

/**
 * Board WebSocket URL 생성
 * @param projectId - 프로젝트 ID
 * @param token - 액세스 토큰
 */
export const getBoardWebSocketUrl = (projectId: string, token: string): string => {
  const encodedToken = encodeURIComponent(token);
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드 - CloudFront를 통해 WebSocket 연결
  // board-service WebSocket 경로: /ws/project/:projectId
  // CloudFront → api.dev.wealist.co.kr → ingress → board-service
  if (isIngressMode) {
    const protocol = getWebSocketProtocol();
    return `${protocol}//${window.location.host}/api/svc/board/ws/project/${projectId}?token=${encodedToken}`;
  }

  // Kind 로컬 개발 - Istio Gateway를 통해 WebSocket 프록시
  if (injectedApiBaseUrl?.includes('localhost')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/board/ws/project/${projectId}?token=${encodedToken}`;
  }

  // 운영 환경 (ALB 라우팅)
  if (injectedApiBaseUrl) {
    const protocol = getWebSocketProtocol(injectedApiBaseUrl);
    const host = injectedApiBaseUrl.replace(/^https?:\/\//, '');
    return `${protocol}//${host}/ws/project/${projectId}?token=${encodedToken}`;
  }

  // Fallback
  const host = window.location.host;
  if (host.includes('localhost') || host.includes('127.0.0.1')) {
    const port = window.location.port || '80';
    return `ws://localhost:${port}/api/svc/board/ws/project/${projectId}?token=${encodedToken}`;
  }

  return `wss://api.wealist.co.kr/ws/project/${projectId}?token=${encodedToken}`;
};

/**
 * Notification SSE Stream URL 생성
 * @param token - 액세스 토큰 (optional, 없으면 localStorage에서 가져옴)
 */
export const getNotificationSSEUrl = (token?: string): string => {
  const accessToken = token || localStorage.getItem('accessToken') || '';
  const encodedToken = encodeURIComponent(accessToken);
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드 - CloudFront를 통해 SSE 연결
  // CloudFront → api.dev.wealist.co.kr → ingress → noti-service
  if (isIngressMode) {
    return `${window.location.origin}/api/svc/noti/api/notifications/stream?token=${encodedToken}`;
  }

  // Kind 로컬 개발 - Istio Gateway를 통해 SSE 프록시
  if (injectedApiBaseUrl?.includes('localhost')) {
    const port = window.location.port || '80';
    return `http://localhost:${port}/api/svc/noti/api/notifications/stream?token=${encodedToken}`;
  }

  // 운영 환경 또는 Fallback
  return `${NOTI_SERVICE_API_URL}/api/notifications/stream?token=${encodedToken}`;
};

/**
 * OAuth2 Base URL 생성 (Google 로그인 등)
 * CloudFront Behaviors를 통해 /oauth2/* 경로가 api.dev.wealist.co.kr로 라우팅됨
 */
export const getOAuthBaseUrl = (): string => {
  const isIngressMode = getIsIngressModeInternal();
  const injectedApiBaseUrl = getInjectedApiBaseUrl();

  // K8s ingress 모드
  if (isIngressMode) {
    const hostname = window.location.hostname;
    // dev 환경: CloudFront /api/* → ALB → Istio HTTPRoute /api/oauth2/* → auth-service
    if (hostname === 'dev.wealist.co.kr') {
      return 'https://api.dev.wealist.co.kr/api';
    }
    // prod 환경: CloudFront /api/* → ALB → Istio HTTPRoute /api/oauth2/* → auth-service
    if (hostname === 'wealist.co.kr' || hostname === 'www.wealist.co.kr') {
      return 'https://api.wealist.co.kr/api';
    }
    // Kind 개발 환경 (localhost): Istio Gateway 사용
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      const port = window.location.port || '80';
      return `http://localhost:${port}/api`;
    }
    return '';
  }

  // Docker-compose (로컬 개발): nginx 게이트웨이 사용 (포트 80)
  // nginx가 /oauth2/* 경로를 auth-service:8080으로 프록시함
  if (injectedApiBaseUrl?.includes('localhost')) {
    return injectedApiBaseUrl;  // http://localhost (nginx gateway)
  }

  // 운영 환경
  if (injectedApiBaseUrl) {
    return `${injectedApiBaseUrl}/api/users`;
  }

  // Fallback
  return 'https://api.wealist.co.kr/api/users';
};

/**
 * Google OAuth2 인증 URL
 */
export const getGoogleAuthUrl = (): string => {
  return `${getOAuthBaseUrl()}/oauth2/authorization/google`;
};
