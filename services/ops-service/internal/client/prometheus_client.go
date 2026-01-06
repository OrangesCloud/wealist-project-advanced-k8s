package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// PrometheusClient handles Prometheus API calls
type PrometheusClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// PrometheusConfig holds Prometheus client configuration
type PrometheusConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(cfg PrometheusConfig, logger *zap.Logger) *PrometheusClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &PrometheusClient{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// PrometheusResponse represents the Prometheus API response
type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}

// PrometheusData holds the result data
type PrometheusData struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusResult `json:"result"`
}

// PrometheusResult holds a single result
type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`  // [timestamp, value]
	Values [][]interface{}   `json:"values"` // for range queries
}

// ServiceMetrics represents metrics for a single service
type ServiceMetrics struct {
	ServiceName    string  `json:"serviceName"`
	RequestRate    float64 `json:"requestRate"`    // requests per second
	ErrorRate      float64 `json:"errorRate"`      // percentage
	AvgLatency     float64 `json:"avgLatency"`     // milliseconds
	P95Latency     float64 `json:"p95Latency"`     // milliseconds
	P99Latency     float64 `json:"p99Latency"`     // milliseconds
	SuccessRate    float64 `json:"successRate"`    // percentage
	ActiveRequests int     `json:"activeRequests"` // concurrent requests
}

// ClusterMetrics represents cluster-level metrics
type ClusterMetrics struct {
	NodeCount       int     `json:"nodeCount"`
	PodCount        int     `json:"podCount"`
	CPUUsage        float64 `json:"cpuUsage"`        // percentage
	MemoryUsage     float64 `json:"memoryUsage"`     // percentage
	TotalCPUCores   float64 `json:"totalCpuCores"`
	TotalMemoryGB   float64 `json:"totalMemoryGb"`
	HealthyPods     int     `json:"healthyPods"`
	UnhealthyPods   int     `json:"unhealthyPods"`
}

// SystemOverview represents overall system metrics
type SystemOverview struct {
	TotalRequests    float64 `json:"totalRequests"`    // requests in last hour
	AvgResponseTime  float64 `json:"avgResponseTime"`  // milliseconds
	ErrorPercentage  float64 `json:"errorPercentage"`  // percentage
	ActiveServices   int     `json:"activeServices"`
	TotalEndpoints   int     `json:"totalEndpoints"`
}

// Query executes a PromQL instant query
func (c *PrometheusClient) Query(ctx context.Context, query string) (*PrometheusResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/query", c.baseURL)

	params := url.Values{}
	params.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus query failed: %s", string(body))
	}

	var result PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// QueryRange executes a PromQL range query
func (c *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*PrometheusResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/query_range", c.baseURL)

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.Unix(), 10))
	params.Set("end", strconv.FormatInt(end.Unix(), 10))
	params.Set("step", step.String())

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus range query failed: %s", string(body))
	}

	var result PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetServiceMetrics returns metrics for all services
func (c *PrometheusClient) GetServiceMetrics(ctx context.Context, namespace string) ([]ServiceMetrics, error) {
	services := []string{
		"auth-service",
		"user-service",
		"board-service",
		"chat-service",
		"noti-service",
		"storage-service",
		"ops-service",
	}

	var metrics []ServiceMetrics
	for _, svc := range services {
		m, err := c.getServiceMetric(ctx, svc, namespace)
		if err != nil {
			c.logger.Warn("Failed to get metrics for service",
				zap.String("service", svc),
				zap.Error(err))
			// Add empty metric instead of failing
			m = &ServiceMetrics{ServiceName: svc}
		}
		metrics = append(metrics, *m)
	}

	return metrics, nil
}

func (c *PrometheusClient) getServiceMetric(ctx context.Context, serviceName, namespace string) (*ServiceMetrics, error) {
	metric := &ServiceMetrics{ServiceName: serviceName}

	// Request rate (requests per second) - using Istio metrics
	rateQuery := fmt.Sprintf(`sum(rate(istio_requests_total{destination_service_name="%s", reporter="destination"}[5m]))`, serviceName)
	if result, err := c.Query(ctx, rateQuery); err == nil && len(result.Data.Result) > 0 {
		metric.RequestRate = c.extractValue(result.Data.Result[0].Value)
	}

	// Error rate (5xx responses)
	errorQuery := fmt.Sprintf(`sum(rate(istio_requests_total{destination_service_name="%s", response_code=~"5..", reporter="destination"}[5m])) / sum(rate(istio_requests_total{destination_service_name="%s", reporter="destination"}[5m])) * 100`, serviceName, serviceName)
	if result, err := c.Query(ctx, errorQuery); err == nil && len(result.Data.Result) > 0 {
		metric.ErrorRate = c.extractValue(result.Data.Result[0].Value)
	}

	// Average latency (P50)
	latencyQuery := fmt.Sprintf(`histogram_quantile(0.50, sum(rate(istio_request_duration_milliseconds_bucket{destination_service_name="%s", reporter="destination"}[5m])) by (le))`, serviceName)
	if result, err := c.Query(ctx, latencyQuery); err == nil && len(result.Data.Result) > 0 {
		metric.AvgLatency = c.extractValue(result.Data.Result[0].Value)
	}

	// P95 latency
	p95Query := fmt.Sprintf(`histogram_quantile(0.95, sum(rate(istio_request_duration_milliseconds_bucket{destination_service_name="%s", reporter="destination"}[5m])) by (le))`, serviceName)
	if result, err := c.Query(ctx, p95Query); err == nil && len(result.Data.Result) > 0 {
		metric.P95Latency = c.extractValue(result.Data.Result[0].Value)
	}

	// P99 latency
	p99Query := fmt.Sprintf(`histogram_quantile(0.99, sum(rate(istio_request_duration_milliseconds_bucket{destination_service_name="%s", reporter="destination"}[5m])) by (le))`, serviceName)
	if result, err := c.Query(ctx, p99Query); err == nil && len(result.Data.Result) > 0 {
		metric.P99Latency = c.extractValue(result.Data.Result[0].Value)
	}

	// Success rate (2xx + 3xx)
	successQuery := fmt.Sprintf(`sum(rate(istio_requests_total{destination_service_name="%s", response_code=~"[23]..", reporter="destination"}[5m])) / sum(rate(istio_requests_total{destination_service_name="%s", reporter="destination"}[5m])) * 100`, serviceName, serviceName)
	if result, err := c.Query(ctx, successQuery); err == nil && len(result.Data.Result) > 0 {
		metric.SuccessRate = c.extractValue(result.Data.Result[0].Value)
	}

	return metric, nil
}

// GetClusterMetrics returns cluster-level metrics
func (c *PrometheusClient) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	metrics := &ClusterMetrics{}

	// Node count
	nodeQuery := `count(kube_node_info)`
	if result, err := c.Query(ctx, nodeQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.NodeCount = int(c.extractValue(result.Data.Result[0].Value))
	}

	// Pod count
	podQuery := `count(kube_pod_info)`
	if result, err := c.Query(ctx, podQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.PodCount = int(c.extractValue(result.Data.Result[0].Value))
	}

	// CPU usage percentage
	cpuQuery := `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	if result, err := c.Query(ctx, cpuQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.CPUUsage = c.extractValue(result.Data.Result[0].Value)
	}

	// Memory usage percentage
	memQuery := `(1 - (sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes))) * 100`
	if result, err := c.Query(ctx, memQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.MemoryUsage = c.extractValue(result.Data.Result[0].Value)
	}

	// Total CPU cores
	coresQuery := `sum(machine_cpu_cores)`
	if result, err := c.Query(ctx, coresQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.TotalCPUCores = c.extractValue(result.Data.Result[0].Value)
	}

	// Total memory in GB
	totalMemQuery := `sum(node_memory_MemTotal_bytes) / 1024 / 1024 / 1024`
	if result, err := c.Query(ctx, totalMemQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.TotalMemoryGB = c.extractValue(result.Data.Result[0].Value)
	}

	// Healthy pods (Running phase)
	healthyQuery := `count(kube_pod_status_phase{phase="Running"})`
	if result, err := c.Query(ctx, healthyQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.HealthyPods = int(c.extractValue(result.Data.Result[0].Value))
	}

	// Unhealthy pods (Failed or Unknown phase)
	unhealthyQuery := `count(kube_pod_status_phase{phase=~"Failed|Unknown|Pending"})`
	if result, err := c.Query(ctx, unhealthyQuery); err == nil && len(result.Data.Result) > 0 {
		metrics.UnhealthyPods = int(c.extractValue(result.Data.Result[0].Value))
	}

	return metrics, nil
}

// GetSystemOverview returns overall system metrics for dashboard
func (c *PrometheusClient) GetSystemOverview(ctx context.Context, namespace string) (*SystemOverview, error) {
	overview := &SystemOverview{}

	// Total requests in last hour
	totalReqQuery := fmt.Sprintf(`sum(increase(istio_requests_total{reporter="destination", destination_service_namespace="%s"}[1h]))`, namespace)
	if result, err := c.Query(ctx, totalReqQuery); err == nil && len(result.Data.Result) > 0 {
		overview.TotalRequests = c.extractValue(result.Data.Result[0].Value)
	}

	// Average response time
	avgRespQuery := fmt.Sprintf(`avg(histogram_quantile(0.50, sum(rate(istio_request_duration_milliseconds_bucket{reporter="destination", destination_service_namespace="%s"}[5m])) by (le, destination_service_name)))`, namespace)
	if result, err := c.Query(ctx, avgRespQuery); err == nil && len(result.Data.Result) > 0 {
		overview.AvgResponseTime = c.extractValue(result.Data.Result[0].Value)
	}

	// Error percentage
	errorPctQuery := fmt.Sprintf(`sum(rate(istio_requests_total{reporter="destination", destination_service_namespace="%s", response_code=~"5.."}[5m])) / sum(rate(istio_requests_total{reporter="destination", destination_service_namespace="%s"}[5m])) * 100`, namespace, namespace)
	if result, err := c.Query(ctx, errorPctQuery); err == nil && len(result.Data.Result) > 0 {
		overview.ErrorPercentage = c.extractValue(result.Data.Result[0].Value)
	}

	// Active services (services with traffic)
	activeQuery := fmt.Sprintf(`count(count by (destination_service_name) (rate(istio_requests_total{reporter="destination", destination_service_namespace="%s"}[5m]) > 0))`, namespace)
	if result, err := c.Query(ctx, activeQuery); err == nil && len(result.Data.Result) > 0 {
		overview.ActiveServices = int(c.extractValue(result.Data.Result[0].Value))
	}

	return overview, nil
}

// extractValue extracts the float value from Prometheus result
func (c *PrometheusClient) extractValue(value []interface{}) float64 {
	if len(value) < 2 {
		return 0
	}

	// Value is at index 1, timestamp is at index 0
	switch v := value[1].(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		// Handle NaN
		if f != f {
			return 0
		}
		return f
	case float64:
		if v != v {
			return 0
		}
		return v
	default:
		return 0
	}
}
