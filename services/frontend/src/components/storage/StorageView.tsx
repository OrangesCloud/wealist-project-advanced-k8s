// src/components/storage/StorageView.tsx - Google Drive 스타일 파일/폴더 뷰

import React, { useCallback, useState, useRef, useEffect } from 'react';
import {
  Folder,
  FileText,
  FileVideo,
  FileAudio,
  FileArchive,
  File,
  MoreVertical,
  Download,
  Edit2,
  Share2,
  Trash2,
  RotateCcw,
  Star,
  Eye,
  FolderOpen,
  FolderPlus,
  ImagePlus,
  Users,
  FileSpreadsheet,
  Presentation,
  Image,
} from 'lucide-react';
import {
  StorageFolder,
  StorageFile,
  ViewMode,
  SelectedItem,
  SharedItem,
  ProjectPermission,
  getFileCategory,
  formatFileSize,
} from '../../types/storage';

interface StorageViewProps {
  viewMode: ViewMode;
  folders: StorageFolder[];
  files: StorageFile[];
  sharedItems?: SharedItem[];
  selectedItems: SelectedItem[];
  onSelectItem: (items: SelectedItem[]) => void;
  onFolderOpen: (folder: StorageFolder) => void;
  onFileDownload: (file: StorageFile) => void;
  onFilePreview: (file: StorageFile) => void;
  onRename: (item: SelectedItem) => void;
  onShare: (item: SelectedItem) => void;
  onDelete: () => void;
  onRestore?: () => void;
  onNewFolder: () => void;
  onUpload: () => void;
  isTrash: boolean;
  isEmpty: boolean;
  activeSection: string;
  // 권한 관련 props
  projectPermission?: ProjectPermission | null;
}

// Google Drive 스타일 파일 아이콘
const getFileIcon = (file: StorageFile, size: 'sm' | 'lg' = 'lg') => {
  const category = getFileCategory(file.extension);
  const iconSize = size === 'lg' ? 'w-12 h-12' : 'w-6 h-6';

  // Google Drive 색상
  const colors = {
    doc: '#4285f4', // Google Docs blue
    sheet: '#0f9d58', // Google Sheets green
    slide: '#f4b400', // Google Slides yellow
    image: '#ea4335', // Red
    video: '#ea4335', // Red
    audio: '#9334ea', // Purple
    pdf: '#ea4335', // Red
    archive: '#f4b400', // Yellow
    default: '#5f6368', // Gray
  };

  switch (category) {
    case 'image':
      return <Image className={iconSize} style={{ color: colors.image }} />;
    case 'video':
      return <FileVideo className={iconSize} style={{ color: colors.video }} />;
    case 'audio':
      return <FileAudio className={iconSize} style={{ color: colors.audio }} />;
    case 'document':
      if (file.extension === '.pdf') {
        return <FileText className={iconSize} style={{ color: colors.pdf }} />;
      }
      if (['.doc', '.docx'].includes(file.extension)) {
        return <FileText className={iconSize} style={{ color: colors.doc }} />;
      }
      if (['.xls', '.xlsx', '.csv'].includes(file.extension)) {
        return <FileSpreadsheet className={iconSize} style={{ color: colors.sheet }} />;
      }
      if (['.ppt', '.pptx'].includes(file.extension)) {
        return <Presentation className={iconSize} style={{ color: colors.slide }} />;
      }
      return <FileText className={iconSize} style={{ color: colors.doc }} />;
    case 'archive':
      return <FileArchive className={iconSize} style={{ color: colors.archive }} />;
    default:
      return <File className={iconSize} style={{ color: colors.default }} />;
  }
};

// 상대 시간 포맷
const formatRelativeTime = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  const minutes = Math.floor(diff / (1000 * 60));
  const hours = Math.floor(diff / (1000 * 60 * 60));
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));

  if (minutes < 1) return '방금';
  if (minutes < 60) return `${minutes}분 전`;
  if (hours < 24) return `${hours}시간 전`;
  if (days < 7) return `${days}일 전`;

  return date.toLocaleDateString('ko-KR', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
};

export const StorageView: React.FC<StorageViewProps> = ({
  viewMode,
  folders,
  files,
  // sharedItems,
  selectedItems,
  onSelectItem,
  onFolderOpen,
  onFileDownload,
  onFilePreview,
  onRename,
  onShare,
  onDelete,
  onRestore,
  onNewFolder,
  onUpload,
  isTrash,
  isEmpty,
  activeSection,
  projectPermission,
}) => {
  // 권한에 따른 편집 가능 여부
  const canEdit = !projectPermission || projectPermission === 'OWNER' || projectPermission === 'EDITOR';
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    item: SelectedItem;
  } | null>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);

  // 외부 클릭 시 컨텍스트 메뉴 닫기
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 항목 선택 핸들러
  const handleItemSelect = useCallback(
    (item: SelectedItem, isCtrlKey: boolean) => {
      if (isCtrlKey) {
        const isSelected = selectedItems.some((s) => s.id === item.id && s.type === item.type);
        if (isSelected) {
          onSelectItem(selectedItems.filter((s) => !(s.id === item.id && s.type === item.type)));
        } else {
          onSelectItem([...selectedItems, item]);
        }
      } else {
        onSelectItem([item]);
      }
    },
    [selectedItems, onSelectItem],
  );

  // 컨텍스트 메뉴 열기
  const handleContextMenu = useCallback((e: React.MouseEvent, item: SelectedItem) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      item,
    });
  }, []);

  // 더블클릭 핸들러
  const handleDoubleClick = useCallback(
    (item: SelectedItem) => {
      if (item.type === 'folder') {
        onFolderOpen(item.data as StorageFolder);
      } else {
        onFilePreview(item.data as StorageFile);
      }
    },
    [onFolderOpen, onFilePreview],
  );

  // 선택 여부 확인
  const isSelected = useCallback(
    (id: string, type: 'file' | 'folder') => {
      return selectedItems.some((s) => s.id === id && s.type === type);
    },
    [selectedItems],
  );

  // 빈 상태 - Google Drive 스타일
  if (isEmpty) {
    const emptyMessages: Record<
      string,
      { title: string; description: string; icon: React.ReactNode }
    > = {
      'my-drive': {
        title: '드라이브로 파일을 드래그하거나 "새로 만들기" 버튼을 사용하세요',
        description: '',
        icon: (
          <div className="w-40 h-40 mb-6">
            <svg viewBox="0 0 120 120" className="w-full h-full">
              <path d="M45 35h30l15 15v45H45V35z" fill="#e8eaed" stroke="#dadce0" strokeWidth="2" />
              <path d="M75 35v15h15" fill="none" stroke="#dadce0" strokeWidth="2" />
              <rect x="55" y="55" width="20" height="2" fill="#bdc1c6" />
              <rect x="55" y="62" width="15" height="2" fill="#bdc1c6" />
              <rect x="55" y="69" width="18" height="2" fill="#bdc1c6" />
            </svg>
          </div>
        ),
      },
      shared: {
        title: '나와 공유된 파일 없음',
        description: '다른 사용자가 파일이나 폴더를 공유하면 여기에 표시됩니다.',
        icon: <Users className="w-24 h-24 text-[#dadce0] mb-6" />,
      },
      recent: {
        title: '최근 파일 없음',
        description: '최근에 열거나 수정한 파일이 여기에 표시됩니다.',
        icon: <FileText className="w-24 h-24 text-[#dadce0] mb-6" />,
      },
      starred: {
        title: '중요 표시된 파일 없음',
        description: '파일이나 폴더에 별표를 추가하면 여기에 표시됩니다.',
        icon: <Star className="w-24 h-24 text-[#dadce0] mb-6" />,
      },
      trash: {
        title: '휴지통이 비어 있습니다',
        description: '',
        icon: <Trash2 className="w-24 h-24 text-[#dadce0] mb-6" />,
      },
    };

    const emptyState = emptyMessages[activeSection] || emptyMessages['my-drive'];

    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 bg-white">
        {emptyState.icon}
        <h3 className="text-base text-[#3c4043] text-center max-w-md">{emptyState.title}</h3>
        {emptyState.description && (
          <p className="mt-2 text-sm text-[#5f6368] text-center max-w-md">
            {emptyState.description}
          </p>
        )}
      </div>
    );
  }

  // 리스트 뷰 - Google Drive 스타일 테이블
  if (viewMode === 'list') {
    return (
      <div className="flex-1 overflow-auto bg-white">
        {/* 테이블 헤더 */}
        <div className="sticky top-0 bg-white border-b border-[#e0e0e0] px-6">
          <div className="flex items-center py-2 text-xs font-medium text-[#5f6368]">
            <div className="flex-1 min-w-0">이름</div>
            <div className="w-32 text-right">소유자</div>
            <div className="w-40 text-right">마지막 수정</div>
            <div className="w-24 text-right">파일 크기</div>
            <div className="w-10"></div>
          </div>
        </div>

        {/* 폴더 목록 */}
        {folders.map((folder) => {
          const selectedItem: SelectedItem = {
            type: 'folder',
            id: folder.id,
            name: folder.name,
            data: folder,
          };
          const selected = isSelected(folder.id, 'folder');

          return (
            <div
              key={folder.id}
              onClick={(e) => handleItemSelect(selectedItem, e.ctrlKey || e.metaKey)}
              onDoubleClick={() => handleDoubleClick(selectedItem)}
              onContextMenu={(e) => handleContextMenu(e, selectedItem)}
              className={`flex items-center px-6 py-2 cursor-pointer border-b border-[#f1f3f4] transition-colors ${
                selected ? 'bg-[#e8f0fe]' : 'hover:bg-[#f1f3f4]'
              }`}
            >
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <Folder
                  className="w-6 h-6 flex-shrink-0"
                  style={{ color: folder.color || '#5f6368' }}
                  fill={folder.color || '#5f6368'}
                />
                <span className="text-sm text-[#3c4043] truncate">{folder.name}</span>
              </div>
              <div className="w-32 text-sm text-[#5f6368] text-right">나</div>
              <div className="w-40 text-sm text-[#5f6368] text-right">
                {formatRelativeTime(folder.updatedAt)}
              </div>
              <div className="w-24 text-sm text-[#5f6368] text-right">—</div>
              <div className="w-10 flex justify-end">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleContextMenu(e, selectedItem);
                  }}
                  className="p-1.5 rounded-full hover:bg-[#dadce0] opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <MoreVertical className="w-5 h-5 text-[#5f6368]" />
                </button>
              </div>
            </div>
          );
        })}

        {/* 파일 목록 */}
        {files.map((file) => {
          const selectedItem: SelectedItem = {
            type: 'file',
            id: file.id,
            name: file.name,
            data: file,
          };
          const selected = isSelected(file.id, 'file');

          return (
            <div
              key={file.id}
              onClick={(e) => handleItemSelect(selectedItem, e.ctrlKey || e.metaKey)}
              onDoubleClick={() => handleDoubleClick(selectedItem)}
              onContextMenu={(e) => handleContextMenu(e, selectedItem)}
              className={`group flex items-center px-6 py-2 cursor-pointer border-b border-[#f1f3f4] transition-colors ${
                selected ? 'bg-[#e8f0fe]' : 'hover:bg-[#f1f3f4]'
              }`}
            >
              <div className="flex items-center gap-3 flex-1 min-w-0">
                {getFileIcon(file, 'sm')}
                <span className="text-sm text-[#3c4043] truncate">{file.name}</span>
              </div>
              <div className="w-32 text-sm text-[#5f6368] text-right">나</div>
              <div className="w-40 text-sm text-[#5f6368] text-right">
                {formatRelativeTime(file.updatedAt)}
              </div>
              <div className="w-24 text-sm text-[#5f6368] text-right">
                {formatFileSize(file.fileSize)}
              </div>
              <div className="w-10 flex justify-end">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleContextMenu(e, selectedItem);
                  }}
                  className="p-1.5 rounded-full hover:bg-[#dadce0] opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <MoreVertical className="w-5 h-5 text-[#5f6368]" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    );
  }

  // 그리드 뷰 - Google Drive 스타일 카드
  return (
    <div className="flex-1 overflow-auto p-4 bg-white">
      {/* 폴더 섹션 */}
      {folders.length > 0 && (
        <div className="mb-6">
          <h3 className="text-sm font-medium text-[#5f6368] px-2 mb-2">폴더</h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
            {folders.map((folder) => {
              const selectedItem: SelectedItem = {
                type: 'folder',
                id: folder.id,
                name: folder.name,
                data: folder,
              };
              const selected = isSelected(folder.id, 'folder');

              return (
                <div
                  key={folder.id}
                  onClick={(e) => handleItemSelect(selectedItem, e.ctrlKey || e.metaKey)}
                  onDoubleClick={() => handleDoubleClick(selectedItem)}
                  onContextMenu={(e) => handleContextMenu(e, selectedItem)}
                  className={`group flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-all border ${
                    selected
                      ? 'bg-[#e8f0fe] border-[#1967d2]'
                      : 'bg-[#f8f9fa] border-transparent hover:bg-[#e8eaed]'
                  }`}
                >
                  <Folder
                    className="w-6 h-6 flex-shrink-0"
                    style={{ color: folder.color || '#5f6368' }}
                    fill={folder.color || '#5f6368'}
                  />
                  <span className="text-sm text-[#3c4043] truncate flex-1">{folder.name}</span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleContextMenu(e, selectedItem);
                    }}
                    className="p-1 rounded-full hover:bg-[#dadce0] opacity-0 group-hover:opacity-100"
                  >
                    <MoreVertical className="w-4 h-4 text-[#5f6368]" />
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* 파일 섹션 */}
      {files.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-[#5f6368] px-2 mb-2">파일</h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {files.map((file) => {
              const selectedItem: SelectedItem = {
                type: 'file',
                id: file.id,
                name: file.name,
                data: file,
              };
              const selected = isSelected(file.id, 'file');

              return (
                <div
                  key={file.id}
                  onClick={(e) => handleItemSelect(selectedItem, e.ctrlKey || e.metaKey)}
                  onDoubleClick={() => handleDoubleClick(selectedItem)}
                  onContextMenu={(e) => handleContextMenu(e, selectedItem)}
                  className={`group rounded-xl cursor-pointer transition-all border overflow-hidden ${
                    selected
                      ? 'border-[#1967d2] shadow-md'
                      : 'border-[#dadce0] hover:border-[#1967d2] hover:shadow-md'
                  }`}
                >
                  {/* 썸네일 영역 */}
                  <div className="aspect-[4/3] bg-[#f8f9fa] flex items-center justify-center relative">
                    {file.isImage && file.fileUrl ? (
                      <img
                        src={file.fileUrl}
                        alt={file.name}
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="flex items-center justify-center">
                        {getFileIcon(file, 'lg')}
                      </div>
                    )}
                    {/* 호버 시 액션 버튼 */}
                    <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleContextMenu(e, selectedItem);
                        }}
                        className="p-1.5 bg-white rounded-full shadow-md hover:bg-gray-100"
                      >
                        <MoreVertical className="w-4 h-4 text-[#5f6368]" />
                      </button>
                    </div>
                  </div>
                  {/* 파일 정보 */}
                  <div className="p-3 bg-white">
                    <p className="text-sm text-[#3c4043] truncate font-medium">{file.name}</p>
                    <p className="text-xs text-[#5f6368] mt-0.5">
                      {formatRelativeTime(file.updatedAt)}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* 컨텍스트 메뉴 - Google Drive 스타일 */}
      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="fixed bg-white rounded-lg shadow-2xl border border-[#dadce0] py-2 z-50 min-w-[200px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          {contextMenu.item.type === 'file' && (
            <>
              <button
                onClick={() => {
                  onFilePreview(contextMenu.item.data as StorageFile);
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <Eye className="w-5 h-5 text-[#5f6368]" />
                미리보기
              </button>
              <button
                onClick={() => {
                  onFileDownload(contextMenu.item.data as StorageFile);
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <Download className="w-5 h-5 text-[#5f6368]" />
                다운로드
              </button>
              <div className="h-px bg-[#e0e0e0] my-1" />
            </>
          )}

          {contextMenu.item.type === 'folder' && !isTrash && (
            <>
              <button
                onClick={() => {
                  onFolderOpen(contextMenu.item.data as StorageFolder);
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <FolderOpen className="w-5 h-5 text-[#5f6368]" />
                열기
              </button>
              <div className="h-px bg-[#e0e0e0] my-1" />
            </>
          )}

          {!isTrash && canEdit && (
            <>
              <button
                onClick={() => {
                  onNewFolder();
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <FolderPlus className="w-5 h-5 text-[#5f6368]" />
                새 폴더
              </button>
              <button
                onClick={() => {
                  onUpload();
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <ImagePlus className="w-5 h-5 text-[#5f6368]" />
                파일 추가
              </button>
              <div className="h-px bg-[#e0e0e0] my-1" />
            </>
          )}

          {!isTrash && (
            <>
              <button
                onClick={() => {
                  onShare(contextMenu.item);
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <Share2 className="w-5 h-5 text-[#5f6368]" />
                공유
              </button>
              {canEdit && (
                <>
                  <div className="h-px bg-[#e0e0e0] my-1" />
                  <button
                    onClick={() => {
                      onRename(contextMenu.item);
                      setContextMenu(null);
                    }}
                    className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
                  >
                    <Edit2 className="w-5 h-5 text-[#5f6368]" />
                    이름 바꾸기
                  </button>
                </>
              )}
            </>
          )}

          {isTrash && onRestore && (
            <button
              onClick={() => {
                onSelectItem([contextMenu.item]);
                onRestore();
                setContextMenu(null);
              }}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#1967d2] hover:bg-[#e8f0fe]"
            >
              <RotateCcw className="w-5 h-5" />
              복원
            </button>
          )}

          {canEdit && (
            <>
              <div className="h-px bg-[#e0e0e0] my-1" />
              <button
                onClick={() => {
                  onSelectItem([contextMenu.item]);
                  onDelete();
                  setContextMenu(null);
                }}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-[#3c4043] hover:bg-[#f1f3f4]"
              >
                <Trash2 className="w-5 h-5 text-[#5f6368]" />
                {isTrash ? '영구 삭제' : '삭제'}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
};
