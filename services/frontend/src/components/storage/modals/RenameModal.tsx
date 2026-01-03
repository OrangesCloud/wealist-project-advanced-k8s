// src/components/storage/modals/RenameModal.tsx

import React, { useState } from 'react';
import { X, Folder, FileText } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';
import type { SelectedItem } from '../../../types/storage';

interface RenameModalProps {
  item: SelectedItem;
  onClose: () => void;
  onRename: (newName: string) => void;
}

export const RenameModal: React.FC<RenameModalProps> = ({ item, onClose, onRename }) => {
  const { theme } = useTheme();
  const [newName, setNewName] = useState(item.name);
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!newName.trim()) {
      setError('이름을 입력해주세요.');
      return;
    }

    if (newName.length > 255) {
      setError('이름은 255자 이내로 입력해주세요.');
      return;
    }

    if (newName.trim() === item.name) {
      onClose();
      return;
    }

    onRename(newName.trim());
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-md ${theme.colors.card} rounded-xl shadow-2xl`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">이름 바꾸기</h2>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-gray-100 transition"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 본문 */}
        <form onSubmit={handleSubmit} className="p-4">
          {/* 아이템 미리보기 */}
          <div className="flex items-center gap-3 mb-4 p-3 bg-gray-50 rounded-lg">
            {item.type === 'folder' ? (
              <Folder className="w-8 h-8 text-blue-500" />
            ) : (
              <FileText className="w-8 h-8 text-gray-500" />
            )}
            <span className="text-sm text-gray-600 truncate">{item.name}</span>
          </div>

          {/* 새 이름 */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              새 이름
            </label>
            <input
              type="text"
              value={newName}
              onChange={(e) => {
                setNewName(e.target.value);
                setError('');
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition"
              placeholder="새 이름 입력"
              autoFocus
            />
            {error && <p className="mt-1 text-sm text-red-500">{error}</p>}
          </div>

          {/* 버튼 */}
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition"
            >
              취소
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              확인
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
