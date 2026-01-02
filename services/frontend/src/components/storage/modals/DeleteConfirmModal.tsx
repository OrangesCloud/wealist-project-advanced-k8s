// src/components/storage/modals/DeleteConfirmModal.tsx

import React from 'react';
import { X, AlertTriangle, Folder, FileText } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import type { SelectedItem } from '../../../types/storage';

interface DeleteConfirmModalProps {
  items: SelectedItem[];
  isPermanent: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  items,
  isPermanent,
  onClose,
  onConfirm,
}) => {
  const { theme } = useTheme();

  const folderCount = items.filter((i) => i.type === 'folder').length;
  const fileCount = items.filter((i) => i.type === 'file').length;

  const getItemDescription = () => {
    const parts = [];
    if (folderCount > 0) parts.push(`폴더 ${folderCount}개`);
    if (fileCount > 0) parts.push(`파일 ${fileCount}개`);
    return parts.join(', ');
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-md ${theme.colors.card} rounded-xl shadow-2xl`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">
            {isPermanent ? '영구 삭제' : '삭제'}
          </h2>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-gray-100 transition"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 본문 */}
        <div className="p-4">
          {/* 경고 아이콘 */}
          <div className="flex justify-center mb-4">
            <div className={`p-3 rounded-full ${isPermanent ? 'bg-red-100' : 'bg-yellow-100'}`}>
              <AlertTriangle
                className={`w-8 h-8 ${isPermanent ? 'text-red-500' : 'text-yellow-500'}`}
              />
            </div>
          </div>

          {/* 메시지 */}
          <div className="text-center mb-4">
            <p className="text-gray-900 font-medium mb-2">
              {isPermanent
                ? `${getItemDescription()}을(를) 영구 삭제하시겠습니까?`
                : `${getItemDescription()}을(를) 휴지통으로 이동하시겠습니까?`}
            </p>
            <p className="text-sm text-gray-500">
              {isPermanent
                ? '이 작업은 되돌릴 수 없습니다.'
                : '휴지통에서 복원할 수 있습니다.'}
            </p>
          </div>

          {/* 선택된 항목 목록 (최대 5개) */}
          {items.length <= 5 && (
            <div className="bg-gray-50 rounded-lg p-3 mb-4 max-h-40 overflow-y-auto">
              {items.map((item) => (
                <div
                  key={`${item.type}-${item.id}`}
                  className="flex items-center gap-2 py-1"
                >
                  {item.type === 'folder' ? (
                    <Folder className="w-4 h-4 text-blue-500" />
                  ) : (
                    <FileText className="w-4 h-4 text-gray-500" />
                  )}
                  <span className="text-sm text-gray-700 truncate">{item.name}</span>
                </div>
              ))}
            </div>
          )}

          {/* 버튼 */}
          <div className="flex justify-end gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition"
            >
              취소
            </button>
            <button
              onClick={onConfirm}
              className={`px-4 py-2 text-white rounded-lg transition ${
                isPermanent
                  ? 'bg-red-600 hover:bg-red-700'
                  : 'bg-blue-600 hover:bg-blue-700'
              }`}
            >
              {isPermanent ? '영구 삭제' : '삭제'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
