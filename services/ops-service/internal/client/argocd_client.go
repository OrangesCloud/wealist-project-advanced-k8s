package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ArgoCDClient handles ArgoCD API calls
type ArgoCDClient struct {
	serverURL string
	token     string
	client    *http.Client
	logger    *zap.Logger
}

// ArgoCDConfig holds ArgoCD client configuration
type ArgoCDConfig struct {
	ServerURL string
	Token     string
	Insecure  bool
}

// NewArgoCDClient creates a new ArgoCD client
func NewArgoCDClient(cfg ArgoCDConfig, logger *zap.Logger) *ArgoCDClient {
	transport := &http.Transport{}
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &ArgoCDClient{
		serverURL: strings.TrimSuffix(cfg.ServerURL, "/"),
		token:     cfg.Token,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		logger: logger,
	}
}

// Application represents an ArgoCD application
type Application struct {
	Metadata ApplicationMetadata `json:"metadata"`
	Spec     ApplicationSpec     `json:"spec"`
	Status   ApplicationStatus   `json:"status"`
}

// ApplicationMetadata holds application metadata
type ApplicationMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ApplicationSpec holds application spec
type ApplicationSpec struct {
	Project string `json:"project"`
}

// ApplicationStatus holds application status
type ApplicationStatus struct {
	Sync   SyncStatus   `json:"sync"`
	Health HealthStatus `json:"health"`
}

// SyncStatus holds sync status
type SyncStatus struct {
	Status string `json:"status"`
}

// HealthStatus holds health status
type HealthStatus struct {
	Status string `json:"status"`
}

// ApplicationList holds a list of applications
type ApplicationList struct {
	Items []Application `json:"items"`
}

// RBACPolicy represents an RBAC policy line
type RBACPolicy struct {
	Role       string `json:"role"`
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	Object     string `json:"object"`
	Permission string `json:"permission"`
}

// GetApplications gets all applications
func (c *ArgoCDClient) GetApplications() ([]Application, error) {
	resp, err := c.doRequest("GET", "/api/v1/applications", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get applications: %s", string(body))
	}

	var appList ApplicationList
	if err := json.NewDecoder(resp.Body).Decode(&appList); err != nil {
		return nil, fmt.Errorf("failed to decode applications: %w", err)
	}

	return appList.Items, nil
}

// GetApplication gets a specific application
func (c *ArgoCDClient) GetApplication(name string) (*Application, error) {
	resp, err := c.doRequest("GET", "/api/v1/applications/"+name, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get application: %s", string(body))
	}

	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode application: %w", err)
	}

	return &app, nil
}

// SyncApplication syncs an application
func (c *ArgoCDClient) SyncApplication(name string) error {
	resp, err := c.doRequest("POST", "/api/v1/applications/"+name+"/sync", strings.NewReader("{}"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to sync application: %s", string(body))
	}

	return nil
}

// GetRBACConfigMap gets the RBAC ConfigMap (argocd-rbac-cm)
// Note: This requires cluster access, typically done via ServiceAccount
func (c *ArgoCDClient) GetRBACConfigMap() (map[string]string, error) {
	// ArgoCD API doesn't directly expose ConfigMap editing
	// This would typically be done via Kubernetes API
	c.logger.Warn("GetRBACConfigMap: Direct ConfigMap access not available via ArgoCD API")
	return nil, fmt.Errorf("RBAC ConfigMap access requires Kubernetes API")
}

func (c *ArgoCDClient) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.serverURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
