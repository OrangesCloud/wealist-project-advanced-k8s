// src/components/layout/ProjectContent.tsx

import React, { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { Plus, ArrowUp, ArrowDown, Users, User } from 'lucide-react';
import { useTheme } from '../../contexts/ThemeContext';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { getDefaultColorByIndex } from '../../constants/colors';
import { ProjectResponse, BoardResponse, Column, ViewState, FieldOption } from '../../types/board';
import { getBoardsByProject, moveBoard } from '../../api/boardService';
import { BoardDetailModal } from '../modals/board/BoardDetailModal';
import { FilterBar } from '../modals/board/FilterBar';
import { useAuth } from '../../contexts/AuthContext';
import { AssigneeAvatarStack } from '../common/AvartarStack';
import { WorkspaceMemberResponse } from '../../types/user';
import { connectWebSocket, disconnectWebSocket, WS_BOARD_MTH } from '../../utils/boardWebsocket';

interface ProjectContentProps {
  // Data
  selectedProject: ProjectResponse;
  workspaceId: string;
  workspaceMembers: WorkspaceMemberResponse[]; // 💡 추가
  fieldOptionsLookup: {
    stages?: FieldOption[];
    roles?: FieldOption[];
    importances?: FieldOption[];
  };

  // Handlers
  onProjectContentUpdate: () => void;
  onManageModalOpen: () => void;

  // Initial States for Modals
  onEditBoard: (data: any) => void;

  showCreateBoard: boolean;
  setShowCreateBoard: (show: boolean) => void;
}
export const ProjectContent: React.FC<ProjectContentProps> = ({
  selectedProject,
  workspaceId,
  workspaceMembers, // 💡 추가
  fieldOptionsLookup,
  onManageModalOpen,
  onEditBoard,
  setShowCreateBoard,
}) => {
  const { theme } = useTheme();
  const { userId } = useAuth(); // useAuth 훅 사용 가정

  // 💡 [Board Data States]
  const [columns, setColumns] = useState<Column[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const {
    roles: roleOptions,
    stages: stageOptions,
    importances: importanceOptions,
  } = fieldOptionsLookup;

  // 💡 [통합된 View/Filter 상태]
  const [viewState, setViewState] = useState<ViewState>({
    currentView: 'stage',
    searchQuery: '',
    filterOption: 'all',
    currentLayout: 'board',
    showCompleted: true, // 🔥 기본값 true로 변경
    showDeleted: true, // 🔥 삭제된 항목 보기 (기본값 true)
    sortColumn: null,
    sortDirection: 'asc',
  });

  // 💡 [UI States]
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null);

  // 알림에서 클릭한 보드 열기
  useEffect(() => {
    const pendingBoardId = localStorage.getItem('pendingBoardId');
    if (pendingBoardId) {
      setSelectedBoardId(pendingBoardId);
      localStorage.removeItem('pendingBoardId');
    }
  }, [selectedProject]);

  // Drag state
  const [draggedBoard, setDraggedBoard] = useState<BoardResponse | null>(null);
  const [draggedFromColumn, setDraggedFromColumn] = useState<string | null>(null);
  const [draggedColumn, setDraggedColumn] = useState<Column | null>(null);
  const [dragOverBoardId, setDragOverBoardId] = useState<string | null>(null);
  const [dragOverColumn, setDragOverColumn] = useState<string | null>(null);

  // 💡 [추가] View State Setter Helper (유지)
  const setViewField = useCallback(<K extends keyof ViewState>(key: K, value: ViewState[K]) => {
    setViewState((prev) => ({ ...prev, [key]: value }));
  }, []);

  const getRoleOption = (roleId: string | undefined) =>
    roleId ? roleOptions?.find((r) => r.optionValue === roleId) : undefined;
  const getImportanceOption = (importanceId: string | undefined) =>
    importanceId ? importanceOptions?.find((i) => i.optionValue === importanceId) : undefined;
  const getStageOption = (stageId: string | undefined) =>
    stageId ? stageOptions?.find((i) => i.optionValue === stageId) : undefined;

  const fetchBoards = useCallback(async () => {
    console.log('gh');
    if (!selectedProject || !stageOptions || stageOptions.length === 0) {
      setColumns([]);
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const stages = stageOptions;
      const boardsResponse = await getBoardsByProject(selectedProject.projectId);

      // 데이터 처리 로직
      const stageMap = new Map<string, { stage: FieldOption; boards: BoardResponse[] }>();

      stages.forEach((stage: FieldOption) => {
        stageMap.set(stage.optionValue, { stage, boards: [] });
      });

      boardsResponse?.forEach((board: BoardResponse) => {
        const stageId = board.customFields?.stage;

        if (stageId && stageMap.has(stageId)) {
          stageMap.get(stageId)!.boards.push(board);
        } else {
          console.warn(`⚠️ 보드 "${board.title}"에 유효하지 않은 Stage ID가 있습니다: ${stageId}`);
        }
      });

      const newColumns: Column[] = Array.from(stageMap.values())
        .sort((a, b) => (a.stage as any).displayOrder - (b.stage as any).displayOrder)
        .map(({ stage, boards }) => ({
          stageId: stage.optionValue,
          title: stage.optionLabel,
          color: (stage as any).color,
          boards: boards,
        }));

      console.log('📌 최종 newColumns:', newColumns);
      setColumns(newColumns);
    } catch (err) {
      const fetchError = err as Error;
      console.error('❌ 보드 로드 실패:', fetchError);
      setError(`보드 로드 실패: ${fetchError.message}`);
      setColumns([]);
    } finally {
      setIsLoading(false);
    }
  }, [selectedProject, stageOptions]);

  useEffect(() => {
    // selectedProject가 변경되었을 때, stageOptions가 로드되면 fetchBoards를 호출
    if (selectedProject && stageOptions && stageOptions.length > 0) {
      fetchBoards();
    }
    // 💡 stageOptions에 대한 의존성을 명확히 합니다.
  }, [fetchBoards, selectedProject, stageOptions]);

  // WebSocket!!
  const wsConnectedRef = React.useRef(false);
  const fetchBoardsRef = useRef(fetchBoards);

  useEffect(() => {
    fetchBoardsRef.current = fetchBoards;
  }, [fetchBoards]);

  // 💡 [수정] useEffect 수정
  useEffect(() => {
    if (!selectedProject?.projectId) return;

    // 🔥 컴포넌트 마운트 시 1번만 연결
    if (!wsConnectedRef.current) {
      wsConnectedRef.current = true;
      console.log('🔌 [WS] 연결 시작:', selectedProject.projectId);

      connectWebSocket(selectedProject.projectId, (event) => {
        console.log('🔊 [WS EVENT] 수신:', event);
        if (WS_BOARD_MTH?.includes(event.type)) {
          fetchBoardsRef.current();
        }
      });
    }

    // 🔥 클린업: 컴포넌트 언마운트 시에만 연결 해제
    return () => {
      if (wsConnectedRef.current) {
        console.log('🔌 [WS] 컴포넌트 언마운트 - 연결 해제');
        disconnectWebSocket();
        wsConnectedRef.current = false;
      }
    };
  }, [selectedProject?.projectId]);

  //

  // 5. 드래그 앤 드롭 및 정렬 로직 (useCallback 유지)
  const handleDragStart = (board: BoardResponse, columnId: string): void => {
    console.log('🔍 [DRAG START] board:', board.title);
    console.log('🔍 [DRAG START] columnId:', columnId);
    console.log('🔍 [DRAG START] currentView:', viewState.currentView);
    setDraggedBoard(board);
    setDraggedFromColumn(columnId);
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>): void => {
    e.preventDefault();
  };

  const handleDragEnd = (): void => {
    setDraggedBoard(null);
    setDraggedFromColumn(null);
    setDraggedColumn(null);
    setDragOverBoardId(null);
    setDragOverColumn(null);
  };

  const handleSort = (column: 'title' | 'stage' | 'role' | 'importance') => {
    if (viewState.sortColumn === column) {
      setViewField('sortDirection', viewState.sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setViewField('sortColumn', column);
      setViewField('sortDirection', 'asc');
    }
  };

  const handleBoardEdit = (boardData: any) => {
    onEditBoard(boardData);
    setSelectedBoardId(null);
  };

  // 6. Table/Board View 공통 데이터 필터링/정렬 로직 (useMemo)
  const allProcessedBoards = useMemo(() => {
    const { searchQuery, sortColumn, showCompleted, showDeleted } = viewState;

    const boardsToProcess = columns.flatMap((column) =>
      column.boards.map((board) => {
        const roleId = board.customFields?.role;
        const importanceId = board.customFields?.importance;
        const stageId = board.customFields?.stage;

        // 🔥 [핵심 수정] stageOption을 먼저 찾아서 optionValue 사용
        const stageOption = getStageOption(stageId);

        return {
          ...board,
          stageName: stageOption?.optionLabel || column.title,
          stageColor: (stageOption as any)?.color || column.color,
          stageId: stageOption?.optionValue || stageId,
          roleOption: getRoleOption(roleId),
          importanceOption: getImportanceOption(importanceId),
          // 💡 현재 사용자 ID가 할당자 또는 참여자인지 확인하는 필터링 기준
          isAssignedOrParticipant:
            board.assigneeId === userId || board.participantIds?.includes(userId as string),
        };
      }),
    );

    let filteredBoardsByCompletion = boardsToProcess;
    if (!showCompleted) {
      const completedStageIds = stageOptions
        ?.filter((s) => s.optionLabel === '완료')
        .map((s) => s.optionValue);

      filteredBoardsByCompletion = boardsToProcess.filter(
        (board) => !completedStageIds?.includes(board.stageId),
      );
    }

    // 2-1. 💡 [추가] '삭제된 항목' 필터링 로직
    if (!showDeleted) {
      const deletedStageIds = stageOptions
        ?.filter((s) => s.optionLabel === '삭제')
        .map((s) => s.optionValue);

      filteredBoardsByCompletion = filteredBoardsByCompletion.filter(
        (board) => !deletedStageIds?.includes(board.stageId),
      );
    }

    // 2. 💡 [추가] '나의 일감' 필터링 로직
    if (viewState?.filterOption === 'my_tasks' && userId) {
      filteredBoardsByCompletion = filteredBoardsByCompletion.filter(
        (board) => board.isAssignedOrParticipant,
      );
    }

    const finalFilteredBoards = searchQuery?.trim()
      ? filteredBoardsByCompletion.filter((board) => {
          const query = searchQuery.toLowerCase();
          const titleMatch = board.title.toLowerCase().includes(query);
          const contentMatch = board.content?.toLowerCase().includes(query);
          return titleMatch || contentMatch;
        })
      : filteredBoardsByCompletion;

    const sortedBoards = [...finalFilteredBoards].sort((a, b) => {
      if (!sortColumn) return 0;
      let aValue: any;
      let bValue: any;
      const direction = viewState.sortDirection === 'asc' ? 1 : -1;

      switch (sortColumn) {
        case 'title':
          aValue = a.title.toLowerCase();
          bValue = b.title.toLowerCase();
          break;
        case 'stage':
          aValue = a.stageName;
          bValue = b.stageName;
          break;
        case 'role':
          aValue = a.roleOption?.optionLabel || '';
          bValue = b.roleOption?.optionLabel || '';
          break;
        case 'importance':
          aValue = (a.importanceOption as any)?.level || 0;
          bValue = (b.importanceOption as any)?.level || 0;
          break;
        case 'assignee':
          aValue = a.assigneeId?.toLowerCase() || '';
          bValue = b.assigneeId?.toLowerCase() || '';
          break;
        case 'dueDate':
          aValue = a.dueDate ? new Date(a.dueDate).getTime() : 0;
          bValue = b.dueDate ? new Date(b.dueDate).getTime() : 0;
          break;
        default:
          return 0;
      }

      if (aValue < bValue) return -1 * direction;
      if (aValue > bValue) return 1 * direction;
      return 0;
    });

    return sortedBoards;
  }, [
    columns,
    viewState,
    roleOptions,
    importanceOptions,
    stageOptions,
    getRoleOption,
    getImportanceOption,
    getStageOption,
    userId,
  ]);

  // 7. 뷰 기준에 따라 컬럼을 재구성 (useMemo)
  const currentViewColumns = useMemo(() => {
    // 💡 [수정] 필터링이 완료된 보드만 사용합니다.
    const filteredBoards = allProcessedBoards;

    // 🔥 보드가 없을 때, 현재 뷰 기준에 맞는 빈 컬럼 구조를 생성합니다.
    if (filteredBoards.length === 0) {
      if (viewState?.currentLayout === 'board') {
        let options: FieldOption[] = [];
        let groupByField: 'stage' | 'role' | 'importance' | undefined = viewState?.currentView;

        if (groupByField === 'stage') {
          options = fieldOptionsLookup?.stages || [];
        } else if (groupByField === 'role') {
          options = fieldOptionsLookup?.roles || [];
        } else if (groupByField === 'importance') {
          options = fieldOptionsLookup?.importances || [];
        }

        const emptyColumns: Column[] = options.map((option) => ({
          stageId: option.optionValue,
          title: option.optionLabel,
          color: (option as any).color,
          boards: [],
        }));

        return emptyColumns;
      }
      return [];
    }

    const groupByField = viewState.currentView;

    // 🔥 stage일 때는 원본 columns를 사용하지 않고,
    //    필터링된 boards를 다시 stage 기준으로 그룹화해야 합니다. (showCompleted 적용을 위해)
    if (groupByField === 'stage') {
      const stageMap = new Map<string, Column>();
      const stages = fieldOptionsLookup?.stages || [];

      stages.forEach((stage: FieldOption) => {
        stageMap.set(stage.optionValue, {
          stageId: stage.optionValue,
          title: stage.optionLabel,
          color: (stage as any).color,
          boards: [],
        } as Column);
      });

      filteredBoards.forEach((board) => {
        const stageId = board.stageId;
        const targetColumn = stageMap.get(stageId);

        if (targetColumn) {
          targetColumn.boards.push(board as any);
        } else {
          console.warn(`⚠️ 보드 "${board.title}"에 유효하지 않은 Stage ID: ${stageId}`);
        }
      });

      const result = Array.from(stageMap.values()).sort((a, b) => {
        const orderA = (stages.find((o) => o.optionValue === a.stageId) as any)?.displayOrder || 0;
        const orderB = (stages.find((o) => o.optionValue === b.stageId) as any)?.displayOrder || 0;
        return orderA - orderB;
      });

      // 완료/삭제 컬럼 필터링
      let filteredResult = result;
      if (!viewState.showCompleted && viewState?.currentView === 'stage') {
        const completedStageIds = stages
          .filter((s) => s.optionLabel === '완료')
          .map((s) => s.optionValue);
        filteredResult = filteredResult.filter((col) => !completedStageIds.includes(col.stageId));
      }
      if (!viewState.showDeleted && viewState?.currentView === 'stage') {
        const deletedStageIds = stages
          .filter((s) => s.optionLabel === '삭제')
          .map((s) => s.optionValue);
        filteredResult = filteredResult.filter((col) => !deletedStageIds.includes(col.stageId));
      }
      return filteredResult;
    }

    // role이나 importance일 때만 재그룹화 (기존 로직 유지, filteredBoards 사용)
    let baseOptions: any[] = [];
    let lookupField: 'roleOption' | 'importanceOption';

    if (groupByField === 'role') {
      baseOptions = fieldOptionsLookup?.roles || [];
      lookupField = 'roleOption';
    } else if (groupByField === 'importance') {
      baseOptions = fieldOptionsLookup?.importances || [];
      lookupField = 'importanceOption';
    } else {
      return [];
    }

    const groupedMap = new Map<string, Column>();

    baseOptions?.forEach((option) => {
      const id = option.optionValue;
      groupedMap?.set(id, {
        stageId: id,
        title: option?.optionLabel,
        color: (option as any).color,
        boards: [],
      });
    });

    // 💡 모든 보드는 stage, role, importance를 갖고 있으므로 반드시 매칭됨
    filteredBoards?.forEach((board) => {
      const optionValue = (board as any)[lookupField]?.optionValue;

      if (optionValue && groupedMap.has(optionValue)) {
        groupedMap.get(optionValue)!.boards.push(board as any);
      } else {
        // 💡 이 경우는 발생하지 않아야 함 (모든 보드가 필수 필드를 가짐)
        console.error(`❌ [심각] 보드 "${board.title}"에 ${lookupField} 값이 없습니다:`, board);
      }
    });

    const result = Array.from(groupedMap.values()).sort((a, b) => {
      const orderA =
        (baseOptions.find((o) => o.optionValue === a.stageId) as any)?.displayOrder || 0;
      const orderB =
        (baseOptions.find((o) => o.optionValue === b.stageId) as any)?.displayOrder || 0;
      return orderA - orderB;
    });

    return result;
  }, [allProcessedBoards, viewState, fieldOptionsLookup, columns, stageOptions]);

  const handleDrop = useCallback(
    async (targetColumnId: string): Promise<void> => {
      if (!draggedBoard || !draggedFromColumn) return;

      // 🔥 수정: columns 대신 currentViewColumns 사용
      const targetColumn = currentViewColumns.find((col) => col.stageId === targetColumnId);
      if (!targetColumn) {
        console.log('❌ [DROP] targetColumn을 찾을 수 없음:', targetColumnId);
        handleDragEnd();
        return;
      }

      // 같은 컬럼이면 무시
      if (draggedFromColumn === targetColumnId) {
        handleDragEnd();
        return;
      }

      // 🔥 현재 뷰 타입에 따라 업데이트할 필드 결정
      const fieldKeyName = viewState.currentView as 'stage' | 'role' | 'importance';

      // 🔥 [핵심 수정] 기존 customFields를 유지하면서 해당 필드만 업데이트
      const currentCustomFields = draggedBoard.customFields || {};
      const updatedCustomFields = {
        ...currentCustomFields,
        [fieldKeyName]: targetColumnId,
      };

      console.log('🔍 [DROP DEBUG] 기존 customFields:', currentCustomFields);
      console.log('🔍 [DROP DEBUG] 업데이트할 필드:', fieldKeyName, '→', targetColumnId);
      console.log('🔍 [DROP DEBUG] 최종 customFields:', updatedCustomFields);

      // 3. 🔥 API 호출 - 실제 DB 업데이트
      try {
        // 💡 [핵심 수정] moveBoard API에 전체 customFields 전송
        await moveBoard(draggedBoard.boardId, {
          projectId: selectedProject.projectId,
          groupByFieldName: fieldKeyName,
          newFieldValue: targetColumnId,
        });

        console.log(
          `✅ [API SUCCESS] 보드 이동 완료: ${draggedBoard.title} → ${targetColumn.title} (${fieldKeyName}: ${targetColumnId})`,
        );

        // 🔥 API 성공 후 데이터 새로고침
        await fetchBoards();
      } catch (error) {
        console.error('❌ [API ERROR] 보드 이동 실패:', error);
        alert('보드 이동에 실패했습니다. 다시 시도해주세요.');
      } finally {
        handleDragEnd();
      }
    },
    [
      draggedBoard,
      draggedFromColumn,
      currentViewColumns,
      viewState.currentView,
      fetchBoards,
      selectedProject.projectId,
    ],
  );

  // 로딩 상태 처리
  if (isLoading && (stageOptions === undefined || stageOptions.length === 0)) {
    return <LoadingSpinner message="보드와 필드 데이터를 로드 중..." />;
  }

  if (error) {
    return (
      <div className="mt-4 p-4 bg-red-50 border border-red-300 rounded-lg text-red-700">
        {error}
      </div>
    );
  }

  return (
    <>
      {/* FilterBar */}
      <FilterBar
        onSearchChange={(query) => setViewField('searchQuery', query)}
        onViewChange={(view) => setViewField('currentView', view)}
        onFilterChange={(filter) => setViewField('filterOption', filter)}
        onManageClick={onManageModalOpen}
        currentView={viewState.currentView}
        onLayoutChange={(layout) => setViewField('currentLayout', layout)}
        onShowCompletedChange={(show) => setViewField('showCompleted', show)}
        onShowDeletedChange={(show) => setViewField('showDeleted', show)}
        currentLayout={viewState.currentLayout}
        showCompleted={viewState.showCompleted}
        showDeleted={viewState.showDeleted}
        stageOptions={fieldOptionsLookup?.stages || []}
        roleOptions={fieldOptionsLookup?.roles || []}
        importanceOptions={fieldOptionsLookup?.importances || []}
        currentFilter={viewState.filterOption as string}
      />

      {/* Boards or Table View */}
      {viewState?.currentLayout === 'table' ? (
        // Table Layout
        <div className="mt-4 overflow-x-auto">
          <table
            className={`w-full ${theme.colors.card} ${theme.effects.borderRadius} overflow-hidden shadow-lg`}
          >
            <thead className="bg-gray-100 border-b border-gray-200">
              <tr>
                {['title', 'stage', 'role', 'importance', 'assignee', 'dueDate'].map((col) => (
                  <th key={col} className="px-4 py-3 text-left">
                    <button
                      onClick={() => handleSort(col as 'title' | 'stage' | 'role' | 'importance')}
                      className="flex items-center gap-2 font-semibold text-sm text-gray-700 hover:text-blue-600 transition"
                    >
                      {col === 'title' && '제목'}
                      {col === 'stage' && '진행 단계'}
                      {col === 'role' && '역할'}
                      {col === 'importance' && '중요도'}
                      {viewState?.sortColumn === col &&
                        (viewState?.sortDirection === 'asc' ? (
                          <ArrowUp className="w-4 h-4" />
                        ) : (
                          <ArrowDown className="w-4 h-4" />
                        ))}
                    </button>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {allProcessedBoards?.map((board) => (
                <tr
                  key={board.boardId}
                  onClick={() => setSelectedBoardId(board.boardId)}
                  className="border-b border-gray-200 hover:bg-gray-50 cursor-pointer transition"
                >
                  <td className="px-4 py-3 font-semibold text-gray-800">{board.title}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" />
                      <span className="text-sm">{board.stageName}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {board?.roleOption ? (
                      <div className="flex items-center gap-2">
                        <span
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: (board?.roleOption as any).color || '#6B7280' }}
                        />
                        <span className="text-sm">{board?.roleOption.optionLabel}</span>
                      </div>
                    ) : (
                      <span className="text-sm text-gray-500">없음</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {board?.importanceOption ? (
                      <div className="flex items-center gap-2">
                        <span
                          className="w-3 h-3 rounded-full"
                          style={{
                            backgroundColor: (board.importanceOption as any).color || '#6B7280',
                          }}
                        />
                        <span className="text-sm">{board.importanceOption.optionLabel}</span>
                      </div>
                    ) : (
                      <span className="text-sm text-gray-500">없음</span>
                    )}
                  </td>
                  {/* 💡 [수정] 작업자 (Participant) 컬럼 */}
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {board.participantIds && board.participantIds.length > 0 ? (
                        <>
                          {/* AvatarStack 대신 텍스트와 카운트 중심 */}
                          <Users className="w-4 h-4 text-orange-500" />
                          <span className="text-sm font-medium text-gray-700">
                            {board.participantIds.length}명
                          </span>
                          {/* 만약 여기서 AvatarStack을 사용하고 싶다면, AvatarStack 컴포넌트가 멤버 객체를 필요로 하므로,
                                   현재 ProjectContent에서는 멤버를 찾을 수 없기 때문에 시각적으로는 AssigneeAvatarStack을 사용해야 합니다. */}
                          <AssigneeAvatarStack
                            assignees={board.participantIds || []}
                            workspaceMembers={workspaceMembers} // 💡 추가
                          />
                        </>
                      ) : (
                        <span className="text-sm text-gray-500">없음</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {board.dueDate ? new Date(board.dueDate).toLocaleDateString('ko-KR') : '없음'}
                  </td>
                </tr>
              ))}

              <tr
                onClick={() => {
                  setShowCreateBoard(true);
                }}
                className="border-t-2 border-gray-300 hover:bg-blue-50 cursor-pointer transition"
              >
                <td colSpan={6} className="px-4 py-4">
                  <div className="flex items-center justify-center gap-2 text-blue-600 font-semibold">
                    <Plus className="w-5 h-5" />
                    <span>보드 추가</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          {allProcessedBoards?.length === 0 && (
            <div className="text-center py-12 text-gray-500">
              보드가 없습니다. 보드를 추가해보세요.
            </div>
          )}
        </div>
      ) : (
        // Board Layout (Kanban)
        <div className="flex flex-col lg:flex-row gap-3 sm:gap-4 min-w-max pb-4 mt-4">
          {currentViewColumns?.map((column, idx) => {
            const columnBoards = column.boards;
            const fieldKeyName = viewState.currentView as 'stage' | 'role' | 'importance';
            const initialData: any = {
              stage: fieldOptionsLookup.stages?.[0]?.optionValue,
              role: fieldOptionsLookup.roles?.[0]?.optionValue,
              importance: fieldOptionsLookup.importances?.[0]?.optionValue,
              [fieldKeyName]: column.stageId,
            };

            return (
              <div
                key={column?.stageId}
                onDragOver={(e) => {
                  handleDragOver(e);
                  if (draggedBoard && !draggedColumn) {
                    setDragOverColumn(column.stageId);
                  }
                }}
                onDragLeave={() => {
                  if (draggedBoard && !draggedColumn) {
                    setDragOverColumn(null);
                  }
                }}
                onDrop={() => {
                  console.log('🔍 [DROP] draggedBoard:', draggedBoard);
                  console.log('🔍 [DROP] column.stageId:', column.stageId);
                  if (draggedBoard) handleDrop(column.stageId);
                }}
                className="w-full lg:w-80 lg:flex-shrink-0 relative transition-all"
              >
                <div
                  className={`relative ${theme.effects.cardBorderWidth} ${
                    dragOverColumn === column.stageId && draggedFromColumn !== column.stageId
                      ? 'border-blue-500 border-2 bg-blue-50 dark:bg-blue-900/20 shadow-lg'
                      : theme.colors.border
                  } p-3 sm:p-4 ${theme.colors.card} ${
                    theme.effects.borderRadius
                  } transition-all duration-200`}
                >
                  <div className={`flex items-center justify-between pb-2`}>
                    <h3
                      className={`font-bold ${theme.colors.text} flex items-center gap-2 ${theme.font.size.xs}`}
                    >
                      <span
                        className={`w-3 h-3 sm:w-4 sm:h-4 ${theme.effects.cardBorderWidth} ${theme.colors.border}`}
                        style={{
                          backgroundColor: column.color || getDefaultColorByIndex(idx).hex,
                        }}
                      ></span>
                      {column.title}
                      <span
                        className={`bg-black text-white px-1 sm:px-2 py-1 ${theme.effects.cardBorderWidth} ${theme.colors.border} text-[8px] sm:text-xs`}
                      >
                        {columnBoards?.length}
                      </span>
                    </h3>
                  </div>

                  <div className="space-y-2 sm:space-y-3">
                    {columnBoards?.map((board, idx) => (
                      <div
                        onDragEnd={handleDragEnd}
                        key={board.boardId + column.stageId + idx}
                        className="relative"
                        onDragOver={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          if (draggedBoard && draggedBoard.boardId !== board.boardId) {
                            setDragOverBoardId(board.boardId);
                          }
                        }}
                        onDragLeave={(e) => {
                          e.stopPropagation();
                          setDragOverBoardId(null);
                        }}
                      >
                        {dragOverBoardId === board.boardId &&
                          draggedBoard &&
                          draggedBoard.boardId !== board.boardId && (
                            <div className="absolute -top-2 left-0 right-0 h-1 bg-blue-500 rounded-full shadow-lg shadow-blue-500/50 z-10"></div>
                          )}
                        <div
                          draggable
                          onDragStart={(e) => {
                            e.stopPropagation();
                            handleDragStart(board, column.stageId);
                          }}
                          onDragEnd={handleDragEnd}
                          onClick={() => setSelectedBoardId(board.boardId)}
                          className={`relative ${theme.colors.card} p-3 sm:p-4 ${
                            theme.effects.cardBorderWidth
                          } ${
                            theme.colors.border
                          } hover:border-blue-500 transition-all cursor-pointer ${
                            theme.effects.borderRadius
                          } 
                            ${
                              draggedBoard?.boardId === board.boardId
                                ? 'opacity-50 scale-95 shadow-2xl rotate-1'
                                : 'opacity-100'
                            }
                          `}
                        >
                          <h3
                            className={`font-bold ${theme.colors.text} mb-2 sm:mb-3 ${theme.font.size.xs} break-words`}
                          >
                            {board.title}
                          </h3>

                          {/* 💡 [수정] 아이콘과 함께 표시할 컨테이너 */}
                          <div className="flex flex-col">
                            {/* 1. 작업 할당자 (Assignee) */}
                            <div className="flex items-center justify-between">
                              <span className="text-xs font-semibold text-gray-500 flex items-center gap-1">
                                <User className="w-3 h-3" /> 할당자
                              </span>
                              <AssigneeAvatarStack
                                assignees={board.assigneeId || 'Unassigned'}
                                workspaceMembers={workspaceMembers} // 💡 추가
                              />
                            </div>

                            {/* 2. 참여자 (Participants) */}
                            {(board.participantIds?.length || 0) > 0 && (
                              <div className="flex items-center justify-between">
                                <span className="text-xs font-semibold text-gray-500 flex items-center gap-1">
                                  <Users className="w-3 h-3" /> 참여 ({board.participantIds?.length}
                                  명)
                                </span>
                                {/* AssigneeAvatarStack이 ID 배열도 받도록 구성되어 있음 */}
                                <AssigneeAvatarStack
                                  assignees={board.participantIds || []}
                                  workspaceMembers={workspaceMembers} // 💡 추가
                                />
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}

                    {columnBoards?.length === 0 &&
                      dragOverColumn === column.stageId &&
                      draggedBoard &&
                      !draggedColumn && (
                        <div className="relative py-2">
                          <div className="h-1 bg-blue-500 rounded-full shadow-lg shadow-blue-500/50"></div>
                        </div>
                      )}

                    <button
                      className={`relative w-full py-3 sm:py-4 ${theme.effects.cardBorderWidth} border-dashed ${theme.colors.border} ${theme.colors.card} hover:bg-gray-100 transition flex items-center justify-center gap-2 ${theme.font.size.xs} ${theme.effects.borderRadius}`}
                      onClick={() => {
                        onEditBoard(initialData);
                        setShowCreateBoard(true);
                      }}
                      onDragOver={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (draggedBoard && !draggedColumn) {
                          setDragOverColumn(column.stageId);
                          setDragOverBoardId(null);
                        }
                      }}
                    >
                      <Plus className="w-3 h-3 sm:w-4 sm:h-4" style={{ strokeWidth: 3 }} />
                      보드 추가
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Board Detail Modal */}
      {selectedBoardId && (
        <BoardDetailModal
          boardId={selectedBoardId}
          workspaceId={workspaceId}
          onClose={() => setSelectedBoardId(null)}
          onBoardUpdated={fetchBoards}
          onBoardDeleted={fetchBoards}
          onEdit={handleBoardEdit}
          fieldOptionsLookup={fieldOptionsLookup}
        />
      )}
    </>
  );
};
