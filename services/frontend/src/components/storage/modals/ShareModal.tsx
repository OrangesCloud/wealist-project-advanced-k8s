// src/components/storage/modals/ShareModal.tsx

import React, { useState, useEffect } from 'react';
import { X, Link2, Copy, Check, Globe, Users, Trash2, ChevronDown } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import type { SelectedItem, PermissionLevel, StorageShare } from '../../../types/storage';
import { createShare, getSharesByEntity, deleteShare } from '../../../api/storageService';
import { getWorkspaceMembers } from '../../../api/userService';
import type { WorkspaceMemberResponse } from '../../../types/user';

interface ShareModalProps {
  item: SelectedItem;
  workspaceId: string;
  onClose: () => void;
}

export const ShareModal: React.FC<ShareModalProps> = ({ item, workspaceId, onClose }) => {
  const { theme } = useTheme();
  const [activeTab, setActiveTab] = useState<'people' | 'link'>('people');
  const [shares, setShares] = useState<StorageShare[]>([]);
  const [members, setMembers] = useState<WorkspaceMemberResponse[]>([]);
  const [selectedMembers, setSelectedMembers] = useState<string[]>([]);
  const [permission, setPermission] = useState<PermissionLevel>('VIEWER');
  const [isPublic, setIsPublic] = useState(false);
  const [shareLink, setShareLink] = useState('');
  const [copied, setCopied] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [showPermissionMenu, setShowPermissionMenu] = useState(false);

  // 데이터 로드
  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true);
      try {
        const [sharesData, membersData] = await Promise.all([
          getSharesByEntity(item.type === 'folder' ? 'FOLDER' : 'FILE', item.id),
          getWorkspaceMembers(workspaceId),
        ]);
        setShares(sharesData);
        setMembers(membersData);

        // 공개 공유 링크가 있는지 확인
        const publicShare = sharesData.find((s) => s.isPublic);
        if (publicShare) {
          setIsPublic(true);
          setShareLink(publicShare.shareUrl || '');
        }
      } catch (err) {
        console.error('Failed to load share data:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadData();
  }, [item, workspaceId]);

  // 공유 추가
  const handleShare = async () => {
    if (selectedMembers.length === 0) return;

    try {
      for (const memberId of selectedMembers) {
        await createShare({
          entityType: item.type === 'folder' ? 'FOLDER' : 'FILE',
          entityId: item.id,
          sharedWithId: memberId,
          permission,
        });
      }

      // 공유 목록 새로고침
      const updatedShares = await getSharesByEntity(
        item.type === 'folder' ? 'FOLDER' : 'FILE',
        item.id,
      );
      setShares(updatedShares);
      setSelectedMembers([]);
    } catch (err) {
      console.error('Failed to share:', err);
    }
  };

  // 공유 삭제
  const handleRemoveShare = async (shareId: string) => {
    try {
      await deleteShare(shareId);
      setShares(shares.filter((s) => s.id !== shareId));
    } catch (err) {
      console.error('Failed to remove share:', err);
    }
  };

  // 공개 링크 생성/제거
  const handleTogglePublic = async () => {
    try {
      if (isPublic) {
        // 공개 공유 삭제
        const publicShare = shares.find((s) => s.isPublic);
        if (publicShare) {
          await deleteShare(publicShare.id);
          setShares(shares.filter((s) => s.id !== publicShare.id));
        }
        setIsPublic(false);
        setShareLink('');
      } else {
        // 공개 공유 생성
        const newShare = await createShare({
          entityType: item.type === 'folder' ? 'FOLDER' : 'FILE',
          entityId: item.id,
          permission: 'VIEWER',
          isPublic: true,
        });
        setShares([...shares, newShare]);
        setIsPublic(true);
        setShareLink(newShare.shareUrl || '');
      }
    } catch (err) {
      console.error('Failed to toggle public:', err);
    }
  };

  // 링크 복사
  const handleCopyLink = () => {
    navigator.clipboard.writeText(shareLink);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // 공유되지 않은 멤버 필터링
  const availableMembers = members.filter((m) => !shares.some((s) => s.sharedWithId === m.userId));

  const permissionLabels: Record<PermissionLevel, string> = {
    VIEWER: '뷰어',
    COMMENTER: '댓글 작성자',
    EDITOR: '편집자',
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-lg ${theme.colors.card} rounded-xl shadow-2xl`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">"{item.name}" 공유</h2>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-gray-100 transition">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 탭 */}
        <div className="flex border-b border-gray-200">
          <button
            onClick={() => setActiveTab('people')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition ${
              activeTab === 'people'
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <Users className="w-4 h-4 inline-block mr-2" />
            사용자와 공유
          </button>
          <button
            onClick={() => setActiveTab('link')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition ${
              activeTab === 'link'
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <Link2 className="w-4 h-4 inline-block mr-2" />
            링크로 공유
          </button>
        </div>

        {/* 본문 */}
        <div className="p-4">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
            </div>
          ) : activeTab === 'people' ? (
            <>
              {/* 사용자 추가 */}
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">사용자 추가</label>
                <div className="flex gap-2">
                  <select
                    multiple
                    value={selectedMembers}
                    onChange={(e) =>
                      setSelectedMembers(Array.from(e.target.selectedOptions, (opt) => opt.value))
                    }
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  >
                    {availableMembers.map((member) => (
                      <option key={member.userId} value={member.userId}>
                        {member.nickName || member.userEmail}
                      </option>
                    ))}
                  </select>

                  {/* 권한 선택 */}
                  <div className="relative">
                    <button
                      onClick={() => setShowPermissionMenu(!showPermissionMenu)}
                      className="px-3 py-2 border border-gray-300 rounded-lg bg-white flex items-center gap-2"
                    >
                      {permissionLabels[permission]}
                      <ChevronDown className="w-4 h-4" />
                    </button>
                    {showPermissionMenu && (
                      <>
                        <div
                          className="fixed inset-0"
                          onClick={() => setShowPermissionMenu(false)}
                        />
                        <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-10">
                          {(['VIEWER', 'COMMENTER', 'EDITOR'] as PermissionLevel[]).map((p) => (
                            <button
                              key={p}
                              onClick={() => {
                                setPermission(p);
                                setShowPermissionMenu(false);
                              }}
                              className="w-full px-4 py-2 text-left text-sm hover:bg-gray-100"
                            >
                              {permissionLabels[p]}
                            </button>
                          ))}
                        </div>
                      </>
                    )}
                  </div>

                  <button
                    onClick={handleShare}
                    disabled={selectedMembers.length === 0}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition"
                  >
                    공유
                  </button>
                </div>
              </div>

              {/* 공유된 사용자 목록 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  액세스 권한이 있는 사용자
                </label>
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {shares
                    .filter((s) => !s.isPublic)
                    .map((share) => (
                      <div
                        key={share.id}
                        className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white text-sm">
                            {share.sharedWithName?.[0]?.toUpperCase() || '?'}
                          </div>
                          <div>
                            <p className="text-sm font-medium text-gray-900">
                              {share.sharedWithName}
                            </p>
                            <p className="text-xs text-gray-500">
                              {permissionLabels[share.permission]}
                            </p>
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveShare(share.id)}
                          className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  {shares.filter((s) => !s.isPublic).length === 0 && (
                    <p className="text-sm text-gray-500 text-center py-4">
                      아직 공유된 사용자가 없습니다.
                    </p>
                  )}
                </div>
              </div>
            </>
          ) : (
            <>
              {/* 공개 링크 설정 */}
              <div className="mb-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Globe className={`w-5 h-5 ${isPublic ? 'text-green-500' : 'text-gray-400'}`} />
                    <span className="font-medium text-gray-900">링크가 있는 모든 사용자</span>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isPublic}
                      onChange={handleTogglePublic}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>
                <p className="text-sm text-gray-500">
                  {isPublic
                    ? '이 링크를 가진 인터넷 사용자는 누구나 볼 수 있습니다.'
                    : '링크 공유가 비활성화되어 있습니다.'}
                </p>
              </div>

              {/* 공유 링크 */}
              {isPublic && shareLink && (
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={shareLink}
                    readOnly
                    className="flex-1 px-4 py-2 bg-gray-50 border border-gray-300 rounded-lg text-sm text-gray-700"
                  />
                  <button
                    onClick={handleCopyLink}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition flex items-center gap-2"
                  >
                    {copied ? (
                      <>
                        <Check className="w-4 h-4" />
                        복사됨
                      </>
                    ) : (
                      <>
                        <Copy className="w-4 h-4" />
                        복사
                      </>
                    )}
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        {/* 푸터 */}
        <div className="flex justify-end p-4 border-t border-gray-200">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
          >
            완료
          </button>
        </div>
      </div>
    </div>
  );
};
