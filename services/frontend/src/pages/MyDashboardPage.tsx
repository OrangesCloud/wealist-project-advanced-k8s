import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';
import {
  getMyWorkspaces,
  createWorkspace,
  getPublicWorkspaces,
  createJoinRequest,
} from '../api/userService';
import { getProjects, getBoards } from '../api/boardService';
import {
  UserWorkspaceResponse,
  WorkspaceResponse,
  CreateWorkspaceRequest,
} from '../types/user';
import { ProjectResponse, BoardResponse } from '../types/board';
import {
  LayoutDashboard,
  FolderKanban,
  CheckCircle2,
  Clock,
  AlertCircle,
  ChevronRight,
  ChevronDown,
  LogOut,
  Briefcase,
  Plus,
  RefreshCw,
  Search,
  X,
  Settings,
  List,
  LayoutGrid,
} from 'lucide-react';
import WorkspaceManagementModal from '../components/modals/user/wsManager/WorkspaceManagementModal';

// Grouped tasks by workspace and project
interface GroupedTasks {
  workspace: UserWorkspaceResponse;
  projects: {
    project: ProjectResponse;
    tasks: BoardResponse[];
  }[];
}

const MyDashboardPage: React.FC = () => {
  const navigate = useNavigate();
  const { theme } = useTheme();
  const { logout, nickName } = useAuth();

  // All workspaces (including pending)
  const [allWorkspaces, setAllWorkspaces] = useState<UserWorkspaceResponse[]>([]);
  const [workspaces, setWorkspaces] = useState<UserWorkspaceResponse[]>([]);
  const [groupedTasks, setGroupedTasks] = useState<GroupedTasks[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'all' | 'pending' | 'inprogress' | 'completed'>('all');

  // Workspace search & create state
  const [searchQuery, setSearchQuery] = useState('');
  const [searchedWorkspaces, setSearchedWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  // Workspace management modal
  const [managingWorkspace, setManagingWorkspace] = useState<UserWorkspaceResponse | null>(null);

  // Workspace section expanded state
  const [isWorkspaceExpanded, setIsWorkspaceExpanded] = useState(true);

  // Task view mode: compact shows only title in one line
  const [isCompactView, setIsCompactView] = useState(false);
  // Collapsed workspace sections in task list
  const [collapsedWorkspaces, setCollapsedWorkspaces] = useState<Set<string>>(new Set());

  const toggleWorkspaceCollapse = (workspaceId: string) => {
    setCollapsedWorkspaces((prev) => {
      const next = new Set(prev);
      if (next.has(workspaceId)) {
        next.delete(workspaceId);
      } else {
        next.add(workspaceId);
      }
      return next;
    });
  };

  // Fetch all data
  const fetchDashboardData = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      // 1. Fetch all workspaces
      const fetchedWorkspaces = await getMyWorkspaces();
      setAllWorkspaces(fetchedWorkspaces);
      const activeWorkspaces = fetchedWorkspaces.filter((ws) => ws.role !== 'PENDING');
      setWorkspaces(activeWorkspaces);

      // 2. For each workspace, fetch projects and boards
      const allGroupedTasks: GroupedTasks[] = [];

      for (const workspace of activeWorkspaces) {
        try {
          const projects = await getProjects(workspace.workspaceId);

          const projectsWithTasks = await Promise.all(
            projects.map(async (project) => {
              try {
                const boards = await getBoards(project.projectId);
                return { project, tasks: boards };
              } catch {
                return { project, tasks: [] };
              }
            })
          );

          // Only add workspaces that have tasks
          const filteredProjects = projectsWithTasks.filter((p) => p.tasks.length > 0);
          if (filteredProjects.length > 0) {
            allGroupedTasks.push({
              workspace,
              projects: filteredProjects,
            });
          }
        } catch {
          continue;
        }
      }

      setGroupedTasks(allGroupedTasks);
    } catch (e: any) {
      console.error('Dashboard data fetch error:', e);
      setError('데이터를 불러오는 중 오류가 발생했습니다.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  // Search workspaces
  const handleSearch = useCallback(async () => {
    if (!searchQuery.trim()) {
      setSearchedWorkspaces([]);
      return;
    }

    setIsSearching(true);
    try {
      const results = await getPublicWorkspaces(searchQuery);
      const myIds = new Set(allWorkspaces.map((w) => w.workspaceId));
      const filteredResults = results.filter((r) => !myIds.has(r.workspaceId));
      setSearchedWorkspaces(filteredResults);
    } catch {
      setSearchedWorkspaces([]);
    } finally {
      setIsSearching(false);
    }
  }, [searchQuery, allWorkspaces]);

  // Join workspace request
  const handleJoinRequest = async (workspace: WorkspaceResponse) => {
    try {
      await createJoinRequest(workspace.workspaceId);
      setSearchQuery('');
      setSearchedWorkspaces([]);
      await fetchDashboardData();
      alert(`'${workspace.workspaceName}'에 가입 요청을 보냈습니다.`);
    } catch (e: any) {
      setError(`가입 요청 실패: ${e.message}`);
    }
  };

  // Create workspace
  const handleCreateWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;

    setIsCreating(true);
    try {
      const createData: CreateWorkspaceRequest = {
        workspaceName: newWorkspaceName,
        workspaceDescription: newDescription || '-',
      };
      await createWorkspace(createData);
      setIsCreating(false);
      alert('워크스페이스가 생성되었습니다!');
      // Alert 확인 후 UI 업데이트
      setShowCreateForm(false);
      setNewWorkspaceName('');
      setNewDescription('');
      await fetchDashboardData();
    } catch (e: any) {
      setError(`워크스페이스 생성 실패: ${e.message}`);
      setIsCreating(false);
    }
  };

  // Filter tasks by status
  const filterTasks = (tasks: BoardResponse[]): BoardResponse[] => {
    if (activeTab === 'all') return tasks;

    return tasks.filter((task) => {
      const stage = task.customFields?.stage?.toLowerCase() || '';
      switch (activeTab) {
        case 'pending':
          // 'pending', 'backlog', 'todo', 빈 값 모두 대기 상태로 처리
          return stage === 'pending' || stage === 'backlog' || stage === 'todo' || stage === '' || stage === 'review';
        case 'inprogress':
          return stage === 'in_progress' || stage === 'in progress';
        case 'completed':
          return stage === 'done' || stage === 'completed' || stage === 'approved';
        default:
          return true;
      }
    });
  };

  // Count tasks
  const countTasks = () => {
    let total = 0;
    let pending = 0;
    let inprogress = 0;
    let completed = 0;

    groupedTasks.forEach((group) => {
      group.projects.forEach((projectGroup) => {
        projectGroup.tasks.forEach((task) => {
          total++;
          const stage = task.customFields?.stage?.toLowerCase() || '';
          // 완료 상태: done, completed, approved
          if (stage === 'done' || stage === 'completed' || stage === 'approved') completed++;
          // 진행중 상태: in_progress
          else if (stage === 'in_progress' || stage === 'in progress') inprogress++;
          // 대기 상태: pending, backlog, todo, review, 빈 값
          else pending++;
        });
      });
    });

    return { total, pending, inprogress, completed };
  };

  const taskCounts = countTasks();
  const pendingWorkspaces = allWorkspaces.filter((ws) => ws.role === 'PENDING');

  // 💡 [추가] 내 워크스페이스 목록 필터링 (검색어로 필터)
  const filteredWorkspaces = searchQuery.trim()
    ? workspaces.filter(
        (ws) =>
          ws.workspaceName.toLowerCase().includes(searchQuery.toLowerCase()) ||
          ws.workspaceDescription?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : workspaces;

  // Get stage badge color and Korean label
  const getStageInfo = (stage?: string): { className: string; label: string } => {
    const stageLower = stage?.toLowerCase() || '';
    if (stageLower === 'done' || stageLower === 'completed') {
      return { className: 'bg-green-100 text-green-700 border-green-300', label: '완료' };
    }
    if (stageLower === 'in_progress' || stageLower === 'in progress') {
      return { className: 'bg-blue-100 text-blue-700 border-blue-300', label: '진행중' };
    }
    if (stageLower === 'todo') {
      return { className: 'bg-yellow-100 text-yellow-700 border-yellow-300', label: '할일' };
    }
    return { className: 'bg-gray-100 text-gray-600 border-gray-300', label: '대기' };
  };

  // Get importance badge and Korean label
  const getImportanceInfo = (importance?: string): { className: string; label: string } => {
    const impLower = importance?.toLowerCase() || '';
    if (impLower === 'high' || impLower === 'urgent') {
      return { className: 'bg-red-100 text-red-700', label: '높음' };
    }
    if (impLower === 'medium' || impLower === 'normal') {
      return { className: 'bg-yellow-100 text-yellow-700', label: '보통' };
    }
    if (impLower === 'low') {
      return { className: 'bg-gray-100 text-gray-500', label: '낮음' };
    }
    return { className: 'bg-gray-100 text-gray-500', label: importance || '' };
  };

  // Navigate to workspace
  const handleGoToWorkspace = (workspace: UserWorkspaceResponse) => {
    if (workspace.role === 'PENDING') return;
    navigate(`/workspace/${workspace.workspaceId}`, { state: { userRole: workspace.role } });
  };

  // Navigate to task
  const handleGoToTask = (workspace: UserWorkspaceResponse, projectId: string, boardId: string) => {
    navigate(`/workspace/${workspace.workspaceId}?project=${projectId}&board=${boardId}`, {
      state: { userRole: workspace.role },
    });
  };

  if (isLoading) {
    return (
      <div className={`min-h-screen ${theme.colors.background} flex items-center justify-center relative`}>
        {/* Grid Pattern Background */}
        <div
          className="absolute inset-0 opacity-5 pointer-events-none z-0"
          style={{
            backgroundImage:
              'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)',
            backgroundSize: '20px 20px',
          }}
        />
        <div className="text-center relative z-10">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p className={`${theme.font.size.lg} ${theme.colors.text}`}>대시보드 로딩 중...</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen ${theme.colors.background} relative`}>
      {/* Grid Pattern Background */}
      <div
        className="absolute inset-0 opacity-5 pointer-events-none z-0"
        style={{
          backgroundImage:
            'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)',
          backgroundSize: '20px 20px',
        }}
      />
      {/* Header */}
      <header className={`${theme.colors.card} border-b ${theme.colors.border} sticky top-0 z-50`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <LayoutDashboard className="w-8 h-8 text-blue-600" />
              <div>
                <h1 className={`text-xl font-bold ${theme.colors.text}`}>My Dashboard</h1>
                <p className={`text-sm ${theme.colors.subText}`}>
                  {nickName ? `${nickName}님의 워크스페이스` : '나의 워크스페이스'}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={fetchDashboardData}
                className="p-2 hover:bg-gray-100 rounded-lg transition"
                title="새로고침"
              >
                <RefreshCw className="w-5 h-5 text-gray-600" />
              </button>
              <button
                onClick={logout}
                className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition"
                title="로그아웃"
              >
                <LogOut className="w-4 h-4" />
                <span className="hidden sm:inline text-sm">로그아웃</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 relative z-10">
        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-700">
            <AlertCircle className="w-5 h-5" />
            {error}
            <button onClick={() => setError(null)} className="ml-auto text-red-500 hover:text-red-700">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Workspaces Section - Left/Top */}
          <div className="lg:col-span-1">
            <div className={`${theme.colors.card} rounded-xl shadow-sm border ${theme.colors.border}`}>
              {/* Workspace Header with Toggle */}
              <div
                className="p-4 border-b border-gray-100 cursor-pointer flex items-center justify-between"
                onClick={() => setIsWorkspaceExpanded(!isWorkspaceExpanded)}
              >
                <h2 className={`text-lg font-semibold ${theme.colors.text} flex items-center gap-2`}>
                  <Briefcase className="w-5 h-5 text-blue-600" />
                  내 워크스페이스
                  <span className="text-sm font-normal text-gray-400">({workspaces.length})</span>
                </h2>
                {isWorkspaceExpanded ? (
                  <ChevronDown className="w-5 h-5 text-gray-400" />
                ) : (
                  <ChevronRight className="w-5 h-5 text-gray-400" />
                )}
              </div>

              {isWorkspaceExpanded && (
                <div className="p-4">
                  {/* Search Workspace */}
                  <div className="relative mb-4">
                    <input
                      type="text"
                      placeholder="워크스페이스 검색..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                      className="w-full px-4 py-2 pr-10 text-sm border border-gray-300 rounded-lg focus:border-blue-500 focus:ring-1 focus:ring-blue-200"
                    />
                    <button
                      onClick={handleSearch}
                      className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-blue-500"
                    >
                      <Search className="w-4 h-4" />
                    </button>
                  </div>

                  {/* Search Results */}
                  {(searchedWorkspaces.length > 0 || isSearching) && (
                    <div className="mb-4 border border-gray-200 rounded-lg overflow-hidden">
                      <div className="px-3 py-2 bg-gray-50 text-xs font-medium text-gray-600">
                        {isSearching ? '검색 중...' : `검색 결과 (${searchedWorkspaces.length})`}
                      </div>
                      <div className="max-h-32 overflow-y-auto">
                        {searchedWorkspaces.map((ws) => (
                          <div
                            key={ws.workspaceId}
                            onClick={() => handleJoinRequest(ws)}
                            className="px-3 py-2 hover:bg-green-50 cursor-pointer flex justify-between items-center border-t border-gray-100"
                          >
                            <div>
                              <span className="text-sm font-medium">{ws.workspaceName}</span>
                              <p className="text-xs text-gray-500">{ws.workspaceDescription}</p>
                            </div>
                            <span className="text-xs text-green-600 bg-green-50 px-2 py-1 rounded">
                              가입 요청
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Pending Workspaces */}
                  {pendingWorkspaces.length > 0 && (
                    <div className="mb-4">
                      <p className="text-xs font-medium text-gray-500 mb-2">승인 대기 중</p>
                      {pendingWorkspaces.map((ws) => (
                        <div
                          key={ws.workspaceId}
                          className="p-2 rounded-lg bg-yellow-50 border border-yellow-200 mb-2"
                        >
                          <div className="flex items-center gap-2">
                            <div className="w-8 h-8 bg-yellow-400 rounded-lg flex items-center justify-center">
                              <Clock className="w-4 h-4 text-white" />
                            </div>
                            <div className="flex-1 min-w-0">
                              <p className="text-sm font-medium truncate">{ws.workspaceName}</p>
                              <p className="text-xs text-yellow-600">승인 대기</p>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Workspace List */}
                  <div className="space-y-2 max-h-[300px] overflow-y-auto">
                    {filteredWorkspaces.length === 0 ? (
                      <div className="text-center py-6">
                        <Briefcase className="w-10 h-10 text-gray-300 mx-auto mb-2" />
                        <p className="text-sm text-gray-500">
                          {searchQuery.trim() ? '검색 결과가 없습니다.' : '소속된 워크스페이스가 없습니다.'}
                        </p>
                      </div>
                    ) : (
                      filteredWorkspaces.map((workspace) => (
                        <div
                          key={workspace.workspaceId}
                          className="p-3 rounded-lg hover:bg-gray-50 cursor-pointer transition border border-transparent hover:border-gray-200 group"
                        >
                          <div className="flex items-center gap-3" onClick={() => handleGoToWorkspace(workspace)}>
                            <div className="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center flex-shrink-0">
                              <span className="text-white font-bold">
                                {workspace.workspaceName.charAt(0).toUpperCase()}
                              </span>
                            </div>
                            <div className="flex-1 min-w-0">
                              <h3 className={`font-medium ${theme.colors.text} truncate text-sm`}>
                                {workspace.workspaceName}
                              </h3>
                              <p className="text-xs text-gray-500 truncate">
                                {workspace.workspaceDescription || '-'}
                              </p>
                            </div>
                            <div className="flex items-center gap-1">
                              {(workspace.owner || workspace.role === 'ADMIN') && (
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setManagingWorkspace(workspace);
                                  }}
                                  className="p-1 hover:bg-gray-200 rounded transition opacity-0 group-hover:opacity-100"
                                  title="설정"
                                >
                                  <Settings className="w-4 h-4 text-gray-500" />
                                </button>
                              )}
                              <ChevronRight className="w-4 h-4 text-gray-400 group-hover:text-blue-600 transition" />
                            </div>
                          </div>
                          <div className="mt-2 flex items-center gap-2 ml-13">
                            <span
                              className={`text-xs px-2 py-0.5 rounded ${
                                workspace.owner ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-600'
                              }`}
                            >
                              {workspace.role}
                            </span>
                          </div>
                        </div>
                      ))
                    )}
                  </div>

                  {/* Create Workspace */}
                  {!showCreateForm ? (
                    <button
                      onClick={() => setShowCreateForm(true)}
                      className="w-full mt-4 p-3 border-2 border-dashed border-gray-300 rounded-lg text-gray-500 hover:border-blue-400 hover:text-blue-600 transition flex items-center justify-center gap-2"
                    >
                      <Plus className="w-4 h-4" />
                      새 워크스페이스 만들기
                    </button>
                  ) : (
                    <div className="mt-4 p-4 border border-blue-200 rounded-lg bg-blue-50">
                      <h4 className="font-medium text-sm text-blue-800 mb-3">새 워크스페이스</h4>
                      <input
                        type="text"
                        placeholder="워크스페이스 이름"
                        value={newWorkspaceName}
                        onChange={(e) => setNewWorkspaceName(e.target.value)}
                        className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg mb-2 focus:border-blue-500"
                      />
                      <input
                        type="text"
                        placeholder="설명 (선택)"
                        value={newDescription}
                        onChange={(e) => setNewDescription(e.target.value)}
                        className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg mb-3 focus:border-blue-500"
                      />
                      <div className="flex gap-2">
                        <button
                          onClick={() => {
                            setShowCreateForm(false);
                            setNewWorkspaceName('');
                            setNewDescription('');
                          }}
                          className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-100"
                        >
                          취소
                        </button>
                        <button
                          onClick={handleCreateWorkspace}
                          disabled={!newWorkspaceName.trim() || isCreating}
                          className="flex-1 px-3 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                        >
                          {isCreating ? '생성 중...' : '생성'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Tasks Section - Right/Bottom */}
          <div className="lg:col-span-2">
            {/* Statistics Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
              <div
                onClick={() => setActiveTab('all')}
                className={`p-3 rounded-xl cursor-pointer transition-all ${
                  activeTab === 'all'
                    ? 'bg-blue-600 text-white shadow-lg scale-105'
                    : 'bg-white hover:bg-blue-50 border border-gray-200'
                }`}
              >
                <div className="flex items-center gap-2">
                  <div className={`p-1.5 rounded-lg ${activeTab === 'all' ? 'bg-blue-500' : 'bg-blue-100'}`}>
                    <Briefcase className={`w-4 h-4 ${activeTab === 'all' ? 'text-white' : 'text-blue-600'}`} />
                  </div>
                  <div>
                    <p className="text-xl font-bold">{taskCounts.total}</p>
                    <p className={`text-xs ${activeTab === 'all' ? 'text-blue-100' : 'text-gray-500'}`}>전체</p>
                  </div>
                </div>
              </div>

              <div
                onClick={() => setActiveTab('pending')}
                className={`p-3 rounded-xl cursor-pointer transition-all ${
                  activeTab === 'pending'
                    ? 'bg-gray-700 text-white shadow-lg scale-105'
                    : 'bg-white hover:bg-gray-50 border border-gray-200'
                }`}
              >
                <div className="flex items-center gap-2">
                  <div className={`p-1.5 rounded-lg ${activeTab === 'pending' ? 'bg-gray-600' : 'bg-gray-100'}`}>
                    <Clock className={`w-4 h-4 ${activeTab === 'pending' ? 'text-white' : 'text-gray-600'}`} />
                  </div>
                  <div>
                    <p className="text-xl font-bold">{taskCounts.pending}</p>
                    <p className={`text-xs ${activeTab === 'pending' ? 'text-gray-300' : 'text-gray-500'}`}>대기</p>
                  </div>
                </div>
              </div>

              <div
                onClick={() => setActiveTab('inprogress')}
                className={`p-3 rounded-xl cursor-pointer transition-all ${
                  activeTab === 'inprogress'
                    ? 'bg-yellow-500 text-white shadow-lg scale-105'
                    : 'bg-white hover:bg-yellow-50 border border-gray-200'
                }`}
              >
                <div className="flex items-center gap-2">
                  <div className={`p-1.5 rounded-lg ${activeTab === 'inprogress' ? 'bg-yellow-400' : 'bg-yellow-100'}`}>
                    <AlertCircle className={`w-4 h-4 ${activeTab === 'inprogress' ? 'text-white' : 'text-yellow-600'}`} />
                  </div>
                  <div>
                    <p className="text-xl font-bold">{taskCounts.inprogress}</p>
                    <p className={`text-xs ${activeTab === 'inprogress' ? 'text-yellow-100' : 'text-gray-500'}`}>진행</p>
                  </div>
                </div>
              </div>

              <div
                onClick={() => setActiveTab('completed')}
                className={`p-3 rounded-xl cursor-pointer transition-all ${
                  activeTab === 'completed'
                    ? 'bg-green-600 text-white shadow-lg scale-105'
                    : 'bg-white hover:bg-green-50 border border-gray-200'
                }`}
              >
                <div className="flex items-center gap-2">
                  <div className={`p-1.5 rounded-lg ${activeTab === 'completed' ? 'bg-green-500' : 'bg-green-100'}`}>
                    <CheckCircle2 className={`w-4 h-4 ${activeTab === 'completed' ? 'text-white' : 'text-green-600'}`} />
                  </div>
                  <div>
                    <p className="text-xl font-bold">{taskCounts.completed}</p>
                    <p className={`text-xs ${activeTab === 'completed' ? 'text-green-100' : 'text-gray-500'}`}>완료</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Tasks List */}
            <div className={`${theme.colors.card} rounded-xl shadow-sm border ${theme.colors.border}`}>
              <div className="p-4 border-b border-gray-100 flex items-center justify-between">
                <h2 className={`text-lg font-semibold ${theme.colors.text} flex items-center gap-2`}>
                  <FolderKanban className="w-5 h-5 text-blue-600" />
                  내 일감
                  <span className="text-sm font-normal text-gray-500">(조직/프로젝트별)</span>
                </h2>
                <button
                  onClick={() => setIsCompactView(!isCompactView)}
                  className={`p-2 rounded-lg transition ${isCompactView ? 'bg-blue-100 text-blue-600' : 'hover:bg-gray-100 text-gray-500'}`}
                  title={isCompactView ? '상세 보기' : '간략히 보기'}
                >
                  {isCompactView ? <LayoutGrid className="w-4 h-4" /> : <List className="w-4 h-4" />}
                </button>
              </div>

              <div className="p-4 max-h-[500px] overflow-y-auto">
                {groupedTasks.length === 0 ? (
                  <div className="text-center py-12">
                    <FolderKanban className="w-16 h-16 text-gray-300 mx-auto mb-4" />
                    <p className={`${theme.colors.subText} mb-2`}>참여 중인 프로젝트의 일감이 없습니다.</p>
                    <p className="text-sm text-gray-400">워크스페이스에 참여하고 프로젝트를 시작해보세요!</p>
                  </div>
                ) : (
                  <div className="space-y-6">
                    {groupedTasks.map((group) => {
                      const filteredProjects = group.projects
                        .map((p) => ({ ...p, tasks: filterTasks(p.tasks) }))
                        .filter((p) => p.tasks.length > 0);

                      if (filteredProjects.length === 0) return null;

                      const totalTaskCount = filteredProjects.reduce((sum, p) => sum + p.tasks.length, 0);
                      const isCollapsed = collapsedWorkspaces.has(group.workspace.workspaceId);

                      return (
                        <div key={group.workspace.workspaceId} className="space-y-3">
                          {/* Workspace Header - Collapsible */}
                          <div className="flex items-center gap-2 p-2 rounded-lg transition hover:bg-gray-50">
                            <button
                              onClick={() => toggleWorkspaceCollapse(group.workspace.workspaceId)}
                              className="p-1 hover:bg-gray-200 rounded"
                            >
                              {isCollapsed ? (
                                <ChevronRight className="w-4 h-4 text-gray-400" />
                              ) : (
                                <ChevronDown className="w-4 h-4 text-gray-400" />
                              )}
                            </button>
                            <div
                              onClick={() => handleGoToWorkspace(group.workspace)}
                              className="flex items-center gap-2 flex-1 cursor-pointer group"
                            >
                              <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
                                <span className="text-white text-xs font-bold">
                                  {group.workspace.workspaceName.charAt(0).toUpperCase()}
                                </span>
                              </div>
                              <span className={`font-semibold text-sm ${theme.colors.text}`}>
                                {group.workspace.workspaceName}
                              </span>
                              <span className="text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">
                                {totalTaskCount}개
                              </span>
                              <ChevronRight className="w-4 h-4 text-gray-400 group-hover:text-blue-600 ml-auto transition opacity-0 group-hover:opacity-100" />
                            </div>
                          </div>

                          {/* Projects and Tasks - Collapsible */}
                          {!isCollapsed && (
                            <div className="ml-4 space-y-3">
                              {filteredProjects.map((projectGroup) => (
                                <div key={projectGroup.project.projectId} className="space-y-2">
                                  <div className="flex items-center gap-2 text-sm text-gray-600">
                                    <FolderKanban className="w-4 h-4" />
                                    <span className="font-medium">{projectGroup.project.name}</span>
                                    <span className="text-gray-400">({projectGroup.tasks.length})</span>
                                  </div>

                                  <div className={`ml-6 ${isCompactView ? 'space-y-1' : 'space-y-2'}`}>
                                    {projectGroup.tasks.slice(0, isCompactView ? 10 : 5).map((task) => {
                                      const stageInfo = getStageInfo(task.customFields?.stage);

                                      // Compact view: single line per task
                                      if (isCompactView) {
                                        return (
                                          <div
                                            key={task.boardId}
                                            onClick={() =>
                                              handleGoToTask(group.workspace, projectGroup.project.projectId, task.boardId)
                                            }
                                            className="flex items-center gap-2 py-1.5 px-2 bg-gray-50 hover:bg-blue-50 rounded cursor-pointer transition group"
                                          >
                                            <span className={`text-xs px-1.5 py-0.5 rounded border ${stageInfo.className}`}>
                                              {stageInfo.label}
                                            </span>
                                            <span className={`flex-1 text-sm ${theme.colors.text} truncate`}>
                                              {task.title}
                                            </span>
                                            {task.dueDate && (
                                              <span className="text-xs text-gray-400">
                                                {new Date(task.dueDate).toLocaleDateString('ko-KR', { month: 'short', day: 'numeric' })}
                                              </span>
                                            )}
                                          </div>
                                        );
                                      }

                                      // Detailed view: full card
                                      return (
                                        <div
                                          key={task.boardId}
                                          onClick={() =>
                                            handleGoToTask(group.workspace, projectGroup.project.projectId, task.boardId)
                                          }
                                          className="p-3 bg-gray-50 hover:bg-blue-50 rounded-lg cursor-pointer transition border border-transparent hover:border-blue-200"
                                        >
                                          <div className="flex items-start justify-between gap-2">
                                            <div className="flex-1 min-w-0">
                                              <h4 className={`font-medium text-sm ${theme.colors.text} truncate`}>
                                                {task.title}
                                              </h4>
                                              {task.content && (
                                                <p className="text-xs text-gray-500 truncate mt-0.5">{task.content}</p>
                                              )}
                                            </div>
                                            <div className="flex flex-col items-end gap-1">
                                              <span className={`text-xs px-1.5 py-0.5 rounded border ${stageInfo.className}`}>
                                                {stageInfo.label}
                                              </span>
                                              {task.customFields?.importance && (() => {
                                                const impInfo = getImportanceInfo(task.customFields?.importance);
                                                return (
                                                  <span className={`text-xs px-1.5 py-0.5 rounded ${impInfo.className}`}>
                                                    {impInfo.label}
                                                  </span>
                                                );
                                              })()}
                                            </div>
                                          </div>
                                          {task.dueDate && (
                                            <div className="flex items-center gap-1 mt-1.5 text-xs text-gray-500">
                                              <Clock className="w-3 h-3" />
                                              <span>마감: {new Date(task.dueDate).toLocaleDateString('ko-KR')}</span>
                                            </div>
                                          )}
                                        </div>
                                      );
                                    })}
                                    {projectGroup.tasks.length > (isCompactView ? 10 : 5) && (
                                      <button
                                        onClick={() => handleGoToWorkspace(group.workspace)}
                                        className="w-full text-center text-xs text-blue-600 hover:text-blue-800 py-2"
                                      >
                                        +{projectGroup.tasks.length - (isCompactView ? 10 : 5)}개 더 보기
                                      </button>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Workspace Management Modal */}
      {managingWorkspace && (
        <WorkspaceManagementModal
          workspaceId={managingWorkspace.workspaceId}
          workspaceName={managingWorkspace.workspaceName}
          onClose={() => setManagingWorkspace(null)}
        />
      )}
    </div>
  );
};

export default MyDashboardPage;
