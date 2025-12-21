package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonclient "github.com/OrangesCloud/wealist-advanced-go-pkg/client"
)

// UserClient defines the interface for User API interactions
type UserClient interface {
	ValidateWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, token string) (bool, error)
}

// userClient implements UserClient interface using common HTTP client
type userClient struct {
	*commonclient.BaseHTTPClient
}

// NewUserClient creates a new User API client
func NewUserClient(baseURL string, timeout time.Duration, logger *zap.Logger) UserClient {
	return &userClient{
		BaseHTTPClient: commonclient.NewBaseHTTPClient(baseURL, timeout, logger),
	}
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
	if err := c.DoRequest(ctx, "GET", url, token, &response); err != nil {
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
