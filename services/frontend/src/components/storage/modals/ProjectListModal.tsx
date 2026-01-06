// src/components/storage/modals/ProjectListModal.tsx

import React, { useState, useEffect } from 'react';
import { X, FolderKanban, HardDrive } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import { getProjects, getProjectMembers } from '../../../api/boardService';
import type { ProjectResponse } from '../../../types/board';
import type { ProjectPermission } from '../../../types/storage';

interface ProjectListModalProps {
  workspaceId: string;
  currentProjectId: string | null;
  onClose: () => void;
  onSelectProject: (project: ProjectResponse | null, permission: ProjectPermission | null) => void;
}

interface ProjectWithPermission extends ProjectResponse {
  myPermission?: ProjectPermission | null;
}

export const ProjectListModal: React.FC<ProjectListModalProps> = ({
  workspaceId,
  currentProjectId,
  onClose,
  onSelectProject,
}) => {
  const { theme } = useTheme();
  const [projects, setProjects] = useState<ProjectWithPermission[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // 프로젝트 목록 로드 (보드 서비스의 프로젝트 사용)
  useEffect(() => {
    const loadProjects = async () => {
      setIsLoading(true);
      try {
        // 보드 서비스에서 워크스페이스의 프로젝트 목록 가져오기
        const projectsData = await getProjects(workspaceId);

        // 각 프로젝트의 멤버 정보에서 현재 사용자 권한 확인
        const projectsWithPermission = await Promise.all(
          projectsData.map(async (project) => {
            try {
              const members = await getProjectMembers(project.projectId);
              // 현재 사용자의 권한 찾기 (localStorage에서 userId 가져오기)
              const currentUserId = localStorage.getItem('userId');
              const myMembership = members.find(m => m.userId === currentUserId);

              let permission: ProjectPermission | null = null;
              if (myMembership) {
                // 보드 서비스의 roleName을 스토리지 권한으로 매핑
                const role = myMembership.roleName?.toUpperCase() || '';
                if (role === 'OWNER' || role === 'ADMIN') {
                  permission = 'OWNER';
                } else if (role === 'MEMBER' || role === 'EDITOR') {
                  permission = 'EDITOR';
                } else {
                  permission = 'VIEWER';
                }
              }

              return { ...project, myPermission: permission };
            } catch {
              // 멤버 조회 실패 시 기본 VIEWER 권한
              return { ...project, myPermission: 'VIEWER' as ProjectPermission };
            }
          })
        );

        setProjects(projectsWithPermission);
      } catch (err) {
        console.error('Failed to load projects:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadProjects();
  }, [workspaceId]);

  // 프로젝트 선택
  const handleSelectProject = (project: ProjectWithPermission) => {
    onSelectProject(project, project.myPermission || null);
    onClose();
  };

  // 내 드라이브 선택 (프로젝트 없음)
  const handleSelectMyDrive = () => {
    onSelectProject(null, null);
    onClose();
  };

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

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-lg ${theme.colors.card} rounded-xl shadow-2xl max-h-[80vh] flex flex-col`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">스토리지 선택</h2>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-gray-100 transition">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 본문 */}
        <div className="p-4 flex-1 overflow-y-auto">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
            </div>
          ) : (
            <>
              {/* 내 드라이브 옵션 (개인 스토리지) */}
              <button
                onClick={handleSelectMyDrive}
                className={`w-full flex items-center gap-3 p-3 rounded-lg mb-2 transition ${
                  currentProjectId === null
                    ? 'bg-blue-50 border-2 border-blue-500'
                    : 'hover:bg-gray-50 border-2 border-transparent'
                }`}
              >
                <HardDrive className="w-6 h-6 text-blue-500" />
                <div className="flex-1 text-left">
                  <p className="font-medium text-gray-900">내 드라이브</p>
                  <p className="text-sm text-gray-500">개인 스토리지 (프로젝트와 무관)</p>
                </div>
              </button>

              {/* 구분선 */}
              <div className="my-4 border-t border-gray-200" />

              {/* 프로젝트 섹션 */}
              <div className="mb-2">
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">프로젝트 스토리지</p>
              </div>

              {/* 프로젝트 목록 */}
              <div className="space-y-2">
                {projects.map((project) => (
                  <button
                    key={project.projectId}
                    onClick={() => handleSelectProject(project)}
                    className={`w-full flex items-center gap-3 p-3 rounded-lg transition ${
                      currentProjectId === project.projectId
                        ? 'bg-blue-50 border-2 border-blue-500'
                        : 'hover:bg-gray-50 border-2 border-transparent'
                    }`}
                  >
                    <FolderKanban className="w-6 h-6 text-blue-500" />
                    <div className="flex-1 min-w-0 text-left">
                      <div className="flex items-center gap-2">
                        <p className="font-medium text-gray-900 truncate">{project.name}</p>
                        {project.myPermission && (
                          <span className={`px-2 py-0.5 text-xs rounded-full flex-shrink-0 ${permissionColors[project.myPermission]}`}>
                            {permissionLabels[project.myPermission]}
                          </span>
                        )}
                      </div>
                      {project.description && (
                        <p className="text-sm text-gray-500 truncate">{project.description}</p>
                      )}
                    </div>
                  </button>
                ))}

                {projects.length === 0 && (
                  <div className="text-center py-4">
                    <p className="text-sm text-gray-500">
                      접근 가능한 프로젝트가 없습니다.
                    </p>
                    <p className="text-xs text-gray-400 mt-1">
                      워크스페이스에서 프로젝트를 생성하면 자동으로 스토리지가 연결됩니다.
                    </p>
                  </div>
                )}
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
