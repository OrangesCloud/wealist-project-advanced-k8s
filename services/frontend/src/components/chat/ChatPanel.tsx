// src/components/chat/ChatPanel.tsx

import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { ChevronLeft, X, Image as ImageIcon, Loader2, ZoomIn } from 'lucide-react';
import { useChatWebSocket } from '../../hooks/useChatWebsocket';
import { getMessages, updateLastRead, getChat, generateChatPresignedURL, uploadChatFileToS3 } from '../../api/chatService';
import { getWorkspaceMembers } from '../../api/userService';
import type { Message } from '../../types/chat';
import type { WorkspaceMemberResponse } from '../../types/user';

// 이미지 메시지 컴포넌트 (로딩 상태 + 썸네일)
const ChatImage: React.FC<{
  src: string;
  alt: string;
  isUploading?: boolean;
  onClickView: (src: string) => void;
}> = ({ src, alt, isUploading, onClickView }) => {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  return (
    <div className="relative group">
      {/* 로딩/업로드 중 표시 */}
      {(isLoading || isUploading) && !hasError && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100 rounded-lg min-h-[60px] min-w-[80px]">
          <div className="flex flex-col items-center gap-1">
            <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
            <span className="text-xs text-gray-500">{isUploading ? '업로드 중...' : '로딩...'}</span>
          </div>
        </div>
      )}
      {/* 에러 표시 */}
      {hasError && (
        <div className="flex items-center justify-center bg-gray-100 rounded-lg p-4 min-h-[60px]">
          <span className="text-xs text-gray-500">이미지를 불러올 수 없습니다</span>
        </div>
      )}
      {/* 실제 이미지 */}
      {!hasError && (
        <img
          src={src}
          alt={alt}
          className={`max-w-full rounded-lg cursor-pointer hover:opacity-90 transition ${isLoading ? 'invisible h-0' : ''}`}
          style={{ maxHeight: '150px', maxWidth: '200px', objectFit: 'cover' }}
          onLoad={() => setIsLoading(false)}
          onError={() => {
            setIsLoading(false);
            setHasError(true);
          }}
          onClick={() => onClickView(src)}
        />
      )}
      {/* 확대 아이콘 */}
      {!isLoading && !hasError && (
        <button
          onClick={() => onClickView(src)}
          className="absolute bottom-1 right-1 p-1 bg-black/50 rounded opacity-0 group-hover:opacity-100 transition"
        >
          <ZoomIn className="w-3 h-3 text-white" />
        </button>
      )}
    </div>
  );
};

// 이미지 모달 뷰어
const ImageModal: React.FC<{
  src: string | null;
  onClose: () => void;
}> = ({ src, onClose }) => {
  if (!src) return null;

  return (
    <div
      className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <button
        onClick={onClose}
        className="absolute top-4 right-4 p-2 bg-white/20 hover:bg-white/30 rounded-full transition"
      >
        <X className="w-6 h-6 text-white" />
      </button>
      <img
        src={src}
        alt="확대 이미지"
        className="max-w-full max-h-full object-contain rounded-lg"
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
};

interface ChatPanelProps {
  chatId: string;
  onClose: () => void;
  onBack?: () => void;
}

export const ChatPanel: React.FC<ChatPanelProps> = ({ chatId, onClose, onBack }) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [members, setMembers] = useState<WorkspaceMemberResponse[]>([]);
  const [workspaceId, setWorkspaceId] = useState<string>('');
  const [pastedImage, setPastedImage] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [modalImage, setModalImage] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // 현재 사용자 ID
  const currentUserId = localStorage.getItem('userId');

  // 🔥 userId -> nickName 매핑 (워크스페이스 멤버 정보에서)
  const userNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    members.forEach((m) => {
      map[m.userId] = m.nickName || m.userEmail || 'Unknown';
    });
    return map;
  }, [members]);

  // WebSocket 연결
  const {
    sendMessage,
    sendFileMessage,
    sendTyping,
    // isConnected
  } = useChatWebSocket({
    chatId,
    onMessage: (event) => {
      console.log('🔊 [ChatPanel] 이벤트 수신:', event);

      if (event.type === 'MESSAGE_RECEIVED') {
        // 🔥 isMine 계산하여 추가
        // 백엔드에서 message 필드 안에 데이터를 보냄
        const messageData = event.message || event.payload || event;
        const newMessage: Message = {
          messageId: messageData.messageId,
          chatId: messageData.chatId,
          userId: messageData.userId,
          userName: messageData.userName,
          content: messageData.content,
          messageType: messageData.messageType,
          fileUrl: messageData.fileUrl,
          fileName: messageData.fileName,
          fileSize: messageData.fileSize,
          createdAt: messageData.createdAt,
          updatedAt: messageData.createdAt,
          isMine: messageData.userId === currentUserId,
        };
        // 🔥 중복 방지: 이미 존재하는 메시지인지 확인 (ID 또는 optimistic 메시지)
        setMessages((prev) => {
          // 동일 messageId 중복 체크
          if (prev.some((m) => m.messageId === newMessage.messageId)) {
            console.log('⚠️ [ChatPanel] 중복 메시지 무시:', newMessage.messageId);
            return prev;
          }
          // 🔥 Optimistic UI 메시지 대체: 내 메시지이고 같은 내용이면 temp 메시지 교체
          if (newMessage.isMine) {
            const tempIndex = prev.findIndex(
              (m) =>
                m.messageId.startsWith('temp-') &&
                m.content === newMessage.content &&
                m.userId === newMessage.userId,
            );
            if (tempIndex !== -1) {
              console.log(
                '✅ [ChatPanel] Optimistic 메시지 대체:',
                prev[tempIndex].messageId,
                '→',
                newMessage.messageId,
              );
              const updated = [...prev];
              updated[tempIndex] = newMessage;
              return updated;
            }
          }
          return [...prev, newMessage];
        });
      }

      if (event.type === 'USER_TYPING') {
        console.log('⌨️ User typing:', event.userId);
      }
    },
  });

  // 메시지 로드 및 읽음 처리
  useEffect(() => {
    const loadMessages = async () => {
      setIsLoading(true);
      try {
        // 🔥 채팅방 정보 + 메시지 동시 로드
        const [chatInfo, msgs] = await Promise.all([getChat(chatId), getMessages(chatId)]);

        setMessages(msgs);

        // 🔥 워크스페이스 멤버 정보 로드 (userName 조회용)
        if (chatInfo.workspaceId) {
          setWorkspaceId(chatInfo.workspaceId);
          const workspaceMembers = await getWorkspaceMembers(chatInfo.workspaceId);
          setMembers(workspaceMembers);
        }

        // 🔥 채팅방 진입 시 lastReadAt 업데이트 (읽음 처리)
        await updateLastRead(chatId);
        console.log('✅ [ChatPanel] lastReadAt 업데이트 완료');
      } catch (error) {
        console.error('Failed to load messages:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadMessages();
  }, [chatId]);

  // 자동 스크롤
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // 이미지 붙여넣기 핸들러
  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      for (const item of items) {
        if (item.type.startsWith('image/')) {
          e.preventDefault();
          const file = item.getAsFile();
          if (file) {
            setPastedImage(file);
            setImagePreview(URL.createObjectURL(file));
          }
          break;
        }
      }
    },
    [],
  );

  // 이미지 미리보기 제거
  const handleRemoveImage = useCallback(() => {
    if (imagePreview) {
      URL.revokeObjectURL(imagePreview);
    }
    setPastedImage(null);
    setImagePreview(null);
  }, [imagePreview]);

  // 이미지 업로드 및 전송
  const handleSendImage = useCallback(async () => {
    if (!pastedImage || !workspaceId || isUploading) return;

    setIsUploading(true);
    try {
      // 🔥 chat-service를 통해 S3 업로드 URL 생성
      const uploadUrlResponse = await generateChatPresignedURL({
        workspaceId,
        fileName: pastedImage.name,
        contentType: pastedImage.type,
        fileSize: pastedImage.size,
      });

      // 🔥 S3에 직접 업로드 (chat-service의 uploadChatFileToS3 사용)
      await uploadChatFileToS3(uploadUrlResponse.uploadUrl, pastedImage);

      // 🔥 chat-service가 제공하는 downloadUrl 사용
      const fileUrl = uploadUrlResponse.downloadUrl;

      // WebSocket으로 이미지 메시지 전송
      const success = sendFileMessage('', {
        messageType: 'IMAGE',
        fileUrl,
        fileName: pastedImage.name,
        fileSize: pastedImage.size,
      });

      if (success) {
        // Optimistic UI
        const optimisticMessage: Message = {
          messageId: `temp-${Date.now()}`,
          chatId,
          userId: currentUserId || '',
          userName: '',
          content: '',
          messageType: 'IMAGE',
          fileUrl,
          fileName: pastedImage.name,
          fileSize: pastedImage.size,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          isMine: true,
        };
        setMessages((prev) => [...prev, optimisticMessage]);
      }

      handleRemoveImage();
    } catch (error) {
      console.error('이미지 업로드 실패:', error);
      alert('이미지 업로드에 실패했습니다.');
    } finally {
      setIsUploading(false);
    }
  }, [pastedImage, workspaceId, isUploading, sendFileMessage, chatId, currentUserId, handleRemoveImage]);

  // 메시지 전송
  const handleSendMessage = async () => {
    // 이미지가 있으면 이미지 먼저 전송
    if (pastedImage) {
      await handleSendImage();
    }

    if (!inputMessage.trim()) return;

    const content = inputMessage.trim();
    const success = sendMessage(content);
    if (success) {
      // 🔥 Optimistic UI Update - 메시지를 즉시 UI에 표시
      const optimisticMessage: Message = {
        messageId: `temp-${Date.now()}`, // 임시 ID
        chatId,
        userId: currentUserId || '',
        userName: '', // 본인 메시지이므로 표시 안됨
        content,
        messageType: 'TEXT',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        isMine: true,
      };
      setMessages((prev) => [...prev, optimisticMessage]);
      setInputMessage('');
    }
  };

  // 타이핑 인디케이터
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputMessage(e.target.value);
    sendTyping(true);
    setTimeout(() => sendTyping(false), 1000);
  };

  return (
    // 🔥 fixed와 right-0 제거! 부모(MainLayout)가 위치 제어
    <div className="h-full w-full bg-white flex flex-col">
      {/* 헤더 */}
      <div className="p-4 border-b bg-gradient-to-r from-blue-600 to-blue-700 text-white">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {onBack && (
              <button
                onClick={onBack}
                className="p-1 hover:bg-white/20 rounded transition"
                title="채팅 목록으로"
              >
                <ChevronLeft className="w-5 h-5" />
              </button>
            )}
            <h3 className="font-bold">채팅</h3>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-white/20 rounded transition">
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* 메시지 영역 */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {isLoading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
          </div>
        ) : messages.length === 0 ? (
          <div className="flex justify-center py-8 text-gray-400 text-sm">
            메시지가 없습니다. 첫 메시지를 보내보세요!
          </div>
        ) : (
          messages.map((msg) => {
            // 🔥 메시지 유효성 검사 및 isMine fallback
            if (!msg || !msg.messageId) return null;
            const isMine = msg.isMine ?? msg.userId === currentUserId;

            return (
              <div
                key={msg.messageId}
                className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[70%] rounded-lg p-3 ${
                    isMine ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-900'
                  }`}
                >
                  {!isMine && (
                    <p className="text-xs font-bold mb-1 text-blue-600">
                      {msg.userName || userNameMap[msg.userId] || 'Unknown'}
                    </p>
                  )}
                  {/* 이미지 메시지 */}
                  {msg.messageType === 'IMAGE' && msg.fileUrl && (
                    <ChatImage
                      src={msg.fileUrl}
                      alt={msg.fileName || '이미지'}
                      isUploading={msg.messageId.startsWith('temp-')}
                      onClickView={setModalImage}
                    />
                  )}
                  {/* 텍스트 메시지 */}
                  {msg.content && (
                    <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
                  )}
                  <p className={`text-xs mt-1 ${isMine ? 'text-blue-100' : 'text-gray-500'}`}>
                    {msg.createdAt
                      ? new Date(msg.createdAt).toLocaleTimeString('ko-KR', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })
                      : ''}
                  </p>
                </div>
              </div>
            );
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* 입력 영역 */}
      <div className="p-4 border-t bg-gray-50">
        {/* 이미지 미리보기 */}
        {imagePreview && (
          <div className="mb-3 relative inline-block">
            <img
              src={imagePreview}
              alt="미리보기"
              className="max-h-24 rounded-lg border border-gray-200"
            />
            <button
              onClick={handleRemoveImage}
              className="absolute -top-2 -right-2 w-5 h-5 bg-red-500 text-white rounded-full flex items-center justify-center hover:bg-red-600 text-xs"
            >
              ×
            </button>
          </div>
        )}
        <div className="flex items-center gap-2">
          <input
            ref={inputRef}
            type="text"
            value={inputMessage}
            onChange={handleInputChange}
            onKeyPress={(e) => e.key === 'Enter' && !isUploading && handleSendMessage()}
            onPaste={handlePaste}
            placeholder={imagePreview ? '메시지와 함께 전송 (선택)...' : '메시지를 입력하세요... (Ctrl+V로 이미지 붙여넣기)'}
            className="flex-1 p-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={isUploading}
          />
          <button
            onClick={handleSendMessage}
            disabled={(!inputMessage.trim() && !pastedImage) || isUploading}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-300 transition flex items-center gap-1"
          >
            {isUploading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>전송중</span>
              </>
            ) : pastedImage ? (
              <>
                <ImageIcon className="w-4 h-4" />
                <span>전송</span>
              </>
            ) : (
              '전송'
            )}
          </button>
        </div>
      </div>

      {/* 이미지 확대 모달 */}
      <ImageModal src={modalImage} onClose={() => setModalImage(null)} />
    </div>
  );
};
