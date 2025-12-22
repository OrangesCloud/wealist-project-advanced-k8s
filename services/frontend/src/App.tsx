import React, { Suspense, lazy } from 'react';
import { ThemeProvider } from './contexts/ThemeContext'; // ✅
import { AuthProvider } from './contexts/AuthContext'; // ✅
// 1. react-router-dom에서 필요한 것들을 임포트합니다.
import { Routes, Route, Navigate, Outlet, useNavigate } from 'react-router-dom';

// Lazy load 페이지들
const AuthPage = lazy(() => import('./pages/Authpage'));
const MyDashboardPage = lazy(() => import('./pages/MyDashboardPage'));
const WorkspacePage = lazy(() => import('./pages/WorkspacePage'));
const OAuthRedirectPage = lazy(() => import('./pages/OAuthRedirectPage'));
const StoragePage = lazy(() => import('./pages/StoragePage'));

const LoadingScreen = ({ msg = '로딩 중..' }) => (
  <div className="text-center min-h-screen flex items-center justify-center bg-gray-50">
    <div className="p-8 bg-white rounded-xl shadow-lg">
      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
      <h1 className="text-xl font-medium text-gray-800">{msg}</h1>
    </div>
  </div>
);

// 2. 인증이 필요한 페이지를 감싸는 '보호 라우트' 컴포넌트
const ProtectedRoute = () => {
  const accessToken = localStorage.getItem('accessToken');
  // 토큰이 없으면 로그인 페이지로 리다이렉트
  if (!accessToken) {
    return <Navigate to="/" replace />;
  }
  // 토큰이 있으면 자식 컴포넌트를 렌더링
  return <Outlet />;
};

const App: React.FC = () => {
  // 로그아웃 핸들러
  const navigate = useNavigate();

  const handleLogout = () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('nickName');
    localStorage.removeItem('userEmail');
    // 로그아웃 후 로그인 페이지로 이동
    navigate('/', { replace: true });
  };

  // 5. renderContent 함수 대신 Routes를 사용합니다.
  return (
    <ThemeProvider>
      <Suspense fallback={<LoadingScreen />}>
        <AuthProvider>
          <Routes>
            {/* 1. 로그인 페이지 */}
            <Route path="/" element={<AuthPage />} />

            {/* 2. OAuth 콜백 페이지 */}
            <Route path="/oauth/callback" element={<OAuthRedirectPage />} />

            {/* 3. 보호되는 라우트 (인증 필요) */}
            <Route element={<ProtectedRoute />}>
              {/* MyDashboardPage - 로그인 후 기본 대시보드 (워크스페이스 목록) */}
              <Route path="/dashboard" element={<MyDashboardPage />} />

              {/* 이전 경로 호환성을 위한 리다이렉트 */}
              <Route path="/workspaces" element={<Navigate to="/dashboard" replace />} />

              {/* WorkspacePage - 워크스페이스 상세 (프로젝트/보드 관리) */}
              <Route
                path="/workspace/:workspaceId"
                element={<WorkspacePage onLogout={handleLogout} />}
              />

              {/* StoragePage - Google Drive 스타일 스토리지 */}
              <Route
                path="/workspace/:workspaceId/storage"
                element={<StoragePage onLogout={handleLogout} />}
              />
            </Route>

            {/* 4. 일치하는 라우트가 없으면 로그인 페이지로 */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </AuthProvider>
      </Suspense>
    </ThemeProvider>
  );
};

export default App;
