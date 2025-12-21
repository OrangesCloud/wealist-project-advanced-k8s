import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { getGoogleAuthUrl } from '../api/apiConfig';
const AuthPage: React.FC = () => {
  const { theme } = useTheme();
  const navigate = useNavigate();

  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // 로그인 상태 체크: 이미 로그인되어 있으면 워크스페이스 선택 페이지로 이동
  useEffect(() => {
    const accessToken = localStorage.getItem('accessToken');
    const userId = localStorage.getItem('userId');

    // 토큰과 유저 ID가 모두 있을 때만 리다이렉트
    if (accessToken && userId) {
      // TODO: 나중에는 백엔드에 토큰 유효성 검증 API 호출하는 것이 더 좋음
      navigate('/dashboard', { replace: true });
    } else if (accessToken || userId) {
      // 불완전한 인증 정보가 있으면 정리
      localStorage.removeItem('accessToken');
      localStorage.removeItem('refreshToken');
      localStorage.removeItem('nickName');
      localStorage.removeItem('userEmail');
    }
  }, [navigate]);

  // Google 로그인 핸들러: 리다이렉션만 수행
  const handleGoogleLogin = () => {
    setError(null);
    setIsLoading(true);

    try {
      // ⚠️ getGoogleAuthUrl()을 클릭 시점에 호출해야 window.__ENV__가 확실히 로드됨
      const googleAuthUrl = getGoogleAuthUrl();
      console.log('GOOGLE_AUTH_URL:', googleAuthUrl);
      window.location.href = googleAuthUrl;
    } catch (e) {
      setIsLoading(false);
      setError('인증 요청 URL 접근 중 오류가 발생했습니다.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div
      className={`min-h-screen ${theme.colors.background} flex items-center justify-center p-4 relative overflow-hidden`}
    >
      <div
        className="absolute inset-0 opacity-5"
        style={{
          backgroundImage:
            'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)',
          backgroundSize: '20px 20px',
        }}
      ></div>

      <div
        className={`${theme.colors.primary} ${theme.effects.borderRadius} p-2 w-full max-w-md relative z-10 shadow-2xl ${theme.effects.borderWidth} ${theme.colors.border}`}
      >
        <div
          className={`${theme.colors.secondary} ${theme.effects.cardBorderWidth} ${theme.colors.border} p-4 sm:p-6 ${theme.effects.borderRadius}`}
        >
          {/* logo.png가 없으므로 임시로 텍스트로 대체 */}
          <img src="/logo.png" alt="Orange Cloud Logo" />
          <p className={`${theme.font.size.xs} ${theme.colors.text} mb-4 sm:mb-6 text-center`}>
            Google 계정으로 로그인하고 워크스페이스를 시작하세요.
          </p>

          {/* 에러 메시지 표시 */}
          {error && <p className="text-red-500 text-center mb-4 text-sm font-medium">{error}</p>}

          <div className="relative mb-4 sm:mb-6">
            <div className="absolute inset-0 flex items-center">
              <div className={`w-full border-t-2 sm:border-t-4 border-gray-200`}></div>
            </div>
            <div className={`relative flex justify-center ${theme.font.size.xs}`}>
              <span
                className={`px-2 ${theme.colors.secondary} ${theme.colors.text} ${theme.effects.cardBorderWidth} ${theme.colors.border} ${theme.effects.borderRadius}`}
              >
                START WITH
              </span>
            </div>
          </div>

          <div className="space-y-2 sm:space-y-3">
            <div className="relative">
              <button
                onClick={handleGoogleLogin}
                disabled={isLoading}
                // 구글 스타일의 버튼을 위해 커스텀 스타일 적용
                className={`relative w-full flex items-center justify-center gap-2 sm:gap-3 px-3 sm:px-4 py-2 sm:py-3 font-bold bg-white text-gray-800 border border-gray-300 hover:bg-gray-100 transition ${
                  theme.effects.borderRadius
                } ${isLoading ? 'opacity-70 cursor-not-allowed' : ''}`}
              >
                {/* 인라인 SVG Google Icon */}
                <svg
                  className="w-4 h-4 sm:w-5 sm:h-5 text-red-500 fill-current"
                  viewBox="0 0 48 48"
                >
                  <path
                    fill="#EA4335"
                    d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.91 3.57 30.29 2 24 2 15.1 2 7.45 6.46 3.96 13.06l7.85 6.18C14.64 13.9 19.34 9.5 24 9.5z"
                  />
                  <path
                    fill="#4285F4"
                    d="M46.7 24.5c0-.82-.07-1.57-.18-2.3H24v4.49h12.66c-.6 3.06-2.97 4.79-5.91 6.78l7.26 5.66c3.41-3.23 5.34-8.03 5.34-14.63z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M11.85 28.36c-.11-.32-.18-.65-.18-1c0-.35.07-.69.18-1l-7.85-6.18C3.84 20.17 3 22.28 3 24.5s.84 4.33 2.15 6.18l7.85-6.18z"
                  />
                  <path
                    fill="#34A853"
                    d="M24 46c5.17 0 9.86-1.95 13.4-5.18l-7.26-5.66c-2.48 1.81-5.63 2.87-9.14 2.87-4.66 0-8.67-2.61-10.63-6.42l-7.85 6.18C7.45 41.54 15.1 46 24 46z"
                  />
                  <path fill="none" d="M0 0h48v48H0z" />
                </svg>

                <span className={`font-bold ${theme.font.size.xs}`}>
                  {isLoading ? '인증 요청 중...' : 'GOOGLE 계정으로 계속하기'}
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AuthPage;
