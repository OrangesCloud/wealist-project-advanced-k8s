// src/components/storage/modals/UploadProgressModal.tsx

import React from 'react';
import { X, Upload, CheckCircle, AlertCircle } from 'lucide-react';
import { useTheme } from '../../../contexts/ThemeContext';

interface UploadItem {
  fileName: string;
  progress: number;
  error?: string;
}

interface UploadProgressModalProps {
  uploads: UploadItem[];
  onClose: () => void;
}

export const UploadProgressModal: React.FC<UploadProgressModalProps> = ({
  uploads,
  onClose,
}) => {
  const { theme } = useTheme();

  const completedCount = uploads.filter((u) => u.progress === 100).length;
  const errorCount = uploads.filter((u) => u.error).length;
  const isAllComplete = completedCount === uploads.length && errorCount === 0;

  return (
    <div className="fixed bottom-6 right-6 z-50">
      <div className={`w-96 ${theme.colors.card} rounded-xl shadow-2xl border border-gray-200 overflow-hidden`}>
        {/* 헤더 */}
        <div className="flex items-center justify-between px-4 py-3 bg-gray-50 border-b border-gray-200">
          <div className="flex items-center gap-2">
            <Upload className="w-5 h-5 text-blue-600" />
            <span className="font-medium text-gray-900">
              {isAllComplete
                ? '업로드 완료'
                : `업로드 중... (${completedCount}/${uploads.length})`}
            </span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-gray-200 transition"
          >
            <X className="w-4 h-4 text-gray-500" />
          </button>
        </div>

        {/* 업로드 목록 */}
        <div className="max-h-60 overflow-y-auto">
          {uploads.map((upload, index) => (
            <div key={index} className="px-4 py-3 border-b border-gray-100 last:border-0">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm text-gray-700 truncate max-w-[200px]">
                  {upload.fileName}
                </span>
                {upload.error ? (
                  <AlertCircle className="w-4 h-4 text-red-500" />
                ) : upload.progress === 100 ? (
                  <CheckCircle className="w-4 h-4 text-green-500" />
                ) : (
                  <span className="text-xs text-gray-500">{upload.progress}%</span>
                )}
              </div>

              {/* 프로그레스 바 */}
              {!upload.error && upload.progress < 100 && (
                <div className="w-full h-1.5 bg-gray-200 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 rounded-full transition-all duration-300"
                    style={{ width: `${upload.progress}%` }}
                  />
                </div>
              )}

              {/* 에러 메시지 */}
              {upload.error && (
                <p className="text-xs text-red-500 mt-1">{upload.error}</p>
              )}
            </div>
          ))}
        </div>

        {/* 푸터 */}
        {isAllComplete && (
          <div className="px-4 py-3 bg-green-50 border-t border-green-100">
            <div className="flex items-center gap-2 text-green-700">
              <CheckCircle className="w-4 h-4" />
              <span className="text-sm">모든 파일이 업로드되었습니다.</span>
            </div>
          </div>
        )}

        {errorCount > 0 && (
          <div className="px-4 py-3 bg-red-50 border-t border-red-100">
            <div className="flex items-center gap-2 text-red-700">
              <AlertCircle className="w-4 h-4" />
              <span className="text-sm">{errorCount}개 파일 업로드 실패</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
