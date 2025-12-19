// Package client provides common HTTP client utilities for service-to-service communication.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// BaseHTTPClient provides common HTTP client functionality for service-to-service communication.
type BaseHTTPClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
	Logger     *zap.Logger
}

// NewBaseHTTPClient creates a new base HTTP client.
func NewBaseHTTPClient(baseURL string, timeout time.Duration, logger *zap.Logger) *BaseHTTPClient {
	return &BaseHTTPClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Timeout: timeout,
		Logger:  logger,
	}
}

// BuildURL constructs the full URL for API calls.
// baseURL should be the service host only (e.g., http://user-service:8081)
// endpoint should be the API path (e.g., /workspaces/123/validate-member/456)
//
// Example:
//   - baseURL: http://user-service:8081, endpoint: /workspaces/123/validate-member/456
//     -> http://user-service:8081/api/workspaces/123/validate-member/456
func (c *BaseHTTPClient) BuildURL(endpoint string) string {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Prepend /api to the endpoint
	finalURL := c.BaseURL + "/api" + endpoint

	c.Logger.Debug("Built URL",
		zap.String("base_url", c.BaseURL),
		zap.String("endpoint", endpoint),
		zap.String("final_url", finalURL),
	)

	return finalURL
}

// DoRequest performs an HTTP request with the given parameters and decodes JSON response.
func (c *BaseHTTPClient) DoRequest(ctx context.Context, method, url, token string, result interface{}) error {
	startTime := time.Now()

	c.Logger.Debug("Making HTTP request",
		zap.String("method", method),
		zap.String("url", url),
		zap.Bool("has_token", token != ""),
		zap.Duration("timeout", c.Timeout),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		c.Logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("method", method),
			zap.String("url", url),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(startTime)

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

	// Read response body
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

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Logger.Error("API returned non-success status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
			zap.String("method", method),
			zap.String("response_body", string(body)),
			zap.Duration("processing_time", processingTime),
		)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	c.Logger.Debug("Received successful response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
		zap.Int("body_length", len(body)),
		zap.Duration("processing_time", processingTime),
	)

	// Parse response
	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			c.Logger.Error("Failed to parse response JSON",
				zap.Error(err),
				zap.String("url", url),
				zap.String("response_body", string(body)),
			)
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// DoRequestWithBody performs an HTTP request with a JSON body.
func (c *BaseHTTPClient) DoRequestWithBody(ctx context.Context, method, url, token string, body interface{}, result interface{}) error {
	startTime := time.Now()

	// Marshal request body
	var reqBody io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(bodyBytes))
	}

	c.Logger.Debug("Making HTTP request with body",
		zap.String("method", method),
		zap.String("url", url),
		zap.Bool("has_token", token != ""),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		c.Logger.Error("Failed to execute HTTP request",
			zap.Error(err),
			zap.String("url", url),
			zap.Duration("processing_time", duration),
		)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Logger.Error("API returned non-success status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
			zap.String("response_body", string(respBody)),
		)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GetStatusCode makes a request and returns the status code (useful for validation endpoints).
func (c *BaseHTTPClient) GetStatusCode(ctx context.Context, method, url, token string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
