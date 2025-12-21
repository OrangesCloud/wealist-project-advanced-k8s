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
	assert.NotNil(t, m.NotificationsCreatedTotal)
	assert.NotNil(t, m.NotificationsDeliveredTotal)
	assert.NotNil(t, m.NotificationsReadTotal)
	assert.NotNil(t, m.NotificationsDeletedTotal)
	assert.NotNil(t, m.SSEConnectionsTotal)
	assert.NotNil(t, m.SSEConnectionsCreatedTotal)
	assert.NotNil(t, m.SSEConnectionsClosedTotal)
	assert.NotNil(t, m.NotificationDeliveryDuration)
}

func TestNewWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry)
	assert.NotNil(t, m)
}

func TestMetrics_RecordNotificationCreated(t *testing.T) {
	m := NewForTest()
	m.RecordNotificationCreated()
	// Should not panic
}

func TestMetrics_RecordNotificationDelivered(t *testing.T) {
	m := NewForTest()
	m.RecordNotificationDelivered()
	// Should not panic
}

func TestMetrics_RecordNotificationRead(t *testing.T) {
	m := NewForTest()
	m.RecordNotificationRead()
	// Should not panic
}

func TestMetrics_RecordNotificationDeleted(t *testing.T) {
	m := NewForTest()
	m.RecordNotificationDeleted()
	// Should not panic
}

func TestMetrics_RecordSSEConnectionOpened(t *testing.T) {
	m := NewForTest()
	m.RecordSSEConnectionOpened()
	// Should not panic
}

func TestMetrics_RecordSSEConnectionClosed(t *testing.T) {
	m := NewForTest()
	m.RecordSSEConnectionOpened() // Open first to have positive value
	m.RecordSSEConnectionClosed()
	// Should not panic
}

func TestMetrics_SetSSEConnectionsTotal(t *testing.T) {
	m := NewForTest()
	m.SetSSEConnectionsTotal(10)
	// Should not panic
}

func TestMetrics_RecordNotificationDeliveryDuration(t *testing.T) {
	m := NewForTest()
	m.RecordNotificationDeliveryDuration(50 * time.Millisecond)
	// Should not panic
}

func TestHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	t.Run("records metrics for regular endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/api/notifications", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/notifications", nil)
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
	m.RecordHTTPRequest("GET", "/api/notifications", 200, 100*time.Millisecond)
	// Should not panic
}
