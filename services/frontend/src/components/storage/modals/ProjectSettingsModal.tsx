// src/components/storage/modals/ProjectSettingsModal.tsx

import React, { useState, useEffect } from 'react';
import { X, UserPlus, Trash2, ChevronDown, Edit2, Check, XCircle } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import type { StorageProject, ProjectMember, ProjectPermission } from '../../../types/storage';
import type { WorkspaceMemberResponse } from '../../../types/user';
import {
  getProjectMembers,
  addProjectMember,
  updateProjectMember,
  removeProjectMember,
  updateProject,
} from '../../../api/storageService';
import { getWorkspaceMembers } from '../../../api/userService';

interface ProjectSettingsModalProps {
  project: StorageProject;
  workspaceId: string;
  onClose: () => void;
  onProjectUpdated: (project: StorageProject) => void;
}

export const ProjectSettingsModal: React.FC<ProjectSettingsModalProps> = ({
  project,
  workspaceId,
  onClose,
  onProjectUpdated,
}) => {
  const { theme } = useTheme();
  const [activeTab, setActiveTab] = useState<'info' | 'members'>('members');
  const [members, setMembers] = useState<ProjectMember[]>([]);
  const [workspaceMembers, setWorkspaceMembers] = useState<WorkspaceMemberResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // 프로젝트 정보 수정
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedName, setEditedName] = useState(project.name);
  const [editedDescription, setEditedDescription] = useState(project.description || '');

  // 멤버 추가
  const [showAddMember, setShowAddMember] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [selectedPermission, setSelectedPermission] = useState<ProjectPermission>('VIEWER');
  const [showPermissionDropdown, setShowPermissionDropdown] = useState<string | null>(null);

  // 데이터 로드
  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true);
      try {
        const [membersData, workspaceMembersData] = await Promise.all([
          getProjectMembers(project.id),
          getWorkspaceMembers(workspaceId),
        ]);
        setMembers(membersData);
        setWorkspaceMembers(workspaceMembersData);
      } catch (err) {
        console.error('Failed to load data:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadData();
  }, [project.id, workspaceId]);

  // 프로젝트 정보 저장
  const handleSaveProjectInfo = async () => {
    try {
      const updatedProject = await updateProject(project.id, {
        name: editedName.trim(),
        description: editedDescription.trim() || undefined,
      });
      onProjectUpdated(updatedProject);
      setIsEditingName(false);
    } catch (err) {
      console.error('Failed to update project:', err);
    }
  };

  // 멤버 추가
  const handleAddMember = async () => {
    if (!selectedUserId) return;

    try {
      const newMember = await addProjectMember(project.id, {
        userId: selectedUserId,
        permission: selectedPermission,
      });

      // 워크스페이스 멤버 정보로 이름/이메일 추가
      const wsm = workspaceMembers.find((m) => m.userId === selectedUserId);
      const memberWithInfo: ProjectMember = {
        ...newMember,
        userName: wsm?.nickName || undefined,
        userEmail: wsm?.userEmail || undefined,
      };

      setMembers([...members, memberWithInfo]);
      setSelectedUserId('');
      setSelectedPermission('VIEWER');
      setShowAddMember(false);
    } catch (err) {
      console.error('Failed to add member:', err);
    }
  };

  // 멤버 권한 수정
  const handleUpdatePermission = async (memberId: string, permission: ProjectPermission) => {
    try {
      await updateProjectMember(project.id, memberId, { permission });
      setMembers(members.map((m) => (m.id === memberId ? { ...m, permission } : m)));
      setShowPermissionDropdown(null);
    } catch (err) {
      console.error('Failed to update permission:', err);
    }
  };

  // 멤버 제거
  const handleRemoveMember = async (memberId: string) => {
    if (!confirm('이 멤버를 프로젝트에서 제거하시겠습니까?')) return;

    try {
      await removeProjectMember(project.id, memberId);
      setMembers(members.filter((m) => m.id !== memberId));
    } catch (err) {
      console.error('Failed to remove member:', err);
    }
  };

  // 추가 가능한 워크스페이스 멤버 (이미 프로젝트 멤버가 아닌 사람들)
  const availableMembers = workspaceMembers.filter(
    (wsm) => !members.some((m) => m.userId === wsm.userId)
  );

  const permissionLabels: Record<ProjectPermission, string> = {
    OWNER: '소유자',
    EDITOR: '편집자',
    VIEWER: '뷰어',
  };

  const permissionColors: Record<ProjectPermission, string> = {
    OWNER: 'bg-purple-100 text-purple-700',
    EDITOR: 'bg-blue-100 text-blue-700',
    VIEWER: 'bg-gray-100 text-gray-700',
  };

  const permissionDescriptions: Record<ProjectPermission, string> = {
    OWNER: '모든 권한 + 프로젝트 관리',
    EDITOR: '파일/폴더 생성, 수정, 삭제',
    VIEWER: '읽기 전용',
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-lg ${theme.colors.card} rounded-xl shadow-2xl max-h-[80vh] flex flex-col`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">프로젝트 설정</h2>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-gray-100 transition">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 탭 */}
        <div className="flex border-b border-gray-200">
          <button
            onClick={() => setActiveTab('info')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition ${
              activeTab === 'info'
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            기본 정보
          </button>
          <button
            onClick={() => setActiveTab('members')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition ${
              activeTab === 'members'
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            멤버 관리
          </button>
        </div>

        {/* 본문 */}
        <div className="p-4 flex-1 overflow-y-auto">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
            </div>
          ) : activeTab === 'info' ? (
            <>
              {/* 프로젝트 이름 */}
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">프로젝트 이름</label>
                {isEditingName ? (
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={editedName}
                      onChange={(e) => setEditedName(e.target.value)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      autoFocus
                    />
                    <button
                      onClick={handleSaveProjectInfo}
                      className="p-2 text-green-600 hover:bg-green-50 rounded-lg transition"
                    >
                      <Check className="w-5 h-5" />
                    </button>
                    <button
                      onClick={() => {
                        setIsEditingName(false);
                        setEditedName(project.name);
                      }}
                      className="p-2 text-gray-500 hover:bg-gray-100 rounded-lg transition"
                    >
                      <XCircle className="w-5 h-5" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="text-gray-900">{project.name}</span>
                    <button
                      onClick={() => setIsEditingName(true)}
                      className="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition"
                    >
                      <Edit2 className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>

              {/* 프로젝트 설명 */}
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">설명</label>
                <textarea
                  value={editedDescription}
                  onChange={(e) => setEditedDescription(e.target.value)}
                  placeholder="프로젝트 설명을 입력하세요"
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-none"
                />
                {editedDescription !== (project.description || '') && (
                  <div className="flex justify-end mt-2">
                    <button
                      onClick={handleSaveProjectInfo}
                      className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                    >
                      저장
                    </button>
                  </div>
                )}
              </div>

              {/* 생성 정보 */}
              <div className="text-sm text-gray-500">
                <p>생성일: {new Date(project.createdAt).toLocaleDateString('ko-KR')}</p>
              </div>
            </>
          ) : (
            <>
              {/* 멤버 추가 버튼 */}
              {!showAddMember ? (
                <button
                  onClick={() => setShowAddMember(true)}
                  disabled={availableMembers.length === 0}
                  className="w-full flex items-center justify-center gap-2 p-3 mb-4 border-2 border-dashed border-gray-300 rounded-lg text-gray-500 hover:border-blue-500 hover:text-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition"
                >
                  <UserPlus className="w-5 h-5" />
                  멤버 추가
                </button>
              ) : (
                <div className="mb-4 p-4 bg-gray-50 rounded-lg">
                  <div className="flex gap-2 mb-3">
                    <select
                      value={selectedUserId}
                      onChange={(e) => setSelectedUserId(e.target.value)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    >
                      <option value="">멤버 선택...</option>
                      {availableMembers.map((member) => (
                        <option key={member.userId} value={member.userId}>
                          {member.nickName || member.userEmail}
                        </option>
                      ))}
                    </select>
                    <select
                      value={selectedPermission}
                      onChange={(e) => setSelectedPermission(e.target.value as ProjectPermission)}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    >
                      <option value="VIEWER">뷰어</option>
                      <option value="EDITOR">편집자</option>
                      <option value="OWNER">소유자</option>
                    </select>
                  </div>
                  <div className="flex justify-end gap-2">
                    <button
                      onClick={() => {
                        setShowAddMember(false);
                        setSelectedUserId('');
                        setSelectedPermission('VIEWER');
                      }}
                      className="px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-200 rounded-lg transition"
                    >
                      취소
                    </button>
                    <button
                      onClick={handleAddMember}
                      disabled={!selectedUserId}
                      className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition"
                    >
                      추가
                    </button>
                  </div>
                </div>
              )}

              {/* 멤버 목록 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  프로젝트 멤버 ({members.length}명)
                </label>
                <div className="space-y-2 max-h-80 overflow-y-auto">
                  {members.map((member) => (
                    <div
                      key={member.id}
                      className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                    >
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-blue-500 rounded-full flex items-center justify-center text-white font-medium">
                          {(member.userName || member.userEmail || '?')[0].toUpperCase()}
                        </div>
                        <div>
                          <p className="font-medium text-gray-900">
                            {member.userName || member.userEmail || member.userId}
                          </p>
                          {member.userEmail && member.userName && (
                            <p className="text-xs text-gray-500">{member.userEmail}</p>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        {/* 권한 드롭다운 */}
                        <div className="relative">
                          <button
                            onClick={() =>
                              setShowPermissionDropdown(
                                showPermissionDropdown === member.id ? null : member.id
                              )
                            }
                            className={`px-3 py-1.5 text-sm rounded-lg flex items-center gap-1 ${permissionColors[member.permission]}`}
                          >
                            {permissionLabels[member.permission]}
                            <ChevronDown className="w-3 h-3" />
                          </button>

                          {showPermissionDropdown === member.id && (
                            <>
                              <div
                                className="fixed inset-0"
                                onClick={() => setShowPermissionDropdown(null)}
                              />
                              <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-10 min-w-[180px]">
                                {(['OWNER', 'EDITOR', 'VIEWER'] as ProjectPermission[]).map((p) => (
                                  <button
                                    key={p}
                                    onClick={() => handleUpdatePermission(member.id, p)}
                                    className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-100 ${
                                      member.permission === p ? 'bg-gray-50' : ''
                                    }`}
                                  >
                                    <div className="font-medium">{permissionLabels[p]}</div>
                                    <div className="text-xs text-gray-500">
                                      {permissionDescriptions[p]}
                                    </div>
                                  </button>
                                ))}
                              </div>
                            </>
                          )}
                        </div>

                        {/* 삭제 버튼 */}
                        <button
                          onClick={() => handleRemoveMember(member.id)}
                          className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}

                  {members.length === 0 && (
                    <p className="text-sm text-gray-500 text-center py-4">
                      아직 멤버가 없습니다.
                    </p>
                  )}
                </div>
              </div>
            </>
          )}
        </div>

        {/* 푸터 */}
        <div className="flex justify-end p-4 border-t border-gray-200">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
          >
            닫기
          </button>
        </div>
      </div>
    </div>
  );
};
