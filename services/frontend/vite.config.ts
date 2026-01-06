import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const devHost = process.env.VITE_HOST || '127.0.0.1';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],

  // 빌드 정보 환경변수 주입
  define: {
    __BUILD_NUMBER__: JSON.stringify(process.env.BUILD_NUMBER || 'local'),
    __BUILD_SHA__: JSON.stringify(process.env.BUILD_SHA || 'dev'),
    __BUILD_TIME__: JSON.stringify(new Date().toISOString()),
  },

  // 💡 HMR 연결 주소와 포트를 설정 (선택 사항이지만 안전합니다)
  server: {
    host: devHost, // Docker 컨테이너 내에서 외부 접근 허용
    port: 5173, // 컨테이너 포트와 일치
    // Hot Module Replacement (HMR) 설정
    hmr: {
      clientPort: 3000, // 호스트 포트 (브라우저가 접속하는 포트)
    },
  },

  // 💡 모듈 해석 확장자를 명시적으로 정의 (TSX/TS 파일이 누락되지 않도록)
  resolve: {
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json'],
  },
});
