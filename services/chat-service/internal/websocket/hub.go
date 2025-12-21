package websocket

import (
	"chat-service/internal/domain"
	"chat-service/internal/middleware"
	"chat-service/internal/response"
	"chat-service/internal/service"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev
	},
}

type Client struct {
	Hub         *Hub
	Conn        *websocket.Conn
	Send        chan []byte
	UserID      uuid.UUID
	ChatID      uuid.UUID
	WorkspaceID uuid.UUID
}

type Hub struct {
	chatRooms       map[uuid.UUID]map[*Client]bool // chatID -> clients
	userConnections map[uuid.UUID]map[*Client]bool // userID -> clients
	mu              sync.RWMutex
	chatService     *service.ChatService
	presenceService *service.PresenceService
	validator       middleware.TokenValidator
	redis           *redis.Client
	logger          *zap.Logger
}

func NewHub(
	chatService *service.ChatService,
	presenceService *service.PresenceService,
	validator middleware.TokenValidator,
	redis *redis.Client,
	logger *zap.Logger,
) *Hub {
	hub := &Hub{
		chatRooms:       make(map[uuid.UUID]map[*Client]bool),
		userConnections: make(map[uuid.UUID]map[*Client]bool),
		chatService:     chatService,
		presenceService: presenceService,
		validator:       validator,
		redis:           redis,
		logger:          logger,
	}

	// Start Redis subscription handler
	if redis != nil {
		go hub.subscribeToRedis()
	}

	return hub
}

func (h *Hub) subscribeToRedis() {
	ctx := context.Background()
	pubsub := h.redis.PSubscribe(ctx, "chat:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Parse chat ID from channel name
		var chatIDStr string
		fmt.Sscanf(msg.Channel, "chat:%s", &chatIDStr)
		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			continue
		}

		h.broadcastToChat(chatID, []byte(msg.Payload))
	}
}

func (h *Hub) HandleChatWebSocket(c *gin.Context) {
	// Get token from query param
	token := c.Query("token")
	if token == "" {
		response.Unauthorized(c, "Token required")
		return
	}

	// Validate token
	userID, err := h.validator.ValidateToken(c.Request.Context(), token)
	if err != nil {
		response.Unauthorized(c, "Invalid token")
		return
	}

	// Get chat ID
	chatIDStr := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid chat ID")
		return
	}

	// Verify user is in chat
	inChat, err := h.chatService.IsUserInChat(c.Request.Context(), chatID, userID)
	if err != nil || !inChat {
		response.Forbidden(c, "Not a participant")
		return
	}

	// Get chat for workspace ID
	chat, err := h.chatService.GetChatByID(c.Request.Context(), chatID)
	if err != nil {
		response.NotFound(c, "Chat not found")
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:         h,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		UserID:      userID,
		ChatID:      chatID,
		WorkspaceID: chat.WorkspaceID,
	}

	h.registerClient(client)

	// Set user online
	h.presenceService.SetUserOnline(c.Request.Context(), userID, chat.WorkspaceID)

	go client.writePump()
	go client.readPump()
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to chat room
	if h.chatRooms[client.ChatID] == nil {
		h.chatRooms[client.ChatID] = make(map[*Client]bool)
	}
	h.chatRooms[client.ChatID][client] = true

	// Add to user connections
	if h.userConnections[client.UserID] == nil {
		h.userConnections[client.UserID] = make(map[*Client]bool)
	}
	h.userConnections[client.UserID][client] = true

	h.logger.Info("Client connected",
		zap.String("userId", client.UserID.String()),
		zap.String("chatId", client.ChatID.String()),
	)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from chat room
	if clients, ok := h.chatRooms[client.ChatID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.chatRooms, client.ChatID)
		}
	}

	// Remove from user connections
	if clients, ok := h.userConnections[client.UserID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.userConnections, client.UserID)
			// User has no more connections, set offline
			h.presenceService.SetUserOffline(context.Background(), client.UserID, client.WorkspaceID)
		}
	}

	close(client.Send)

	h.logger.Info("Client disconnected",
		zap.String("userId", client.UserID.String()),
		zap.String("chatId", client.ChatID.String()),
	)
}

func (h *Hub) broadcastToChat(chatID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.chatRooms[chatID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(clients, client)
			}
		}
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.userConnections[userID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		c.handleMessage(message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Close()

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var msg struct {
		Type        string  `json:"type"`
		Content     string  `json:"content,omitempty"`
		MessageType string  `json:"messageType,omitempty"`
		ChatID      string  `json:"chatId,omitempty"`
		MessageID   string  `json:"messageId,omitempty"`
		FileURL     *string `json:"fileUrl,omitempty"`
		FileName    *string `json:"fileName,omitempty"`
		FileSize    *int64  `json:"fileSize,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("INVALID_MESSAGE", "Invalid message format")
		return
	}

	ctx := context.Background()

	switch msg.Type {
	case "MESSAGE":
		messageType := domain.MessageTypeText
		if msg.MessageType != "" {
			messageType = domain.MessageType(msg.MessageType)
		}

		message, err := c.Hub.chatService.SendMessage(ctx, c.ChatID, c.UserID, &domain.SendMessageRequest{
			Content:     msg.Content,
			MessageType: messageType,
			FileURL:     msg.FileURL,
			FileName:    msg.FileName,
			FileSize:    msg.FileSize,
		})
		if err != nil {
			c.sendError("SEND_FAILED", "Failed to send message")
			return
		}

		// Broadcast via Redis (already handled in service)
		// But also send locally for immediate feedback
		response, _ := json.Marshal(map[string]interface{}{
			"type":    "MESSAGE_RECEIVED",
			"message": message,
		})
		c.Hub.broadcastToChat(c.ChatID, response)

	case "TYPING_START":
		response, _ := json.Marshal(map[string]interface{}{
			"type":   "USER_TYPING",
			"userId": c.UserID.String(),
			"chatId": c.ChatID.String(),
		})
		c.Hub.broadcastToChat(c.ChatID, response)

	case "TYPING_STOP":
		response, _ := json.Marshal(map[string]interface{}{
			"type":   "USER_TYPING_STOP",
			"userId": c.UserID.String(),
			"chatId": c.ChatID.String(),
		})
		c.Hub.broadcastToChat(c.ChatID, response)

	case "READ_MESSAGE":
		messageID, err := uuid.Parse(msg.MessageID)
		if err != nil {
			c.sendError("INVALID_MESSAGE_ID", "Invalid message ID")
			return
		}

		c.Hub.chatService.MarkMessagesAsRead(ctx, []uuid.UUID{messageID}, c.UserID)
		c.Hub.chatService.UpdateLastReadAt(ctx, c.ChatID, c.UserID)

		response, _ := json.Marshal(map[string]interface{}{
			"type":      "MESSAGE_READ",
			"messageId": messageID.String(),
			"userId":    c.UserID.String(),
		})
		c.Hub.broadcastToChat(c.ChatID, response)
	}
}

func (c *Client) sendError(code, message string) {
	response, _ := json.Marshal(map[string]interface{}{
		"type":    "ERROR",
		"code":    code,
		"message": message,
	})
	c.Send <- response
}

// PresenceClient represents a connected client for presence tracking
type PresenceClient struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID uuid.UUID
}

// HandlePresenceWebSocket handles global presence WebSocket connections
func (h *Hub) HandlePresenceWebSocket(c *gin.Context) {
	// Get token from query param
	token := c.Query("token")
	if token == "" {
		response.Unauthorized(c, "Token required")
		return
	}

	// Validate token
	userID, err := h.validator.ValidateToken(c.Request.Context(), token)
	if err != nil {
		response.Unauthorized(c, "Invalid token")
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Presence WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &PresenceClient{
		Hub:    h,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	h.registerPresenceClient(client)

	// Set user online (with nil workspaceID for global presence)
	h.presenceService.SetUserOnline(c.Request.Context(), userID, uuid.Nil)

	h.logger.Info("Presence client connected", zap.String("userId", userID.String()))

	go client.presenceWritePump()
	go client.presenceReadPump()
}

// presenceClients stores presence-only clients
var presenceClients = struct {
	sync.RWMutex
	clients map[uuid.UUID]map[*PresenceClient]bool
}{clients: make(map[uuid.UUID]map[*PresenceClient]bool)}

func (h *Hub) registerPresenceClient(client *PresenceClient) {
	presenceClients.Lock()
	defer presenceClients.Unlock()

	if presenceClients.clients[client.UserID] == nil {
		presenceClients.clients[client.UserID] = make(map[*PresenceClient]bool)
	}
	presenceClients.clients[client.UserID][client] = true
}

func (h *Hub) unregisterPresenceClient(client *PresenceClient) {
	presenceClients.Lock()
	defer presenceClients.Unlock()

	if clients, ok := presenceClients.clients[client.UserID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(presenceClients.clients, client.UserID)
			// User has no more presence connections, set offline
			h.presenceService.SetUserOffline(context.Background(), client.UserID, uuid.Nil)
			h.logger.Info("Presence client disconnected - user now offline", zap.String("userId", client.UserID.String()))
		}
	}

	close(client.Send)
}

func (pc *PresenceClient) presenceReadPump() {
	defer func() {
		pc.Hub.unregisterPresenceClient(pc)
		pc.Conn.Close()
	}()

	pc.Conn.SetReadLimit(512 * 1024) // 512KB
	pc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	pc.Conn.SetPongHandler(func(string) error {
		pc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := pc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				pc.Hub.logger.Error("Presence WebSocket read error", zap.Error(err))
			}
			break
		}

		// Handle heartbeat
		var msg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg.Type == "heartbeat" {
				// Refresh presence
				pc.Hub.presenceService.SetUserOnline(context.Background(), pc.UserID, uuid.Nil)
				// Send pong
				response, _ := json.Marshal(map[string]string{"type": "pong"})
				select {
				case pc.Send <- response:
				default:
				}
			}
		}
	}
}

func (pc *PresenceClient) presenceWritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		pc.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-pc.Send:
			pc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				pc.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := pc.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Close()

		case <-ticker.C:
			pc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := pc.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
