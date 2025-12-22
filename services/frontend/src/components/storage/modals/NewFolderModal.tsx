// src/components/storage/modals/NewFolderModal.tsx

import React, { useState } from 'react';
import { X, Folder } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';

interface NewFolderModalProps {
  onClose: () => void;
  onCreate: (name: string, color?: string) => void;
}

const FOLDER_COLORS = [
  '#60A5FA', // blue
  '#34D399', // green
  '#FBBF24', // yellow
  '#F87171', // red
  '#A78BFA', // purple
  '#F472B6', // pink
  '#FB923C', // orange
  '#6B7280', // gray
];

export const NewFolderModal: React.FC<NewFolderModalProps> = ({ onClose, onCreate }) => {
  const { theme } = useTheme();
  const [folderName, setFolderName] = useState('새 폴더');
  const [selectedColor, setSelectedColor] = useState(FOLDER_COLORS[0]);
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!folderName.trim()) {
      setError('폴더 이름을 입력해주세요.');
      return;
    }

    if (folderName.length > 255) {
      setError('폴더 이름은 255자 이내로 입력해주세요.');
      return;
    }

    onCreate(folderName.trim(), selectedColor);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className={`w-full max-w-md ${theme.colors.card} rounded-xl shadow-2xl`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">새 폴더</h2>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-gray-100 transition"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* 본문 */}
        <form onSubmit={handleSubmit} className="p-4">
          {/* 폴더 미리보기 */}
          <div className="flex justify-center mb-6">
            <Folder
              className="w-20 h-20"
              style={{ color: selectedColor }}
            />
          </div>

          {/* 폴더 이름 */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              폴더 이름
            </label>
            <input
              type="text"
              value={folderName}
              onChange={(e) => {
                setFolderName(e.target.value);
                setError('');
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition"
              placeholder="폴더 이름 입력"
              autoFocus
            />
            {error && <p className="mt-1 text-sm text-red-500">{error}</p>}
          </div>

          {/* 색상 선택 */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              폴더 색상
            </label>
            <div className="flex gap-2 flex-wrap">
              {FOLDER_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  onClick={() => setSelectedColor(color)}
                  className={`w-8 h-8 rounded-full transition-all ${
                    selectedColor === color
                      ? 'ring-2 ring-offset-2 ring-blue-500 scale-110'
                      : 'hover:scale-110'
                  }`}
                  style={{ backgroundColor: color }}
                />
              ))}
            </div>
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
              만들기
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
