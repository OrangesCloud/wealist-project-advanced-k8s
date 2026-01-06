/// <reference types="vite/client" />

// 빌드 정보 전역 변수 (vite.config.ts의 define에서 주입)
declare const __BUILD_NUMBER__: string;
declare const __BUILD_SHA__: string;
declare const __BUILD_TIME__: string;
