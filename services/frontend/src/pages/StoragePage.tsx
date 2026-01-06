// src/pages/StoragePage.tsx - Google Drive 스타일 스토리지 페이지

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useParams } from 'react-router-dom';
import MainLayout from '../components/layout/MainLayout';
import { StorageSidebar } from '../components/storage/StorageSidebar';
import { StorageHeader } from '../components/storage/StorageHeader';
import { StorageView } from '../components/storage/StorageView';
import { NewFolderModal } from '../components/storage/modals/NewFolderModal';
import { RenameModal } from '../components/storage/modals/RenameModal';
import { ShareModal } from '../components/storage/modals/ShareModal';
import { DeleteConfirmModal } from '../components/storage/modals/DeleteConfirmModal';
import { FilePreviewModal } from '../components/storage/modals/FilePreviewModal';
import { UploadProgressModal } from '../components/storage/modals/UploadProgressModal';
import { ProjectListModal } from '../components/storage/modals/ProjectListModal';
import { ProjectSettingsModal } from '../components/storage/modals/ProjectSettingsModal';
import { LoadingSpinner } from '../components/common/LoadingSpinner';

import {
  getRootContents,
  getFolderContents,
  getRecentFiles,
  getTrashFolders,
  getTrashFiles,
  getStarredItems,
  createFolder,
  deleteFolder,
  deleteFile,
  updateFolder,
  updateFile,
  restoreFolder,
  restoreFile,
  deleteFolderPermanent,
  deleteFilePermanent,
  emptyTrash,
  uploadFile,
  downloadFile,
  searchStorage,
  getStorageUsage,
} from '../api/storageService';

import type {
  StorageFolder,
  StorageFile,
  StorageProject,
  ProjectPermission,
  ViewMode,
  SortBy,
  SortDirection,
  SelectedItem,
  BreadcrumbItem,
  StorageUsage,
} from '../types/storage';

interface StoragePageProps {
  onLogout: () => void;
}

// 네비게이션 섹션 타입 (사이드바에서는 recent, starred, trash만 표시, 내부적으로 my-drive 사용)
type NavigationSection = 'my-drive' | 'recent' | 'starred' | 'trash';

const StoragePage: React.FC<StoragePageProps> = ({ onLogout }) => {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const currentWorkspaceId = workspaceId || '';

  // 파일 입력 ref
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dropZoneRef = useRef<HTMLDivElement>(null);

  // 상태 관리
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  // 뷰 상태
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [sortBy, setSortBy] = useState<SortBy>('name');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [searchQuery, setSearchQuery] = useState('');

  // 네비게이션 상태
  const [activeSection, setActiveSection] = useState<NavigationSection>('my-drive');
  const [currentFolderId, setCurrentFolderId] = useState<string | null>(null);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([
    { id: null, name: '내 드라이브', path: '/' },
  ]);

  // 데이터 상태
  const [folders, setFolders] = useState<StorageFolder[]>([]);
  const [files, setFiles] = useState<StorageFile[]>([]);
  const [storageUsage, setStorageUsage] = useState<StorageUsage | null>(null);

  // 선택 상태
  const [selectedItems, setSelectedItems] = useState<SelectedItem[]>([]);

  // 모달 상태
  const [showNewFolderModal, setShowNewFolderModal] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showPreviewModal, setShowPreviewModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [showProjectSettingsModal, setShowProjectSettingsModal] = useState(false);
  const [renameTarget, setRenameTarget] = useState<SelectedItem | null>(null);
  const [shareTarget, setShareTarget] = useState<SelectedItem | null>(null);
  const [previewFile, setPreviewFile] = useState<StorageFile | null>(null);
  const [settingsProject, setSettingsProject] = useState<StorageProject | null>(null);

  // 프로젝트 상태
  const [currentProject, setCurrentProject] = useState<StorageProject | null>(null);
  const [currentProjectPermission, setCurrentProjectPermission] = useState<ProjectPermission | null>(null);

  // 업로드 상태
  const [uploadProgress, setUploadProgress] = useState<{ fileName: string; progress: number }[]>(
    [],
  );
  const [isUploading, setIsUploading] = useState(false);

  // 사이드바 접기 상태
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  // 데이터 로드
  const loadContents = useCallback(async () => {
    if (!currentWorkspaceId) return;

    setIsLoading(true);
    setError(null);
    setSelectedItems([]);

    try {
      switch (activeSection) {
        case 'my-drive':
          if (currentFolderId) {
            const folderContents = await getFolderContents(currentWorkspaceId, currentFolderId);
            setFolders(folderContents?.children || []);
            setFiles(folderContents?.files || []);
          } else {
            const rootContents = await getRootContents(currentWorkspaceId);
            setFolders(rootContents?.children || []);
            setFiles(rootContents?.files || []);
          }
          break;

        case 'recent':
          const recentFiles = await getRecentFiles(currentWorkspaceId, 50);
          setFolders([]);
          setFiles(recentFiles);
          break;

        case 'starred':
          const starredItems = await getStarredItems();
          setFolders(starredItems.folders || []);
          setFiles(starredItems.files || []);
          break;

        case 'trash':
          const [trashFolders, trashFiles] = await Promise.all([
            getTrashFolders(currentWorkspaceId),
            getTrashFiles(currentWorkspaceId),
          ]);
          setFolders(trashFolders);
          setFiles(trashFiles);
          break;
      }

      // 스토리지 사용량 조회
      const usage = await getStorageUsage(currentWorkspaceId);
      setStorageUsage(usage);
    } catch (err: any) {
      setError(err.message || '데이터를 불러오는데 실패했습니다.');
    } finally {
      setIsLoading(false);
    }
  }, [currentWorkspaceId, currentFolderId, activeSection]);

  useEffect(() => {
    loadContents();
  }, [loadContents]);

  // 섹션 변경
  const handleSectionChange = useCallback((section: NavigationSection) => {
    setActiveSection(section);
    setCurrentFolderId(null);
    setSearchQuery('');

    // 브레드크럼 초기화
    const sectionNames: Record<NavigationSection, string> = {
      'my-drive': '내 드라이브',
      recent: '최근 항목',
      starred: '중요 항목',
      trash: '휴지통',
    };
    setBreadcrumbs([{ id: null, name: sectionNames[section], path: '/' }]);
  }, []);

  // 폴더 열기
  const handleFolderOpen = useCallback(
    (folder: StorageFolder) => {
      if (activeSection === 'trash') return; // 휴지통에서는 폴더 열기 불가

      setCurrentFolderId(folder.id);
      setBreadcrumbs((prev) => [...prev, { id: folder.id, name: folder.name, path: folder.path }]);
    },
    [activeSection],
  );

  // 브레드크럼 네비게이션
  const handleBreadcrumbClick = useCallback((item: BreadcrumbItem) => {
    setCurrentFolderId(item.id);

    // 클릭한 항목까지만 브레드크럼 유지
    setBreadcrumbs((prev) => {
      const index = prev.findIndex((b) => b.id === item.id);
      return prev.slice(0, index + 1);
    });
  }, []);

  // 파일 다운로드
  const handleFileDownload = useCallback(async (file: StorageFile) => {
    try {
      await downloadFile(file.id, file.name);
    } catch (err: any) {
      setError('파일 다운로드에 실패했습니다.');
    }
  }, []);

  // 파일 미리보기
  const handleFilePreview = useCallback((file: StorageFile) => {
    setPreviewFile(file);
    setShowPreviewModal(true);
  }, []);

  // 새 폴더 생성
  const handleCreateFolder = useCallback(
    async (name: string, color?: string) => {
      try {
        await createFolder({
          workspaceId: currentWorkspaceId,
          projectId: currentProject?.id,
          parentId: currentFolderId || undefined,
          name,
          color,
        });
        setShowNewFolderModal(false);
        loadContents();
      } catch (err: any) {
        setError('폴더 생성에 실패했습니다.');
      }
    },
    [currentWorkspaceId, currentProject, currentFolderId, loadContents],
  );

  // 프로젝트 선택
  const handleSelectProject = useCallback((project: StorageProject | null, permission: ProjectPermission | null) => {
    setCurrentProject(project);
    setCurrentProjectPermission(permission);
    setCurrentFolderId(null);
    setBreadcrumbs([{ id: null, name: project ? project.name : '내 드라이브', path: '/' }]);
  }, []);

  // 프로젝트 설정 열기
  const handleOpenProjectSettings = useCallback((project: StorageProject) => {
    setSettingsProject(project);
    setShowProjectModal(false);
    setShowProjectSettingsModal(true);
  }, []);

  // 프로젝트 정보 업데이트
  const handleProjectUpdated = useCallback((updatedProject: StorageProject) => {
    setSettingsProject(updatedProject);
    if (currentProject?.id === updatedProject.id) {
      setCurrentProject(updatedProject);
    }
  }, [currentProject]);

  // 이름 변경
  const handleRename = useCallback(
    async (newName: string) => {
      if (!renameTarget) return;

      try {
        if (renameTarget.type === 'folder') {
          await updateFolder(renameTarget.id, { name: newName });
        } else {
          await updateFile(renameTarget.id, { name: newName });
        }
        setShowRenameModal(false);
        setRenameTarget(null);
        loadContents();
      } catch (err: any) {
        setError('이름 변경에 실패했습니다.');
      }
    },
    [renameTarget, loadContents],
  );

  // 삭제
  const handleDelete = useCallback(async () => {
    try {
      const isPermanent = activeSection === 'trash';

      for (const item of selectedItems) {
        if (item.type === 'folder') {
          isPermanent ? await deleteFolderPermanent(item.id) : await deleteFolder(item.id);
        } else {
          isPermanent ? await deleteFilePermanent(item.id) : await deleteFile(item.id);
        }
      }

      setShowDeleteModal(false);
      setSelectedItems([]);
      loadContents();
    } catch (err: any) {
      setError('삭제에 실패했습니다.');
    }
  }, [selectedItems, activeSection, loadContents]);

  // 복원 (휴지통)
  const handleRestore = useCallback(async () => {
    try {
      for (const item of selectedItems) {
        if (item.type === 'folder') {
          await restoreFolder(item.id);
        } else {
          await restoreFile(item.id);
        }
      }
      setSelectedItems([]);
      loadContents();
    } catch (err: any) {
      setError('복원에 실패했습니다.');
    }
  }, [selectedItems, loadContents]);

  // 휴지통 비우기
  const handleEmptyTrash = useCallback(async () => {
    if (!window.confirm('휴지통을 비우시겠습니까? 이 작업은 되돌릴 수 없습니다.')) return;

    try {
      await emptyTrash(currentWorkspaceId);
      loadContents();
    } catch (err: any) {
      setError('휴지통 비우기에 실패했습니다.');
    }
  }, [currentWorkspaceId, loadContents]);

  // 파일 업로드
  const handleFileUpload = useCallback(
    async (files: FileList) => {
      if (!files.length) return;

      setIsUploading(true);
      const uploadList = Array.from(files);
      setUploadProgress(uploadList.map((f) => ({ fileName: f.name, progress: 0 })));

      try {
        for (let i = 0; i < uploadList.length; i++) {
          const file = uploadList[i];
          await uploadFile(file, currentWorkspaceId, currentFolderId || undefined, (progress) => {
            setUploadProgress((prev) => prev.map((p, idx) => (idx === i ? { ...p, progress } : p)));
          });
        }
        loadContents();
      } catch (err: any) {
        setError('파일 업로드에 실패했습니다.');
      } finally {
        setIsUploading(false);
        setUploadProgress([]);
      }
    },
    [currentWorkspaceId, currentFolderId, loadContents],
  );

  // 파일 선택 트리거
  const triggerFileUpload = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  // 드래그앤드롭 핸들러
  const handleDragEnter = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (activeSection !== 'trash' && activeSection !== 'shared') {
        setIsDragging(true);
      }
    },
    [activeSection],
  );

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    // 드롭존 영역을 벗어났는지 확인
    const relatedTarget = e.relatedTarget as Node;
    if (!dropZoneRef.current?.contains(relatedTarget)) {
      setIsDragging(false);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);

      if (activeSection === 'trash' || activeSection === 'shared') {
        return;
      }

      const files = e.dataTransfer.files;
      if (files && files.length > 0) {
        handleFileUpload(files);
      }
    },
    [activeSection, handleFileUpload],
  );

  // 검색
  const handleSearch = useCallback(
    async (query: string) => {
      if (!query.trim()) {
        loadContents();
        return;
      }

      setIsLoading(true);
      try {
        const results = await searchStorage(currentWorkspaceId, query);
        setFolders(results.folders);
        setFiles(results.files);
      } catch (err: any) {
        setError('검색에 실패했습니다.');
      } finally {
        setIsLoading(false);
      }
    },
    [currentWorkspaceId, loadContents],
  );

  // 정렬
  const handleSort = useCallback(
    (newSortBy: SortBy) => {
      if (sortBy === newSortBy) {
        setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
      } else {
        setSortBy(newSortBy);
        setSortDirection('asc');
      }
    },
    [sortBy],
  );

  // 정렬된 데이터
  const sortedFolders = [...folders].sort((a, b) => {
    let compare = 0;
    switch (sortBy) {
      case 'name':
        compare = a.name.localeCompare(b.name);
        break;
      case 'modifiedAt':
        compare = new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime();
        break;
      default:
        compare = a.name.localeCompare(b.name);
    }
    return sortDirection === 'asc' ? compare : -compare;
  });

  const sortedFiles = [...files].sort((a, b) => {
    let compare = 0;
    switch (sortBy) {
      case 'name':
        compare = a.name.localeCompare(b.name);
        break;
      case 'modifiedAt':
        compare = new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime();
        break;
      case 'size':
        compare = a.fileSize - b.fileSize;
        break;
      case 'type':
        compare = a.extension.localeCompare(b.extension);
        break;
      default:
        compare = a.name.localeCompare(b.name);
    }
    return sortDirection === 'asc' ? compare : -compare;
  });

  // 컨텍스트 메뉴 핸들러
  const openRenameModal = useCallback((item: SelectedItem) => {
    setRenameTarget(item);
    setShowRenameModal(true);
  }, []);

  const openShareModal = useCallback((item: SelectedItem) => {
    setShareTarget(item);
    setShowShareModal(true);
  }, []);

  const openDeleteModal = useCallback(() => {
    if (selectedItems.length > 0) {
      setShowDeleteModal(true);
    }
  }, [selectedItems]);

  return (
    <MainLayout onLogout={onLogout} workspaceId={currentWorkspaceId} onProfileModalOpen={() => {}}>
      <div
        ref={dropZoneRef}
        className="flex h-screen bg-gray-50 relative"
        onDragEnter={handleDragEnter}
        onDragLeave={handleDragLeave}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        {/* 드래그 오버레이 */}
        {isDragging && (
          <div className="absolute inset-0 bg-blue-500/20 border-4 border-dashed border-blue-500 z-50 flex items-center justify-center pointer-events-none">
            <div className="bg-white rounded-xl shadow-2xl p-8 text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-blue-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-blue-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                  />
                </svg>
              </div>
              <p className="text-lg font-medium text-gray-900">파일을 여기에 놓으세요</p>
              <p className="text-sm text-gray-500 mt-1">파일을 드롭하여 업로드합니다</p>
            </div>
          </div>
        )}
        {/* 스토리지 사이드바 */}
        <StorageSidebar
          activeSection={activeSection}
          onSectionChange={handleSectionChange}
          onNewFolder={() => setShowNewFolderModal(true)}
          onUpload={triggerFileUpload}
          storageUsage={storageUsage}
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
          currentProject={currentProject}
          currentProjectPermission={currentProjectPermission}
          onOpenProjectModal={() => setShowProjectModal(true)}
        />

        {/* 메인 콘텐츠 영역 - flexbox로 자동 조절 */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* 헤더 */}
          <StorageHeader
            breadcrumbs={breadcrumbs}
            onBreadcrumbClick={handleBreadcrumbClick}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            sortBy={sortBy}
            sortDirection={sortDirection}
            onSort={handleSort}
            searchQuery={searchQuery}
            onSearch={handleSearch}
            onSearchChange={setSearchQuery}
            selectedCount={selectedItems.length}
            onDelete={openDeleteModal}
            onRestore={activeSection === 'trash' ? handleRestore : undefined}
            onEmptyTrash={activeSection === 'trash' ? handleEmptyTrash : undefined}
            isTrash={activeSection === 'trash'}
            projectPermission={currentProjectPermission}
          />

          {/* 에러 메시지 */}
          {error && (
            <div className="mx-6 mt-4 p-4 bg-red-50 border border-red-300 rounded-lg text-red-700">
              {error}
              <button
                onClick={() => setError(null)}
                className="ml-4 text-red-500 hover:text-red-700"
              >
                닫기
              </button>
            </div>
          )}

          {/* 파일/폴더 뷰 */}
          {isLoading ? (
            <div className="flex-1 flex items-center justify-center">
              <LoadingSpinner message="로딩 중..." />
            </div>
          ) : (
            <StorageView
              viewMode={viewMode}
              folders={sortedFolders}
              files={sortedFiles}
              sharedItems={undefined}
              selectedItems={selectedItems}
              onSelectItem={setSelectedItems}
              onFolderOpen={handleFolderOpen}
              onFileDownload={handleFileDownload}
              onFilePreview={handleFilePreview}
              onRename={openRenameModal}
              onShare={openShareModal}
              onDelete={openDeleteModal}
              onRestore={activeSection === 'trash' ? handleRestore : undefined}
              onNewFolder={() => setShowNewFolderModal(true)}
              onUpload={triggerFileUpload}
              isTrash={activeSection === 'trash'}
              isEmpty={folders.length === 0 && files.length === 0}
              projectPermission={currentProjectPermission}
              activeSection={activeSection}
            />
          )}
        </div>
      </div>

      {/* 숨겨진 파일 입력 */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={(e) => e.target.files && handleFileUpload(e.target.files)}
      />

      {/* 모달들 */}
      {showNewFolderModal && (
        <NewFolderModal
          onClose={() => setShowNewFolderModal(false)}
          onCreate={handleCreateFolder}
        />
      )}

      {showRenameModal && renameTarget && (
        <RenameModal
          item={renameTarget}
          onClose={() => {
            setShowRenameModal(false);
            setRenameTarget(null);
          }}
          onRename={handleRename}
        />
      )}

      {showShareModal && shareTarget && (
        <ShareModal
          item={shareTarget}
          workspaceId={currentWorkspaceId}
          onClose={() => {
            setShowShareModal(false);
            setShareTarget(null);
          }}
        />
      )}

      {showDeleteModal && (
        <DeleteConfirmModal
          items={selectedItems}
          isPermanent={activeSection === 'trash'}
          onClose={() => setShowDeleteModal(false)}
          onConfirm={handleDelete}
        />
      )}

      {showPreviewModal && previewFile && (
        <FilePreviewModal
          file={previewFile}
          onClose={() => {
            setShowPreviewModal(false);
            setPreviewFile(null);
          }}
          onDownload={() => handleFileDownload(previewFile)}
        />
      )}

      {isUploading && uploadProgress.length > 0 && (
        <UploadProgressModal
          uploads={uploadProgress}
          onClose={() => {
            setIsUploading(false);
            setUploadProgress([]);
          }}
        />
      )}

      {showProjectModal && (
        <ProjectListModal
          workspaceId={currentWorkspaceId}
          currentProjectId={currentProject?.id || null}
          onClose={() => setShowProjectModal(false)}
          onSelectProject={handleSelectProject}
          onOpenSettings={handleOpenProjectSettings}
        />
      )}

      {showProjectSettingsModal && settingsProject && (
        <ProjectSettingsModal
          project={settingsProject}
          workspaceId={currentWorkspaceId}
          onClose={() => {
            setShowProjectSettingsModal(false);
            setSettingsProject(null);
          }}
          onProjectUpdated={handleProjectUpdated}
        />
      )}
    </MainLayout>
  );
};

export default StoragePage;
