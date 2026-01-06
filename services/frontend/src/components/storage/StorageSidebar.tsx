// src/components/storage/StorageSidebar.tsx - Google Drive 스타일 사이드바

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
    { id: 'recent', label: '최근 항목', icon: <Clock className="w-5 h-5" /> },
    { id: 'starred', label: '중요 항목', icon: <Star className="w-5 h-5" /> },
    { id: 'trash', label: '휴지통', icon: <Trash2 className="w-5 h-5" /> },
  ];

  // 스토리지 사용량 퍼센트 계산 (기본 15GB 제한 가정)
  const storageLimit = 15 * 1024 * 1024 * 1024; // 15GB in bytes
  const usedPercent = storageUsage
    ? Math.min((storageUsage.totalSize / storageLimit) * 100, 100)
    : 0;

  return (
    <div
      className={`h-full flex flex-col bg-[#f8f9fa] border-r border-[#dadce0] transition-all duration-300 flex-shrink-0 relative ${
        isCollapsed ? 'w-[68px]' : 'w-[256px]'
      }`}
    >
      {/* 접기/펼치기 버튼 */}
      <button
        onClick={onToggleCollapse}
        className="absolute -right-3 top-4 w-6 h-6 bg-white border border-[#dadce0] rounded-full shadow-sm flex items-center justify-center hover:bg-[#f1f3f4] transition-colors z-30"
        title={isCollapsed ? '사이드바 펼치기' : '사이드바 접기'}
      >
        {isCollapsed ? (
          <ChevronRight className="w-4 h-4 text-[#5f6368]" />
        ) : (
          <ChevronLeft className="w-4 h-4 text-[#5f6368]" />
        )}
      </button>

      {/* 드라이브 선택 버튼 - Google Drive 스타일 */}
      <div className={`p-3 ${isCollapsed ? 'px-2' : ''}`}>
        <button
          onClick={onOpenProjectModal}
          className={`flex items-center gap-3 rounded-2xl transition-all ${
            isCollapsed
              ? 'w-full justify-center p-3 bg-white shadow-md hover:shadow-lg border border-[#dadce0]'
              : 'w-full px-4 py-3 bg-white shadow-md hover:shadow-lg border border-[#dadce0]'
          }`}
          title={currentProject ? currentProject.name : '내 드라이브'}
        >
          <HardDrive className="w-5 h-5 text-[#1967d2]" />
          {!isCollapsed && (
            <>
              <div className="flex-1 text-left min-w-0">
                <p className="text-sm font-medium text-[#3c4043] truncate">
                  {currentProject ? currentProject.name : '내 드라이브'}
                </p>
                {currentProjectPermission && (
                  <p className="text-xs text-[#5f6368]">
                    {currentProjectPermission === 'OWNER'
                      ? '소유자'
                      : currentProjectPermission === 'EDITOR'
                      ? '편집자'
                      : '뷰어'}
                  </p>
                )}
              </div>
              <ChevronDown className="w-4 h-4 text-[#5f6368]" />
            </>
          )}
        </button>
      </div>

      {/* 구분선 */}
      <div className={`mx-3 border-t border-[#e0e0e0] ${isCollapsed ? 'mx-2' : ''}`} />

      {/* 네비게이션 메뉴 */}
      <nav className={`flex-1 overflow-y-auto py-2 ${isCollapsed ? 'px-2' : 'px-3'}`}>
        <ul className="space-y-0.5">
          {navItems.map((item) => (
            <li key={item.id}>
              <button
                onClick={() => onSectionChange(item.id)}
                className={`w-full flex items-center gap-3 transition-all text-sm rounded-full ${
                  isCollapsed ? 'justify-center p-3' : 'px-6 py-2.5'
                } ${
                  activeSection === item.id
                    ? 'bg-[#c2e7ff] text-[#001d35] font-medium'
                    : 'text-[#3c4043] hover:bg-[#e8eaed]'
                }`}
                title={isCollapsed ? item.label : undefined}
              >
                <span className={activeSection === item.id ? 'text-[#001d35]' : 'text-[#5f6368]'}>
                  {item.icon}
                </span>
                {!isCollapsed && <span>{item.label}</span>}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      {/* 스토리지 사용량 - Google Drive 스타일 */}
      <div className={`border-t border-[#e0e0e0] ${isCollapsed ? 'p-2' : 'p-4'}`}>
        {!isCollapsed ? (
          <div className="space-y-3">
            <div className="flex items-center gap-3 text-sm text-[#5f6368]">
              <Cloud className="w-5 h-5" />
              <span>저장용량</span>
            </div>
            <div>
              <div className="w-full h-1 bg-[#e0e0e0] rounded-full overflow-hidden mb-2">
                <div
                  className={`h-full rounded-full transition-all ${
                    usedPercent > 90 ? 'bg-[#d93025]' : usedPercent > 70 ? 'bg-[#f9ab00]' : 'bg-[#1a73e8]'
                  }`}
                  style={{ width: `${Math.max(usedPercent, 1)}%` }}
                />
              </div>
              <p className="text-xs text-[#5f6368]">
                {storageUsage ? formatFileSize(storageUsage.totalSize) : '0 B'} / 15GB 사용 중
              </p>
            </div>
          </div>
        ) : (
          <div
            className="flex justify-center py-2"
            title={`${storageUsage ? formatFileSize(storageUsage.totalSize) : '0 B'} / 15GB 사용 중`}
          >
            <div className="relative">
              <Cloud className="w-5 h-5 text-[#5f6368]" />
              {usedPercent > 70 && (
                <div
                  className={`absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full ${
                    usedPercent > 90 ? 'bg-[#d93025]' : 'bg-[#f9ab00]'
                  }`}
                />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
