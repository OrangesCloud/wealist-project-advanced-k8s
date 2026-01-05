// src/components/modals/board/BoardManageModal.tsx

import React, { useState, useEffect, useRef } from 'react';
import {
  X,
  Plus,
  Settings,
  User,
  Users,
  ChevronDown,
  CheckSquare as CheckSquareIcon,
  Paperclip,
  Calendar,
} from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import {
  CreateBoardRequest,
  FieldOption,
  IEditCustomFields,
  UpdateBoardRequest,
} from '../../../types/board';
import {
  createBoard,
  updateBoard,
  uploadAttachment, // 💡 Presigned URL 업로드 함수 임포트
} from '../../../api/boardService';
import { getWorkspaceMembers } from '../../../api/userService';
import { WorkspaceMemberResponse } from '../../../types/user';
import { AvatarStack, MemberAvatar } from '../../common/AvartarStack';
import Portal from '../../common/Portal';
import { useFileUpload } from '../../../hooks/useFileUpload';

interface BoardManageModalProps {
  projectId: string;
  editData?: {
    boardId: string;
    projectId: string;
    title: string;
    content: string;
    stage: string;
    role: string;
    dueDate: string;
    startDate: string;
    importance: string;
    assigneeId?: string;
    participantIds?: string[];
    attachments?: Array<{
      // 💡 추가
      id: string;
      fileName: string;
      fileUrl: string;
      fileSize: number;
      contentType: string;
    }>;
  } | null;
  workspaceId: string;
  onClose: () => void;
  onBoardCreated: () => void;
  fieldOptionsLookup: {
    stages?: FieldOption[];
    roles?: FieldOption[];
    importances?: FieldOption[];
  };
  handleCustomField: (editFieldData: IEditCustomFields | null) => void;
}

const getMember = (members: WorkspaceMemberResponse[], userId: string) => {
  return members.find((m) => m.userId === userId);
};

export const BoardManageModal: React.FC<BoardManageModalProps> = ({
  projectId,
  editData,
  workspaceId,
  onClose,
  onBoardCreated,
  fieldOptionsLookup,
  handleCustomField,
}) => {
  const { theme } = useTheme();

  // Form state
  const [title, setTitle] = useState(editData?.title || '');
  const [content, setContent] = useState(editData?.content || '');
  const [selectedStageId, setSelectedStageId] = useState(
    editData?.stage || fieldOptionsLookup.stages?.[0]?.optionValue || '',
  );
  const [selectedRoleId, setSelectedRoleId] = useState(
    editData?.role || fieldOptionsLookup.roles?.[0]?.optionValue || '',
  );
  const [selectedImportanceId, setSelectedImportanceId] = useState(
    editData?.importance || fieldOptionsLookup.importances?.[0]?.optionValue || '',
  );

  const [selectedAssigneeId, setSelectedAssigneeId] = useState(editData?.assigneeId || '');
  const [selectedParticipantIds, setSelectedParticipantIds] = useState<string[]>(
    editData?.participantIds || [],
  );

  const [workspaceMembers, setWorkspaceMembers] = useState<WorkspaceMemberResponse[]>([]);

  // Dates
  const [dueDate, setDueDate] = useState(
    editData?.dueDate ? editData.dueDate.substring(0, 10) : '',
  );
  const [startDate, setStartDate] = useState(
    editData?.startDate ? editData.startDate.substring(0, 10) : '',
  );

  // UI state
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingFields, _setIsLoadingFields] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Dropdown states
  const [showRoleDropdown, setShowRoleDropdown] = useState(false);
  const [showStageDropdown, setShowStageDropdown] = useState(false);
  const [showImportanceDropdown, setShowImportanceDropdown] = useState(false);
  const [showAssigneeDropdown, setShowAssigneeDropdown] = useState(false);
  const [showParticipantDropdown, setShowParticipantDropdown] = useState(false);

  // 💡 기존 첨부파일 state 추가
  const [existingAttachment, setExistingAttachment] = useState<{
    id: string;
    fileName: string;
    fileUrl: string;
  } | null>(editData?.attachments?.[0] || null);

  // 파일 업로드 훅
  const { selectedFile, handleFileSelect, handleRemoveFile } = useFileUpload();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 외부 클릭 감지
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest('.role-dropdown-container')) setShowRoleDropdown(false);
      if (!target.closest('.stage-dropdown-container')) setShowStageDropdown(false);
      if (!target.closest('.importance-dropdown-container')) setShowImportanceDropdown(false);
      if (!target.closest('.assignee-dropdown-container')) setShowAssigneeDropdown(false);
      if (!target.closest('.participant-dropdown-container')) setShowParticipantDropdown(false);
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 멤버 조회
  useEffect(() => {
    const fetchMembers = async () => {
      try {
        const members = await getWorkspaceMembers(workspaceId);
        setWorkspaceMembers(members);
      } catch (err) {
        console.error('❌ 워크스페이스 멤버 로드 실패:', err);
      }
    };
    if (workspaceId) fetchMembers();
  }, [workspaceId]);

  // ✅ editData 변경 시 폼 상태 초기화
  useEffect(() => {
    if (editData) {
      setTitle(editData.title || '');
      setContent(editData.content || '');
      setSelectedStageId(editData.stage || fieldOptionsLookup.stages?.[0]?.optionValue || '');
      setSelectedRoleId(editData.role || fieldOptionsLookup.roles?.[0]?.optionValue || '');
      setSelectedImportanceId(
        editData.importance || fieldOptionsLookup.importances?.[0]?.optionValue || '',
      );
      setSelectedAssigneeId(editData.assigneeId || '');
      setSelectedParticipantIds(editData.participantIds || []); // ✅ 핵심
      setDueDate(editData.dueDate ? editData.dueDate.substring(0, 10) : '');
      setStartDate(editData.startDate ? editData.startDate.substring(0, 10) : '');
      setExistingAttachment(editData.attachments?.[0] || null);

      console.log('✅ 보드 수정 데이터 로드:', {
        boardId: editData.boardId,
        participantIds: editData.participantIds,
        participantCount: editData.participantIds?.length || 0,
      });
    } else {
      // Create 모드 초기화
      setTitle('');
      setContent('');
      setSelectedStageId(fieldOptionsLookup.stages?.[0]?.optionValue || '');
      setSelectedRoleId(fieldOptionsLookup.roles?.[0]?.optionValue || '');
      setSelectedImportanceId(fieldOptionsLookup.importances?.[0]?.optionValue || '');
      setSelectedAssigneeId('');
      setSelectedParticipantIds([]);
      setDueDate('');
      setStartDate('');
      setExistingAttachment(null);
    }
  }, [editData, fieldOptionsLookup]);

  const toggleParticipant = (userId: string) => {
    setSelectedParticipantIds((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId],
    );
  };

  // ==========================================================================
  // 💡 수정된 Submit 핸들러 (Presigned URL 방식 적용)
  // ==========================================================================
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!title.trim()) {
      setError('보드 제목은 필수입니다.');
      return;
    }
    if (!selectedStageId || !selectedRoleId || !selectedImportanceId) {
      setError('필수 필드(단계, 역할, 중요도)를 선택해주세요.');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      let attachmentIdsPayload: string[] | undefined = undefined;

      if (selectedFile) {
        const uploadedAttachment = await uploadAttachment(selectedFile, 'BOARD', workspaceId);
        attachmentIdsPayload = [uploadedAttachment.attachmentId];
      } else if (
        editData?.boardId &&
        !existingAttachment &&
        editData.attachments &&
        editData.attachments.length > 0
      ) {
        attachmentIdsPayload = [];
      }

      const customFields = {
        stage: selectedStageId,
        role: selectedRoleId,
        importance: selectedImportanceId,
      };

      const isEditing = !!editData?.boardId;

      const boardData: CreateBoardRequest | UpdateBoardRequest = {
        projectId,
        title: title.trim(),
        content: content.trim() || undefined,
        customFields,
        assigneeId: selectedAssigneeId || undefined,
        participants: selectedParticipantIds.length > 0 ? selectedParticipantIds : undefined, // ✅ 수정
        dueDate: dueDate ? `${dueDate}T00:00:00Z` : undefined,
        startDate: startDate ? `${startDate}T00:00:00Z` : undefined,
        attachmentIds: attachmentIdsPayload,
      };

      // ✅ 디버깅 로그 추가
      console.log('📤 전송할 보드 데이터:', {
        ...boardData,
        participantCount: selectedParticipantIds.length,
        participants: selectedParticipantIds,
      });

      if (isEditing) {
        await updateBoard(editData!.boardId, boardData);
        alert('✅ 보드가 수정되었습니다!');
      } else {
        await createBoard(boardData as CreateBoardRequest);
        alert('✅ 보드가 생성되었습니다!');
      }

      onBoardCreated();
      onClose();
    } catch (err: any) {
      const errorMsg = err.response?.data?.error?.message || err.message;

      // ✅ 상세 에러 로그 추가
      console.error('❌ 보드 저장 실패:', {
        error: err,
        message: errorMsg,
        response: err.response?.data,
        selectedParticipants: selectedParticipantIds,
      });

      setError(errorMsg || '작업에 실패했습니다.');
    } finally {
      setIsLoading(false);
    }
  };
  const currentAssignee = getMember(workspaceMembers, selectedAssigneeId);

  return (
    <Portal>
      <div
        className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-[9999] modal-container"
        onClick={onClose}
      >
        <div
          className={`relative w-full max-w-2xl ${theme.colors.card} ${theme.effects.borderRadius} shadow-xl max-h-[90vh] flex flex-col overflow-hidden`}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 pt-6 pb-4 flex-shrink-0">
            <h2 className="text-xl font-bold text-gray-800">
              {editData?.boardId ? '보드 수정' : '새 보드 만들기'}
            </h2>
            <button
              onClick={onClose}
              className="p-2 rounded-full hover:bg-gray-100 text-gray-500 hover:text-gray-700 transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto px-6 custom-scrollbar">
            {error && (
              <div className="mb-4 p-3 bg-red-50 border border-red-300 rounded-lg text-red-700 text-sm">
                {error}
              </div>
            )}

            {isLoadingFields ? (
              <div className="py-12 text-center text-gray-500">로딩 중...</div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-5 pb-6">
                {/* Title */}
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-1">
                    제목 <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="보드 제목을 입력하세요"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                    disabled={isLoading}
                    maxLength={200}
                  />
                </div>

                {/* Content & File Upload (Compact Style) */}
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-1">설명</label>
                  <div className="relative">
                    <textarea
                      value={content}
                      onChange={(e) => setContent(e.target.value)}
                      placeholder="내용을 입력하세요"
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm resize-none min-h-[100px]"
                      rows={4}
                      disabled={isLoading}
                    />

                    {/* 파일 첨부 영역 */}
                    <div className="mt-2 flex items-center gap-2">
                      <input
                        type="file"
                        ref={fileInputRef}
                        onChange={(e) => {
                          if (e.target.files && e.target.files.length > 0) {
                            handleFileSelect(e as any);
                            // 💡 새 파일 선택 시 기존 파일 제거
                            setExistingAttachment(null);
                          }
                        }}
                        className="hidden"
                        accept="image/*, .pdf, .doc, .docx, .xls, .xlsx"
                      />

                      <button
                        type="button"
                        onClick={() => fileInputRef.current?.click()}
                        className="flex items-center gap-1 px-2 py-1 text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded transition text-xs font-medium"
                        disabled={isLoading}
                      >
                        <Paperclip size={14} />
                        <span>파일 첨부</span>
                      </button>

                      {/* 💡 새로 선택한 파일 표시 */}
                      {selectedFile && (
                        <div className="flex items-center gap-1 pl-2 pr-1 py-0.5 bg-blue-50 text-blue-700 rounded-full text-xs border border-blue-100 max-w-[250px]">
                          <span className="truncate max-w-[200px]">{selectedFile.name}</span>
                          <button
                            type="button"
                            onClick={() => {
                              handleRemoveFile();
                              if (fileInputRef.current) fileInputRef.current.value = '';
                            }}
                            className="p-0.5 text-blue-400 hover:text-blue-600 hover:bg-blue-100 rounded-full"
                          >
                            <X size={12} />
                          </button>
                        </div>
                      )}

                      {/* 💡 기존 첨부파일 표시 (새 파일이 없을 때만) */}
                      {!selectedFile && existingAttachment && (
                        <div className="flex items-center gap-1 pl-2 pr-1 py-0.5 bg-green-50 text-green-700 rounded-full text-xs border border-green-100 max-w-[250px]">
                          <span className="truncate max-w-[200px]">
                            {existingAttachment.fileName}
                          </span>
                          <button
                            type="button"
                            onClick={() => setExistingAttachment(null)}
                            className="p-0.5 text-green-400 hover:text-green-600 hover:bg-green-100 rounded-full"
                          >
                            <X size={12} />
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
                <hr className="border-gray-100" />

                {/* Date Inputs */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      <Calendar className="w-4 h-4 inline mr-1 text-gray-500" /> 시작일
                    </label>
                    <input
                      type="date"
                      value={startDate}
                      onChange={(e) => setStartDate(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                      disabled={isLoading}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      <Calendar className="w-4 h-4 inline mr-1 text-red-500" /> 마감일
                    </label>
                    <input
                      type="date"
                      value={dueDate}
                      onChange={(e) => setDueDate(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                      disabled={isLoading}
                    />
                  </div>
                </div>

                {/* Assignee & Participants */}
                <div className="grid grid-cols-2 gap-4">
                  {/* Assignee */}
                  <div className="relative assignee-dropdown-container">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      <User className="w-4 h-4 inline mr-1 text-green-500" /> 할당자
                    </label>
                    <button
                      type="button"
                      onClick={() => setShowAssigneeDropdown(!showAssigneeDropdown)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-white text-sm text-left flex items-center justify-between hover:bg-gray-50"
                      disabled={isLoading}
                    >
                      <span className="flex items-center gap-2 truncate">
                        {currentAssignee ? (
                          <>
                            <MemberAvatar member={currentAssignee} index={0} size="sm" />
                            {currentAssignee.nickName || currentAssignee.userEmail || 'Unknown'}
                          </>
                        ) : (
                          <span className="text-gray-400">선택 안함</span>
                        )}
                      </span>
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    </button>
                    {showAssigneeDropdown && (
                      <div className="absolute z-20 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto p-1">
                        <button
                          type="button"
                          onClick={() => {
                            setSelectedAssigneeId('');
                            setShowAssigneeDropdown(false);
                          }}
                          className="w-full px-3 py-2 text-left text-sm text-gray-500 hover:bg-gray-100 rounded-md flex items-center gap-2"
                        >
                          <X className="w-4 h-4" /> 할당 해제
                        </button>
                        {workspaceMembers.map((m, idx) => (
                          <button
                            key={m.userId}
                            type="button"
                            onClick={() => {
                              setSelectedAssigneeId(m.userId);
                              setShowAssigneeDropdown(false);
                            }}
                            className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-100 rounded-md flex items-center gap-2 ${
                              selectedAssigneeId === m.userId ? 'bg-blue-50 text-blue-700' : ''
                            }`}
                          >
                            <MemberAvatar member={m} index={idx} size="sm" />
                            {m.nickName || m.userEmail || 'Unknown'}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Participants */}
                  <div className="relative participant-dropdown-container">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      <Users className="w-4 h-4 inline mr-1 text-orange-500" /> 작업자
                    </label>
                    <button
                      type="button"
                      onClick={() => setShowParticipantDropdown(!showParticipantDropdown)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-white text-sm text-left flex items-center justify-between hover:bg-gray-50 h-[38px]"
                      disabled={isLoading}
                    >
                      <span className="flex items-center gap-2 truncate">
                        {selectedParticipantIds.length > 0 ? (
                          <>
                            <AvatarStack
                              members={workspaceMembers.filter((m) =>
                                selectedParticipantIds.includes(m.userId),
                              )}
                              interactive={false}
                            />
                            <span>{selectedParticipantIds.length}명</span>
                          </>
                        ) : (
                          <span className="text-gray-400">선택 안함</span>
                        )}
                      </span>
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    </button>
                    {showParticipantDropdown && (
                      <div className="absolute z-20 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto p-1">
                        {workspaceMembers.map((m, idx) => (
                          <button
                            key={m.userId}
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              toggleParticipant(m.userId);
                            }}
                            className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-100 rounded-md flex items-center gap-2 justify-between ${
                              selectedParticipantIds.includes(m.userId)
                                ? 'bg-blue-50 text-blue-700'
                                : ''
                            }`}
                          >
                            <div className="flex items-center gap-2">
                              <MemberAvatar member={m} index={idx} size="sm" />
                              {m.nickName || m.userEmail || 'Unknown'}
                            </div>
                            {selectedParticipantIds.includes(m.userId) && (
                              <CheckSquareIcon className="w-4 h-4 text-blue-500" />
                            )}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                {/* Stage & Role */}
                <div className="grid grid-cols-2 gap-4">
                  {/* Stage */}
                  <div className="relative stage-dropdown-container">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      진행 단계 <span className="text-red-500">*</span>
                    </label>
                    <button
                      type="button"
                      onClick={() => setShowStageDropdown(!showStageDropdown)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-white text-sm text-left flex items-center justify-between hover:bg-gray-50"
                      disabled={isLoading}
                    >
                      <span className="flex items-center gap-2">
                        {selectedStageId &&
                        fieldOptionsLookup?.stages?.find(
                          (s) => s.optionValue === selectedStageId,
                        ) ? (
                          <>
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{
                                backgroundColor:
                                  (
                                    fieldOptionsLookup.stages.find(
                                      (s) => s.optionValue === selectedStageId,
                                    ) as any
                                  )?.color || '#ccc',
                              }}
                            />
                            {
                              fieldOptionsLookup.stages.find(
                                (s) => s.optionValue === selectedStageId,
                              )?.optionLabel
                            }
                          </>
                        ) : (
                          '선택'
                        )}
                      </span>
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    </button>
                    {showStageDropdown && (
                      <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto p-1">
                        {fieldOptionsLookup?.stages?.map((opt) => (
                          <button
                            key={opt.optionId}
                            type="button"
                            onClick={() => {
                              setSelectedStageId(opt.optionValue);
                              setShowStageDropdown(false);
                            }}
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 rounded-md flex items-center gap-2"
                          >
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{ backgroundColor: (opt as any).color }}
                            />
                            {opt.optionLabel}
                          </button>
                        ))}
                        <button
                          onClick={() => {
                            setShowStageDropdown(false);
                            handleCustomField({
                              name: '진행 단계',
                              fieldType: 'multi_select',
                              options: fieldOptionsLookup?.stages,
                            });
                          }}
                          className="w-full px-3 py-2 text-left text-sm text-blue-600 hover:bg-blue-50 rounded-md border-t mt-1 flex items-center gap-2"
                        >
                          <Settings size={14} /> 관리
                        </button>
                      </div>
                    )}
                  </div>

                  {/* Role */}
                  <div className="relative role-dropdown-container">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      역할 <span className="text-red-500">*</span>
                    </label>
                    <button
                      type="button"
                      onClick={() => setShowRoleDropdown(!showRoleDropdown)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-white text-sm text-left flex items-center justify-between hover:bg-gray-50"
                      disabled={isLoading}
                    >
                      <span className="flex items-center gap-2">
                        {selectedRoleId &&
                        fieldOptionsLookup?.roles?.find((r) => r.optionValue === selectedRoleId) ? (
                          <>
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{
                                backgroundColor:
                                  (
                                    fieldOptionsLookup.roles.find(
                                      (r) => r.optionValue === selectedRoleId,
                                    ) as any
                                  )?.color || '#ccc',
                              }}
                            />
                            {
                              fieldOptionsLookup.roles.find((r) => r.optionValue === selectedRoleId)
                                ?.optionLabel
                            }
                          </>
                        ) : (
                          '선택'
                        )}
                      </span>
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    </button>
                    {showRoleDropdown && (
                      <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto p-1">
                        {fieldOptionsLookup?.roles?.map((opt) => (
                          <button
                            key={opt.optionId}
                            type="button"
                            onClick={() => {
                              setSelectedRoleId(opt.optionValue);
                              setShowRoleDropdown(false);
                            }}
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 rounded-md flex items-center gap-2"
                          >
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{ backgroundColor: (opt as any).color }}
                            />
                            {opt.optionLabel}
                          </button>
                        ))}
                        <button
                          onClick={() => {
                            setShowRoleDropdown(false);
                            handleCustomField({
                              name: '역할',
                              fieldType: 'multi_select',
                              options: fieldOptionsLookup?.roles,
                            });
                          }}
                          className="w-full px-3 py-2 text-left text-sm text-blue-600 hover:bg-blue-50 rounded-md border-t mt-1 flex items-center gap-2"
                        >
                          <Settings size={14} /> 관리
                        </button>
                      </div>
                    )}
                  </div>
                </div>

                {/* Importance & Field Add */}
                <div className="grid grid-cols-2 gap-4">
                  {/* Importance */}
                  <div className="relative importance-dropdown-container">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      중요도 <span className="text-red-500">*</span>
                    </label>
                    <button
                      type="button"
                      onClick={() => setShowImportanceDropdown(!showImportanceDropdown)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-white text-sm text-left flex items-center justify-between hover:bg-gray-50"
                      disabled={isLoading}
                    >
                      <span className="flex items-center gap-2">
                        {selectedImportanceId &&
                        fieldOptionsLookup?.importances?.find(
                          (i) => i.optionValue === selectedImportanceId,
                        ) ? (
                          <>
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{
                                backgroundColor:
                                  (
                                    fieldOptionsLookup.importances.find(
                                      (i) => i.optionValue === selectedImportanceId,
                                    ) as any
                                  )?.color || '#ccc',
                              }}
                            />
                            {
                              fieldOptionsLookup.importances.find(
                                (i) => i.optionValue === selectedImportanceId,
                              )?.optionLabel
                            }
                          </>
                        ) : (
                          '선택'
                        )}
                      </span>
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    </button>
                    {showImportanceDropdown && (
                      <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto p-1">
                        {fieldOptionsLookup?.importances?.map((opt) => (
                          <button
                            key={opt.optionId}
                            type="button"
                            onClick={() => {
                              setSelectedImportanceId(opt.optionValue);
                              setShowImportanceDropdown(false);
                            }}
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 rounded-md flex items-center gap-2"
                          >
                            <span
                              className="w-2.5 h-2.5 rounded-full"
                              style={{ backgroundColor: (opt as any).color }}
                            />
                            {opt.optionLabel}
                          </button>
                        ))}
                        <button
                          onClick={() => {
                            setShowImportanceDropdown(false);
                            handleCustomField({
                              name: '중요도',
                              fieldType: 'multi_select',
                              options: fieldOptionsLookup?.importances,
                            });
                          }}
                          className="w-full px-3 py-2 text-left text-sm text-blue-600 hover:bg-blue-50 rounded-md border-t mt-1 flex items-center gap-2"
                        >
                          <Settings size={14} /> 관리
                        </button>
                      </div>
                    )}
                  </div>
                  {/* Add Field */}
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-1">
                      필드 추가
                    </label>
                    <button
                      type="button"
                      onClick={() => handleCustomField(null)}
                      className="w-full px-3 py-2 border border-dashed border-gray-300 rounded-lg text-sm text-gray-500 hover:bg-gray-50 hover:text-blue-600 flex items-center justify-between transition"
                      disabled={isLoading}
                    >
                      <span>커스텀 필드 생성</span>
                      <Plus size={16} />
                    </button>
                  </div>
                </div>

                {/* Footer Actions */}
                <div className="flex gap-3 pt-4 sticky bottom-0 bg-white border-t border-gray-100 mt-2">
                  <button
                    type="button"
                    onClick={onClose}
                    className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 font-semibold hover:bg-gray-50 transition"
                    disabled={isLoading}
                  >
                    취소
                  </button>
                  <button
                    type="submit"
                    className="flex-1 px-4 py-2 bg-blue-500 text-white font-semibold rounded-lg hover:bg-blue-600 transition disabled:opacity-50"
                    disabled={isLoading}
                  >
                    {isLoading ? '처리 중...' : editData?.boardId ? '수정 완료' : '생성 하기'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      </div>
    </Portal>
  );
};
