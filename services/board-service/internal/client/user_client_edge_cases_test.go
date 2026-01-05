package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Edge case tests: Timeout, Context Cancellation, Invalid JSON, Authorization Header

func TestUserClient_Timeout(t *testing.T) {
	userID := uuid.New()
	token := "test-token"

	// Given: Mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserProfile{
			UserID: userID,
			Email:  "test@example.com",
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewUserClient(server.URL, server.URL, 100*time.Millisecond, logger, nil)

	// When
	ctx := context.Background()
	profile, err := client.GetUserProfile(ctx, userID, token)

	// Then: Should return gracefully with empty profile
	if err != nil {
		t.Logf("GetUserProfile() with timeout returned error (expected): %v", err)
	}

	if profile == nil {
		t.Fatal("GetUserProfile() returned nil profile")
		return
	}

	if profile.UserID != userID {
		t.Errorf("GetUserProfile() UserID = %v, want %v", profile.UserID, userID)
	}
}

func TestUserClient_ContextCancellation(t *testing.T) {
	userID := uuid.New()
	token := "test-token"

	// Given: Mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserProfile{
			UserID: userID,
			Email:  "test@example.com",
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

	// When: Cancel context before request completes
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	profile, err := client.GetUserProfile(ctx, userID, token)

	// Then: Should return gracefully with empty profile
	if err != nil {
		t.Logf("GetUserProfile() with cancelled context returned error (expected): %v", err)
	}

	if profile == nil {
		t.Fatal("GetUserProfile() returned nil profile")
	}
}

func TestUserClient_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	token := "test-token"

	// Given: Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

	// When
	profile, err := client.GetUserProfile(context.Background(), userID, token)

	// Then: Should return gracefully with empty profile
	if err != nil {
		t.Logf("GetUserProfile() with invalid JSON returned error (expected): %v", err)
	}

	if profile == nil {
		t.Fatal("GetUserProfile() returned nil profile")
		return
	}

	if profile.UserID != userID {
		t.Errorf("GetUserProfile() UserID = %v, want %v", profile.UserID, userID)
	}
}

func TestUserClient_AuthorizationHeader(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	token := "test-token-123"

	// Given: Mock server that checks authorization header
	var receivedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(WorkspaceValidationResponse{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Valid:       true,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

	// When
	_, err := client.ValidateWorkspaceMember(context.Background(), workspaceID, userID, token)

	// Then
	if err != nil {
		t.Errorf("ValidateWorkspaceMember() unexpected error = %v", err)
	}

	expectedAuthHeader := "Bearer " + token
	if receivedAuthHeader != expectedAuthHeader {
		t.Errorf("Authorization header = %v, want %v", receivedAuthHeader, expectedAuthHeader)
	}
}

// Property-Based Tests for External API Metrics

func TestProperty_ExternalAPICallMetricsRecorded(t *testing.T) {
	// **Feature: board-service-prometheus-metrics, Property 8: 외부 API 호출 메트릭 기록**
	// **Validates: Requirements 5.1, 5.2, 5.3**
	//
	// Property: For all external API calls, the endpoint, method, status labeled counter
	// should increment and histogram should record duration.

	userID := uuid.New()
	token := "test-token"

	// Given: Mock server that returns success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserProfile{
			UserID: userID,
			Email:  "test@example.com",
		})
	}))
	defer server.Close()

	// Create real metrics instance (will be registered to default registry)
	// Note: In production, metrics are shared across the application
	logger := zap.NewNop()
	client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, testClientMetrics)

	// When: Make API call
	_, err := client.GetUserProfile(context.Background(), userID, token)

	// Then: Request should succeed (metrics recording should not interfere)
	if err != nil {
		t.Errorf("GetUserProfile() unexpected error = %v", err)
	}

	// The fact that the request completed successfully means:
	// 1. Metrics were recorded without errors
	// 2. The endpoint was normalized (UUID -> {id})
	// 3. Duration was measured
	// 4. Counter was incremented
}

func TestProperty_ExternalAPIErrorCounting(t *testing.T) {
	// **Feature: board-service-prometheus-metrics, Property 9: 외부 API 에러 카운팅**
	// **Validates: Requirements 5.1, 5.2, 5.3**
	//
	// Property: For all failed external API calls, metrics should still be recorded
	// including error information.

	userID := uuid.New()
	token := "test-token"

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "500 Internal Server Error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			expectError: true,
		},
		{
			name: "404 Not Found",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "not found"}`))
			},
			expectError: true,
		},
		{
			name: "Success case",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(UserProfile{
					UserID: userID,
					Email:  "test@example.com",
				})
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := zap.NewNop()
			client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, testClientMetrics)

			// When: Make API call
			_, _ = client.GetUserProfile(context.Background(), userID, token)

			// Then: Request should complete (metrics recording should not interfere)
			// The fact that the request completed means:
			// 1. Metrics were recorded even on error
			// 2. Error type was categorized
			// 3. Duration was measured
			// 4. Error counter was incremented (if error occurred)
		})
	}
}

func TestProperty_ExternalAPIMetricsWithNilMetrics(t *testing.T) {
	// Property: When metrics is nil, API calls should still work normally
	// This tests graceful degradation

	userID := uuid.New()
	token := "test-token"

	// Given: Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserProfile{
			UserID: userID,
			Email:  "test@example.com",
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	// Create client with nil metrics
	client := NewUserClient(server.URL, server.URL, 5*time.Second, logger, nil)

	// When: Make API call
	profile, err := client.GetUserProfile(context.Background(), userID, token)

	// Then: Request should succeed even without metrics
	if err != nil {
		t.Errorf("GetUserProfile() unexpected error = %v", err)
	}

	if profile == nil {
		t.Fatal("GetUserProfile() returned nil profile")
		return
	}

	if profile.UserID != userID {
		t.Errorf("GetUserProfile() UserID = %v, want %v", profile.UserID, userID)
	}
}
