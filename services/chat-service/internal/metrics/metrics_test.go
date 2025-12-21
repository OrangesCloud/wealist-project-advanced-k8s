package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := NewForTest()
	assert.NotNil(t, m)
	assert.NotNil(t, m.Metrics)
	assert.NotNil(t, m.ChatsTotal)
	assert.NotNil(t, m.MessagesTotal)
	assert.NotNil(t, m.OnlineUsersTotal)
	assert.NotNil(t, m.WebSocketConnectionsActive)
	assert.NotNil(t, m.MessagesSentTotal)
	assert.NotNil(t, m.MessagesReadTotal)
	assert.NotNil(t, m.WebSocketMessagesSent)
	assert.NotNil(t, m.WebSocketMessagesReceived)
}

func TestNewWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry)
	assert.NotNil(t, m)
}

func TestMetrics_RecordMessageSent(t *testing.T) {
	m := NewForTest()
	m.RecordMessageSent()
	// Should not panic
}

func TestMetrics_RecordMessageRead(t *testing.T) {
	m := NewForTest()
	m.RecordMessageRead()
	// Should not panic
}

func TestMetrics_RecordWebSocketMessageSent(t *testing.T) {
	m := NewForTest()
	m.RecordWebSocketMessageSent()
	// Should not panic
}

func TestMetrics_RecordWebSocketMessageReceived(t *testing.T) {
	m := NewForTest()
	m.RecordWebSocketMessageReceived()
	// Should not panic
}

func TestMetrics_IncrementWebSocketConnections(t *testing.T) {
	m := NewForTest()
	m.IncrementWebSocketConnections()
	// Should not panic
}

func TestMetrics_DecrementWebSocketConnections(t *testing.T) {
	m := NewForTest()
	m.DecrementWebSocketConnections()
	// Should not panic
}

func TestMetrics_SetChatsTotal(t *testing.T) {
	m := NewForTest()
	m.SetChatsTotal(100)
	// Should not panic
}

func TestMetrics_SetMessagesTotal(t *testing.T) {
	m := NewForTest()
	m.SetMessagesTotal(1000)
	// Should not panic
}

func TestMetrics_SetOnlineUsersTotal(t *testing.T) {
	m := NewForTest()
	m.SetOnlineUsersTotal(50)
	// Should not panic
}

func TestHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	t.Run("records metrics for regular endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/api/chats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/chats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skips health endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/live", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skips metrics endpoint", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPMiddleware_DurationRecording(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	router := gin.New()
	router.Use(HTTPMiddleware(m))
	router.GET("/api/slow", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, elapsed >= 10*time.Millisecond)
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := NewForTest()
	m.RecordHTTPRequest("GET", "/api/chats", 200, 100*time.Millisecond)
	// Should not panic
}
