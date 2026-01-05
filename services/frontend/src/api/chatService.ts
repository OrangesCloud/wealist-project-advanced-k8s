// src/api/chat/chatService.ts

import { chatServiceClient } from './apiConfig';
import { AxiosResponse } from 'axios';
import type { Chat, Message, CreateChatRequest, SendMessageRequest } from '../types/chat';

// chat-service 응답 wrapper 타입
interface ChatServiceResponse<T> {
  message: T;
  success: boolean;
}

// 응답에서 실제 데이터 추출 헬퍼
const extractData = <T>(response: AxiosResponse<ChatServiceResponse<T> | T>): T => {
  const data = response.data;
  // wrapper 형태인지 확인 (message 필드가 있고 success 필드가 있는 경우)
  if (data && typeof data === 'object' && 'message' in data && 'success' in data) {
    return (data as ChatServiceResponse<T>).message;
  }
  // 직접 데이터인 경우
  return data as T;
};

/**
 * 🔥 DM 채팅방 생성 또는 기존 채팅방 가져오기
 * @param targetUserId 대화 상대방 userId
 * @param workspaceId 워크스페이스 ID
 * @returns chatId
 */
export const createOrGetDMChat = async (
  targetUserId: string,
  workspaceId: string,
): Promise<string> => {
  try {
    console.log('🔍 DM 채팅방 찾기/생성:', { targetUserId, workspaceId });

    // 1. 내 채팅방 목록 가져오기
    const myChats = await getMyChats();
    console.log('📋 내 채팅방 목록:', myChats.length, '개');

    // 2. 이미 존재하는 DM 채팅방 찾기
    const existingDM = myChats.find((chat) => {
      if (chat.chatType !== 'DM') return false;

      // participants가 있으면 확인
      if (chat.participants) {
        const participantUserIds = chat.participants.map((p) => p.userId);
        return participantUserIds.includes(targetUserId);
      }

      return false;
    });

    if (existingDM) {
      console.log('✅ 기존 DM 채팅방 사용:', existingDM.chatId);
      return existingDM.chatId;
    }

    // 3. 없으면 새로 생성
    console.log('🆕 새 DM 채팅방 생성 중...');
    const newChat = await createChat({
      workspaceId,
      chatType: 'DM',
      chatName: 'DM',
      participants: [targetUserId],
    });

    console.log('✅ 새 채팅방 생성 완료:', newChat.chatId);
    return newChat.chatId;
  } catch (error) {
    console.error('❌ Failed to create or get DM chat:', error);
    throw error;
  }
};

/**
 * 채팅방 생성
 * [API] POST /api/chats
 */
export const createChat = async (data: CreateChatRequest): Promise<Chat> => {
  const response = await chatServiceClient.post('', data);
  return extractData<Chat>(response);
};

/**
 * 내 채팅방 목록 조회
 * [API] GET /api/chats/my
 */
export const getMyChats = async (): Promise<Chat[]> => {
  const response = await chatServiceClient.get('/my');
  const data = extractData<Chat[]>(response);
  return Array.isArray(data) ? data : [];
};

/**
 * 워크스페이스 채팅방 목록 조회
 * [API] GET /api/chats/workspace/{workspaceId}
 */
export const getWorkspaceChats = async (workspaceId: string): Promise<Chat[]> => {
  const response = await chatServiceClient.get(`/workspace/${workspaceId}`);
  const data = extractData<Chat[]>(response);
  return Array.isArray(data) ? data : [];
};

/**
 * 프로젝트 채팅방 조회 (필터링)
 * 워크스페이스의 모든 채팅방 중 특정 프로젝트 채팅만 필터링
 */
export const getProjectChats = async (projectId: string): Promise<Chat[]> => {
  const chats = await getMyChats();
  return chats.filter((chat) => chat.projectId === projectId || chat.chatType === 'PROJECT');
};

/**
 * 채팅방 상세 조회
 * [API] GET /api/chats/{chatId}
 */
export const getChat = async (chatId: string): Promise<Chat> => {
  const response = await chatServiceClient.get(`/${chatId}`);
  return extractData<Chat>(response);
};

/**
 * 채팅방 삭제
 * [API] DELETE /api/chats/{chatId}
 */
export const deleteChat = async (chatId: string): Promise<void> => {
  await chatServiceClient.delete(`/${chatId}`);
};

/**
 * 참여자 추가
 * [API] POST /api/chats/{chatId}/participants
 */
export const addParticipants = async (chatId: string, userIds: string[]): Promise<void> => {
  await chatServiceClient.post(`/${chatId}/participants`, { userIds });
};

/**
 * 참여자 제거
 * [API] DELETE /api/chats/{chatId}/participants/{userId}
 */
export const removeParticipant = async (chatId: string, userId: string): Promise<void> => {
  await chatServiceClient.delete(`/${chatId}/participants/${userId}`);
};

/**
 * 메시지 히스토리 조회
 * [API] GET /api/chats/messages/{chatId}
 */
export const getMessages = async (chatId: string, limit = 50, offset = 0): Promise<Message[]> => {
  const response = await chatServiceClient.get(`/messages/${chatId}`, {
    params: { limit, offset },
  });

  // 🔥 wrapper 응답에서 데이터 추출
  const data = extractData<Message[]>(response);

  // 🔥 null/undefined 체크
  if (!data || !Array.isArray(data)) {
    console.warn('[getMessages] 메시지 데이터가 비어있거나 배열이 아님:', data);
    return [];
  }

  // 🔥 현재 사용자 ID 가져오기
  const currentUserId = localStorage.getItem('userId');

  // isMine 플래그 추가
  return data.map((msg) => ({
    ...msg,
    isMine: msg.userId === currentUserId,
  }));
};

/**
 * 메시지 전송 (REST fallback)
 * [API] POST /api/chats/messages/{chatId}
 */
export const sendMessage = async (chatId: string, content: string): Promise<Message> => {
  const requestData: SendMessageRequest = { content };
  const response = await chatServiceClient.post(`/messages/${chatId}`, requestData);
  return extractData<Message>(response);
};

/**
 * 메시지 삭제
 * [API] DELETE /api/chats/messages/{messageId}
 */
export const deleteMessage = async (messageId: string): Promise<void> => {
  await chatServiceClient.delete(`/messages/${messageId}`);
};

/**
 * 메시지 읽음 처리
 * [API] POST /api/chats/messages/read
 */
export const markMessagesAsRead = async (messageIds: string[]): Promise<void> => {
  await chatServiceClient.post('/messages/read', { messageIds });
};

/**
 * 읽지 않은 메시지 수 조회
 * [API] GET /api/chats/messages/{chatId}/unread
 */
export const getUnreadCount = async (chatId: string): Promise<number> => {
  const response = await chatServiceClient.get(`/messages/${chatId}/unread`);
  const data = extractData<{ unreadCount: number }>(response);
  return data?.unreadCount ?? 0;
};

/**
 * 마지막 읽은 시간 업데이트
 * [API] PUT /api/chats/messages/{chatId}/last-read
 */
export const updateLastRead = async (chatId: string): Promise<void> => {
  await chatServiceClient.put(`/messages/${chatId}/last-read`);
};

// ============================================================================
// 🔥 File Upload API (채팅 이미지 업로드)
// ============================================================================

/**
 * Presigned URL 요청 타입
 */
interface ChatPresignedURLRequest {
  workspaceId: string;
  fileName: string;
  contentType: string;
  fileSize: number;
}

/**
 * Presigned URL 응답 타입
 */
interface ChatPresignedURLResponse {
  uploadUrl: string;
  downloadUrl: string; // 업로드 후 파일 접근 URL
  fileKey: string;
  expiresIn: number;
}

/**
 * 채팅 파일 업로드용 Presigned URL 생성
 * [API] POST /api/chats/files/presigned-url
 */
export const generateChatPresignedURL = async (
  data: ChatPresignedURLRequest,
): Promise<ChatPresignedURLResponse> => {
  const response = await chatServiceClient.post('/files/presigned-url', data);
  return extractData<ChatPresignedURLResponse>(response);
};

/**
 * S3에 직접 파일 업로드 (presigned URL 사용)
 */
export const uploadChatFileToS3 = async (uploadUrl: string, file: File): Promise<void> => {
  await fetch(uploadUrl, {
    method: 'PUT',
    body: file,
    headers: {
      'Content-Type': file.type,
    },
  });
};

// ============================================================================
// 🔥 Presence API (온라인 상태)
// ============================================================================

/**
 * 온라인 사용자 목록 조회
 * [API] GET /api/chats/presence/online
 */
export const getOnlineUsers = async (): Promise<string[]> => {
  const response = await chatServiceClient.get('/presence/online');
  const data = extractData<{ onlineUsers: string[]; count: number }>(response);
  return data?.onlineUsers ?? [];
};

/**
 * 특정 사용자 온라인 여부 확인
 * [API] GET /api/chats/presence/status/{userId}
 * @param userId 확인할 사용자 ID
 * @returns true: 온라인, false: 오프라인
 */
export const checkUserStatus = async (userId: string): Promise<boolean> => {
  const response = await chatServiceClient.get(`/presence/status/${userId}`);
  const data = extractData<{ userId: string; isOnline: boolean }>(response);
  return data?.isOnline ?? false;
};

/**
 * 여러 사용자의 온라인 상태 일괄 확인
 * @param userIds 확인할 사용자 ID 배열
 * @returns Map<userId, isOnline>
 */
export const checkMultipleUserStatus = async (userIds: string[]): Promise<Map<string, boolean>> => {
  try {
    // 온라인 사용자 목록 가져오기
    const onlineUsers = await getOnlineUsers();
    const onlineSet = new Set(onlineUsers);

    // Map으로 변환
    const statusMap = new Map<string, boolean>();
    userIds.forEach((userId) => {
      statusMap.set(userId, onlineSet.has(userId));
    });

    return statusMap;
  } catch (error) {
    console.error('Failed to check multiple user status:', error);
    // 에러 시 모두 오프라인으로 처리
    const statusMap = new Map<string, boolean>();
    userIds.forEach((userId) => {
      statusMap.set(userId, false);
    });
    return statusMap;
  }
};
