// src/components/modals/board/ProjectManageModal.tsx

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  X,
  Paperclip,
  Download,
  Edit2,
  BarChart3,
  Lock,
  Loader2,
  User as UserIcon,
  Trash2,
} from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import { createProject, updateProject, getBoardsByProject, deleteProject, getProjectMembers } from '../../../api/boardService';
import {
  ProjectResponse,
  BoardResponse,
  CreateProjectRequest,
  UpdateProjectRequest,
  AttachmentResponse,
  ProjectMemberResponse,
} from '../../../types/board';
import { IROLES } from '../../../types/common';
import Portal from '../../common/Portal';

import { useFileUpload } from '../../../hooks/useFileUpload';
import { FileUploader } from '../../common/FileUploader';

/**
 * 모달 모드 타입 정의
 */
type ProjectModalMode = 'create' | 'detail' | 'edit';

interface ProjectManageModalProps {
  workspaceId: string;
  project?: ProjectResponse;
  onClose: () => void;
  onProjectSaved: () => void;
  onProjectCreated?: (createObj: ProjectResponse) => void;
  userRole: IROLES;
  initialMode: ProjectModalMode;
}

// 파일 다운로드 핸들러 (Detail 모드용)
const handleFileDownload = (fileUrl: string, fileName: string) => {
  if (!fileUrl) return;

  const link = document.createElement('a');
  link.href = fileUrl;
  link.setAttribute('download', fileName);
  link.setAttribute('target', '_blank');
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

// 이미지 파일 여부 확인 함수
const isImageFile = (contentType?: string, fileName?: string): boolean => {
  if (contentType) {
    return contentType.startsWith('image/');
  }
  if (fileName) {
    const ext = fileName.split('.').pop()?.toLowerCase();
    return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext || '');
  }
  return false;
};

export const ProjectManageModal: React.FC<ProjectManageModalProps> = ({
  workspaceId,
  project,
  onClose,
  onProjectSaved,
  onProjectCreated,
  userRole,
  initialMode = 'create',
}) => {
  const { theme } = useTheme();
  const isExistingProject = !!project;
  const [mode, setMode] = useState<ProjectModalMode>(isExistingProject ? initialMode : 'create');

  // 💡 [추가] 모달 내에서 프로젝트 데이터의 최신 상태를 관리하기 위한 상태
  const [currentProject, setCurrentProject] = useState<ProjectResponse | undefined>(project);

  // Form state (currentProject를 기반으로 초기화)
  const [name, setName] = useState(currentProject?.name || '');
  const [description, setDescription] = useState(currentProject?.description || '');
  const [startDate, setStartDate] = useState(
    currentProject?.startDate ? currentProject.startDate.substring(0, 10) : '',
  );
  const [dueDate, setDueDate] = useState(
    currentProject?.dueDate ? currentProject.dueDate.substring(0, 10) : '',
  );

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Board & Member state
  const [boards, setBoards] = useState<BoardResponse[]>([]);
  const [isBoardsLoading, setIsBoardsLoading] = useState(false);
  const [projectMembers, setProjectMembers] = useState<ProjectMemberResponse[]>([]);
  const [isMembersLoading, setIsMembersLoading] = useState(false);

  // Attachment state (for UI display and hook initialization)
  const [firstAttachmentState, setFirstAttachmentState] = useState<AttachmentResponse | undefined>(
    currentProject?.attachments?.[0],
  );
  const [currentAttachmentId, setCurrentAttachmentId] = useState<string | undefined>(
    currentProject?.attachments?.[0]?.attachmentId,
  );

  // File Upload Hook
  const {
    selectedFile,
    previewUrl,
    handleFileSelect,
    handleRemoveFile,
    upload,
    setInitialFile,
    setAttachmentId,
  } = useFileUpload();

  // UI state
  const [previewImage, setPreviewImage] = useState<string | null>(null);

  const canEdit = useMemo(() => {
    return (
      isExistingProject &&
      (userRole === 'OWNER' || userRole === 'ADMIN' || userRole === 'ORGANIZER')
    );
  }, [isExistingProject, userRole]);

  // 프로젝트 멤버 API 조회
  const fetchProjectMembers = useCallback(async () => {
    const projectToFetch = currentProject || project;
    if (!projectToFetch?.projectId) {
      setProjectMembers([]);
      return;
    }
    setIsMembersLoading(true);
    try {
      const membersData = await getProjectMembers(projectToFetch.projectId);
      setProjectMembers(membersData || []);
    } catch (err) {
      console.error('❌ Failed to fetch project members:', err);
      setProjectMembers([]);
    } finally {
      setIsMembersLoading(false);
    }
  }, [currentProject, project]);

  useEffect(() => {
    fetchProjectMembers();
  }, [fetchProjectMembers]);

  // 프로젝트 데이터 로드 및 파일 상태 초기화 (메인 useEffect)
  useEffect(() => {
    // 💡 [수정] project prop 또는 currentProject 로컬 상태를 사용
    const projectToUse = currentProject || project;

    if (projectToUse) {
      setName(projectToUse.name);
      setDescription(projectToUse.description || '');
      setStartDate(projectToUse.startDate ? projectToUse.startDate.substring(0, 10) : '');
      setDueDate(projectToUse.dueDate ? projectToUse.dueDate.substring(0, 10) : '');

      const initialAttachment = projectToUse.attachments?.[0];
      setFirstAttachmentState(initialAttachment);
      setCurrentAttachmentId(initialAttachment?.attachmentId);

      if (initialAttachment) {
        setInitialFile(initialAttachment.fileUrl, initialAttachment.fileName);
        setAttachmentId(initialAttachment.attachmentId);
      } else {
        handleRemoveFile();
      }
    } else if (mode === 'create') {
      // 생성 모드 초기화
      setName('');
      setDescription('');
      setStartDate('');
      setDueDate('');
      setFirstAttachmentState(undefined);
      setCurrentAttachmentId(undefined);
      handleRemoveFile();
    }
    setError(null);
  }, [
    project,
    currentProject, // 💡 [추가] currentProject 변경 시 재실행
    mode,
    setInitialFile,
    handleRemoveFile,
    setAttachmentId,
  ]);

  const fetchBoards = useCallback(async () => {
    // 💡 [수정] currentProject를 사용하거나 project prop을 사용
    const projectToFetch = currentProject || project;
    if (!projectToFetch || mode !== 'detail') {
      setBoards([]);
      return;
    }
    setIsBoardsLoading(true);
    try {
      const response = await getBoardsByProject(projectToFetch.projectId);
      setBoards(response || []);
    } catch (err) {
      console.error('❌ Failed to fetch boards for statistics:', err);
      setBoards([]);
    } finally {
      setIsBoardsLoading(false);
    }
  }, [currentProject, project, mode]); // 💡 [수정] currentProject 의존성 추가

  useEffect(() => {
    fetchBoards();
  }, [fetchBoards]);

  const projectStats = useMemo(() => {
    const totalBoards = boards.length;
    const inProgressBoards = boards.filter((b) => (b as any).status === 'IN_PROGRESS').length;
    const delayedBoards = boards.filter((b) => (b as any).isDelayed).length;
    return {
      totalBoards,
      inProgressBoards,
      delayedBoards,
    };
  }, [boards]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      setError('프로젝트 이름은 필수입니다.');
      return;
    }

    setIsLoading(true);
    setError(null);

    let attachmentIdsPayload: string[] | undefined = undefined;

    try {
      if (selectedFile) {
        // 1. 새 파일 업로드
        const uploadResult = await upload(workspaceId, 'project');
        if (uploadResult) {
          attachmentIdsPayload = [uploadResult.attachmentId];
        }
      } else if (mode === 'edit' && !previewUrl && currentAttachmentId) {
        // 2. 기존 파일 삭제 의도
        attachmentIdsPayload = [];
      }

      const projectBaseData = {
        name: name.trim(),
        description: description.trim() || undefined,
        startDate: startDate ? `${startDate}T00:00:00Z` : undefined,
        dueDate: dueDate ? `${dueDate}T00:00:00Z` : undefined,
      };

      if (mode === 'edit' && project) {
        const updatePayload: UpdateProjectRequest = {
          ...projectBaseData,
          attachmentIds: attachmentIdsPayload,
        };

        const updatedProject = await updateProject(project.projectId, updatePayload);

        alert(`✅ ${name} 프로젝트가 수정되었습니다!`);

        // 💡 [핵심 수정] 서버 응답으로 로컬 프로젝트 상태를 갱신
        setCurrentProject(updatedProject);

        // 수정 성공 후, 최신 첨부 파일 정보를 상태에 반영
        const newAttachment = updatedProject.attachments?.[0];
        setFirstAttachmentState(newAttachment);
        setCurrentAttachmentId(newAttachment?.attachmentId);

        // 파일 업로드 훅 상태 갱신
        if (newAttachment) {
          setInitialFile(newAttachment.fileUrl, newAttachment.fileName);
          setAttachmentId(newAttachment.attachmentId);
        } else {
          handleRemoveFile();
        }

        // 보드 정보 fetch를 통해 상세 보기 모드의 통계 갱신 (currentProject 갱신으로 fetchBoards가 재실행됨)
        // await fetchBoards(); // currentProject 의존성이 추가되었으므로 불필요

        onProjectSaved();
        setMode('detail');
      } else if (mode === 'create') {
        const createPayload: CreateProjectRequest = {
          workspaceId: workspaceId,
          ...projectBaseData,
          attachmentIds: attachmentIdsPayload,
        };
        const newProjectResponse: ProjectResponse = await createProject(createPayload);
        alert(`✅ ${name} 프로젝트가 생성되었습니다!`);
        if (newProjectResponse) {
          onProjectCreated?.(newProjectResponse);
        }
        onProjectSaved();
        onClose();
      }
    } catch (err: any) {
      const errorMsg = err.response?.data?.error?.message || err.message;
      console.error(mode === 'create' ? '❌ 생성 실패:' : '❌ 수정 실패:', errorMsg);
      setError(errorMsg || '작업 처리에 실패했습니다.');
    } finally {
      setIsLoading(false);
    }
  };

  // ========================================
  // 프로젝트 삭제 핸들러
  // ========================================

  const handleDeleteProject = async () => {
    if (!project) return;

    const confirmed = window.confirm(
      `정말로 "${project.name}" 프로젝트를 삭제하시겠습니까?\n\n삭제된 프로젝트는 복구할 수 없으며, 모든 보드와 데이터가 함께 삭제됩니다.`
    );

    if (!confirmed) return;

    try {
      setIsLoading(true);
      setError(null);
      await deleteProject(project.projectId);
      alert('프로젝트가 삭제되었습니다.');
      onProjectSaved();
      onClose();
    } catch (err: any) {
      console.error('❌ 프로젝트 삭제 실패:', err);
      const errorMsg = err.response?.data?.error?.message || err.message;
      setError(`프로젝트 삭제에 실패했습니다: ${errorMsg}`);
    } finally {
      setIsLoading(false);
    }
  };

  // 💡 [추가] 렌더링에 사용할 최종 프로젝트 데이터
  const projectToDisplay = currentProject || project;

  const modalTitle = useMemo(() => {
    switch (mode) {
      case 'create':
        return '새 프로젝트 만들기';
      case 'edit':
        return `${projectToDisplay?.name || '프로젝트'} 수정`; // 💡 projectToDisplay 사용
      default:
        return `${projectToDisplay?.name || '프로젝트'} 상세 정보`; // 💡 projectToDisplay 사용
    }
  }, [mode, projectToDisplay?.name]);

  // 상세 보기용 파일 정보 (firstAttachmentState 사용)
  const detailFileUrl = firstAttachmentState?.fileUrl || '';
  const detailFileName = firstAttachmentState?.fileName || 'project_file_attachment';
  const hasAttachments = !!firstAttachmentState;

  // 공통 입력 필드 스타일
  const inputBaseStyle = `w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm`;
  const detailInputStyle = `${inputBaseStyle} bg-gray-100 text-gray-700`;
  const editInputStyle = `${inputBaseStyle} bg-white`;

  // 입력 필드 렌더링 함수 (생략)
  const renderInputField = (
    label: string,
    value: string,
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void,
    type: 'text' | 'date' | 'textarea',
    required: boolean = false,
    maxLength?: number,
    placeholder?: string,
    rows?: number,
  ) => {
    const isDetailMode = mode === 'detail';
    const inputClass = isDetailMode ? detailInputStyle : editInputStyle;

    const labelElement = (
      <label className="block text-sm font-semibold text-gray-700 mb-2">
        {label} {required && <span className="text-red-500">*</span>}
      </label>
    );

    let inputElement;
    if (type === 'textarea') {
      inputElement = (
        <textarea
          value={value}
          onChange={onChange}
          disabled={isDetailMode || isLoading}
          className={`${inputClass} resize-none`}
          rows={rows || 5}
          maxLength={maxLength}
          placeholder={placeholder}
        />
      );
    } else {
      inputElement = (
        <input
          type={type}
          value={value}
          onChange={onChange}
          disabled={isDetailMode || isLoading}
          className={inputClass}
          maxLength={maxLength}
          placeholder={placeholder}
          autoFocus={type === 'text' && mode === 'create'}
        />
      );
    }

    return (
      <div>
        {labelElement}
        {inputElement}
      </div>
    );
  };

  // ========================================
  // 🎨 Detail / Edit Mode 렌더링
  // ========================================
  const renderDetailOrEditContent = () => (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-3 gap-6">
        <div className="col-span-2 space-y-4">
          {/* Name */}
          {renderInputField(
            '프로젝트 이름',
            name,
            (e) => setName(e.target.value),
            'text',
            true,
            100,
          )}

          {/* Dates */}
          <div className="grid grid-cols-2 gap-4">
            {renderInputField('시작일', startDate, (e) => setStartDate(e.target.value), 'date')}
            {renderInputField('마감일', dueDate, (e) => setDueDate(e.target.value), 'date')}
          </div>

          {/* Description */}
          {renderInputField(
            '프로젝트 설명',
            description,
            (e) => setDescription(e.target.value),
            'textarea',
            false,
            800,
            undefined,
            10,
          )}

          {/* Files 섹션 */}
          <div className="pt-0">
            {mode === 'edit' ? (
              // ✏️ 수정 모드: 파일 업로더 표시
              <FileUploader
                selectedFile={selectedFile}
                previewUrl={previewUrl}
                onFileSelect={handleFileSelect}
                onRemoveFile={handleRemoveFile}
                existingFileName={firstAttachmentState?.fileName}
                disabled={isLoading}
                label="첨부 파일 수정"
              />
            ) : (
              // 📖 상세 보기 모드: 다운로드 UI 표시
              <>
                <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-1">
                  <Paperclip className="w-4 h-4 text-blue-500" />
                  첨부 파일
                </label>
                <div
                  className="p-2 bg-gray-50 border border-gray-200 rounded-lg flex items-center justify-between text-sm relative"
                  onMouseEnter={() => {
                    if (
                      detailFileUrl &&
                      firstAttachmentState?.contentType &&
                      isImageFile(firstAttachmentState.contentType, detailFileName)
                    ) {
                      setPreviewImage(detailFileUrl);
                    }
                  }}
                  onMouseLeave={() => setPreviewImage(null)}
                >
                  <span className="text-gray-700 truncate flex items-center gap-1">
                    {hasAttachments ? (
                      <span className="text-gray-700">{detailFileName}</span>
                    ) : (
                      <span className="text-gray-500">첨부 파일 없음</span>
                    )}
                  </span>
                  {hasAttachments ? (
                    <button
                      type="button"
                      onClick={() => handleFileDownload(detailFileUrl, detailFileName)}
                      className="flex items-center gap-1 text-blue-600 hover:text-blue-700 transition font-medium ml-2 flex-shrink-0"
                    >
                      <Download className="w-4 h-4" />
                      <span className="text-xs">다운로드</span>
                    </button>
                  ) : (
                    <span className="text-gray-400 text-xs flex-shrink-0">다운로드 불가</span>
                  )}
                  {/* 이미지 미리보기 툴팁 */}
                  {previewImage && mode === 'detail' && (
                    <div className="absolute left-0 bottom-full **mb-6** z-50 pointer-events-none">
                      <div className="bg-white border-2 border-gray-300 rounded-lg shadow-2xl p-2">
                        <img
                          src={previewImage}
                          alt="미리보기"
                          className="max-w-xs max-h-64 rounded"
                          style={{ objectFit: 'contain' }}
                        />
                      </div>
                    </div>
                  )}
                </div>
              </>
            )}
          </div>
        </div>

        {/* Right Section */}
        <div className="col-span-1 space-y-4 divide-y divide-gray-200 pl-4 border-l border-gray-200">
          {projectToDisplay && ( // 💡 [수정] projectToDisplay 사용
            <div className="pb-4">
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-1">
                <UserIcon className="w-4 h-4 text-gray-500" />
                프로젝트 소유자
              </label>
              <div className="text-sm font-medium text-gray-700 ml-1">
                {projectToDisplay?.ownerName}
              </div>{' '}
              {/* 💡 [수정] projectToDisplay 사용 */}
            </div>
          )}

          <div className="pt-4">
            <h3 className="text-md font-bold text-gray-800 mb-2">
              소속 멤버 ({projectMembers?.length}명)
            </h3>
            <div className="max-h-56 overflow-y-auto space-y-2">
              {isMembersLoading ? (
                <div className="flex justify-center items-center py-4 text-gray-500">
                  <Loader2 className="w-4 h-4 animate-spin mr-2" />
                  멤버 로딩 중...
                </div>
              ) : projectMembers?.length === 0 ? (
                <p className="text-sm text-gray-500 py-2">소속 멤버가 없습니다.</p>
              ) : (
                projectMembers?.map((member) => (
                  <div
                    key={member?.memberId || member?.userId}
                    className="flex items-center justify-between p-2 rounded-lg hover:bg-gray-100 transition"
                  >
                    <span className="text-sm">{member?.userName || member?.userEmail || 'Unknown'}</span>
                    <span
                      className={`text-xs px-2 py-0.5 rounded-full ${
                        member?.roleName === 'OWNER'
                          ? 'bg-red-100 text-red-600'
                          : member?.roleName === 'MEMBER'
                          ? 'bg-blue-100 text-blue-600'
                          : 'bg-gray-100 text-gray-500'
                      }`}
                    >
                      {member?.roleName}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {mode === 'detail' && (
            <div className="pt-4">
              <h3 className="text-md font-bold text-gray-800 flex items-center gap-2 mb-3">
                <BarChart3 className="w-5 h-5 text-indigo-500" /> 프로젝트 현황
              </h3>
              {isBoardsLoading ? (
                <div className="flex justify-center items-center py-4 text-gray-500">
                  <Loader2 className="w-5 h-5 animate-spin mr-2" />
                  통계 데이터 로드 중...
                </div>
              ) : (
                <div className="grid grid-cols-3 gap-3 text-center">
                  <div className="p-3 bg-indigo-50 rounded-lg border border-indigo-200">
                    <p className="text-2xl font-bold text-indigo-700">{projectStats.totalBoards}</p>
                    <p className="text-xs text-indigo-500 mt-1">총 보드 수</p>
                  </div>
                  <div className="p-3 bg-green-50 rounded-lg border border-green-200">
                    <p className="text-2xl font-bold text-green-700">
                      {projectStats.inProgressBoards}
                    </p>
                    <p className="text-xs text-green-500 mt-1">진행 중</p>
                  </div>
                  <div className="p-3 bg-red-50 rounded-lg border border-red-200">
                    <p className="text-2xl font-bold text-red-700">{projectStats.delayedBoards}</p>
                    <p className="text-xs text-red-500 mt-1">지연</p>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* 위험 구역 - 프로젝트 삭제 (detail 모드에서만, OWNER/ADMIN만) */}
      {mode === 'detail' && canEdit && (
        <div className="mt-6 pt-4 border-t border-gray-200">
          <div className="flex items-center justify-between p-3 bg-red-50 rounded-lg">
            <div>
              <p className="text-sm font-medium text-red-600">프로젝트 삭제</p>
              <p className="text-xs text-gray-500">
                모든 보드와 데이터가 함께 삭제됩니다
              </p>
            </div>
            <button
              type="button"
              onClick={handleDeleteProject}
              disabled={isLoading}
              className={`p-2 rounded-lg text-red-500 hover:bg-red-100 transition ${
                isLoading ? 'opacity-50 cursor-not-allowed' : ''
              }`}
              title="프로젝트 삭제"
            >
              <Trash2 className="w-5 h-5" />
            </button>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-3 pt-4 px-6 sticky bottom-0 bg-white border-t border-gray-300">
        {mode === 'edit' && (
          <button
            type="button"
            onClick={() => {
              setMode('detail');
              // 수정 취소 시 폼 상태 및 파일 상태를 project prop 기준으로 재설정
              if (project) {
                // 💡 [수정] currentProject를 prop 기준으로 초기화하여 취소 시에도 prop을 따르도록 함
                setCurrentProject(project);

                // 나머지 폼 상태는 useEffect가 currentProject를 보고 재설정할 것임.
              }
            }}
            className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 font-semibold rounded-lg hover:bg-gray-50 transition"
            disabled={isLoading}
          >
            취소 (상세 보기로)
          </button>
        )}

        {mode === 'detail' && (
          <button
            type="button"
            onClick={onClose}
            className="flex-1 px-4 py-2 bg-gray-200 text-gray-700 font-semibold rounded-lg hover:bg-gray-300 transition"
          >
            닫기
          </button>
        )}

        {(mode === 'edit' || mode === 'create') && (
          <button
            type="submit"
            className={`flex-1 px-4 py-2 bg-blue-500 text-white font-semibold rounded-lg hover:bg-blue-600 transition ${
              isLoading ? 'opacity-50 cursor-not-allowed' : ''
            }`}
            disabled={isLoading}
          >
            {isLoading
              ? mode === 'edit'
                ? '저장 중...'
                : '생성 중...'
              : mode === 'edit'
              ? '수정 내용 저장'
              : '프로젝트 만들기'}
          </button>
        )}
      </div>
    </form>
  );

  // ----------------------------------------------------
  // 🎨 Create Mode 렌더링
  // ----------------------------------------------------
  const renderCreateContent = () => (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-3 gap-6">
        <div className="col-span-2 space-y-4">
          {/* Name */}
          {renderInputField(
            '프로젝트 이름',
            name,
            (e) => setName(e.target.value),
            'text',
            true,
            100,
            '예: Wealist 서비스 개발',
          )}

          {/* Dates */}
          <div className="grid grid-cols-2 gap-4">
            {renderInputField(
              '시작일 (선택)',
              startDate,
              (e) => setStartDate(e.target.value),
              'date',
            )}
            {renderInputField('마감일 (선택)', dueDate, (e) => setDueDate(e.target.value), 'date')}
          </div>

          {/* Description */}
          {renderInputField(
            '프로젝트 설명 (선택)',
            description,
            (e) => setDescription(e.target.value),
            'textarea',
            false,
            500,
            '프로젝트에 대한 간단한 설명을 입력하세요',
            5,
          )}

          {/* FileUploader */}
          <div>
            <FileUploader
              selectedFile={selectedFile}
              previewUrl={previewUrl}
              onFileSelect={handleFileSelect}
              onRemoveFile={handleRemoveFile}
              disabled={isLoading}
              label="첨부 파일 (선택)"
            />
          </div>
        </div>

        {/* Right Section (Instructions) */}
        <div className="col-span-1 space-y-4 divide-y divide-gray-200 pl-4 border-l border-gray-200">
          <div className="pb-4">
            <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-1">
              <UserIcon className="w-4 h-4 text-gray-500" />
              프로젝트 생성 안내
            </label>
            <p className="text-xs text-gray-500">
              프로젝트 생성 시, 자동으로 소유자(Owner) 역할을 갖게 됩니다. 생성 후 멤버를
              초대하거나, 설정을 변경할 수 있습니다.
            </p>
          </div>

          <div className="pt-4">
            <h3 className="text-md font-bold text-gray-800 mb-2">마감일 설정 Tip</h3>
            <p className="text-xs text-gray-600">
              시작일과 마감일을 명확히 설정하면, 보드 현황판에서 **지연된 보드**를 정확하게 파악할
              수 있습니다.
            </p>
          </div>
        </div>
      </div>

      <div className="flex gap-3 pt-4 px-6 sticky bottom-0 bg-white border-t border-gray-300">
        <button
          type="submit"
          className={`flex-1 px-4 py-2 bg-blue-500 text-white font-semibold rounded-lg hover:bg-blue-600 transition ${
            isLoading ? 'opacity-50 cursor-not-allowed' : ''
          }`}
          disabled={isLoading}
        >
          {isLoading ? '생성 중...' : '프로젝트 만들기'}
        </button>
        <button
          type="button"
          onClick={onClose}
          className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 font-semibold rounded-lg hover:bg-gray-50 transition"
          disabled={isLoading}
        >
          취소
        </button>
      </div>
    </form>
  );

  return (
    <Portal>
      <div
        className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-[9999]"
        onClick={onClose}
      >
        <div
          className={`relative w-full max-w-4xl ${theme.colors.card} p-6 ${theme.effects.borderRadius} shadow-xl max-h-[90vh] overflow-y-auto`}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between mb-4 pb-2">
            <div className="flex items-center">
              <h2 className="text-xl font-bold text-gray-800">{modalTitle}</h2>
              {mode !== 'create' && canEdit && (
                <div className="flex items-center gap-3">
                  {mode === 'detail' ? (
                    <button
                      onClick={() => setMode('edit')}
                      title="프로젝트 수정"
                      className="p-2 rounded-full hover:bg-yellow-50 text-yellow-600 transition"
                    >
                      <Edit2 className="w-5 h-5" />
                    </button>
                  ) : (
                    <button
                      onClick={() => setMode('detail')}
                      title="수정 취소"
                      className="p-2 rounded-full hover:bg-gray-100 text-gray-600 transition"
                    >
                      <Lock className="w-5 h-5" />
                    </button>
                  )}
                </div>
              )}
            </div>
            <button
              onClick={onClose}
              className="p-2 rounded-full hover:bg-gray-100 text-gray-500 hover:text-gray-700 transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-300 rounded-lg text-red-700 text-sm mx-6">
              {error}
            </div>
          )}

          {mode === 'create' ? renderCreateContent() : renderDetailOrEditContent()}
        </div>
      </div>
    </Portal>
  );
};
