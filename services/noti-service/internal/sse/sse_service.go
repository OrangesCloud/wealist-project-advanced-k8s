package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type SSEClient struct {
	UserID     uuid.UUID
	Writer     gin.ResponseWriter
	Flusher    http.Flusher
	Done       chan struct{}
	cancelFunc context.CancelFunc
}

type SSEService struct {
	clients map[string][]*SSEClient // userID -> clients
	mu      sync.RWMutex
	redis   *redis.Client
	logger  *zap.Logger
}

func NewSSEService(redis *redis.Client, logger *zap.Logger) *SSEService {
	return &SSEService{
		clients: make(map[string][]*SSEClient),
		redis:   redis,
		logger:  logger,
	}
}

func (s *SSEService) AddClient(c *gin.Context, userID uuid.UUID) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())

	client := &SSEClient{
		UserID:     userID,
		Writer:     c.Writer,
		Flusher:    flusher,
		Done:       make(chan struct{}),
		cancelFunc: cancel,
	}

	// Add client to map
	s.mu.Lock()
	userKey := userID.String()
	s.clients[userKey] = append(s.clients[userKey], client)
	s.mu.Unlock()

	s.logger.Info("SSE client connected",
		zap.String("userId", userID.String()),
		zap.Int("totalClients", s.GetConnectedClientsCount()),
	)

	// Send initial connected event
	s.sendEvent(client, "connected", map[string]string{"status": "connected"})

	// Start Redis subscription for this user
	go s.subscribeToUserChannel(ctx, client)

	// Start keep-alive ping
	go s.startPingLoop(ctx, client)

	// Wait for client disconnect
	<-c.Request.Context().Done()

	// Cleanup
	s.removeClient(userID, client)
	cancel()
	close(client.Done)
}

func (s *SSEService) subscribeToUserChannel(ctx context.Context, client *SSEClient) {
	if s.redis == nil {
		return
	}

	channel := fmt.Sprintf("notifications:user:%s", client.UserID.String())
	pubsub := s.redis.Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var notification map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
				s.logger.Error("failed to unmarshal notification", zap.Error(err))
				continue
			}

			s.sendEvent(client, "notification", notification)
		}
	}
}

func (s *SSEService) startPingLoop(ctx context.Context, client *SSEClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendPing(client)
		}
	}
}

func (s *SSEService) sendEvent(client *SSEClient, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("failed to marshal SSE data", zap.Error(err))
		return
	}

	select {
	case <-client.Done:
		return
	default:
		_, _ = fmt.Fprintf(client.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
		client.Flusher.Flush()
	}
}

func (s *SSEService) sendPing(client *SSEClient) {
	select {
	case <-client.Done:
		return
	default:
		_, _ = fmt.Fprintf(client.Writer, ":ping\n\n")
		client.Flusher.Flush()
	}
}

func (s *SSEService) removeClient(userID uuid.UUID, client *SSEClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userKey := userID.String()
	clients := s.clients[userKey]

	for i, c := range clients {
		if c == client {
			s.clients[userKey] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(s.clients[userKey]) == 0 {
		delete(s.clients, userKey)
	}

	s.logger.Info("SSE client disconnected",
		zap.String("userId", userID.String()),
		zap.Int("totalClients", s.getConnectedClientsCountLocked()),
	)
}

func (s *SSEService) SendToUser(userID uuid.UUID, event string, data interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := s.clients[userID.String()]
	for _, client := range clients {
		s.sendEvent(client, event, data)
	}
}

func (s *SSEService) GetConnectedClientsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getConnectedClientsCountLocked()
}

func (s *SSEService) getConnectedClientsCountLocked() int {
	count := 0
	for _, clients := range s.clients {
		count += len(clients)
	}
	return count
}

func (s *SSEService) GetUserClientsCount(userID uuid.UUID) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients[userID.String()])
}
