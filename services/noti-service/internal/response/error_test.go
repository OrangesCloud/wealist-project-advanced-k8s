package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		appErr := apperrors.New(apperrors.ErrCodeNotFound, "notification not found", "")
		Error(c, appErr)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp["success"].(bool))

	errorData := resp["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errorData["code"])
	assert.Equal(t, "notification not found", errorData["message"])
}

func TestBadRequest(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		BadRequest(c, "invalid input")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidationError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		ValidationError(c, "validation failed")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnauthorized(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		Unauthorized(c, "invalid token")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestForbidden(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		Forbidden(c, "access denied")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestNotFound(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		NotFound(c, "notification not found")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInternalError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		InternalError(c, "something went wrong")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSuccess(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		Success(c, "operation completed")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "operation completed", resp["message"])
}

func TestOK(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		OK(c, gin.H{"status": "healthy"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestOKWithPagination(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		OKWithPagination(c, []string{"item1", "item2"}, 100, 1, 20)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.Equal(t, float64(100), resp["total"])
	assert.Equal(t, float64(1), resp["page"])
	assert.Equal(t, float64(20), resp["limit"])
}

func TestCreated(t *testing.T) {
	router := setupTestRouter()
	router.POST("/test", func(c *gin.Context) {
		Created(c, gin.H{"id": "123"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["success"].(bool))
}

func TestNoContent(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/test", func(c *gin.Context) {
		NoContent(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCustomError(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		CustomError(c, http.StatusConflict, "DUPLICATE", "Notification already exists")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp["success"].(bool))

	errorData := resp["error"].(map[string]interface{})
	assert.Equal(t, "DUPLICATE", errorData["code"])
	assert.Equal(t, "Notification already exists", errorData["message"])
}
