package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserClient defines the interface for User API interactions
type UserClient interface {
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
}

// WorkspaceValidationResponse represents the response from workspace validation endpoint
type WorkspaceValidationResponse struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	UserID      uuid.UUID `json:"userId"`
	Valid       bool      `json:"valid"`
	IsValid     bool      `json:"isValid"`
	IsMember    bool      `json:"isMember"` // User Service returns this field
}

// userClient implements UserClient interface
type userClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	logger     *zap.Logger
}

// NewUserClient creates a new User API client
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

// ValidateWorkspaceMember validates if a user is a member of a workspace
func (c *userClient) ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error) {
	url := c.buildURL(fmt.Sprintf("/workspaces/%s/validate-member/%s", workspaceID.String(), userID.String()))

	c.logger.Debug("Validating workspace member",
		zap.String("url", url),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	var response WorkspaceValidationResponse
	if err := c.doRequest(ctx, "GET", url, token, &response); err != nil {
		c.logger.Error("Failed to validate workspace member",
			zap.Error(err),
			zap.String("workspace_id", workspaceID.String()),
			zap.String("user_id", userID.String()),
		)
		// Graceful degradation: return false on error
		return false, err
	}

	// Check Valid, IsValid, and IsMember fields for compatibility
	// User Service returns "isMember" field
	isValid := response.Valid || response.IsValid || response.IsMember

	c.logger.Debug("Workspace member validation result",
		zap.Bool("is_valid", isValid),
		zap.Bool("response_valid", response.Valid),
		zap.Bool("response_is_valid", response.IsValid),
		zap.Bool("response_is_member", response.IsMember),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("user_id", userID.String()),
	)

	return isValid, nil
}

// doRequest performs an HTTP request with the given parameters
func (c *userClient) doRequest(ctx context.Context, method, url, token string, result interface{}) error {
	// Track request start time for processing time calculation
	startTime := time.Now()

	// Enhanced logging: Log detailed request information before making the call
	c.logger.Info("Making request to User Service",
		zap.String("method", method),
		zap.String("url", url),
		zap.String("base_url", c.baseURL),
		zap.Bool("has_token", token != ""),
		zap.Duration("timeout", c.timeout),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		c.logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Error("Failed to execute HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
			zap.Duration("processing_time", duration),
		)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body",
			zap.Error(err),
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("processing_time", time.Since(startTime)),
		)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate processing time
	processingTime := time.Since(startTime)

	// Check status code and log accordingly
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Enhanced error logging with status code and response body
		c.logger.Error("User API returned non-success status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
			zap.String("method", method),
			zap.String("response_body", string(body)),
			zap.Bool("has_token", token != ""),
			zap.Duration("processing_time", processingTime),
		)

		// Special handling for 403 Forbidden errors
		if resp.StatusCode == http.StatusForbidden {
			c.logger.Error("403 Forbidden error from User Service",
				zap.String("requested_url", url),
				zap.String("method", method),
				zap.Bool("token_present", token != ""),
				zap.String("response_body", string(body)),
				zap.Duration("processing_time", processingTime),
			)
		}

		return fmt.Errorf("user API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Success response logging with processing time
	c.logger.Info("Received successful response from User Service",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
		zap.String("method", method),
		zap.Int("body_length", len(body)),
		zap.Duration("processing_time", processingTime),
	)

	// Parse response
	if err := json.Unmarshal(body, result); err != nil {
		c.logger.Error("Failed to parse response JSON",
			zap.Error(err),
			zap.String("url", url),
			zap.String("response_body", string(body)),
			zap.Duration("processing_time", processingTime),
		)
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Debug("Successfully parsed response",
		zap.String("url", url),
		zap.Duration("total_processing_time", processingTime),
	)

	return nil
}
