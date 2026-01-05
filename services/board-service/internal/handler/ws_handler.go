package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"project-board-api/internal/client"
	"project-board-api/internal/database"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ============================================================================
// 💡 WebSocket 타임아웃 설정
// ============================================================================
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // 54초마다 ping
	maxMessageSize = 512
)

// getLogger is defined in error_handler.go with trace context support

type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	BoardID string      `json:"boardId,omitempty"`
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
}

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	projectID string
	userID    string // 🔥 온라인 상태 추적용
}

type WSHandler struct {
	Logger     *zap.Logger
	AuthClient client.UserClient
}

func NewWSHandler(log *zap.Logger, authClient client.UserClient) *WSHandler {
	return &WSHandler{
		Logger:     log,
		AuthClient: authClient,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients   = make(map[string]map[*Client]bool)
	clientsMu sync.RWMutex
	wsLogger  *zap.Logger // Package-level logger for WebSocket operations
)

// InitWSLogger initializes the package-level WebSocket logger
func InitWSLogger(log *zap.Logger) {
	wsLogger = log
}

// getWSLogger returns the package-level logger or a nop logger if not initialized
func getWSLogger() *zap.Logger {
	if wsLogger != nil {
		return wsLogger
	}
	return zap.NewNop()
}

// HandleWebSocket godoc
// @Summary      WebSocket 실시간 연결
// @Description  프로젝트의 실시간 이벤트를 구독하기 위한 WebSocket 연결을 설정합니다
// @Description  연결 후 BOARD_CREATED, BOARD_UPDATED, BOARD_MOVED, BOARD_DELETED 이벤트를 실시간으로 수신합니다
// @Description  인증은 쿼리 파라미터로 전달된 JWT 토큰을 통해 수행됩니다
// @Tags         websocket
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Param        token query string true "JWT Access Token"
// @Success      101 {string} string "Switching Protocols - WebSocket 연결 성공"
// @Failure      401 {string} string "인증 실패"
// @Failure      500 {string} string "서버 에러"
// @Router       /ws/project/{projectId} [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	projectID := c.Param("projectId")
	log := getLogger(c)

	log.Info("WebSocket connection attempt", zap.String("projectId", projectID))

	tokenStr := c.Query("token")
	if tokenStr == "" {
		log.Warn("WS connection attempt without token", zap.String("projectId", projectID))
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	authCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	userID, err := h.AuthClient.ValidateToken(authCtx, tokenStr)
	if err != nil {
		log.Error("WebSocket Auth Failed", zap.Error(err), zap.String("projectId", projectID))
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.Info("WebSocket auth successful", zap.String("projectId", projectID), zap.String("userId", userID.String()))

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket Upgrade Failed", zap.Error(err), zap.String("projectId", projectID))
		return
	}

	log.Info("WebSocket upgrade successful", zap.String("projectId", projectID))

	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		projectID: projectID,
		userID:    userID.String(), // 🔥 사용자 ID 저장
	}

	clientsMu.Lock()
	if clients[projectID] == nil {
		clients[projectID] = make(map[*Client]bool)
		log.Info("Created new client map for project", zap.String("projectId", projectID))
	}
	clients[projectID][client] = true
	currentClientCount := len(clients[projectID])
	clientsMu.Unlock()

	log.Info("WebSocket client registered",
		zap.String("projectId", projectID),
		zap.Int("totalClients", currentClientCount))

	go h.writePump(client, log)
	go h.readPump(client, log)
	go subscribeToRedis(projectID, client, log)

	<-c.Request.Context().Done()
	log.Info("WebSocket context done", zap.String("projectId", projectID))
}

// ============================================================================
// readPump: 클라이언트로부터 메시지 수신 + Pong 처리
// ============================================================================
func (h *WSHandler) readPump(client *Client, log *zap.Logger) {
	defer func() {
		log.Info("🔌 readPump: Client disconnected", zap.String("projectId", client.projectID))

		clientsMu.Lock()
		delete(clients[client.projectID], client)
		if len(clients[client.projectID]) == 0 {
			delete(clients, client.projectID)
		}
		clientsMu.Unlock()

		close(client.send)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))

	client.conn.SetPongHandler(func(string) error {
		log.Info("🏓 Pong received from client", zap.String("projectId", client.projectID))
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	log.Info("readPump started", zap.String("projectId", client.projectID))

	for {
		messageType, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("❌ Unexpected WebSocket close", zap.Error(err), zap.String("projectId", client.projectID))
			} else {
				log.Info("⚠️ Normal WebSocket close", zap.String("projectId", client.projectID))
			}
			break
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
					log.Info("🏓 Ping received (JSON), sending pong", zap.String("projectId", client.projectID))
					pongMsg, _ := json.Marshal(map[string]string{"type": "pong"})
					select {
					case client.send <- pongMsg:
					default:
						log.Warn("⚠️ Client send channel full", zap.String("projectId", client.projectID))
					}
					continue
				}
			}
		}

		log.Info("📨 Message received from client",
			zap.String("projectId", client.projectID),
			zap.Int("type", int(messageType)),
			zap.Int("length", len(message)))
	}

	log.Info("readPump exiting", zap.String("projectId", client.projectID))
}

// ============================================================================
// writePump: Ping 전송 + 메시지 전송
// ============================================================================
func (h *WSHandler) writePump(client *Client, log *zap.Logger) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
		log.Info("writePump exiting", zap.String("projectId", client.projectID))
	}()

	log.Info("writePump started", zap.String("projectId", client.projectID))

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error("❌ WriteMessage failed", zap.Error(err), zap.String("projectId", client.projectID))
				return
			}
			log.Info("✅ Message sent to client",
				zap.String("projectId", client.projectID),
				zap.String("message", string(message)))

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("❌ Ping failed", zap.Error(err), zap.String("projectId", client.projectID))
				return
			}
			log.Info("🏓 Ping sent to client", zap.String("projectId", client.projectID))
		}
	}
}

// ============================================================================
// subscribeToRedis: Redis Pub/Sub 구독
// ============================================================================
func subscribeToRedis(projectID string, client *Client, log *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered from panic in subscribeToRedis",
				zap.String("projectId", projectID),
				zap.Any("panic", r))
		}
	}()

	rdb := database.GetRedis()
	if rdb == nil {
		log.Warn("Redis client not available")
		return
	}

	ctx := context.Background()
	channel := fmt.Sprintf("kanban:project:%s", projectID)
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()

	log.Info("Redis subscription started", zap.String("projectId", projectID), zap.String("channel", channel))

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			log.Error("Redis subscription error", zap.Error(err), zap.String("projectId", projectID))
			return
		}

		select {
		case client.send <- []byte(msg.Payload):
			log.Debug("Message sent to WebSocket client",
				zap.String("projectId", projectID),
				zap.String("message", msg.Payload))
		case <-time.After(1 * time.Second):
			log.Warn("Failed to send message: channel full or closed",
				zap.String("projectId", projectID))
			return
		}
	}
}

// GetOnlineUsersForProject returns all online user IDs for a given project
// 🔥 프로젝트에 연결된 사용자 목록 반환
func GetOnlineUsersForProject(projectID string) []string {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	userSet := make(map[string]bool)
	if projectClients, ok := clients[projectID]; ok {
		for client := range projectClients {
			if client.userID != "" {
				userSet[client.userID] = true
			}
		}
	}

	users := make([]string, 0, len(userSet))
	for userID := range userSet {
		users = append(users, userID)
	}
	return users
}

// HandleGetOnlineUsers godoc
// @Summary      프로젝트 온라인 사용자 목록 조회
// @Description  현재 프로젝트에 WebSocket으로 연결된 온라인 사용자 목록을 반환합니다
// @Tags         websocket
// @Produce      json
// @Param        projectId path string true "Project ID (UUID)"
// @Success      200 {object} map[string]interface{} "onlineUsers: []string, count: int"
// @Router       /api/projects/{projectId}/online-users [get]
func (h *WSHandler) HandleGetOnlineUsers(c *gin.Context) {
	projectID := c.Param("projectId")
	log := getLogger(c)

	// 🔥 디버그: 현재 연결된 모든 프로젝트 로깅
	clientsMu.RLock()
	allProjects := make([]string, 0, len(clients))
	totalClients := 0
	for pid, projectClients := range clients {
		allProjects = append(allProjects, pid)
		totalClients += len(projectClients)
	}
	clientsMu.RUnlock()

	users := GetOnlineUsersForProject(projectID)

	log.Info("🔍 Online users requested",
		zap.String("requestedProjectId", projectID),
		zap.Int("onlineCount", len(users)),
		zap.Strings("users", users),
		zap.Strings("allConnectedProjects", allProjects),
		zap.Int("totalClientsAcrossProjects", totalClients))

	c.JSON(http.StatusOK, gin.H{
		"onlineUsers": users,
		"count":       len(users),
	})
}

// BroadcastEvent broadcasts a WebSocket event to all clients subscribed to the given project.
func BroadcastEvent(projectID string, event WSEvent) {
	log := getWSLogger()
	payload, _ := json.Marshal(event)

	clientsMu.RLock()
	defer clientsMu.RUnlock()

	log.Debug("Broadcasting event",
		zap.String("projectID", projectID),
		zap.Int("clientCount", len(clients[projectID])),
		zap.String("eventType", event.Type))

	if projectClients, ok := clients[projectID]; ok {
		for client := range projectClients {
			select {
			case client.send <- payload:
				log.Debug("Message sent to client")
			default:
				log.Warn("Client channel full, closing")
				close(client.send)
			}
		}
	} else {
		log.Debug("No clients found for project", zap.String("projectID", projectID))
	}

	redis := database.GetRedis()
	if redis != nil {
		redis.Publish(context.Background(), "kanban:project:"+projectID, payload)
	}
}
