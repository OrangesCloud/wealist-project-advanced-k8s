// src/components/layout/UserMenu.tsx

import React from 'react';
import { LogOut, User as UserIcon } from 'lucide-react';
import { useTheme } from '../../contexts/ThemeContext';
import type { UserProfileResponse } from '../../types/user';

interface UserMenuProps {
  userProfile: UserProfileResponse | null;
  onProfileModalOpen: () => void;
  onLogout: () => void;
  onClose: () => void;
}

export const UserMenu: React.FC<UserMenuProps> = ({
  userProfile,
  onProfileModalOpen,
  onLogout,
  onClose,
}) => {
  const { theme } = useTheme();

  return (
    <div
      className={`fixed bottom-20 left-[72px] w-64 ${theme.colors.card} ${theme.effects.cardBorderWidth} ${theme.colors.border} z-[9999] ${theme.effects.borderRadius} shadow-2xl`}
      onMouseDown={(e) => e.stopPropagation()}
    >
      <div className="p-3 pb-3 mb-2 border-b border-gray-200">
        <div className="flex items-center gap-3">
          <div
            className={`w-10 h-10 ${theme.colors.primary} flex items-center justify-center text-white text-base font-bold rounded-md overflow-hidden`}
          >
            {userProfile?.profileImageUrl ? (
              <img
                src={userProfile.profileImageUrl}
                alt={userProfile.nickName}
                className="w-full h-full object-cover"
              />
            ) : (
              userProfile?.nickName?.[0]?.toUpperCase() || 'U'
            )}
          </div>
          <div>
            <h3 className="font-bold text-lg text-gray-900">{userProfile?.nickName}</h3>
            <div className="flex items-center text-green-600 text-xs mt-1">
              <span className="w-2 h-2 bg-green-500 rounded-full mr-1"></span>
              대화 가능
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-1 p-2 pt-0">
        <button
          onClick={() => {
            onProfileModalOpen();
            onClose();
          }}
          className="w-full text-left px-2 py-1.5 text-sm text-gray-800 hover:bg-blue-50 hover:text-blue-700 rounded transition flex items-center gap-2"
        >
          <UserIcon className="w-4 h-4" /> 프로필 설정
        </button>
      </div>

      <div className="pt-2 pb-2 border-t border-gray-200 mx-2">
        <button
          onClick={onLogout}
          className="w-full text-left px-2 py-1.5 text-sm text-gray-800 hover:bg-red-50 hover:text-red-700 rounded transition flex items-center gap-2"
        >
          <LogOut className="w-4 h-4" /> 로그아웃
        </button>
      </div>
    </div>
  );
};
