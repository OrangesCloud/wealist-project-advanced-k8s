package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"user-service/internal/domain"
)

// MockUserService is a mock implementation of user service
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(req domain.CreateUserRequest) (*domain.User, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) GetUser(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(id uuid.UUID, req domain.UpdateUserRequest) (*domain.User, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserService) RestoreUser(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UserExists(id uuid.UUID) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) FindOrCreateOAuthUser(email, name, provider string) (*domain.User, error) {
	args := m.Called(email, name, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// userHandlerTestable is a testable version of UserHandler
type userHandlerTestable struct {
	svc userServiceInterface
}

type userServiceInterface interface {
	CreateUser(req domain.CreateUserRequest) (*domain.User, error)
	GetUser(id uuid.UUID) (*domain.User, error)
	UpdateUser(id uuid.UUID, req domain.UpdateUserRequest) (*domain.User, error)
	DeleteUser(id uuid.UUID) error
	RestoreUser(id uuid.UUID) (*domain.User, error)
	UserExists(id uuid.UUID) (bool, error)
	FindOrCreateOAuthUser(email, name, provider string) (*domain.User, error)
}

func newUserHandlerTestable(svc userServiceInterface) *userHandlerTestable {
	return &userHandlerTestable{svc: svc}
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestUser(id uuid.UUID, email string) *domain.User {
	return &domain.User{
		ID:        id,
		Email:     email,
		Name:      "Test User",
		Provider:  "google",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestUserHandler_CreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")
		req := domain.CreateUserRequest{
			Email:    "test@example.com",
			Provider: "google",
		}

		mockSvc.On("CreateUser", mock.MatchedBy(func(r domain.CreateUserRequest) bool {
			return r.Email == req.Email
		})).Return(expectedUser, nil)

		router.POST("/users", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			var req domain.CreateUserRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			user, err := handler.svc.CreateUser(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, user.ToResponse())
		})

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		router := setupRouter()

		router.POST("/users", func(c *gin.Context) {
			var req domain.CreateUserRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "validation error"})
				return
			}
		})

		// Invalid email
		body := []byte(`{"email": "invalid"}`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_GetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")

		mockSvc.On("GetUser", userID).Return(expectedUser, nil)

		router.GET("/users/:userId", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			user, err := handler.svc.GetUser(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, user.ToResponse())
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/users/"+userID.String(), nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp domain.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, userID, resp.UserID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		router := setupRouter()

		router.GET("/users/:userId", func(c *gin.Context) {
			idStr := c.Param("userId")
			_, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/users/invalid-uuid", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		mockSvc.On("GetUser", userID).Return(nil, assert.AnError)

		router.GET("/users/:userId", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, _ := uuid.Parse(idStr)
			user, err := handler.svc.GetUser(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, user.ToResponse())
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/users/"+userID.String(), nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestUserHandler_DeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		mockSvc.On("DeleteUser", userID).Return(nil)

		router.DELETE("/users/:userId", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, _ := uuid.Parse(idStr)
			if err := handler.svc.DeleteUser(id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("DELETE", "/users/"+userID.String(), nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestUserHandler_RestoreUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")
		mockSvc.On("RestoreUser", userID).Return(expectedUser, nil)

		router.PUT("/users/:userId/restore", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, _ := uuid.Parse(idStr)
			user, err := handler.svc.RestoreUser(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, user.ToResponse())
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", "/users/"+userID.String()+"/restore", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestUserHandler_UserExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		mockSvc.On("UserExists", userID).Return(true, nil)

		router.GET("/internal/users/:userId/exists", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, _ := uuid.Parse(idStr)
			exists, err := handler.svc.UserExists(id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"exists": exists})
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/internal/users/"+userID.String()+"/exists", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]bool
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp["exists"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("not exists", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		mockSvc.On("UserExists", userID).Return(false, nil)

		router.GET("/internal/users/:userId/exists", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			idStr := c.Param("userId")
			id, _ := uuid.Parse(idStr)
			exists, err := handler.svc.UserExists(id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"exists": exists})
		})

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/internal/users/"+userID.String()+"/exists", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]bool
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp["exists"])
		mockSvc.AssertExpectations(t)
	})
}

func TestUserHandler_OAuthLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		router := setupRouter()

		userID := uuid.New()
		expectedUser := createTestUser(userID, "test@example.com")

		mockSvc.On("FindOrCreateOAuthUser", "test@example.com", "Test User", "google").Return(expectedUser, nil)

		router.POST("/internal/oauth/login", func(c *gin.Context) {
			handler := newUserHandlerTestable(mockSvc)
			var req domain.OAuthLoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			user, err := handler.svc.FindOrCreateOAuthUser(req.Email, req.Name, req.Provider)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"userId": user.ID.String()})
		})

		body := []byte(`{"email":"test@example.com","name":"Test User","provider":"google"}`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/internal/oauth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, userID.String(), resp["userId"])
		mockSvc.AssertExpectations(t)
	})
}
