// src/components/modals/board/FilterBar.tsx

import React, { useState, useEffect, useRef, useMemo } from 'react';
import {
  Search,
  ChevronDown,
  Eye,
  Table,
  LayoutGrid,
  UserCheck,
  Check,
} from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import { FieldOption, TLayout, TView } from '../../../types/board';

interface FilterBarProps {
  onSearchChange: (search: string) => void;
  onViewChange: (view: TView) => void;
  onFilterChange: (filter: string) => void;
  onManageClick: () => void;
  currentView: TView;
  onLayoutChange?: (layout: TLayout) => void;
  onShowCompletedChange?: (show: boolean) => void;
  onShowDeletedChange?: (show: boolean) => void;
  currentLayout?: TLayout;
  showCompleted?: boolean;
  showDeleted?: boolean;

  // 💡 [수정] FieldOption 타입 사용
  stageOptions: FieldOption[];
  roleOptions: FieldOption[];
  importanceOptions: FieldOption[];

  // 💡 [추가] 현재 선택된 필터 상태를 받습니다.
  currentFilter: string;
}

// 💡 상수 정의
const FILTER_ALL = 'all';
const FILTER_MY_TASKS = 'my_tasks';

export const FilterBar: React.FC<FilterBarProps> = ({
  onSearchChange,
  onViewChange,
  onFilterChange, // 💡 사용
  onLayoutChange,
  onShowCompletedChange,
  onShowDeletedChange,
  currentView,
  currentLayout = 'board',
  showCompleted = false,
  showDeleted = true, // 기본값 true
  currentFilter, // 💡 추가
  // stageOptions,
  // roleOptions,
  // importanceOptions,
}) => {
  const { theme } = useTheme();
  const [searchValue, setSearchValue] = useState('');

  const [showViewDropdown, setShowViewDropdown] = useState(false);
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);

  // Refs for outside click detection
  const viewDropdownRef = useRef<HTMLDivElement>(null);
  const filterDropdownRef = useRef<HTMLDivElement>(null);

  const handleSearchChange = (value: string) => {
    setSearchValue(value);
    onSearchChange(value);
  };

  const handleViewChange = (view: TView) => {
    onViewChange(view);
    setShowViewDropdown(false);
  };

  // 💡 [추가] 필터 변경 핸들러 (클릭 시 해제 로직 포함)
  const handleFilterToggle = (filter: string) => {
    // 현재 선택된 필터와 동일하면 해제(all)하고, 아니면 선택합니다.
    const newFilter = currentFilter === filter ? FILTER_ALL : filter;
    onFilterChange(newFilter);
    setShowFilterDropdown(false);
  };

  // 외부 클릭 감지 (기존 로직 유지)
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;

      if (viewDropdownRef.current && !viewDropdownRef.current.contains(target)) {
        setShowViewDropdown(false);
      }

      if (filterDropdownRef.current && !filterDropdownRef.current.contains(target)) {
        setShowFilterDropdown(false);
      }
    };

    if (showViewDropdown || showFilterDropdown) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showViewDropdown, showFilterDropdown]);

  // 💡 [추가] 현재 필터 라벨 결정
  const filterLabel = useMemo(() => {
    switch (currentFilter) {
      case FILTER_MY_TASKS:
        return '나의 일감';
      case FILTER_ALL:
      default:
        return '나만의 필터';
    }
  }, [currentFilter]);

  return (
    <div
      className={`flex items-center gap-3 p-4 ${theme.colors.card} border-b ${theme.colors.border} flex-shrink-0`}
    >
      {/* Search */}
      <div className="relative flex-1 max-w-md">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          value={searchValue}
          onChange={(e) => handleSearchChange(e.target.value)}
          placeholder="보드 검색..."
          className={`w-full pl-10 pr-4 py-2 border ${theme.colors.border} rounded-md ${theme.colors.card} focus:outline-none focus:ring-2 focus:ring-blue-500`}
        />
      </div>

      {/* View Dropdown Button (기존 로직 유지) */}
      <div className="relative" ref={viewDropdownRef}>
        <button
          onClick={() => {
            setShowViewDropdown(!showViewDropdown);
            setShowFilterDropdown(false);
          }}
          className={`flex items-center gap-2 px-4 py-2 border ${theme.colors.border} rounded-md ${theme.colors.card} hover:bg-gray-50 transition-colors`}
        >
          <Eye className="w-4 h-4" />
          <span className="text-sm font-medium">보기</span>
          <ChevronDown
            className={`w-4 h-4 transition-transform ${showViewDropdown ? 'rotate-180' : ''}`}
          />
        </button>
        {showViewDropdown && (
          <div
            className={`absolute top-full mt-2 right-0 w-64 ${theme.colors.card} border ${theme.colors.border} rounded-lg shadow-lg z-10 p-4`}
          >
            {/* Layout Selection */}
            <div className="mb-4">
              <h4 className="text-xs font-semibold text-gray-500 mb-2">레이아웃</h4>
              <div className="flex gap-2">
                <button
                  onClick={() => {
                    onLayoutChange?.('table');
                    setShowViewDropdown(false);
                  }}
                  className={`flex-1 flex flex-col items-center gap-2 px-3 py-3 rounded-md border-2 transition-all ${
                    currentLayout === 'table'
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                  }`}
                >
                  <Table
                    className={`w-6 h-6 ${
                      currentLayout === 'table' ? 'text-blue-600' : 'text-gray-600'
                    }`}
                  />
                  <span className="text-sm font-medium">표</span>
                </button>
                <button
                  onClick={() => {
                    onLayoutChange?.('board');
                    setShowViewDropdown(false);
                  }}
                  className={`flex-1 flex flex-col items-center gap-2 px-3 py-3 rounded-md border-2 transition-all ${
                    currentLayout === 'board'
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                  }`}
                >
                  <LayoutGrid
                    className={`w-6 h-6 ${
                      currentLayout === 'board' ? 'text-blue-600' : 'text-gray-600'
                    }`}
                  />
                  <span className="text-sm font-medium">보드</span>
                </button>
              </div>
            </div>

            {/* Divider */}
            <div className="border-t border-gray-200 my-3"></div>

            {/* View By (Group By) */}
            <div>
              <h4 className="text-xs font-semibold text-gray-500 mb-2">그룹 기준</h4>
              <button
                onClick={() => handleViewChange('stage')}
                className={`w-full px-3 py-2 text-left text-sm rounded hover:bg-gray-100 ${
                  currentView === 'stage' ? 'bg-blue-100 text-blue-700' : ''
                }`}
              >
                작업단계 기준
              </button>
              <button
                onClick={() => handleViewChange('importance')}
                className={`w-full px-3 py-2 text-left text-sm rounded hover:bg-gray-100 ${
                  currentView === 'importance' ? 'bg-blue-100 text-blue-700' : ''
                }`}
              >
                중요도 기준
              </button>
              <button
                onClick={() => handleViewChange('role')}
                className={`w-full px-3 py-2 text-left text-sm rounded hover:bg-gray-100 ${
                  currentView === 'role' ? 'bg-blue-100 text-blue-700' : ''
                }`}
              >
                역할 기준
              </button>
            </div>

            {/* Divider */}
            <div className="border-t border-gray-200 my-3"></div>

            {/* Show Completed Toggle */}
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-700">완료된 항목 보기</span>
              <button
                onClick={() => {
                  onShowCompletedChange?.(!showCompleted);
                }}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  showCompleted ? 'bg-blue-500' : 'bg-gray-300'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    showCompleted ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>

            {/* Show Deleted Toggle */}
            <div className="flex items-center justify-between mt-3">
              <span className="text-sm font-medium text-gray-700">삭제된 항목 보기</span>
              <button
                onClick={() => {
                  onShowDeletedChange?.(!showDeleted);
                }}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  showDeleted ? 'bg-blue-500' : 'bg-gray-300'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    showDeleted ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Filter Dropdown */}
      <div className="relative" ref={filterDropdownRef}>
        <button
          onClick={() => {
            setShowFilterDropdown(!showFilterDropdown);
            setShowViewDropdown(false);
          }}
          className={`flex items-center gap-2 px-4 py-2 border ${theme.colors.border} rounded-md ${
            theme.colors.card
          } hover:bg-gray-50 transition-colors ${
            currentFilter !== FILTER_ALL ? 'border-blue-500 text-blue-600 bg-blue-50' : ''
          }`}
        >
          <span className="text-sm font-medium">{filterLabel}</span>
          <ChevronDown className="w-4 h-4" />
        </button>
        {showFilterDropdown && (
          <div
            className={`absolute top-full mt-2 left-0 w-64 ${theme.colors.card} ${theme.effects.cardBorderWidth} ${theme.colors.border} ${theme.effects.borderRadius} shadow-lg z-10`}
          >
            <div className="pt-2 pb-2 space-y-1">
              {/* 💡 [추가] 나의 일감 필터 버튼 */}
              <button
                onClick={() => handleFilterToggle(FILTER_MY_TASKS)}
                className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 transition ${
                  currentFilter === FILTER_MY_TASKS
                    ? 'bg-blue-100 text-blue-700 font-semibold'
                    : 'hover:bg-gray-100 text-gray-700'
                }`}
              >
                <UserCheck className="w-4 h-4" />
                나의 일감
                {currentFilter === FILTER_MY_TASKS && (
                  <Check className="w-4 h-4 ml-auto text-blue-600" />
                )}
              </button>

              {/* <div className="border-t border-gray-200 my-2"></div>

              <button
                onClick={() => {
                  onManageClick();
                  setShowFilterDropdown(false);
                }}
                className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 text-blue-500 hover:bg-gray-100 transition`}
              >
                <Settings className="w-4 h-4" />
                필드 옵션 관리
              </button> */}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
