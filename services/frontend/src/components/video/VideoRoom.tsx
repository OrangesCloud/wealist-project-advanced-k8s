import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Mic,
  MicOff,
  Video,
  VideoOff,
  PhoneOff,
  Users,
  Monitor,
  MessageSquare,
  Copy,
  Check,
  Minimize2,
  Maximize2,
  GripHorizontal,
  Send,
  Captions,
  CaptionsOff,
  FlipHorizontal,
} from 'lucide-react';
import { VideoRoom as VideoRoomType } from '../../api/videoService';
import {
  Room,
  RoomEvent,
  VideoPresets,
  Track,
  LocalParticipant,
  RemoteParticipant,
  RemoteTrack,
  RemoteTrackPublication,
  Participant,
  DataPacket_Kind,
  LocalTrackPublication,
  DisconnectReason,
} from 'livekit-client';

// Display mode types
type DisplayMode = 'full' | 'mini';

interface UserProfile {
  id: string;
  nickName: string;
  profileImageUrl?: string | null;
}

interface VideoRoomProps {
  room: VideoRoomType;
  token: string;
  wsUrl: string;
  onLeave: () => void;
  onTokenRefresh?: () => Promise<{ token: string; wsUrl: string } | null>;
  userProfile?: UserProfile | null;
}

// LiveKit connection state types
type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error' | 'reconnecting';

interface ChatMessage {
  id: string;
  sender: string;
  senderName: string;
  message: string;
  timestamp: Date;
  isLocal: boolean;
}

interface TranscriptLine {
  id: string;
  speakerId: string;
  speakerName: string;
  text: string;
  timestamp: Date;
  isFinal: boolean;
}

// DataPacket message types for real-time sync
interface DataMessage {
  type: 'chat' | 'subtitle';
  // Chat fields
  message?: string;
  // Subtitle fields
  text?: string;
  speakerId?: string;
  speakerName?: string;
  isFinal?: boolean;
}

// Participant metadata stored in LiveKit
interface ParticipantMetadata {
  id?: string;
  nickName?: string;
  profileImageUrl?: string | null;
}

// Remote subtitle state
interface RemoteSubtitle {
  speakerId: string;
  speakerName: string;
  text: string;
  timestamp: number;
}

// Web Speech API type declarations (Keeping this for subtitles)
interface SpeechRecognitionEvent {
  results: SpeechRecognitionResultList;
  resultIndex: number;
}

interface SpeechRecognitionResultList {
  length: number;
  item(index: number): SpeechRecognitionResult;
  [index: number]: SpeechRecognitionResult;
}

interface SpeechRecognitionResult {
  isFinal: boolean;
  length: number;
  item(index: number): SpeechRecognitionAlternative;
  [index: number]: SpeechRecognitionAlternative;
}

interface SpeechRecognitionAlternative {
  transcript: string;
  confidence: number;
}

interface SpeechRecognition extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

declare global {
  interface Window {
    SpeechRecognition: new () => SpeechRecognition;
    webkitSpeechRecognition: new () => SpeechRecognition;
  }
}

// Helper component to render video track
const VideoTrackRenderer: React.FC<{
  track?: Track | null;
  isLocal: boolean;
  isMirrored: boolean;
}> = ({ track, isLocal, isMirrored }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const el = videoRef.current;
    if (el && track) {
      track.attach(el);
      return () => {
        track.detach(el);
      };
    }
  }, [track]);

  if (!track)
    return (
      <div className="w-full h-full bg-gray-800 flex items-center justify-center text-gray-500">
        카메라 꺼짐
      </div>
    );

  return (
    <video
      ref={videoRef}
      className="w-full h-full object-cover"
      style={{ transform: isLocal && isMirrored ? 'scaleX(-1)' : 'none' }}
    />
  );
};

export const VideoRoom: React.FC<VideoRoomProps> = ({
  room: roomInfo,
  token: initialToken,
  wsUrl: initialWsUrl,
  onLeave,
  onTokenRefresh,
  userProfile,
}) => {
  const [room, setRoom] = useState<Room | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting');
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [activeSpeakers, setActiveSpeakers] = useState<string[]>([]);
  const [remoteSubtitles, setRemoteSubtitles] = useState<Map<string, RemoteSubtitle>>(new Map());

  // Token state for rejoin
  const [currentToken, setCurrentToken] = useState(initialToken);
  const [currentWsUrl, setCurrentWsUrl] = useState(initialWsUrl);
  const rejoinAttemptsRef = useRef(0);
  const maxRejoinAttempts = 3;
  // Track if we've ever successfully connected (to distinguish initial failure from disconnection)
  const hasConnectedOnceRef = useRef(false);

  // Local state
  const [isMuted, setIsMuted] = useState(false);
  const [isVideoEnabled, setIsVideoEnabled] = useState(false);
  const [isScreenSharing, setIsScreenSharing] = useState(false);
  const [isMirrored, setIsMirrored] = useState(true);
  const [_isBlurEnabled, _setIsBlurEnabled] = useState(false); // Not implemented with real SDK yet
  const [showParticipants, setShowParticipants] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Chat state
  const [showChat, setShowChat] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState('');
  const chatEndRef = useRef<HTMLDivElement>(null);

  // Mini mode state
  const [displayMode, setDisplayMode] = useState<DisplayMode>('full');
  const [miniPosition, setMiniPosition] = useState({ x: 20, y: 20 });
  const [isDragging, setIsDragging] = useState(false);
  const dragOffsetRef = useRef({ x: 0, y: 0 });

  // Subtitle/Transcript state
  const [isSubtitleEnabled, setIsSubtitleEnabled] = useState(false);
  const [currentSubtitle, setCurrentSubtitle] = useState('');
  const [_transcript, setTranscript] = useState<TranscriptLine[]>([]);
  const recognitionRef = useRef<SpeechRecognition | null>(null);

  // Helper to get user display name
  const getMyDisplayName = useCallback(() => {
    return userProfile?.nickName || room?.localParticipant?.name || '나';
  }, [userProfile, room]);

  // Join notification state
  const [joinNotification, setJoinNotification] = useState<string | null>(null);

  const miniContainerRef = useRef<HTMLDivElement>(null);

  // Initialize Room
  useEffect(() => {
    const newRoom = new Room({
      adaptiveStream: true,
      dynacast: true,
      videoCaptureDefaults: {
        resolution: VideoPresets.h720.resolution,
      },
      // Reconnection options
      reconnectPolicy: {
        nextRetryDelayInMs: (context) => {
          // Exponential backoff: 300ms, 600ms, 1200ms, ... up to 10s
          const delay = Math.min(300 * Math.pow(2, context.retryCount), 10000);
          console.log(`[VideoRoom] Reconnect attempt ${context.retryCount + 1}, delay: ${delay}ms`);
          return delay;
        },
      },
      disconnectOnPageLeave: true,
    });

    setRoom(newRoom);

    return () => {
      newRoom.disconnect();
    };
  }, []);

  // Rejoin room with new token
  const attemptRejoin = useCallback(async () => {
    if (!onTokenRefresh || rejoinAttemptsRef.current >= maxRejoinAttempts) {
      console.log('[VideoRoom] Max rejoin attempts reached or no token refresh callback');
      setConnectionState('error');
      setError('연결을 복구할 수 없습니다. 다시 참가해주세요.');
      return;
    }

    rejoinAttemptsRef.current += 1;
    console.log(`[VideoRoom] Attempting rejoin ${rejoinAttemptsRef.current}/${maxRejoinAttempts}`);
    setConnectionState('reconnecting');

    try {
      const newCredentials = await onTokenRefresh();
      if (!newCredentials) {
        throw new Error('Failed to get new token');
      }

      setCurrentToken(newCredentials.token);
      setCurrentWsUrl(newCredentials.wsUrl);
      console.log('[VideoRoom] Got new token, will reconnect');
    } catch (e: any) {
      console.error('[VideoRoom] Failed to refresh token:', e);

      // Check if room is gone (410) or not found (404) - don't retry
      const status = e?.response?.status;
      if (status === 410 || status === 404) {
        console.log('[VideoRoom] Room is gone or ended, stopping rejoin attempts');
        setConnectionState('error');
        setError('통화가 종료되었습니다.');
        return;
      }

      // Try again with exponential backoff for other errors
      const delay = Math.min(1000 * Math.pow(2, rejoinAttemptsRef.current), 10000);
      setTimeout(attemptRejoin, delay);
    }
  }, [onTokenRefresh, maxRejoinAttempts]);

  // Connect to LiveKit
  useEffect(() => {
    if (!room || !currentToken || !currentWsUrl) return;

    const connect = async () => {
      try {
        console.log(
          '[VideoRoom] Connecting with WS URL:',
          currentWsUrl,
          'Token:',
          currentToken?.slice(0, 10) + '...',
        );
        await room.connect(currentWsUrl, currentToken);
        setConnectionState('connected');
        hasConnectedOnceRef.current = true; // Mark that we've successfully connected
        rejoinAttemptsRef.current = 0; // Reset rejoin attempts on successful connection
        updateParticipants();

        // Set participant metadata (profile info) - non-blocking
        // Metadata is optional, so don't fail connection if it times out
        if (userProfile) {
          const metadata: ParticipantMetadata = {
            id: userProfile.id,
            nickName: userProfile.nickName,
            profileImageUrl: userProfile.profileImageUrl,
          };
          room.localParticipant
            .setMetadata(JSON.stringify(metadata))
            .then(() => console.log('[VideoRoom] Set participant metadata:', metadata))
            .catch((e) => console.warn('[VideoRoom] Failed to set metadata (non-critical):', e));
        }

        // Publish initial state (Audio default on if available, Video off)
        // Note: Browser policy might require user interaction, but we'll try
        try {
          await room.localParticipant.setMicrophoneEnabled(true);
          setIsMuted(false);
        } catch (e) {
          console.warn('Autoplay policy prevented microphone connect', e);
          setIsMuted(true);
        }
      } catch (e) {
        console.error('Failed to connect', e);
        setConnectionState('error');
        setError(e instanceof Error ? e.message : 'Connection failed');
      }
    };

    connect();

    // Event listeners
    const onParticipantConnected = (participant: RemoteParticipant) => {
      setJoinNotification(`${participant.identity}님이 참여하셨습니다`); // identity as fallback, better to use name if available
      setTimeout(() => setJoinNotification(null), 3000);
      updateParticipants();
    };

    const onParticipantDisconnected = (_participant: RemoteParticipant) => {
      updateParticipants();
    };

    const onTrackSubscribed = (
      _track: RemoteTrack,
      _publication: RemoteTrackPublication,
      _participant: RemoteParticipant,
    ) => {
      updateParticipants(); // Trigger re-render to show video
    };

    const onTrackUnsubscribed = (
      _track: RemoteTrack,
      _publication: RemoteTrackPublication,
      _participant: RemoteParticipant,
    ) => {
      updateParticipants();
    };

    const onLocalTrackPublished = (
      _publication: LocalTrackPublication,
      _participant: LocalParticipant,
    ) => {
      updateParticipants();
    };

    const onLocalTrackUnpublished = (
      _publication: LocalTrackPublication,
      _participant: LocalParticipant,
    ) => {
      updateParticipants();
    };

    const onDataReceived = (
      payload: Uint8Array,
      participant?: RemoteParticipant,
      _kind?: DataPacket_Kind,
    ) => {
      const str = new TextDecoder().decode(payload);
      try {
        const data = JSON.parse(str) as DataMessage;

        if (data.type === 'chat') {
          const newMessage: ChatMessage = {
            id: Date.now().toString() + Math.random(),
            sender: participant?.identity || 'Anonymous',
            senderName: participant?.name || participant?.identity || 'Anonymous',
            message: data.message || '',
            timestamp: new Date(),
            isLocal: false,
          };
          setChatMessages((prev) => [...prev, newMessage]);
        } else if (data.type === 'subtitle' && data.text) {
          // Handle subtitle from other participants
          const speakerId = data.speakerId || participant?.identity || 'unknown';
          const speakerName =
            data.speakerName || participant?.name || participant?.identity || 'Unknown';

          // Update remote subtitles map
          setRemoteSubtitles((prev) => {
            const newMap = new Map(prev);
            newMap.set(speakerId, {
              speakerId,
              speakerName,
              text: data.text!,
              timestamp: Date.now(),
            });
            return newMap;
          });

          // If final, add to transcript
          if (data.isFinal) {
            const newLine: TranscriptLine = {
              id: Date.now().toString() + speakerId,
              speakerId,
              speakerName,
              text: data.text,
              timestamp: new Date(),
              isFinal: true,
            };
            setTranscript((prev) => [...prev, newLine]);

            // Clear this speaker's subtitle after a short delay
            setTimeout(() => {
              setRemoteSubtitles((prev) => {
                const newMap = new Map(prev);
                newMap.delete(speakerId);
                return newMap;
              });
            }, 500);
          }
        }
      } catch (e) {
        console.error('Failed to parse data message', e);
      }
    };

    const onActiveSpeakersChanged = (speakers: Participant[]) => {
      setActiveSpeakers(speakers.map((s) => s.identity));
      updateParticipants(); // Re-render to show speaking indicator if needed
    };

    // Connection state event handlers
    const onDisconnected = (reason?: DisconnectReason) => {
      console.log('[VideoRoom] Disconnected from room, reason:', reason);
      setConnectionState('disconnected');

      // Only attempt to rejoin if we've successfully connected at least once
      // This prevents infinite reconnection loops when the initial connection fails
      if (!hasConnectedOnceRef.current) {
        console.log('[VideoRoom] Initial connection failed, not attempting rejoin');
        setConnectionState('error');
        setError('통화 연결에 실패했습니다. 다시 시도해주세요.');
        return;
      }

      // If we have a token refresh callback, attempt to rejoin
      // This handles cases where LiveKit's built-in reconnection fails
      if (onTokenRefresh && rejoinAttemptsRef.current < maxRejoinAttempts) {
        console.log('[VideoRoom] Will attempt to rejoin with new token');
        // Wait a bit before attempting rejoin to avoid rapid reconnection loops
        setTimeout(() => {
          attemptRejoin();
        }, 2000);
      }
    };

    const onReconnecting = () => {
      console.log('[VideoRoom] Reconnecting to room...');
      setConnectionState('reconnecting');
    };

    const onReconnected = () => {
      console.log('[VideoRoom] Successfully reconnected to room');
      setConnectionState('connected');
      rejoinAttemptsRef.current = 0; // Reset rejoin attempts
      updateParticipants();
    };

    room.on(RoomEvent.ParticipantConnected, onParticipantConnected);
    room.on(RoomEvent.ParticipantDisconnected, onParticipantDisconnected);
    room.on(RoomEvent.TrackSubscribed, onTrackSubscribed);
    room.on(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed);
    room.on(RoomEvent.LocalTrackPublished, onLocalTrackPublished);
    room.on(RoomEvent.LocalTrackUnpublished, onLocalTrackUnpublished);
    room.on(RoomEvent.DataReceived, onDataReceived);
    room.on(RoomEvent.ActiveSpeakersChanged, onActiveSpeakersChanged);
    room.on(RoomEvent.Disconnected, onDisconnected);
    room.on(RoomEvent.Reconnecting, onReconnecting);
    room.on(RoomEvent.Reconnected, onReconnected);

    // Also listed to mute/unmute events for UI updates
    room.on(RoomEvent.TrackMuted, updateParticipants);
    room.on(RoomEvent.TrackUnmuted, updateParticipants);

    return () => {
      room.off(RoomEvent.ParticipantConnected, onParticipantConnected);
      room.off(RoomEvent.ParticipantDisconnected, onParticipantDisconnected);
      room.off(RoomEvent.TrackSubscribed, onTrackSubscribed);
      room.off(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed);
      room.off(RoomEvent.LocalTrackPublished, onLocalTrackPublished);
      room.off(RoomEvent.LocalTrackUnpublished, onLocalTrackUnpublished);
      room.off(RoomEvent.DataReceived, onDataReceived);
      room.off(RoomEvent.ActiveSpeakersChanged, onActiveSpeakersChanged);
      room.off(RoomEvent.Disconnected, onDisconnected);
      room.off(RoomEvent.Reconnecting, onReconnecting);
      room.off(RoomEvent.Reconnected, onReconnected);
      room.off(RoomEvent.TrackMuted, updateParticipants);
      room.off(RoomEvent.TrackUnmuted, updateParticipants);
    };
  }, [room, currentToken, currentWsUrl, onTokenRefresh, attemptRejoin]);

  const updateParticipants = useCallback(() => {
    if (!room) return;
    const remotes = Array.from(room.remoteParticipants.values());
    setParticipants([room.localParticipant, ...remotes]);
  }, [room]);

  const toggleMute = useCallback(async () => {
    if (!room) return;
    // const enable = !isMuted;

    try {
      await room.localParticipant.setMicrophoneEnabled(isMuted);
      setIsMuted(!isMuted);
    } catch (e) {
      console.error('Failed to toggle mute', e);
    }
  }, [room, isMuted]);

  const toggleVideo = useCallback(async () => {
    if (!room) return;
    try {
      await room.localParticipant.setCameraEnabled(!isVideoEnabled);
      setIsVideoEnabled(!isVideoEnabled);
    } catch (e) {
      console.error('Failed to toggle video', e);
    }
  }, [room, isVideoEnabled]);

  const toggleScreenShare = useCallback(async () => {
    if (!room) return;
    try {
      await room.localParticipant.setScreenShareEnabled(!isScreenSharing);
      setIsScreenSharing(!isScreenSharing);
    } catch (e) {
      console.error('Failed to toggle screen share', e);
      setIsScreenSharing(false);
    }
  }, [room, isScreenSharing]);

  const handleLeave = useCallback(() => {
    room?.disconnect();
    onLeave();
  }, [room, onLeave]);

  // Chat functions
  const sendChatMessage = useCallback(async () => {
    if (!chatInput.trim() || !room) return;

    const msgData = {
      type: 'chat',
      message: chatInput.trim(),
    };

    try {
      const strData = JSON.stringify(msgData);
      const encoder = new TextEncoder();
      await room.localParticipant.publishData(encoder.encode(strData), { reliable: true });

      const newMessage: ChatMessage = {
        id: Date.now().toString(),
        sender: room.localParticipant.identity,
        senderName: room.localParticipant.name || '나',
        message: chatInput.trim(),
        timestamp: new Date(),
        isLocal: true,
      };
      setChatMessages((prev) => [...prev, newMessage]);
      setChatInput('');
    } catch (e) {
      console.error('Failed to send message', e);
    }
  }, [chatInput, room]);

  const handleChatKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendChatMessage();
    }
  };

  // Scroll to bottom when new message arrives
  useEffect(() => {
    if (chatEndRef.current && showChat) {
      chatEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [chatMessages, showChat]);

  const formatChatTime = (date: Date) => {
    return date.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' });
  };

  const copyRoomLink = () => {
    navigator.clipboard.writeText(`${window.location.origin}/video/${roomInfo.id}`);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Broadcast subtitle to other participants
  const broadcastSubtitle = useCallback(
    async (text: string, isFinal: boolean) => {
      if (!room || !text) return;

      const myName = getMyDisplayName();
      const myId = userProfile?.id || room.localParticipant.identity;

      const subtitleData: DataMessage = {
        type: 'subtitle',
        text,
        speakerId: myId,
        speakerName: myName,
        isFinal,
      };

      try {
        const encoder = new TextEncoder();
        await room.localParticipant.publishData(
          encoder.encode(JSON.stringify(subtitleData)),
          { reliable: isFinal }, // Use reliable for final subtitles
        );
      } catch (e) {
        console.error('Failed to broadcast subtitle:', e);
      }
    },
    [room, userProfile, getMyDisplayName],
  );

  // Speech Recognition (Subtitle) functions - Keeping logic but needs audio stream source updates ideally
  // For now, we'll keep the browser Native Speech API as it uses the physical mic independently of WebRTC
  const startSpeechRecognition = useCallback(() => {
    const SpeechRecognitionAPI = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRecognitionAPI) {
      console.warn('Speech Recognition not supported');
      return;
    }

    const recognition = new SpeechRecognitionAPI();
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = 'ko-KR';

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let interimTranscript = '';
      let finalTranscript = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          finalTranscript += result[0].transcript;
        } else {
          interimTranscript += result[0].transcript;
        }
      }

      const displayText = interimTranscript || finalTranscript;
      const myName = getMyDisplayName();
      setCurrentSubtitle(displayText ? `${myName}: ${displayText}` : '');

      // Broadcast to other participants
      if (displayText) {
        broadcastSubtitle(displayText, !!finalTranscript);
      }

      if (finalTranscript) {
        const myId = userProfile?.id || room?.localParticipant?.identity || 'me';
        const newLine: TranscriptLine = {
          id: Date.now().toString(),
          speakerId: myId,
          speakerName: myName,
          text: finalTranscript,
          timestamp: new Date(),
          isFinal: true,
        };
        setTranscript((prev) => [...prev, newLine]);
        setTimeout(() => setCurrentSubtitle(''), 500);
      }
    };

    recognition.onend = () => {
      if (isSubtitleEnabled && recognitionRef.current) {
        try {
          recognitionRef.current.start();
        } catch (e) {
          console.error('Failed to restart recognition:', e);
        }
      }
    };

    recognitionRef.current = recognition;
    recognition.start();
  }, [isSubtitleEnabled, getMyDisplayName, broadcastSubtitle, userProfile, room]);

  const stopSpeechRecognition = useCallback(() => {
    if (recognitionRef.current) {
      recognitionRef.current.onend = null;
      recognitionRef.current.stop();
      recognitionRef.current = null;
    }
    setCurrentSubtitle('');
  }, []);

  const toggleSubtitle = () => {
    if (isSubtitleEnabled) {
      stopSpeechRecognition();
    } else {
      startSpeechRecognition();
    }
    setIsSubtitleEnabled(!isSubtitleEnabled);
  };

  useEffect(() => {
    return () => {
      if (recognitionRef.current) {
        recognitionRef.current.onend = null;
        recognitionRef.current.stop();
      }
    };
  }, []);

  const toggleDisplayMode = () => {
    setDisplayMode((prev) => (prev === 'full' ? 'mini' : 'full'));
  };

  // Drag handlers for mini mode
  const handleDragStart = (e: React.MouseEvent) => {
    if (displayMode !== 'mini') return;
    e.preventDefault();
    setIsDragging(true);
    const container = miniContainerRef.current;
    if (container) {
      const rect = container.getBoundingClientRect();
      dragOffsetRef.current = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
    }
  };

  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      const container = miniContainerRef.current;
      if (!container) return;

      const containerWidth = container.offsetWidth;
      const containerHeight = container.offsetHeight;

      let newX = window.innerWidth - e.clientX - containerWidth + dragOffsetRef.current.x;
      let newY = window.innerHeight - e.clientY - containerHeight + dragOffsetRef.current.y;

      newX = Math.max(0, Math.min(newX, window.innerWidth - containerWidth));
      newY = Math.max(0, Math.min(newY, window.innerHeight - containerHeight));

      setMiniPosition({ x: newX, y: newY });
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging]);

  if (connectionState === 'error') {
    return (
      <div className="fixed inset-0 bg-gray-900 flex items-center justify-center z-50">
        <div className="text-center text-white">
          <VideoOff className="w-16 h-16 mx-auto mb-4 opacity-50" />
          <h2 className="text-xl font-semibold mb-2">연결 오류</h2>
          <p className="text-gray-400 mb-4">{error}</p>
          <button
            onClick={onLeave}
            className="px-6 py-2 bg-red-500 hover:bg-red-600 rounded-lg transition"
          >
            나가기
          </button>
        </div>
      </div>
    );
  }

  // Reconnecting overlay - shown on top of the video room
  const ReconnectingOverlay = () => {
    if (connectionState !== 'reconnecting' && connectionState !== 'disconnected') return null;

    return (
      <div className="absolute inset-0 bg-black/70 flex items-center justify-center z-[60] backdrop-blur-sm">
        <div className="text-center text-white">
          <div className="w-16 h-16 mx-auto mb-4 relative">
            <div className="absolute inset-0 border-4 border-blue-500/30 rounded-full" />
            <div className="absolute inset-0 border-4 border-transparent border-t-blue-500 rounded-full animate-spin" />
          </div>
          <h2 className="text-xl font-semibold mb-2">
            {connectionState === 'reconnecting' ? '재연결 중...' : '연결이 끊어졌습니다'}
          </h2>
          <p className="text-gray-400 mb-4">
            {connectionState === 'reconnecting'
              ? '잠시만 기다려주세요'
              : '재연결을 시도하고 있습니다'}
          </p>
          <button
            onClick={onLeave}
            className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition text-sm"
          >
            나가기
          </button>
        </div>
      </div>
    );
  };

  // Helper to get video track
  const getParticipantVideoTrack = (p: Participant) => {
    const pub = p.getTrackPublication(Track.Source.Camera);
    if (pub && pub.isSubscribed && pub.track) {
      return pub.track;
    }
    // For local participant, track is available even if not 'subscribed' in remote sense
    if (p instanceof LocalParticipant) {
      const localPub = p.getTrackPublication(Track.Source.Camera);
      return localPub?.track;
    }
    return null;
  };

  // Helper to get participant metadata
  const getParticipantMetadata = (p: Participant): ParticipantMetadata | null => {
    if (!p.metadata) return null;
    try {
      return JSON.parse(p.metadata) as ParticipantMetadata;
    } catch {
      return null;
    }
  };

  // Helper to check if participant is speaking
  const isSpeaking = (p: Participant): boolean => {
    return activeSpeakers.includes(p.identity);
  };

  if (displayMode === 'mini') {
    const localP = room?.localParticipant;
    const localVideoTrack = localP ? getParticipantVideoTrack(localP) : null;

    return (
      <>
        {joinNotification && (
          <div className="fixed top-4 right-4 z-[60] animate-fade-in">
            <div className="bg-green-500 text-white px-3 py-1.5 rounded-lg shadow-lg flex items-center gap-2">
              <Users className="w-3 h-3" />
              <span className="text-xs font-medium">{joinNotification}</span>
            </div>
          </div>
        )}
        <div
          ref={miniContainerRef}
          className="fixed z-50 bg-gray-900 rounded-xl shadow-2xl overflow-hidden"
          style={{
            width: '320px',
            height: '220px',
            right: `${miniPosition.x}px`,
            bottom: `${miniPosition.y}px`,
            cursor: isDragging ? 'grabbing' : 'default',
          }}
        >
          <div
            className="flex items-center justify-between px-3 py-2 bg-gray-800 cursor-grab active:cursor-grabbing"
            onMouseDown={handleDragStart}
          >
            <div className="flex items-center gap-2 flex-1">
              <GripHorizontal className="w-4 h-4 text-gray-500" />
              <div
                className={`w-2 h-2 rounded-full ${
                  connectionState === 'connected'
                    ? 'bg-green-500'
                    : connectionState === 'connecting' || connectionState === 'reconnecting'
                    ? 'bg-yellow-500 animate-pulse'
                    : 'bg-red-500'
                }`}
              />
              <span className="text-white text-sm font-medium truncate flex-1">
                {roomInfo.name}
              </span>
            </div>
            <button
              onClick={toggleDisplayMode}
              className="p-1 rounded hover:bg-gray-700 text-white transition"
              title="전체 화면"
            >
              <Maximize2 className="w-4 h-4" />
            </button>
          </div>

          <div className="relative flex-1" style={{ height: 'calc(100% - 88px)' }}>
            <VideoTrackRenderer track={localVideoTrack} isLocal={true} isMirrored={isMirrored} />
            {!localVideoTrack && (
              <div className="absolute inset-0 flex items-center justify-center bg-gray-800">
                <div className="w-16 h-16 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-xl font-bold text-white border-2 border-gray-600">
                  {localP?.name?.[0]?.toUpperCase() || '나'}
                </div>
              </div>
            )}
            {/* Mini mode reconnecting overlay */}
            {(connectionState === 'reconnecting' || connectionState === 'disconnected') && (
              <div className="absolute inset-0 bg-black/70 flex items-center justify-center backdrop-blur-sm">
                <div className="text-center text-white">
                  <div className="w-8 h-8 mx-auto mb-2 relative">
                    <div className="absolute inset-0 border-2 border-blue-500/30 rounded-full" />
                    <div className="absolute inset-0 border-2 border-transparent border-t-blue-500 rounded-full animate-spin" />
                  </div>
                  <span className="text-xs">
                    {connectionState === 'reconnecting' ? '재연결 중...' : '연결 끊김'}
                  </span>
                </div>
              </div>
            )}
            <div className="absolute top-2 right-2 px-2 py-1 bg-black/50 rounded-full flex items-center gap-1">
              <Users className="w-3 h-3 text-white" />
              <span className="text-white text-xs">{participants.length}</span>
            </div>
          </div>

          <div className="flex items-center justify-center gap-2 py-2 bg-gray-800">
            <button
              onClick={toggleMute}
              className={`p-2 rounded-full transition ${
                isMuted
                  ? 'bg-red-500 hover:bg-red-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-white'
              }`}
            >
              {isMuted ? <MicOff className="w-4 h-4" /> : <Mic className="w-4 h-4" />}
            </button>

            <button
              onClick={toggleVideo}
              className={`p-2 rounded-full transition ${
                !isVideoEnabled
                  ? 'bg-red-500 hover:bg-red-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-white'
              }`}
            >
              {isVideoEnabled ? <Video className="w-4 h-4" /> : <VideoOff className="w-4 h-4" />}
            </button>

            <button
              onClick={handleLeave}
              className="p-2 rounded-full bg-red-500 hover:bg-red-600 text-white transition"
            >
              <PhoneOff className="w-4 h-4" />
            </button>
          </div>
        </div>
      </>
    );
  }

  // Full Mode
  return (
    <div className="fixed inset-0 bg-gray-900 z-50 flex flex-col">
      {/* Reconnecting overlay */}
      <ReconnectingOverlay />

      {joinNotification && (
        <div className="absolute top-16 right-4 z-50 animate-fade-in">
          <div className="bg-green-500 text-white px-4 py-2 rounded-lg shadow-lg flex items-center gap-2">
            <Users className="w-4 h-4" />
            <span className="text-sm font-medium">{joinNotification}</span>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-gray-800">
        <div className="flex items-center gap-3">
          <div
            className={`w-3 h-3 rounded-full ${
              connectionState === 'connected'
                ? 'bg-green-500'
                : connectionState === 'connecting'
                ? 'bg-yellow-500 animate-pulse'
                : 'bg-red-500'
            }`}
          />
          <h1 className="text-white font-medium">{roomInfo.name}</h1>
          <span className="text-gray-400 text-sm">({participants.length}명 참여중)</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={toggleDisplayMode}
            className="p-2 rounded-lg bg-gray-700 hover:bg-gray-600 text-white transition"
            title="미니 모드"
          >
            <Minimize2 className="w-5 h-5" />
          </button>
          <button
            onClick={copyRoomLink}
            className="flex items-center gap-2 px-3 py-1.5 bg-gray-700 hover:bg-gray-600 rounded-lg text-white text-sm transition"
          >
            {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
            {copied ? '복사됨!' : '초대 링크'}
          </button>
          <button
            onClick={() => setShowParticipants(!showParticipants)}
            className={`p-2 rounded-lg transition ${
              showParticipants
                ? 'bg-blue-600 text-white'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
          >
            <Users className="w-5 h-5" />
          </button>
          <button
            onClick={() => setShowChat(!showChat)}
            className={`p-2 rounded-lg transition relative ${
              showChat ? 'bg-blue-600 text-white' : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
          >
            <MessageSquare className="w-5 h-5" />
            {/* Unread badge could be added here */}
          </button>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex overflow-hidden">
        {/* Video Grid */}
        <div className="flex-1 p-4 overflow-y-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 auto-rows-max">
            {participants.map((p) => {
              const isLocal = p.identity === room?.localParticipant?.identity;
              const videoTrack = getParticipantVideoTrack(p);
              const metadata = getParticipantMetadata(p);
              const profileImageUrl = isLocal
                ? userProfile?.profileImageUrl
                : metadata?.profileImageUrl;
              const displayName = isLocal
                ? userProfile?.nickName || p.name || p.identity
                : metadata?.nickName || p.name || p.identity;
              // Check if audio is muted
              const isAudioMuted = p.getTrackPublication(Track.Source.Microphone)?.isMuted ?? true;
              const speaking = isSpeaking(p);

              return (
                <div
                  key={p.identity}
                  className={`relative aspect-video bg-gray-800 rounded-xl overflow-hidden shadow-lg transition-all duration-300 ${
                    speaking
                      ? 'ring-4 ring-green-500 ring-opacity-75 shadow-green-500/30'
                      : 'border border-gray-700'
                  }`}
                >
                  <VideoTrackRenderer
                    track={videoTrack}
                    isLocal={isLocal}
                    isMirrored={isLocal && isMirrored}
                  />

                  {/* Speaking indicator */}
                  {speaking && (
                    <div className="absolute top-3 right-3 px-2 py-1 bg-green-500 rounded-full flex items-center gap-1">
                      <div className="w-2 h-2 rounded-full bg-white animate-pulse" />
                      <span className="text-white text-xs font-medium">말하는 중</span>
                    </div>
                  )}

                  {/* Status Overlay */}
                  <div className="absolute bottom-3 left-3 flex items-center gap-2 bg-black/60 px-3 py-1.5 rounded-full backdrop-blur-sm">
                    <span className="text-white text-sm font-medium">
                      {displayName} {isLocal && '(나)'}
                    </span>
                    {isAudioMuted ? (
                      <MicOff className="w-3 h-3 text-red-400" />
                    ) : (
                      <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                    )}
                  </div>

                  {/* Avatar when no video */}
                  {!videoTrack && (
                    <div className="absolute inset-0 flex items-center justify-center">
                      {profileImageUrl ? (
                        <img
                          src={profileImageUrl}
                          alt={displayName}
                          className={`w-20 h-20 rounded-full object-cover border-4 ${
                            speaking ? 'border-green-500' : 'border-gray-600'
                          }`}
                        />
                      ) : (
                        <div
                          className={`w-20 h-20 rounded-full bg-gradient-to-br from-gray-600 to-gray-700 flex items-center justify-center text-2xl font-bold text-white border-4 ${
                            speaking ? 'border-green-500' : 'border-gray-600'
                          }`}
                        >
                          {displayName?.[0]?.toUpperCase() || '?'}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Right Sidebar (Chat/Participants) */}
        {(showParticipants || showChat) && (
          <div className="w-80 bg-gray-800 border-l border-gray-700 flex flex-col transition-all duration-300">
            {showParticipants && (
              <div
                className={`flex-1 flex flex-col ${
                  showChat ? 'h-1/2 border-b border-gray-700' : 'h-full'
                }`}
              >
                <div className="p-4 border-b border-gray-700 bg-gray-800/50">
                  <h3 className="text-white font-medium flex items-center gap-2">
                    <Users className="w-4 h-4" />
                    참여자 ({participants.length})
                  </h3>
                </div>
                <div className="flex-1 overflow-y-auto p-4 space-y-3">
                  {participants.map((p) => {
                    const isLocal = p.identity === room?.localParticipant?.identity;
                    const metadata = getParticipantMetadata(p);
                    const profileImageUrl = isLocal
                      ? userProfile?.profileImageUrl
                      : metadata?.profileImageUrl;
                    const displayName = isLocal
                      ? userProfile?.nickName || p.name || p.identity
                      : metadata?.nickName || p.name || p.identity;
                    const speaking = isSpeaking(p);
                    const isAudioMuted =
                      p.getTrackPublication(Track.Source.Microphone)?.isMuted ?? true;

                    return (
                      <div
                        key={p.identity}
                        className={`flex items-center justify-between p-2 rounded-lg transition-all ${
                          speaking
                            ? 'bg-green-500/20 ring-1 ring-green-500'
                            : 'hover:bg-gray-700/50'
                        }`}
                      >
                        <div className="flex items-center gap-3">
                          {profileImageUrl ? (
                            <img
                              src={profileImageUrl}
                              alt={displayName}
                              className={`w-8 h-8 rounded-full object-cover border-2 ${
                                speaking ? 'border-green-500' : 'border-transparent'
                              }`}
                            />
                          ) : (
                            <div
                              className={`w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-white text-sm font-medium border-2 ${
                                speaking ? 'border-green-500' : 'border-transparent'
                              }`}
                            >
                              {displayName?.[0]?.toUpperCase() || '?'}
                            </div>
                          )}
                          <div>
                            <p className="text-sm text-white font-medium">
                              {displayName}
                              {isLocal && <span className="text-gray-400 ml-1">(나)</span>}
                            </p>
                            <p className="text-xs text-gray-400">
                              {speaking ? (
                                <span className="text-green-400">말하는 중</span>
                              ) : isAudioMuted ? (
                                '음소거됨'
                              ) : (
                                '대기 중'
                              )}
                            </p>
                          </div>
                        </div>
                        {isAudioMuted ? (
                          <MicOff className="w-4 h-4 text-gray-500" />
                        ) : speaking ? (
                          <div className="flex items-center gap-1">
                            <div className="w-1 h-3 bg-green-500 rounded animate-pulse" />
                            <div className="w-1 h-4 bg-green-500 rounded animate-pulse delay-75" />
                            <div className="w-1 h-2 bg-green-500 rounded animate-pulse delay-150" />
                          </div>
                        ) : (
                          <Mic className="w-4 h-4 text-green-500" />
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {showChat && (
              <div className={`flex-1 flex flex-col ${showParticipants ? 'h-1/2' : 'h-full'}`}>
                <div className="p-4 border-b border-gray-700 bg-gray-800/50">
                  <h3 className="text-white font-medium flex items-center gap-2">
                    <MessageSquare className="w-4 h-4" />
                    채팅
                  </h3>
                </div>
                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                  {chatMessages.map((msg) => (
                    <div
                      key={msg.id}
                      className={`flex flex-col ${msg.isLocal ? 'items-end' : 'items-start'}`}
                    >
                      <div className="flex items-baseline gap-2 mb-1">
                        <span className="text-xs text-gray-300 font-medium">{msg.senderName}</span>
                        <span className="text-[10px] text-gray-500">
                          {formatChatTime(msg.timestamp)}
                        </span>
                      </div>
                      <div
                        className={`px-3 py-2 rounded-lg text-sm max-w-[85%] break-words ${
                          msg.isLocal
                            ? 'bg-blue-600 text-white rounded-br-none'
                            : 'bg-gray-700 text-gray-100 rounded-bl-none'
                        }`}
                      >
                        {msg.message}
                      </div>
                    </div>
                  ))}
                  <div ref={chatEndRef} />
                </div>
                <div className="p-4 bg-gray-800 border-t border-gray-700">
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      onKeyDown={handleChatKeyPress}
                      placeholder="메시지를 입력하세요..."
                      className="flex-1 bg-gray-700 text-white placeholder-gray-400 px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                    />
                    <button
                      onClick={sendChatMessage}
                      disabled={!chatInput.trim()}
                      className="p-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <Send className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Subtitles Overlay - Shows both local and remote subtitles */}
      {isSubtitleEnabled && (currentSubtitle || remoteSubtitles.size > 0) && (
        <div className="absolute bottom-24 left-1/2 transform -translate-x-1/2 z-50 flex flex-col gap-2 max-w-3xl w-full px-4">
          {/* Local subtitle */}
          {currentSubtitle && (
            <div className="animate-fade-in-up">
              <div className="bg-blue-600/80 backdrop-blur-md px-6 py-3 rounded-2xl text-white text-lg font-medium shadow-xl text-center">
                {currentSubtitle}
              </div>
            </div>
          )}
          {/* Remote subtitles */}
          {Array.from(remoteSubtitles.values()).map((subtitle) => (
            <div key={subtitle.speakerId} className="animate-fade-in-up">
              <div className="bg-gray-800/80 backdrop-blur-md px-6 py-3 rounded-2xl text-white text-lg font-medium shadow-xl text-center">
                <span className="text-yellow-400">{subtitle.speakerName}:</span> {subtitle.text}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Footer Controls */}
      <div className="h-20 bg-gray-800 border-t border-gray-700 px-4 flex items-center justify-between">
        <div className="flex items-center gap-4 text-white">
          <span className="text-xl font-bold tracking-tight">
            {new Date().toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })}
          </span>
          <div className="h-8 w-px bg-gray-600" />
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-300 truncate max-w-[200px]">{roomInfo.name}</span>
          </div>
        </div>

        <div className="flex items-center gap-3 absolute left-1/2 transform -translate-x-1/2">
          <button
            onClick={toggleMute}
            className={`p-3 rounded-full transition duration-200 transform hover:scale-105 ${
              isMuted
                ? 'bg-red-500 hover:bg-red-600 text-white shadow-lg shadow-red-500/20'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title={isMuted ? '마이크 켜기' : '마이크 끄기'}
          >
            {isMuted ? <MicOff className="w-6 h-6" /> : <Mic className="w-6 h-6" />}
          </button>

          <button
            onClick={toggleVideo}
            className={`p-3 rounded-full transition duration-200 transform hover:scale-105 ${
              !isVideoEnabled
                ? 'bg-red-500 hover:bg-red-600 text-white shadow-lg shadow-red-500/20'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title={isVideoEnabled ? '카메라 끄기' : '카메라 켜기'}
          >
            {isVideoEnabled ? <Video className="w-6 h-6" /> : <VideoOff className="w-6 h-6" />}
          </button>

          <button
            onClick={toggleScreenShare}
            className={`p-3 rounded-full transition duration-200 transform hover:scale-105 ${
              isScreenSharing
                ? 'bg-green-500 hover:bg-green-600 text-white shadow-lg shadow-green-500/20'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title="화면 공유"
          >
            <Monitor className="w-6 h-6" />
          </button>

          <button
            onClick={toggleSubtitle}
            className={`p-3 rounded-full transition duration-200 transform hover:scale-105 ${
              isSubtitleEnabled
                ? 'bg-purple-500 hover:bg-purple-600 text-white shadow-lg shadow-purple-500/20'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title="자막"
          >
            {isSubtitleEnabled ? (
              <Captions className="w-6 h-6" />
            ) : (
              <CaptionsOff className="w-6 h-6" />
            )}
          </button>

          <button
            onClick={() => setIsMirrored(!isMirrored)}
            className={`p-3 rounded-full transition duration-200 transform hover:scale-105 ${
              isMirrored
                ? 'bg-blue-500 hover:bg-blue-600 text-white shadow-lg shadow-blue-500/20'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title="좌우 반전"
          >
            <FlipHorizontal className="w-6 h-6" />
          </button>

          <button
            onClick={handleLeave}
            className="p-3 rounded-full bg-red-500 hover:bg-red-600 text-white shadow-lg shadow-red-500/20 transition duration-200 transform hover:scale-105 ml-4"
            title="통화 종료"
          >
            <PhoneOff className="w-6 h-6" />
          </button>
        </div>

        <div className="flex items-center gap-2">
          {/* Right side spacer or additional controls */}
        </div>
      </div>
    </div>
  );
};
