// src/api/videoService.ts - Video Service API 호출

import { videoServiceClient } from './apiConfig';
import type {
  VideoRoom,
  VideoParticipant,
  CreateRoomRequest,
  JoinRoomResponse,
  CallHistory,
  CallHistoryResponse,
  Transcript,
  VideoApiResponse,
} from '../types/video';

export const videoService = {
  // Create a new video room
  async createRoom(request: CreateRoomRequest): Promise<VideoRoom> {
    const response = await videoServiceClient.post<VideoApiResponse<VideoRoom>>(
      '/api/video/rooms',
      request
    );
    return response.data.data;
  },

  // Get rooms for a workspace
  async getWorkspaceRooms(workspaceId: string, activeOnly: boolean = true): Promise<VideoRoom[]> {
    const response = await videoServiceClient.get<VideoApiResponse<VideoRoom[]>>(
      `/api/video/rooms/workspace/${workspaceId}`,
      { params: { active: activeOnly } }
    );
    return response.data.data || [];
  },

  // Get room details
  async getRoom(roomId: string): Promise<VideoRoom> {
    const response = await videoServiceClient.get<VideoApiResponse<VideoRoom>>(
      `/api/video/rooms/${roomId}`
    );
    return response.data.data;
  },

  // Join a video room
  async joinRoom(roomId: string, userName?: string): Promise<JoinRoomResponse> {
    const response = await videoServiceClient.post<VideoApiResponse<JoinRoomResponse>>(
      `/api/video/rooms/${roomId}/join`,
      {},
      { params: userName ? { userName } : {} }
    );
    return response.data.data;
  },

  // Leave a video room
  async leaveRoom(roomId: string): Promise<void> {
    await videoServiceClient.post(`/api/video/rooms/${roomId}/leave`, {});
  },

  // End a video room (creator only)
  async endRoom(roomId: string): Promise<void> {
    await videoServiceClient.post(`/api/video/rooms/${roomId}/end`, {});
  },

  // Get room participants
  async getParticipants(roomId: string): Promise<VideoParticipant[]> {
    const response = await videoServiceClient.get<VideoApiResponse<VideoParticipant[]>>(
      `/api/video/rooms/${roomId}/participants`
    );
    return response.data.data || [];
  },

  // Get call history for a workspace
  async getWorkspaceCallHistory(
    workspaceId: string,
    limit: number = 20,
    offset: number = 0
  ): Promise<CallHistoryResponse> {
    const response = await videoServiceClient.get<CallHistoryResponse>(
      `/api/video/history/workspace/${workspaceId}`,
      { params: { limit, offset } }
    );
    return response.data;
  },

  // Get current user's call history
  async getMyCallHistory(
    limit: number = 20,
    offset: number = 0
  ): Promise<CallHistoryResponse> {
    const response = await videoServiceClient.get<CallHistoryResponse>(
      `/api/video/history/me`,
      { params: { limit, offset } }
    );
    return response.data;
  },

  // Get single call history by ID
  async getCallHistory(historyId: string): Promise<CallHistory | null> {
    try {
      const response = await videoServiceClient.get<VideoApiResponse<CallHistory>>(
        `/api/video/history/${historyId}`
      );
      return response.data.data;
    } catch (error) {
      console.error('Failed to fetch call history:', error);
      return null;
    }
  },

  // Save transcript for a room
  async saveTranscript(roomId: string, content: string): Promise<Transcript> {
    const response = await videoServiceClient.post<VideoApiResponse<Transcript>>(
      `/api/video/rooms/${roomId}/transcript`,
      { content }
    );
    return response.data.data;
  },

  // Get transcript for a call history
  async getTranscript(historyId: string): Promise<Transcript | null> {
    const response = await videoServiceClient.get<VideoApiResponse<Transcript | null>>(
      `/api/video/history/${historyId}/transcript`
    );
    return response.data.data;
  },
};

export default videoService;

// Re-export types for convenience
export type {
  VideoRoom,
  VideoParticipant,
  CreateRoomRequest,
  JoinRoomResponse,
  CallHistory,
  CallHistoryResponse,
  CallHistoryParticipant,
  Transcript,
  VideoApiResponse,
} from '../types/video';
