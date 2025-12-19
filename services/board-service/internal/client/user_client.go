package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonclient "github.com/OrangesCloud/wealist-advanced-go-pkg/client"
	"project-board-api/internal/metrics"
)

// Type aliases for backward compatibility with existing code
type (
	UserProfile                 = commonclient.UserProfile
	WorkspaceProfile            = commonclient.WorkspaceProfile
	Workspace                   = commonclient.Workspace
	WorkspaceValidationResponse = commonclient.WorkspaceValidationResponse
	TokenValidationResponse     = commonclient.TokenValidationResponse
)

// UserClient defines the interface for ALL User API and Auth interactions
type UserClient interface {
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
	GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*commonclient.UserProfile, error)
	GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*commonclient.WorkspaceProfile, error)
	GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*commonclient.Workspace, error)
	ValidateToken(ctx context.Context, tokenStr string) (uuid.UUID, error)
}

// userClient implements UserClient interface with metrics support
type userClient struct {
	*commonclient.BaseHTTPClient
	authBaseURL string // Auth service URL for token validation
	metrics     *metrics.Metrics
}

// NewUserClient creates a new User API client
// authBaseURL is used for ValidateToken, baseURL is used for user-related APIs
func NewUserClient(baseURL string, authBaseURL string, timeout time.Duration, logger *zap.Logger, m *metrics.Metrics) UserClient {
	return &userClient{
		BaseHTTPClient: commonclient.NewBaseHTTPClient(baseURL, timeout, logger),
		authBaseURL:    authBaseURL,
		metrics:        m,
	}
}

// ValidateToken validates a token via auth-service (POST /api/auth/validate)
func (c *userClient) ValidateToken(ctx context.Context, tokenStr string) (uuid.UUID, error) {
	url := fmt.Sprintf("%s/api/auth/validate", c.authBaseURL)

	c.Logger.Debug("ValidateToken request",
		zap.String("auth_base_url", c.authBaseURL),
		zap.String("url", url),
	)

	reqBody, err := json.Marshal(map[string]string{"token": tokenStr})
	if err != nil {
		c.Logger.Error("Failed to marshal request body", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		c.Logger.Error("Failed to create validation request", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("Auth service API connection failed", zap.Error(err))
		return uuid.Nil, fmt.Errorf("auth service connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger.Warn("Token validation failed via Auth Service", zap.Int("status", resp.StatusCode))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return uuid.Nil, errors.New("token validation failed: unauthorized or forbidden")
		}
		return uuid.Nil, fmt.Errorf("auth service returned unexpected status: %d", resp.StatusCode)
	}

	var validationResponse commonclient.TokenValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResponse); err != nil {
		c.Logger.Error("Failed to decode validation response", zap.Error(err))
		return uuid.Nil, fmt.Errorf("invalid response format from auth service")
	}

	if !validationResponse.Valid {
		return uuid.Nil, fmt.Errorf("token explicitly marked invalid: %s", validationResponse.Message)
	}

	userID, err := uuid.Parse(validationResponse.UserID)
	if err != nil {
		c.Logger.Error("Invalid UUID format in validation response", zap.String("id", validationResponse.UserID))
		return uuid.Nil, errors.New("invalid user ID format received")
	}

	return userID, nil
}

// ValidateWorkspaceMember validates if a user is a member of a workspace
func (c *userClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	url := c.BuildURL(fmt.Sprintf("/workspaces/%s/validate-member/%s", workspaceID.String(), userID.String()))

	c.Logger.Debug("Validating workspace member",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	var response commonclient.WorkspaceValidationResponse
	if err := c.doRequestWithMetrics(ctx, "GET", url, token, &response); err != nil {
		c.Logger.Error("Failed to validate workspace member",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		return false, err
	}

	isValid := response.IsWorkspaceMember()

	c.Logger.Debug("Workspace member validation result",
		zap.Bool("is_valid", isValid),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	return isValid, nil
}

// GetUserProfile retrieves user profile information
func (c *userClient) GetUserProfile(ctx context.Context, userID uuid.UUID, token string) (*commonclient.UserProfile, error) {
	url := c.BuildURL(fmt.Sprintf("/users/%s", userID.String()))

	c.Logger.Debug("Getting user profile",
		zap.String("url", url),
		zap.String("user_id", userID.String()),
	)

	var profile commonclient.UserProfile
	if err := c.doRequestWithMetrics(ctx, "GET", url, token, &profile); err != nil {
		c.Logger.Error("Failed to get user profile",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return empty profile
		return &commonclient.UserProfile{UserID: userID, Email: ""}, nil
	}

	c.Logger.Debug("User profile retrieved",
		zap.String("user_id", userID.String()),
		zap.String("email", profile.Email),
	)

	return &profile, nil
}

// GetWorkspaceProfile retrieves workspace-specific user profile
func (c *userClient) GetWorkspaceProfile(ctx context.Context, workspaceID, userID uuid.UUID, token string) (*commonclient.WorkspaceProfile, error) {
	url := c.BuildURL(fmt.Sprintf("/profiles/workspace/%s/user/%s", workspaceID.String(), userID.String()))

	c.Logger.Debug("Getting workspace profile",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	var profile commonclient.WorkspaceProfile
	if err := c.doRequestWithMetrics(ctx, "GET", url, token, &profile); err != nil {
		c.Logger.Error("Failed to get workspace profile",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return empty profile
		return &commonclient.WorkspaceProfile{
			WorkspaceID: workspaceID,
			UserID:      userID,
			NickName:    "",
			Email:       "",
		}, nil
	}

	c.Logger.Debug("Workspace profile retrieved",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
		zap.String("nickname", profile.NickName),
	)

	return &profile, nil
}

// GetWorkspace retrieves workspace information
func (c *userClient) GetWorkspace(ctx context.Context, workspaceID uuid.UUID, token string) (*commonclient.Workspace, error) {
	url := c.BuildURL(fmt.Sprintf("/workspaces/%s", workspaceID.String()))

	c.Logger.Debug("Getting workspace",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
	)

	var workspace commonclient.Workspace
	if err := c.doRequestWithMetrics(ctx, "GET", url, token, &workspace); err != nil {
		c.Logger.Error("Failed to get workspace",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
		)
		// Graceful degradation: return empty workspace
		return &commonclient.Workspace{ID: workspaceID, Name: ""}, nil
	}

	c.Logger.Debug("Workspace retrieved",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("name", workspace.Name),
	)

	return &workspace, nil
}

// doRequestWithMetrics performs an HTTP request with metrics recording
func (c *userClient) doRequestWithMetrics(ctx context.Context, method, url, token string, result interface{}) error {
	startTime := time.Now()

	c.Logger.Info("Making request to User Service",
		zap.String("method", method),
		zap.String("url", url),
		zap.Bool("has_token", token != ""),
		zap.Duration("timeout", c.Timeout),
	)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		c.Logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(startTime)

	// Record metrics
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if c.metrics != nil {
		c.metrics.RecordExternalAPICall(url, method, statusCode, duration, err)
	}

	if err != nil {
		c.Logger.Error("Failed to execute HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
			zap.Duration("processing_time", duration),
		)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger.Error("Failed to read response body",
			zap.Error(err),
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
		)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	processingTime := time.Since(startTime)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Logger.Error("User API returned non-success status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
			zap.String("method", method),
			zap.String("response_body", string(body)),
			zap.Duration("processing_time", processingTime),
		)
		return fmt.Errorf("user API returned status %d: %s", resp.StatusCode, string(body))
	}

	c.Logger.Info("Received successful response from User Service",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
		zap.Int("body_length", len(body)),
		zap.Duration("processing_time", processingTime),
	)

	if err := json.Unmarshal(body, result); err != nil {
		c.Logger.Error("Failed to parse response JSON",
			zap.Error(err),
			zap.String("url", url),
			zap.String("response_body", string(body)),
		)
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}
