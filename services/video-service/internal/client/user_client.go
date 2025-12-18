package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserClient handles communication with user-service
type UserClient interface {
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
}

// WorkspaceValidationResponse represents the response from workspace validation endpoint
type WorkspaceValidationResponse struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	UserID      uuid.UUID `json:"userId"`
	Valid       bool      `json:"valid"`
	IsValid     bool      `json:"isValid"`
	IsMember    bool      `json:"isMember"`
}

type userClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	logger     *zap.Logger
}

// NewUserClient creates a new user-service client
func NewUserClient(baseURL string, timeout time.Duration, logger *zap.Logger) UserClient {
	return &userClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		logger:  logger,
	}
}

// buildURL constructs the full URL for User Service API calls
// baseURL should be the service host only (e.g., http://user-service:8081)
// endpoint should be the API path (e.g., /workspaces/123/validate-member/456)
//
// Example:
//   - baseURL: http://user-service:8081, endpoint: /workspaces/123/validate-member/456
//     -> http://user-service:8081/api/workspaces/123/validate-member/456
func (c *userClient) buildURL(endpoint string) string {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// user-service uses /api as base path, so we prepend /api to the endpoint
	finalURL := c.baseURL + "/api" + endpoint

	c.logger.Debug("Built URL for User Service",
		zap.String("base_url", c.baseURL),
		zap.String("endpoint", endpoint),
		zap.String("final_url", finalURL),
	)

	return finalURL
}

// ValidateWorkspaceMember checks if a user is a member of a workspace
func (c *userClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	url := c.buildURL(fmt.Sprintf("/workspaces/%s/validate-member/%s", workspaceID.String(), userID.String()))

	c.logger.Debug("Validating workspace member",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to call user-service",
			zap.Error(err),
			zap.String("url", url),
		)
		return false, fmt.Errorf("failed to call user-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("User-service returned non-200 status",
			zap.Int("status", resp.StatusCode),
			zap.String("url", url),
		)
		// 403 = not a member, 404 = workspace not found
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("user-service returned status %d", resp.StatusCode)
	}

	var response WorkspaceValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode response", zap.Error(err))
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check all possible fields
	isValid := response.Valid || response.IsValid || response.IsMember

	c.logger.Debug("Workspace member validation result",
		zap.Bool("is_valid", isValid),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	return isValid, nil
}
