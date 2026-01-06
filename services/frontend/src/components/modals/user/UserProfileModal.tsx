// src/components/modals/user/UserProfileModal.tsx

/**
 * 사용자 프로필 모달 컴포넌트 (단순화)
 *
 * - 기본 프로필: useAuth().nickName을 기본값으로 사용
 * - 워크스페이스 프로필: 해당 워크스페이스의 프로필이 없으면 기본 프로필(default)을 fallback으로 사용
 */

import React, { useState, useRef, ChangeEvent, useEffect, useMemo } from 'react';
import { X, Camera } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import {
  updateMyProfile,
  getAllMyProfiles,
  getMyWorkspaces,
  uploadProfileImage,
  updateProfileImage,
  deleteWorkspaceProfile,
} from '../../../api/userService';
import {
  UserProfileResponse,
  UpdateProfileRequest,
  UserWorkspaceResponse,
  AttachmentResponse,
} from '../../../types/user';
import Portal from '../../common/Portal';
import { useAuth } from '../../../contexts/AuthContext';

const DEFAULT_WORKSPACE_ID = '00000000-0000-0000-0000-000000000000';

interface UserProfileModalProps {
  onClose: () => void;
  onProfileUpdated?: () => Promise<void>; // 🔥 프로필 업데이트 후 호출될 콜백
}

const UserProfileModal: React.FC<UserProfileModalProps> = ({ onClose, onProfileUpdated }) => {
  const { theme } = useTheme();
  const { nickName: authNickName, refreshNickName } = useAuth();

  const [activeTab, setActiveTab] = useState<'default' | 'workspace'>('default');
  const [allProfiles, setAllProfiles] = useState<UserProfileResponse[]>([]);
  const [workspaces, setWorkspaces] = useState<UserWorkspaceResponse[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string>('');

  const fileInputRef = useRef<HTMLInputElement>(null);

  // 단일 닉네임 상태 (useAuth의 닉네임을 기본값으로)
  const [nickName, setNickName] = useState(authNickName || '');
  const [avatarPreviewUrl, setAvatarPreviewUrl] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 워크스페이스에서 기본 프로필 사용 여부 (워크스페이스별 프로필이 없으면 true)
  const [useDefaultProfile, setUseDefaultProfile] = useState<boolean>(true);

  // ========================================
  // 프로필 데이터 계산 (useMemo)
  // ========================================

  const defaultProfile = useMemo(
    () => allProfiles.find((p) => p.workspaceId === DEFAULT_WORKSPACE_ID) || null,
    [allProfiles]
  );

  const workspaceProfile = useMemo(
    () => allProfiles.find((p) => p.workspaceId === selectedWorkspaceId) || null,
    [allProfiles, selectedWorkspaceId]
  );

  // 현재 프로필: 워크스페이스 탭인데 해당 프로필이 없으면 기본 프로필 fallback
  const currentProfile = useMemo(
    () => (activeTab === 'default' ? defaultProfile : workspaceProfile || defaultProfile),
    [activeTab, defaultProfile, workspaceProfile]
  );

  const userId = currentProfile?.userId || allProfiles[0]?.userId;

  // ========================================
  // 초기 데이터 로드
  // ========================================

  useEffect(() => {
    const loadInitialData = async () => {
      try {
        setLoading(true);
        const [profiles, workspaceList] = await Promise.all([
          getAllMyProfiles(),
          getMyWorkspaces(),
        ]);

        setAllProfiles(profiles);
        setWorkspaces(workspaceList);

        if (workspaceList.length > 0) {
          const firstWorkspaceId = workspaceList[0].workspaceId;
          setSelectedWorkspaceId(firstWorkspaceId);

          // 첫 번째 워크스페이스의 프로필 존재 여부에 따라 체크박스 초기 상태 설정
          const hasWorkspaceProfile = profiles.some((p) => p.workspaceId === firstWorkspaceId);
          setUseDefaultProfile(!hasWorkspaceProfile);
        }

        // 기본 프로필 닉네임으로 초기화 (없으면 useAuth 닉네임 유지)
        const defaultProf = profiles.find((p) => p.workspaceId === DEFAULT_WORKSPACE_ID);
        if (defaultProf?.nickName) {
          setNickName(defaultProf.nickName);
        }
      } catch (err) {
        console.error('[Initial Data Load Error]', err);
        setError('프로필 정보를 불러오는데 실패했습니다.');
      } finally {
        setLoading(false);
      }
    };
    loadInitialData();
  }, []);

  // 탭/워크스페이스 변경 시 닉네임 & 아바타 동기화
  useEffect(() => {
    // 워크스페이스 프로필이 있으면 해당 닉네임, 없으면 기본 프로필 닉네임, 그것도 없으면 useAuth 닉네임
    const profileNickName =
      activeTab === 'workspace' && workspaceProfile?.nickName
        ? workspaceProfile.nickName
        : defaultProfile?.nickName || authNickName || '';

    setNickName(profileNickName);

    // 아바타 미리보기 동기화 (새 파일 선택 안 했을 때만)
    if (!selectedFile) {
      setAvatarPreviewUrl(currentProfile?.profileImageUrl || null);
    }

    // 워크스페이스 탭에서 워크스페이스별 프로필 존재 여부에 따라 체크박스 상태 설정
    if (activeTab === 'workspace') {
      setUseDefaultProfile(!workspaceProfile);
    }
  }, [activeTab, selectedWorkspaceId, workspaceProfile, defaultProfile, authNickName, currentProfile, selectedFile]);

  // ========================================
  // 이미지 업로드 핸들러
  // ========================================

  const handleAvatarChangeClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      if (avatarPreviewUrl) {
        URL.revokeObjectURL(avatarPreviewUrl);
      }
      setAvatarPreviewUrl(URL.createObjectURL(file));
      setSelectedFile(file);
    } else {
      setSelectedFile(null);
      setAvatarPreviewUrl(currentProfile?.profileImageUrl || null);
    }
  };

  // ========================================
  // 저장 핸들러
  // ========================================

  const handleSave = async () => {
    const trimmedNickName = nickName.trim();

    // 기본 프로필 탭이거나 워크스페이스별 프로필을 사용하는 경우 닉네임 필수
    if (!useDefaultProfile || activeTab === 'default') {
      if (!trimmedNickName) {
        setError('닉네임은 필수입니다.');
        return;
      }
    }

    if (!userId) {
      setError('프로필을 불러오는 중입니다. 잠시 후 다시 시도해주세요.');
      return;
    }

    setLoading(true);
    setError(null);

    const targetWorkspaceId = activeTab === 'default' ? DEFAULT_WORKSPACE_ID : selectedWorkspaceId;
    let updatedProfile: UserProfileResponse | undefined;

    try {
      // 워크스페이스 탭에서 "기본 프로필 사용" 체크 시 워크스페이스별 프로필 삭제
      if (activeTab === 'workspace' && useDefaultProfile) {
        // 기존 워크스페이스별 프로필이 있으면 삭제
        if (workspaceProfile) {
          await deleteWorkspaceProfile(selectedWorkspaceId);

          // 로컬 상태에서 해당 프로필 제거
          setAllProfiles((prev) => prev.filter((p) => p.workspaceId !== selectedWorkspaceId));
        }

        // MainLayout/WorkspacePage에 프로필 업데이트 알림
        if (onProfileUpdated) {
          await onProfileUpdated();
        }

        alert('이 워크스페이스에서 기본 프로필을 사용합니다.');
        return;
      }

      // 1. 이미지 업로드 (새 파일 선택 시)
      if (selectedFile) {
        const attachmentResponse: AttachmentResponse = await uploadProfileImage(
          selectedFile,
          targetWorkspaceId
        );
        updatedProfile = await updateProfileImage(targetWorkspaceId, attachmentResponse.attachmentId);
      }

      // 2. 닉네임 업데이트 (변경 시 또는 이미지만 업로드한 경우)
      const isNickNameChanged = updatedProfile
        ? updatedProfile.nickName !== trimmedNickName
        : currentProfile?.nickName !== trimmedNickName;

      if (isNickNameChanged || !updatedProfile) {
        const updateData: UpdateProfileRequest = {
          nickName: trimmedNickName,
          workspaceId: targetWorkspaceId,
          userId: userId,
        };
        updatedProfile = await updateMyProfile(updateData);
      }

      if (!updatedProfile) throw new Error('API 응답이 유효하지 않습니다.');

      // 3. 로컬 상태 업데이트
      setAllProfiles((prev) => {
        const index = prev.findIndex((p) => p.workspaceId === targetWorkspaceId);
        const newProfile: UserProfileResponse = { ...updatedProfile!, workspaceId: targetWorkspaceId };

        if (index !== -1) {
          const updated = [...prev];
          updated[index] = newProfile;
          return updated;
        }
        return [...prev, newProfile];
      });

      // 4. 기본 프로필 저장 시 AuthContext 닉네임도 갱신
      if (activeTab === 'default') {
        refreshNickName();
      }

      // 🔥 5. MainLayout/WorkspacePage에 프로필 업데이트 알림
      if (onProfileUpdated) {
        await onProfileUpdated();
      }

      setSelectedFile(null);
      alert('프로필이 저장되었습니다!');
    } catch (err: any) {
      const errorMsg = err.response?.data?.error?.message || err.message;
      console.error('[Profile Save Error]', errorMsg);
      setError(errorMsg || '프로필 저장에 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  // ========================================
  // 모달 닫기 핸들러
  // ========================================

  const handleClose = () => {
    if (avatarPreviewUrl && selectedFile) {
      URL.revokeObjectURL(avatarPreviewUrl);
    }
    onClose();
  };

  // ========================================
  // 렌더링
  // ========================================

  if (!defaultProfile && loading) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
        <div className="bg-white p-8 rounded-xl shadow-lg">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p className="text-gray-700">프로필 정보를 불러오는 중...</p>
        </div>
      </div>
    );
  }

  return (
    <Portal>
      <div
        className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
        onClick={handleClose}
      >
        <div className="relative w-full max-w-md" onClick={(e) => e.stopPropagation()}>
          <div
            className={`relative ${theme.colors.card} ${theme.effects.borderWidth} ${theme.colors.border} ${theme.effects.borderRadius} shadow-xl`}
          >
            {/* 헤더 */}
            <div className="flex items-center justify-between p-4 pb-3">
              <h2 className={`${theme.font.size.base} font-bold text-gray-800`}>
                사용자 프로필 설정
              </h2>
              <button
                onClick={handleClose}
                className="p-2 hover:bg-gray-100 rounded-lg transition"
                title="닫기"
              >
                <X className="w-4 h-4 text-gray-600" />
              </button>
            </div>

            {/* 탭 메뉴 */}
            <div className="flex border-b border-gray-200 px-6">
              <button
                onClick={() => setActiveTab('default')}
                className={`flex-1 py-3 text-sm font-medium transition-colors relative ${
                  activeTab === 'default' ? 'text-blue-600' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                기본 프로필
                {activeTab === 'default' && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600"></div>
                )}
              </button>
              <button
                onClick={() => setActiveTab('workspace')}
                className={`flex-1 py-3 text-sm font-medium transition-colors relative ${
                  activeTab === 'workspace' ? 'text-blue-600' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                워크스페이스별 프로필
                {activeTab === 'workspace' && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600"></div>
                )}
              </button>
            </div>

            {/* 탭 컨텐츠 */}
            <div className="p-6 space-y-5">
              {/* 에러 메시지 */}
              {error && (
                <div className="p-3 bg-red-100 border border-red-400 text-red-700 rounded-md text-sm">
                  {error}
                </div>
              )}

              {/* 워크스페이스 선택 */}
              <div className={activeTab === 'default' ? 'hidden' : ''}>
                <label className={`block ${theme.font.size.xs} mb-2 text-gray-500 font-medium`}>
                  워크스페이스 선택:
                </label>
                <select
                  value={selectedWorkspaceId}
                  onChange={(e) => {
                    const newWorkspaceId = e.target.value;
                    setSelectedWorkspaceId(newWorkspaceId);
                    // 선택한 워크스페이스의 프로필 존재 여부에 따라 체크박스 즉시 업데이트
                    const hasWorkspaceProfile = allProfiles.some((p) => p.workspaceId === newWorkspaceId);
                    setUseDefaultProfile(!hasWorkspaceProfile);
                  }}
                  className={`w-full px-3 py-2 ${theme.effects.cardBorderWidth} ${theme.colors.border} ${theme.colors.card} ${theme.font.size.xs} ${theme.effects.borderRadius} focus:outline-none focus:ring-2 focus:ring-blue-500`}
                  disabled={workspaces.length === 0}
                >
                  {workspaces.map((workspace) => (
                    <option key={workspace.workspaceId} value={workspace.workspaceId}>
                      {workspace.workspaceName}
                    </option>
                  ))}
                </select>

                {/* 기본 프로필 사용 체크박스 */}
                <label className="flex items-center mt-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={useDefaultProfile}
                    onChange={(e) => {
                      setUseDefaultProfile(e.target.checked);
                      // 기본 프로필 사용으로 전환 시 기본 프로필 데이터로 UI 초기화
                      if (e.target.checked && defaultProfile) {
                        setNickName(defaultProfile.nickName);
                        setAvatarPreviewUrl(defaultProfile.profileImageUrl || null);
                        setSelectedFile(null);
                      }
                    }}
                    className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500"
                  />
                  <span className="ml-2 text-sm text-gray-700">기본 프로필 사용</span>
                </label>
                <p className="mt-1 text-xs text-gray-500">
                  {useDefaultProfile
                    ? '이 워크스페이스에서는 기본 프로필이 사용됩니다'
                    : '이 워크스페이스에 별도의 프로필을 설정합니다'}
                </p>
              </div>
              {/* 기본 탭일 때 높이 유지를 위한 공간 */}
              {activeTab === 'default' && <div style={{ height: '110px' }} className="w-full"></div>}

              {/* 프로필 이미지 */}
              <div className={`flex flex-col items-center mb-4 ${activeTab === 'workspace' && useDefaultProfile ? 'opacity-50' : ''}`}>
                <div className="relative">
                  {(avatarPreviewUrl || currentProfile?.profileImageUrl) ? (
                    <img
                      src={avatarPreviewUrl || currentProfile?.profileImageUrl || ''}
                      alt="프로필 미리보기"
                      className="w-24 h-24 object-cover border-2 border-gray-300 rounded-full"
                    />
                  ) : (
                    <div className="w-24 h-24 bg-blue-500 border-2 border-gray-300 flex items-center justify-center text-white text-3xl font-bold rounded-full">
                      {nickName[0] || 'U'}
                    </div>
                  )}

                  <input
                    type="file"
                    ref={fileInputRef}
                    onChange={handleFileChange}
                    accept="image/*"
                    className="hidden"
                    disabled={activeTab === 'workspace' && useDefaultProfile}
                  />

                  <button
                    onClick={handleAvatarChangeClick}
                    disabled={activeTab === 'workspace' && useDefaultProfile}
                    className={`absolute bottom-0 right-0 p-2 bg-gray-700 text-white rounded-full transition shadow-md ${
                      activeTab === 'workspace' && useDefaultProfile
                        ? 'cursor-not-allowed'
                        : 'hover:bg-gray-800'
                    }`}
                    title={activeTab === 'workspace' && useDefaultProfile ? '기본 프로필 사용 중' : '프로필 사진 변경'}
                  >
                    <Camera className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* 닉네임 */}
              <div className={activeTab === 'workspace' && useDefaultProfile ? 'opacity-50' : ''}>
                <label className={`block ${theme.font.size.xs} mb-2 text-gray-500 font-medium`}>
                  닉네임:
                </label>
                <input
                  type="text"
                  value={nickName}
                  onChange={(e) => setNickName(e.target.value)}
                  disabled={activeTab === 'workspace' && useDefaultProfile}
                  className={`w-full px-3 py-2 ${theme.effects.cardBorderWidth} ${theme.colors.border} ${theme.colors.card} ${theme.font.size.xs} ${theme.effects.borderRadius} focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    activeTab === 'workspace' && useDefaultProfile ? 'cursor-not-allowed bg-gray-100' : ''
                  }`}
                  placeholder="닉네임을 입력하세요"
                />
              </div>

              {/* 버튼 영역 */}
              <div className="flex gap-2 pt-4">
                <button
                  onClick={handleSave}
                  disabled={loading || !userId || !nickName.trim()}
                  className={`flex-1 ${theme.colors.primary} text-white py-3 ${
                    theme.effects.borderRadius
                  } font-semibold transition ${
                    loading || !userId || !nickName.trim() ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90'
                  }`}
                >
                  {loading ? '저장 중...' : '저장'}
                </button>
                <button
                  onClick={handleClose}
                  className="flex-1 bg-gray-300 text-gray-800 py-3 rounded-lg font-semibold hover:bg-gray-400 transition"
                >
                  취소
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Portal>
  );
};

export default UserProfileModal;
