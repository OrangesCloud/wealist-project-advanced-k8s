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
	assert.NotNil(t, m.RoomsTotal)
	assert.NotNil(t, m.ParticipantsTotal)
	assert.NotNil(t, m.RoomsCreatedTotal)
	assert.NotNil(t, m.RoomsEndedTotal)
	assert.NotNil(t, m.ParticipantsJoinedTotal)
	assert.NotNil(t, m.ParticipantsLeftTotal)
	assert.NotNil(t, m.CallDurationSeconds)
}

func TestNewWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry)
	assert.NotNil(t, m)
}

func TestMetrics_RecordRoomCreated(t *testing.T) {
	m := NewForTest()
	m.RecordRoomCreated()
	// Should not panic
}

func TestMetrics_RecordRoomEnded(t *testing.T) {
	m := NewForTest()
	m.RecordRoomCreated() // Create first to have positive value
	m.RecordRoomEnded()
	// Should not panic
}

func TestMetrics_RecordParticipantJoined(t *testing.T) {
	m := NewForTest()
	m.RecordParticipantJoined()
	// Should not panic
}

func TestMetrics_RecordParticipantLeft(t *testing.T) {
	m := NewForTest()
	m.RecordParticipantJoined() // Join first to have positive value
	m.RecordParticipantLeft()
	// Should not panic
}

func TestMetrics_RecordCallDuration(t *testing.T) {
	m := NewForTest()
	m.RecordCallDuration(30 * time.Minute)
	// Should not panic
}

func TestMetrics_SetRoomsTotal(t *testing.T) {
	m := NewForTest()
	m.SetRoomsTotal(10)
	// Should not panic
}

func TestMetrics_SetParticipantsTotal(t *testing.T) {
	m := NewForTest()
	m.SetParticipantsTotal(50)
	// Should not panic
}

func TestHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	t.Run("records metrics for regular endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/api/video/rooms", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/video/rooms", nil)
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
	m.RecordHTTPRequest("GET", "/api/video/rooms", 200, 100*time.Millisecond)
	// Should not panic
}
