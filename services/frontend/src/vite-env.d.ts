/// <reference types="vite/client" />

// 빌드 정보 전역 변수 (vite.config.ts의 define에서 주입)
declare const __BUILD_NUMBER__: string;
declare const __BUILD_SHA__: string;
declare const __BUILD_TIME__: string;

// 런타임 설정 (config.js에서 주입)
interface RuntimeEnv {
  API_BASE_URL?: string;
  API_DOMAIN?: string;
  VERSION?: string;
  BUILD_SHA?: string;
  BUILD_TIME?: string;
  ENVIRONMENT?: string;
}

declare global {
  interface Window {
    __ENV__?: RuntimeEnv;
  }
}

interface ImportMetaEnv {
  // Vite가 기본적으로 제공하는 환경 변수 (예: 개발/운영 모드)
  readonly VITE_APP_ENV: string;

  // 사용자가 정의한 환경 변수를 여기에 readonly로 추가합니다.
  // 이전에 사용하셨던 변수들을 모두 정의해야 합니다.
  readonly VITE_API_URL: string;
  readonly VITE_AUTH_SECRET: string;
  readonly VITE_SOME_API_URL: string;
  readonly VITE_PYTHON_API_URL: string; // 이전 대화에서 언급된 변수
  readonly VITE_GO_API_URL: string; // 이전 대화에서 언급된 변수
  // 필요에 따라 다른 VITE_ 접두사 변수들을 추가하세요.
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
