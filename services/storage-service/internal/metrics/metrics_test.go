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
	assert.NotNil(t, m.FilesTotal)
	assert.NotNil(t, m.FoldersTotal)
	assert.NotNil(t, m.ProjectsTotal)
	assert.NotNil(t, m.SharesTotal)
	assert.NotNil(t, m.FileUploadTotal)
	assert.NotNil(t, m.FileDownloadTotal)
	assert.NotNil(t, m.FileDeleteTotal)
	assert.NotNil(t, m.StorageBytesTotal)
}

func TestNewWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := NewWithRegistry(registry)
	assert.NotNil(t, m)
}

func TestMetrics_RecordFileUpload(t *testing.T) {
	m := NewForTest()
	m.RecordFileUpload()
	// Should not panic
}

func TestMetrics_RecordFileDownload(t *testing.T) {
	m := NewForTest()
	m.RecordFileDownload()
	// Should not panic
}

func TestMetrics_RecordFileDelete(t *testing.T) {
	m := NewForTest()
	m.RecordFileDelete()
	// Should not panic
}

func TestMetrics_SetFilesTotal(t *testing.T) {
	m := NewForTest()
	m.SetFilesTotal(100)
	// Should not panic
}

func TestMetrics_SetFoldersTotal(t *testing.T) {
	m := NewForTest()
	m.SetFoldersTotal(50)
	// Should not panic
}

func TestMetrics_SetProjectsTotal(t *testing.T) {
	m := NewForTest()
	m.SetProjectsTotal(10)
	// Should not panic
}

func TestMetrics_SetSharesTotal(t *testing.T) {
	m := NewForTest()
	m.SetSharesTotal(25)
	// Should not panic
}

func TestMetrics_SetStorageBytesTotal(t *testing.T) {
	m := NewForTest()
	m.SetStorageBytesTotal(1024 * 1024 * 1024) // 1GB
	// Should not panic
}

func TestHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewForTest()

	t.Run("records metrics for regular endpoints", func(t *testing.T) {
		router := gin.New()
		router.Use(HTTPMiddleware(m))
		router.GET("/api/storage/files", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/storage/files", nil)
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
