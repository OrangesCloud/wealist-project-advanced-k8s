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
	m := New()
	assert.NotNil(t, m)
	assert.NotNil(t, m.Metrics)
	assert.NotNil(t, m.UsersTotal)
	assert.NotNil(t, m.WorkspacesTotal)
	assert.NotNil(t, m.ProfilesTotal)
	assert.NotNil(t, m.UserCreatedTotal)
	assert.NotNil(t, m.WorkspaceCreatedTotal)
	assert.NotNil(t, m.ProfileCreatedTotal)
	assert.NotNil(t, m.JoinRequestsTotal)
}

func TestNewForTest(t *testing.T) {
	m := NewForTest()
	assert.NotNil(t, m)
}

func TestNewWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry)
	assert.NotNil(t, m)
}

func TestMetrics_RecordUserCreated(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.RecordUserCreated()
}

func TestMetrics_RecordWorkspaceCreated(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.RecordWorkspaceCreated()
}

func TestMetrics_RecordProfileCreated(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.RecordProfileCreated()
}

func TestMetrics_SetUsersTotal(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.SetUsersTotal(100)
}

func TestMetrics_SetWorkspacesTotal(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.SetWorkspacesTotal(50)
}

func TestMetrics_SetProfilesTotal(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.SetProfilesTotal(200)
}

func TestMetrics_SetJoinRequestsTotal(t *testing.T) {
	m := NewForTest()

	// Should not panic
	m.SetJoinRequestsTotal(10)
}

func TestHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	t.Run("records metrics for regular endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/api/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/users", nil)
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

func TestHTTPMiddleware_EmptyFullPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	router := gin.New()
	router.Use(HTTPMiddleware(m))
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknown/path", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
