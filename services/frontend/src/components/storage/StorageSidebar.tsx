// src/components/storage/StorageSidebar.tsx - Google Drive 스타일 사이드바 (접기 기능 추가)

import React from 'react';
import {
  HardDrive,
  Clock,
  Star,
  Trash2,
  Cloud,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
} from 'lucide-react';
import {
  StorageUsage,
  formatFileSize,
  ProjectPermission,
} from '../../types/storage';
import type { ProjectResponse } from '../../types/board';

// 사이드바에서 표시되는 섹션 타입
type SidebarSection = 'recent' | 'starred' | 'trash';
// StoragePage에서 사용하는 전체 섹션 타입 (my-drive 포함)
type NavigationSection = 'my-drive' | 'recent' | 'starred' | 'trash';

interface StorageSidebarProps {
  activeSection: NavigationSection;
  onSectionChange: (section: SidebarSection) => void;
  storageUsage: StorageUsage | null;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
  // 프로젝트 관련 props (보드 서비스 프로젝트와 1:1 매핑)
  currentProject: ProjectResponse | null;
  currentProjectPermission: ProjectPermission | null;
  onOpenProjectModal: () => void;
}

interface NavItem {
  id: SidebarSection;
  label: string;
  icon: React.ReactNode;
}

export const StorageSidebar: React.FC<StorageSidebarProps> = ({
  activeSection,
  onSectionChange,
  storageUsage,
  isCollapsed,
  onToggleCollapse,
  currentProject,
  currentProjectPermission,
  onOpenProjectModal,
}) => {
  const navItems: NavItem[] = [
    { id: 'recent', label: '최근 문서함', icon: <Clock className="w-5 h-5" /> },
    { id: 'starred', label: '중요 문서함', icon: <Star className="w-5 h-5" /> },
    { id: 'trash', label: '휴지통', icon: <Trash2 className="w-5 h-5" /> },
  ];

  // 스토리지 사용량 퍼센트 계산 (기본 15GB 제한 가정)
  const storageLimit = 15 * 1024 * 1024 * 1024; // 15GB in bytes
  const usedPercent = storageUsage
    ? Math.min((storageUsage.totalSize / storageLimit) * 100, 100)
    : 0;

  return (
    <div
      className={`h-full flex flex-col bg-gray-50 border-r border-gray-200 transition-all duration-300 flex-shrink-0 relative ${
        isCollapsed ? 'w-[72px]' : 'w-[240px]'
      }`}
    >
      {/* 접기/펼치기 버튼 */}
      <button
        onClick={onToggleCollapse}
        className="absolute -right-3 top-6 w-6 h-6 bg-white border border-gray-300 rounded-full shadow-md flex items-center justify-center hover:bg-gray-100 transition-colors z-30"
        title={isCollapsed ? '사이드바 펼치기' : '사이드바 접기'}
      >
        {isCollapsed ? (
          <ChevronRight className="w-4 h-4 text-gray-600" />
        ) : (
          <ChevronLeft className="w-4 h-4 text-gray-600" />
        )}
      </button>

      {/* 프로젝트/스토리지 선택 */}
      <div className={`px-3 py-2 ${isCollapsed ? 'hidden' : ''}`}>
        <button
          onClick={onOpenProjectModal}
          className="w-full flex items-center gap-2 px-3 py-2.5 rounded-lg bg-white border border-gray-200 hover:bg-gray-50 hover:border-gray-300 transition-all shadow-sm"
        >
          <HardDrive className={`w-5 h-5 ${currentProject ? 'text-blue-500' : 'text-blue-500'}`} />
          <div className="flex-1 text-left min-w-0">
            <p className="text-sm font-medium text-gray-900 truncate">
              {currentProject ? currentProject.name : '내 드라이브'}
            </p>
            {currentProjectPermission && (
              <p className="text-xs text-gray-500">
                {currentProjectPermission === 'OWNER'
                  ? '소유자'
                  : currentProjectPermission === 'EDITOR'
                  ? '편집자'
                  : '뷰어'}
              </p>
            )}
          </div>
          <ChevronDown className="w-4 h-4 text-gray-400" />
        </button>
      </div>

      {/* 접힌 상태에서 스토리지 아이콘 */}
      {isCollapsed && (
        <div className="px-3 py-2">
          <button
            onClick={onOpenProjectModal}
            className={`w-full flex justify-center p-3 rounded-xl transition-all ${
              currentProject ? 'bg-blue-50 text-blue-600' : 'bg-blue-50 text-blue-600'
            }`}
            title={currentProject ? currentProject.name : '내 드라이브'}
          >
            <HardDrive className="w-5 h-5" />
          </button>
        </div>
      )}

      {/* 네비게이션 메뉴 */}
      <nav className="flex-1 px-3 py-2 overflow-y-auto">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.id}>
              <button
                onClick={() => onSectionChange(item.id)}
                className={`w-full flex items-center gap-3 transition-all text-sm ${
                  isCollapsed ? 'justify-center p-3 rounded-lg' : 'px-3 py-2 rounded-lg'
                } ${
                  activeSection === item.id
                    ? 'bg-blue-100 text-blue-700 font-medium'
                    : 'text-gray-700 hover:bg-gray-100'
                }`}
                title={isCollapsed ? item.label : undefined}
              >
                <span className={activeSection === item.id ? 'text-blue-700' : 'text-gray-500'}>
                  {item.icon}
                </span>
                {!isCollapsed && <span>{item.label}</span>}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      {/* 스토리지 사용량 */}
      <div className={`p-4 border-t border-gray-200 ${isCollapsed ? 'hidden' : ''}`}>
        <div className="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 mb-2">
          <Cloud className="w-5 h-5 text-gray-500" />
          <span>저장용량</span>
        </div>

        <div className="px-3">
          <div className="w-full h-2 bg-gray-200 rounded-full overflow-hidden mb-2">
            <div
              className={`h-full rounded-full transition-all ${
                usedPercent > 90 ? 'bg-red-500' : usedPercent > 70 ? 'bg-yellow-500' : 'bg-blue-600'
              }`}
              style={{ width: `${Math.max(usedPercent, 1)}%` }}
            />
          </div>
          <p className="text-xs text-gray-500">
            15GB 중 {storageUsage ? formatFileSize(storageUsage.totalSize) : '0 B'} 사용
          </p>
        </div>
      </div>

      {/* 접힌 상태에서 스토리지 아이콘만 표시 */}
      {isCollapsed && (
        <div className="p-4 border-t border-gray-200 flex justify-center">
          <div
            className="relative"
            title={`15GB 중 ${storageUsage ? formatFileSize(storageUsage.totalSize) : '0 B'} 사용`}
          >
            <Cloud className="w-5 h-5 text-gray-500" />
            {usedPercent > 70 && (
              <div
                className={`absolute -top-1 -right-1 w-2 h-2 rounded-full ${
                  usedPercent > 90 ? 'bg-red-500' : 'bg-yellow-500'
                }`}
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
};
