// src/components/storage/modals/FilePreviewModal.tsx

import React from 'react';
import {
  X,
  Download,
  ExternalLink,
  FileText,
  FileImage,
  FileVideo,
  FileAudio,
  File,
} from 'lucide-react';
import type { StorageFile } from '../../../types/storage';
import { formatFileSize, getFileCategory } from '../../../types/storage';

interface FilePreviewModalProps {
  file: StorageFile;
  onClose: () => void;
  onDownload: () => void;
}

export const FilePreviewModal: React.FC<FilePreviewModalProps> = ({
  file,
  onClose,
  onDownload,
}) => {
  const category = getFileCategory(file.extension);

  // 미리보기 가능한 파일 유형인지 확인
  // const isPreviewable = file.isImage || category === 'video' || category === 'audio' || file.extension === '.pdf';

  // 파일 아이콘 가져오기
  const getFileIcon = () => {
    const iconClass = 'w-24 h-24';
    switch (category) {
      case 'image':
        return <FileImage className={`${iconClass} text-green-500`} />;
      case 'video':
        return <FileVideo className={`${iconClass} text-red-500`} />;
      case 'audio':
        return <FileAudio className={`${iconClass} text-purple-500`} />;
      case 'document':
        return <FileText className={`${iconClass} text-blue-500`} />;
      default:
        return <File className={`${iconClass} text-gray-500`} />;
    }
  };

  // 미리보기 렌더링
  const renderPreview = () => {
    if (file.isImage) {
      return (
        <div className="flex-1 flex items-center justify-center bg-gray-900 p-4">
          <img
            src={file.fileUrl}
            alt={file.name}
            className="max-w-full max-h-[70vh] object-contain rounded-lg shadow-2xl"
          />
        </div>
      );
    }

    if (category === 'video') {
      return (
        <div className="flex-1 flex items-center justify-center bg-gray-900 p-4">
          <video
            src={file.fileUrl}
            controls
            className="max-w-full max-h-[70vh] rounded-lg shadow-2xl"
          >
            비디오를 재생할 수 없습니다.
          </video>
        </div>
      );
    }

    if (category === 'audio') {
      return (
        <div className="flex-1 flex flex-col items-center justify-center bg-gray-100 p-8">
          <FileAudio className="w-32 h-32 text-purple-500 mb-8" />
          <audio src={file.fileUrl} controls className="w-full max-w-md">
            오디오를 재생할 수 없습니다.
          </audio>
        </div>
      );
    }

    if (file.extension === '.pdf') {
      return (
        <div className="flex-1 bg-gray-100">
          <iframe src={file.fileUrl} title={file.name} className="w-full h-full border-0" />
        </div>
      );
    }

    // 미리보기 불가능한 파일
    return (
      <div className="flex-1 flex flex-col items-center justify-center bg-gray-50 p-8">
        {getFileIcon()}
        <p className="mt-6 text-lg font-medium text-gray-900">{file.name}</p>
        <p className="mt-2 text-gray-500">미리보기를 사용할 수 없습니다.</p>
        <button
          onClick={onDownload}
          className="mt-6 flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
        >
          <Download className="w-5 h-5" />
          다운로드
        </button>
      </div>
    );
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex flex-col z-50">
      {/* 헤더 */}
      <div className="flex items-center justify-between px-6 py-4 bg-black/40">
        <div className="flex items-center gap-4">
          <h2 className="text-lg font-semibold text-white truncate max-w-md">{file.name}</h2>
          <span className="text-sm text-gray-400">{formatFileSize(file.fileSize)}</span>
        </div>
        <div className="flex items-center gap-2">
          {file.fileUrl && (
            <a
              href={file.fileUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-white hover:bg-white/10 rounded-lg transition"
              title="새 탭에서 열기"
            >
              <ExternalLink className="w-5 h-5" />
            </a>
          )}
          <button
            onClick={onDownload}
            className="p-2 text-white hover:bg-white/10 rounded-lg transition"
            title="다운로드"
          >
            <Download className="w-5 h-5" />
          </button>
          <button
            onClick={onClose}
            className="p-2 text-white hover:bg-white/10 rounded-lg transition ml-2"
            title="닫기"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* 미리보기 영역 */}
      {renderPreview()}

      {/* 파일 정보 */}
      <div className="px-6 py-3 bg-black/40 flex items-center gap-6 text-sm text-gray-400">
        <span>크기: {formatFileSize(file.fileSize)}</span>
        <span>유형: {file.contentType}</span>
        <span>
          수정됨:{' '}
          {new Date(file.updatedAt).toLocaleDateString('ko-KR', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })}
        </span>
      </div>
    </div>
  );
};
