// src/types/video.ts - Video Service 관련 타입 정의

export interface VideoRoom {
  id: string;
  name: string;
  workspaceId: string;
  creatorId: string;
  maxParticipants: number;
  isActive: boolean;
  participantCount: number;
  participants: VideoParticipant[];
  createdAt: string;
  updatedAt: string;
}

export interface VideoParticipant {
  id: string;
  userId: string;
  joinedAt: string;
  leftAt?: string;
  isActive: boolean;
}

export interface CreateRoomRequest {
  name: string;
  workspaceId: string;
  maxParticipants?: number;
}

export interface JoinRoomResponse {
  room: VideoRoom;
  token: string;
  wsUrl: string;
}

export interface CallHistoryParticipant {
  userId: string;
  joinedAt: string;
  leftAt: string;
  durationSeconds: number;
}

export interface CallHistory {
  id: string;
  roomName: string;
  workspaceId: string;
  creatorId: string;
  startedAt: string;
  endedAt: string;
  durationSeconds: number;
  totalParticipants: number;
  participants: CallHistoryParticipant[];
}

export interface CallHistoryResponse {
  success: boolean;
  data: CallHistory[];
  total: number;
  limit: number;
  offset: number;
}

export interface Transcript {
  id: string;
  callHistoryId: string;
  roomId: string;
  content: string;
  createdAt: string;
}

export interface VideoApiResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
  };
}
