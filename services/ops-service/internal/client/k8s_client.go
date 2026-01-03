package client

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClient handles Kubernetes API calls
type K8sClient struct {
	clientset *kubernetes.Clientset
	logger    *zap.Logger
}

// K8sConfig holds Kubernetes client configuration
type K8sConfig struct {
	InCluster  bool   // Use in-cluster config
	KubeConfig string // Path to kubeconfig file (for local development)
}

// NewK8sClient creates a new Kubernetes client
func NewK8sClient(cfg K8sConfig, logger *zap.Logger) (*K8sClient, error) {
	var config *rest.Config
	var err error

	if cfg.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
		logger.Info("Using in-cluster Kubernetes config")
	} else {
		kubeconfig := cfg.KubeConfig
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
		logger.Info("Using kubeconfig", zap.String("path", kubeconfig))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &K8sClient{
		clientset: clientset,
		logger:    logger,
	}, nil
}

// ArgoCDRBACConfig holds ArgoCD RBAC configuration from ConfigMap
type ArgoCDRBACConfig struct {
	Policy     string            `json:"policy"`
	PolicyCSV  string            `json:"policyCSV"`
	Scopes     string            `json:"scopes"`
	AdminUsers []string          `json:"adminUsers"`
	RawData    map[string]string `json:"rawData"`
}

// GetArgoCDRBACConfigMap gets the ArgoCD RBAC ConfigMap
func (c *K8sClient) GetArgoCDRBACConfigMap(ctx context.Context, namespace string) (*ArgoCDRBACConfig, error) {
	if namespace == "" {
		namespace = "argocd"
	}

	cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "argocd-rbac-cm", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get argocd-rbac-cm: %w", err)
	}

	config := &ArgoCDRBACConfig{
		Policy:    cm.Data["policy.default"],
		PolicyCSV: cm.Data["policy.csv"],
		Scopes:    cm.Data["scopes"],
		RawData:   cm.Data,
	}

	// Parse admin users from policy.csv
	config.AdminUsers = parseAdminUsers(config.PolicyCSV)

	return config, nil
}

// UpdateArgoCDRBACConfigMap updates the ArgoCD RBAC ConfigMap
func (c *K8sClient) UpdateArgoCDRBACConfigMap(ctx context.Context, namespace string, data map[string]string) error {
	if namespace == "" {
		namespace = "argocd"
	}

	cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "argocd-rbac-cm", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get argocd-rbac-cm: %w", err)
	}

	// Update the data
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	for k, v := range data {
		cm.Data[k] = v
	}

	_, err = c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update argocd-rbac-cm: %w", err)
	}

	c.logger.Info("Updated argocd-rbac-cm ConfigMap", zap.String("namespace", namespace))
	return nil
}

// AddArgoCDAdmin adds an admin user to ArgoCD RBAC
func (c *K8sClient) AddArgoCDAdmin(ctx context.Context, namespace, email string) error {
	config, err := c.GetArgoCDRBACConfigMap(ctx, namespace)
	if err != nil {
		return err
	}

	// Check if already admin
	for _, admin := range config.AdminUsers {
		if admin == email {
			return fmt.Errorf("user %s is already an admin", email)
		}
	}

	// Add the user to policy.csv
	newPolicyCSV := addAdminToPolicy(config.PolicyCSV, email)

	return c.UpdateArgoCDRBACConfigMap(ctx, namespace, map[string]string{
		"policy.csv": newPolicyCSV,
	})
}

// RemoveArgoCDAdmin removes an admin user from ArgoCD RBAC
func (c *K8sClient) RemoveArgoCDAdmin(ctx context.Context, namespace, email string) error {
	config, err := c.GetArgoCDRBACConfigMap(ctx, namespace)
	if err != nil {
		return err
	}

	// Check if user is admin
	found := false
	for _, admin := range config.AdminUsers {
		if admin == email {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %s is not an admin", email)
	}

	// Remove the user from policy.csv
	newPolicyCSV := removeAdminFromPolicy(config.PolicyCSV, email)

	return c.UpdateArgoCDRBACConfigMap(ctx, namespace, map[string]string{
		"policy.csv": newPolicyCSV,
	})
}

// GetConfigMap gets a ConfigMap by name
func (c *K8sClient) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

// parseAdminUsers extracts admin users from policy.csv
func parseAdminUsers(policyCSV string) []string {
	var admins []string
	lines := strings.Split(policyCSV, "\n")

	// Look for lines like: g, user@example.com, role:admin
	adminRolePattern := regexp.MustCompile(`g,\s*([^,]+),\s*role:admin`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := adminRolePattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			admins = append(admins, strings.TrimSpace(matches[1]))
		}
	}

	return admins
}

// addAdminToPolicy adds an admin user to policy.csv
func addAdminToPolicy(policyCSV, email string) string {
	// Add the admin role assignment
	newLine := fmt.Sprintf("g, %s, role:admin", email)

	if policyCSV == "" {
		return newLine
	}

	// Check if already exists (shouldn't happen, but double-check)
	if strings.Contains(policyCSV, newLine) {
		return policyCSV
	}

	return policyCSV + "\n" + newLine
}

// removeAdminFromPolicy removes an admin user from policy.csv
func removeAdminFromPolicy(policyCSV, email string) string {
	lines := strings.Split(policyCSV, "\n")
	var newLines []string

	// Pattern to match the user's admin role assignment
	pattern := fmt.Sprintf("g, %s, role:admin", email)
	patternAlt := fmt.Sprintf("g,%s,role:admin", email) // Without spaces

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Keep the line if it doesn't match the user's admin assignment
		if trimmed != pattern && trimmed != patternAlt &&
			!strings.Contains(trimmed, fmt.Sprintf("g, %s, role:admin", email)) &&
			!strings.Contains(trimmed, fmt.Sprintf("g,%s,role:admin", email)) {
			newLines = append(newLines, line)
		}
	}

	return strings.Join(newLines, "\n")
}
