// src/components/storage/modals/ProjectListModal.tsx

import React, { useState, useEffect } from 'react';
import { X, FolderKanban, Settings, Trash2, MoreVertical, HardDrive } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import type { StorageProject, ProjectPermission } from '../../../types/storage';
import {
  getWorkspaceProjects,
  deleteProject,
  getMyProjectPermission,
} from '../../../api/storageService';

interface ProjectListModalProps {
  workspaceId: string;
  currentProjectId: string | null;
  onClose: () => void;
  onSelectProject: (project: StorageProject | null, permission: ProjectPermission | null) => void;
  onOpenSettings: (project: StorageProject) => void;
}

interface ProjectWithPermission extends StorageProject {
  myPermission?: ProjectPermission | null;
}

export const ProjectListModal: React.FC<ProjectListModalProps> = ({
  workspaceId,
  currentProjectId,
  onClose,
  onSelectProject,
  onOpenSettings,
}) => {
  const { theme } = useTheme();
  const [projects, setProjects] = useState<ProjectWithPermission[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [activeMenuId, setActiveMenuId] = useState<string | null>(null);

  // 프로젝트 목록 로드
  useEffect(() => {
    const loadProjects = async () => {
      setIsLoading(true);
      try {
        const projectsData = await getWorkspaceProjects(workspaceId);

        // 각 프로젝트의 권한 정보 로드
        const projectsWithPermission = await Promise.all(
          projectsData.map(async (project) => {
            const permission = await getMyProjectPermission(project.id);
            return { ...project, myPermission: permission };
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

  // 프로젝트 삭제
  const handleDeleteProject = async (projectId: string) => {
    if (!confirm('정말 이 프로젝트를 삭제하시겠습니까? 프로젝트 내 모든 파일이 삭제됩니다.')) {
      return;
    }

    try {
      await deleteProject(projectId);
      setProjects(projects.filter((p) => p.id !== projectId));

      if (currentProjectId === projectId) {
        onSelectProject(null, null);
      }
    } catch (err) {
      console.error('Failed to delete project:', err);
    }
    setActiveMenuId(null);
  };

  // 프로젝트 선택
  const handleSelectProject = (project: ProjectWithPermission) => {
    onSelectProject(project, project.myPermission || null);
    onClose();
  };

  // 전체 스토리지 선택 (프로젝트 없음)
  const handleSelectAll = () => {
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
                onClick={handleSelectAll}
                className={`w-full flex items-center gap-3 p-3 rounded-lg mb-2 transition ${
                  currentProjectId === null
                    ? 'bg-blue-50 border-2 border-blue-500'
                    : 'hover:bg-gray-50 border-2 border-transparent'
                }`}
              >
                <HardDrive className="w-6 h-6 text-blue-500" />
                <div className="flex-1 text-left">
                  <p className="font-medium text-gray-900">내 드라이브</p>
                  <p className="text-sm text-gray-500">개인 스토리지 (조직/프로젝트와 무관)</p>
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
                  <div
                    key={project.id}
                    className={`flex items-center gap-3 p-3 rounded-lg transition ${
                      currentProjectId === project.id
                        ? 'bg-blue-50 border-2 border-blue-500'
                        : 'hover:bg-gray-50 border-2 border-transparent'
                    }`}
                  >
                    <button
                      onClick={() => handleSelectProject(project)}
                      className="flex-1 flex items-center gap-3 text-left"
                    >
                      <FolderKanban className="w-6 h-6 text-blue-500" />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-gray-900 truncate">{project.name}</p>
                          {project.myPermission && (
                            <span className={`px-2 py-0.5 text-xs rounded-full ${permissionColors[project.myPermission]}`}>
                              {permissionLabels[project.myPermission]}
                            </span>
                          )}
                        </div>
                        {project.description && (
                          <p className="text-sm text-gray-500 truncate">{project.description}</p>
                        )}
                      </div>
                    </button>

                    {/* 메뉴 버튼 (OWNER만) */}
                    {project.myPermission === 'OWNER' && (
                      <div className="relative">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setActiveMenuId(activeMenuId === project.id ? null : project.id);
                          }}
                          className="p-1.5 rounded hover:bg-gray-200 transition"
                        >
                          <MoreVertical className="w-4 h-4 text-gray-500" />
                        </button>

                        {activeMenuId === project.id && (
                          <>
                            <div
                              className="fixed inset-0"
                              onClick={() => setActiveMenuId(null)}
                            />
                            <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-10 min-w-[120px]">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setActiveMenuId(null);
                                  onOpenSettings(project);
                                }}
                                className="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 flex items-center gap-2"
                              >
                                <Settings className="w-4 h-4" />
                                설정
                              </button>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleDeleteProject(project.id);
                                }}
                                className="w-full px-4 py-2 text-left text-sm hover:bg-red-50 text-red-600 flex items-center gap-2"
                              >
                                <Trash2 className="w-4 h-4" />
                                삭제
                              </button>
                            </div>
                          </>
                        )}
                      </div>
                    )}
                  </div>
                ))}

                {projects.length === 0 && (
                  <div className="text-center py-4">
                    <p className="text-sm text-gray-500">
                      접근 가능한 프로젝트가 없습니다.
                    </p>
                    <p className="text-xs text-gray-400 mt-1">
                      프로젝트에 초대받으면 여기에 표시됩니다.
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
