/// <reference types="vite/client" />

// 빌드 정보 전역 변수 (vite.config.ts의 define에서 주입)
declare const __BUILD_NUMBER__: string;
declare const __BUILD_SHA__: string;
declare const __BUILD_TIME__: string;

// 런타임 설정 (config.js에서 주입) - Window 인터페이스 확장
interface Window {
  __ENV__?: {
    API_BASE_URL?: string;
    API_DOMAIN?: string;
    VERSION?: string;
    BUILD_SHA?: string;
    BUILD_TIME?: string;
    ENVIRONMENT?: string;
  };
}

interface ImportMetaEnv {
  readonly VITE_APP_ENV: string;
  readonly VITE_API_URL: string;
  readonly VITE_AUTH_SECRET: string;
  readonly VITE_SOME_API_URL: string;
  readonly VITE_PYTHON_API_URL: string;
  readonly VITE_GO_API_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
