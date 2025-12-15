// src/components/layout/MainLayout.tsx

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { UserProfileResponse, WorkspaceMemberResponse } from '../../types/user';
import { getMyProfile } from '../../api/userService';
import { createOrGetDMChat, getMyChats } from '../../api/chatService';
import { videoService, VideoRoom as VideoRoomType } from '../../api/videoService';
import { Sidebar } from './Sidebar';
// import { ChatPanel } from '../chat/chatPanel';
import { ChatListPanel } from '../chat/ChatListPanel';
import { ChatPanel } from '../chat/ChatPanel';
import { NotificationPanel } from '../notification/NotificationPanel';
import { VideoCallPanel } from '../video/VideoCallPanel';
import { VideoRoom } from '../video/VideoRoom';
import { LogOut, UserIcon } from 'lucide-react';
import { usePresence } from '../../hooks/usePresence';
import { useNotifications } from '../../hooks/useNotifications';
import type { Notification } from '../../types/notification';

// 🔥 Render prop 타입: handleStartChat을 children에 전달
type StartChatHandler = (member: WorkspaceMemberResponse) => Promise<void>;

interface MainLayoutProps {
  onLogout: () => void;
  workspaceId: string;
  projectId?: string;
  children: React.ReactNode | ((handleStartChat: StartChatHandler) => React.ReactNode);
  onProfileModalOpen: () => void;
  onNotificationClick?: (notification: Notification) => void;
}

const MainLayout: React.FC<MainLayoutProps> = ({
  onLogout,
  workspaceId,
  // projectId,
  children,
  onProfileModalOpen,
  onNotificationClick,
}) => {
  const { theme } = useTheme();

  // States
  const [userProfile, setUserProfile] = useState<UserProfileResponse | null>(null);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [isLoadingProfile, setIsLoadingProfile] = useState(true);
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [isNotificationOpen, setIsNotificationOpen] = useState(false);
  const [isVideoOpen, setIsVideoOpen] = useState(false);
  const [activeChatId, setActiveChatId] = useState<string | null>(null);
  const [isLoadingChat, setIsLoadingChat] = useState(false);
  const [chatListRefreshKey, setChatListRefreshKey] = useState(0); // 🔥 채팅 목록 갱신용
  const [totalUnreadCount, setTotalUnreadCount] = useState(0); // 🔥 총 읽지 않은 메시지 수
  const [activeVideoRoomCount, setActiveVideoRoomCount] = useState(0); // 활성 영상통화방 수
  const [currentVideoRoom, setCurrentVideoRoom] = useState<{
    room: VideoRoomType;
    token: string;
    wsUrl: string;
  } | null>(null);

  // 알림 훅
  const {
    notifications,
    unreadCount: notificationUnreadCount,
    isLoading: isNotificationLoading,
    hasMore: hasMoreNotifications,
    loadMore: loadMoreNotifications,
    markNotificationAsRead,
    markAllNotificationsAsRead,
    removeNotification,
  } = useNotifications({ workspaceId, enabled: true });

  // Ref
  const userMenuRef = useRef<HTMLDivElement>(null);
  const refreshUnreadCountRef = useRef<() => void>(() => {}); // 🔥 Ref for callback
  const sidebarWidthPx = '5rem'; // 80px - CSS value for margin (sm: size)
  const chatPanelWidth = '20rem'; // 320px

  // 🔥 읽지 않은 메시지 수 확인
  const refreshUnreadCount = useCallback(async () => {
    try {
      const chats = await getMyChats();
      const filteredChats = chats.filter(
        (chat) => String(chat.workspaceId) === String(workspaceId),
      );
      // 🔥 총 읽지 않은 메시지 수 계산
      const total = filteredChats.reduce((sum, chat) => sum + (chat.unreadCount || 0), 0);
      setTotalUnreadCount(total);
    } catch (error) {
      console.error('Failed to check unread messages:', error);
    }
  }, [workspaceId]);

  // 🔥 Ref 업데이트 (usePresence에서 사용)
  useEffect(() => {
    refreshUnreadCountRef.current = refreshUnreadCount;
  }, [refreshUnreadCount]);

  // 🔥 Global Presence - 앱 접속 시 자동으로 온라인 상태 등록
  usePresence({
    onStatusChange: (data) => {
      if (data.type === 'USER_STATUS') {
        console.log(`👤 [Presence] User ${data.userId} is now ${data.payload?.status}`);
      }
      // 🔥 새 메시지 알림 수신 시 읽지 않은 카운트 즉시 갱신
      if (data.type === 'NEW_MESSAGE_NOTIFICATION') {
        console.log('📬 [Presence] New message notification received:', data);
        refreshUnreadCountRef.current();
      }
    },
  });

  // 프로필 로드
  useEffect(() => {
    const fetchUserProfile = async () => {
      try {
        const profile = await getMyProfile(workspaceId);
        setUserProfile(profile);
      } catch (e) {
        console.error('기본 프로필 로드 실패:', e);
      } finally {
        setIsLoadingProfile(false);
      }
    };
    fetchUserProfile();
  }, [workspaceId]);

  // 영상통화방 카운트 로드
  useEffect(() => {
    const fetchVideoRoomCount = async () => {
      try {
        const rooms = await videoService.getWorkspaceRooms(workspaceId, true);
        setActiveVideoRoomCount(rooms.length);
      } catch (e) {
        console.error('영상통화방 카운트 로드 실패:', e);
      }
    };
    fetchVideoRoomCount();
    const interval = setInterval(fetchVideoRoomCount, 10000); // 10초마다 갱신
    return () => clearInterval(interval);
  }, [workspaceId]);

  // 영상통화 참여 핸들러
  const handleJoinVideoRoom = (room: VideoRoomType, token: string, wsUrl: string) => {
    setCurrentVideoRoom({ room, token, wsUrl });
    setIsVideoOpen(false);
  };

  // 영상통화 종료 핸들러
  const handleLeaveVideoRoom = async () => {
    if (currentVideoRoom) {
      try {
        await videoService.leaveRoom(currentVideoRoom.room.id);
      } catch (e) {
        console.error('영상통화 종료 실패:', e);
      }
    }
    setCurrentVideoRoom(null);
    setIsVideoOpen(true); // 통화 종료 후 패널 다시 열기
  };

  // 영상통화 토큰 갱신 핸들러 (재연결 시 사용)
  const handleVideoTokenRefresh = useCallback(async () => {
    if (!currentVideoRoom || !userProfile) return null;
    try {
      console.log('[MainLayout] Refreshing video token for room:', currentVideoRoom.room.id);
      const response = await videoService.joinRoom(currentVideoRoom.room.id, userProfile.nickName);
      // Update current video room state with new token
      setCurrentVideoRoom({
        room: response.room,
        token: response.token,
        wsUrl: response.wsUrl,
      });
      return { token: response.token, wsUrl: response.wsUrl };
    } catch (e) {
      console.error('Failed to refresh video token:', e);
      return null;
    }
  }, [currentVideoRoom, userProfile]);

  useEffect(() => {
    refreshUnreadCount();

    // 🔥 5초마다 확인 (더 빠른 응답성)
    const interval = setInterval(refreshUnreadCount, 5000);

    // 🔥 탭이 다시 활성화될 때 즉시 갱신
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refreshUnreadCount();
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      clearInterval(interval);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [refreshUnreadCount, chatListRefreshKey]);

  // 🔥 채팅 패널 열 때 읽지 않은 카운트 갱신
  useEffect(() => {
    if (isChatOpen) {
      refreshUnreadCount();
    }
  }, [isChatOpen, refreshUnreadCount]);

  // 🔥 채팅방 열거나 닫을 때 읽지 않은 카운트 갱신
  useEffect(() => {
    // activeChatId가 null이 되면 (채팅방에서 나올 때) 즉시 갱신
    if (activeChatId === null) {
      refreshUnreadCount();
    } else {
      // 채팅방 진입 시 updateLastRead 완료 후 갱신 (약간의 딜레이)
      const timer = setTimeout(refreshUnreadCount, 500);
      return () => clearTimeout(timer);
    }
  }, [activeChatId, refreshUnreadCount]);

  // 🔥 채팅 시작 핸들러
  const handleStartChat = async (member: WorkspaceMemberResponse) => {
    setIsLoadingChat(true);
    try {
      console.log('🔵 채팅 시작:', member.nickName || member.userEmail);

      // 1. DM 채팅방 생성 또는 기존 채팅방 가져오기
      const chatId = await createOrGetDMChat(member.userId, workspaceId);
      console.log('✅ 채팅방 ID:', chatId);

      // 2. ChatPanel 열기
      setActiveChatId(chatId);
      setIsChatOpen(true);

      // 3. 🔥 채팅 목록 갱신 (새 채팅방이 목록에 표시되도록)
      setChatListRefreshKey((prev) => prev + 1);
    } catch (error) {
      console.error('❌ Failed to start chat:', error);
      alert('채팅방을 열 수 없습니다.');
    } finally {
      setIsLoadingChat(false);
    }
  };

  // 외부 클릭 감지 (UserMenu)
  useEffect(() => {
    if (!showUserMenu) return;

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest('[data-user-menu]')) {
        setShowUserMenu(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showUserMenu]);

  if (isLoadingProfile) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className={`min-h-screen flex ${theme.colors.background} relative`}>
      {/* 백그라운드 패턴 */}
      <div
        className="fixed inset-0 opacity-5"
        style={{
          backgroundImage:
            'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)',
          backgroundSize: '20px 20px',
        }}
      />

      {/* 사이드바 */}
      <Sidebar
        workspaceId={workspaceId}
        userProfile={userProfile}
        isChatActive={isChatOpen}
        isNotificationActive={isNotificationOpen}
        isVideoActive={isVideoOpen}
        onChatToggle={() => {
          setIsChatOpen(!isChatOpen);
          setIsNotificationOpen(false); // 채팅 열면 알림 닫기
          setIsVideoOpen(false); // 채팅 열면 영상통화 닫기
          if (isChatOpen) {
            setActiveChatId(null);
          }
        }}
        onNotificationToggle={() => {
          setIsNotificationOpen(!isNotificationOpen);
          setIsChatOpen(false); // 알림 열면 채팅 닫기
          setIsVideoOpen(false); // 알림 열면 영상통화 닫기
          setActiveChatId(null);
        }}
        onVideoToggle={() => {
          setIsVideoOpen(!isVideoOpen);
          setIsChatOpen(false); // 영상통화 열면 채팅 닫기
          setIsNotificationOpen(false); // 영상통화 열면 알림 닫기
          setActiveChatId(null);
        }}
        onUserMenuToggle={() => setShowUserMenu(!showUserMenu)}
        onStartChat={handleStartChat}
        totalUnreadCount={totalUnreadCount}
        notificationUnreadCount={notificationUnreadCount}
        activeVideoRoomCount={activeVideoRoomCount}
      />

      {/* 🔥 ChatPanel 또는 ChatList (왼쪽에 float) */}
      {isChatOpen && (
        <div className="fixed inset-0 z-40" onClick={() => setIsChatOpen(false)}>
          {/* 배경 오버레이 */}
          <div className="absolute inset-0 bg-black/20" />
          {/* 패널 */}
          <div
            className="absolute top-0 h-full bg-white shadow-2xl left-16 sm:left-20"
            style={{ width: chatPanelWidth }}
            onClick={(e) => e.stopPropagation()}
          >
            {activeChatId ? (
              <ChatPanel
                chatId={activeChatId}
                onClose={() => {
                  setActiveChatId(null);
                  setIsChatOpen(false);
                }}
                onBack={() => setActiveChatId(null)}
              />
            ) : (
              <ChatListPanel
                key={chatListRefreshKey}
                workspaceId={workspaceId}
                onChatSelect={(chatId) => setActiveChatId(chatId)}
                onClose={() => setIsChatOpen(false)}
                onUnreadCountChange={(count) => setTotalUnreadCount(count)}
              />
            )}
          </div>
        </div>
      )}

      {/* 알림 패널 */}
      <NotificationPanel
        isOpen={isNotificationOpen}
        onClose={() => setIsNotificationOpen(false)}
        notifications={notifications}
        unreadCount={notificationUnreadCount}
        isLoading={isNotificationLoading}
        hasMore={hasMoreNotifications}
        onLoadMore={loadMoreNotifications}
        onMarkAsRead={markNotificationAsRead}
        onMarkAllAsRead={markAllNotificationsAsRead}
        onDelete={removeNotification}
        onNotificationClick={onNotificationClick}
      />

      {/* 영상통화 패널 */}
      {isVideoOpen && (
        <VideoCallPanel
          workspaceId={workspaceId}
          userProfile={
            userProfile ? { id: userProfile.userId || '', nickName: userProfile.nickName } : null
          }
          onClose={() => setIsVideoOpen(false)}
          onJoinRoom={handleJoinVideoRoom}
          currentRoomId={currentVideoRoom?.room.id}
          onLeaveCurrentRoom={() => setCurrentVideoRoom(null)}
        />
      )}

      {/* 영상통화 룸 (전체화면) */}
      {currentVideoRoom && (
        <VideoRoom
          room={currentVideoRoom.room}
          token={currentVideoRoom.token}
          wsUrl={currentVideoRoom.wsUrl}
          onLeave={handleLeaveVideoRoom}
          onTokenRefresh={handleVideoTokenRefresh}
          userProfile={
            userProfile
              ? {
                  id: userProfile.userId || '',
                  nickName: userProfile.nickName,
                  profileImageUrl: userProfile.profileImageUrl,
                }
              : null
          }
        />
      )}

      {/* 메인 콘텐츠 영역 - Chat/Notification/Video는 float되므로 margin 변경 없음 */}
      <main
        className="flex-grow flex flex-col relative z-10"
        style={{
          marginLeft: sidebarWidthPx,
          minHeight: '100vh',
        }}
      >
        {/* 🔥 Render prop 지원: children이 함수면 handleStartChat 전달 */}
        {typeof children === 'function' ? children(handleStartChat) : children}
      </main>

      {/* 유저 메뉴 드롭다운 (사이드바 위에 팝업) */}
      {showUserMenu && (
        <div
          ref={userMenuRef}
          className={`absolute bottom-16 left-12 sm:left-16 w-64 ${theme.colors.card} ${theme.effects.cardBorderWidth} ${theme.colors.border} z-50 ${theme.effects.borderRadius} shadow-2xl`}
          onMouseDown={(e) => e.stopPropagation()} // 💡 [수정] 메뉴 내부 클릭 시 닫히는 현상 방지
        >
          <div className="p-3 pb-3 mb-2 border-b border-gray-200">
            <div className="flex items-center gap-3">
              <div
                className={`w-10 h-10 ${theme.colors.primary} flex items-center justify-center text-white text-base font-bold rounded-md overflow-hidden`}
              >
                {userProfile?.profileImageUrl ? (
                  <img
                    src={userProfile?.profileImageUrl}
                    alt={userProfile?.nickName}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  userProfile?.nickName[0]?.toUpperCase() || 'U'
                )}
              </div>
              <div>
                <h3 className="font-bold text-lg text-gray-900">{userProfile?.nickName}</h3>
                <div className="flex items-center text-green-600 text-xs mt-1">
                  <span className="w-2 h-2 bg-green-500 rounded-full mr-1"></span>
                  대화 가능
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-1 p-2 pt-0">
            <button
              onClick={() => {
                // 프로필 모달 열기
                onProfileModalOpen();
                setShowUserMenu(false);
              }}
              className="w-full text-left px-2 py-1.5 text-sm text-gray-800 hover:bg-blue-50 hover:text-blue-700 rounded transition flex items-center gap-2"
            >
              <UserIcon className="w-4 h-4" /> 프로필 설정
            </button>
          </div>

          <div className="pt-2 pb-2 border-t border-gray-200 mx-2">
            <button
              onClick={onLogout}
              className="w-full text-left px-2 py-1.5 text-sm text-gray-800 hover:bg-red-50 hover:text-red-700 rounded transition flex items-center gap-2"
            >
              <LogOut className="w-4 h-4" /> 로그아웃
            </button>
          </div>
        </div>
      )}

      {/* 🔥 채팅 로딩 오버레이 */}
      {isLoadingChat && (
        <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 shadow-xl">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto" />
            <p className="mt-3 text-sm text-gray-600">채팅방을 여는 중...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default MainLayout;
