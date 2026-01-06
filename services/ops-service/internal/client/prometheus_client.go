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

// =============================================================================
// Error Tracker Types and Methods
// =============================================================================

// RecentError represents a single error occurrence
type RecentError struct {
	ServiceName  string  `json:"serviceName"`
	RequestPath  string  `json:"requestPath"`
	ResponseCode string  `json:"responseCode"`
	ErrorCount   float64 `json:"errorCount"`
	Timestamp    int64   `json:"timestamp"`
}

// ErrorTrendPoint represents error rate at a point in time
type ErrorTrendPoint struct {
	Timestamp int64   `json:"timestamp"`
	ErrorRate float64 `json:"errorRate"` // percentage
	ErrorCount float64 `json:"errorCount"`
}

// ServiceErrorSummary represents error statistics for a service
type ServiceErrorSummary struct {
	ServiceName   string  `json:"serviceName"`
	TotalErrors   float64 `json:"totalErrors"`
	ErrorRate     float64 `json:"errorRate"` // percentage
	Error5xxCount float64 `json:"error5xxCount"`
	Error4xxCount float64 `json:"error4xxCount"`
}

// ErrorOverview represents overall error statistics
type ErrorOverview struct {
	TotalErrors     float64 `json:"totalErrors"`
	ErrorRate       float64 `json:"errorRate"`
	MostErrorService string `json:"mostErrorService"`
	MostErrorCount   float64 `json:"mostErrorCount"`
}

// GetRecentErrors returns recent 5xx errors from the last hour
func (c *PrometheusClient) GetRecentErrors(ctx context.Context, namespace string, limit int) ([]RecentError, error) {
	if limit <= 0 {
		limit = 50
	}

	// Query for recent 5xx errors grouped by service, path, and response code
	query := fmt.Sprintf(`topk(%d, sum by (destination_service_name, request_path, response_code) (
		increase(istio_requests_total{
			reporter="destination",
			destination_service_namespace="%s",
			response_code=~"5.."
		}[1h])
	))`, limit, namespace)

	result, err := c.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent errors: %w", err)
	}

	var errors []RecentError
	for _, r := range result.Data.Result {
		errors = append(errors, RecentError{
			ServiceName:  r.Metric["destination_service_name"],
			RequestPath:  r.Metric["request_path"],
			ResponseCode: r.Metric["response_code"],
			ErrorCount:   c.extractValue(r.Value),
			Timestamp:    time.Now().Unix(),
		})
	}

	return errors, nil
}

// GetErrorTrend returns error rate trend over the last hour with 5-minute intervals
func (c *PrometheusClient) GetErrorTrend(ctx context.Context, namespace string) ([]ErrorTrendPoint, error) {
	end := time.Now()
	start := end.Add(-1 * time.Hour)
	step := 5 * time.Minute

	// Error rate query
	rateQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s",
		response_code=~"5.."
	}[5m])) / sum(rate(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s"
	}[5m])) * 100`, namespace, namespace)

	rateResult, err := c.QueryRange(ctx, rateQuery, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("failed to query error trend: %w", err)
	}

	// Error count query
	countQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s",
		response_code=~"5.."
	}[5m]))`, namespace)

	countResult, err := c.QueryRange(ctx, countQuery, start, end, step)
	if err != nil {
		c.logger.Warn("Failed to query error count trend", zap.Error(err))
	}

	var trend []ErrorTrendPoint
	if len(rateResult.Data.Result) > 0 {
		for i, v := range rateResult.Data.Result[0].Values {
			point := ErrorTrendPoint{
				Timestamp: int64(v[0].(float64)),
				ErrorRate: c.extractValueFromRange(v),
			}
			// Add error count if available
			if countResult != nil && len(countResult.Data.Result) > 0 && i < len(countResult.Data.Result[0].Values) {
				point.ErrorCount = c.extractValueFromRange(countResult.Data.Result[0].Values[i])
			}
			trend = append(trend, point)
		}
	}

	return trend, nil
}

// GetErrorsByService returns error statistics per service
func (c *PrometheusClient) GetErrorsByService(ctx context.Context, namespace string) ([]ServiceErrorSummary, error) {
	// 5xx errors per service
	error5xxQuery := fmt.Sprintf(`sum by (destination_service_name) (
		increase(istio_requests_total{
			reporter="destination",
			destination_service_namespace="%s",
			response_code=~"5.."
		}[1h])
	)`, namespace)

	error5xxResult, err := c.Query(ctx, error5xxQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query 5xx errors: %w", err)
	}

	// 4xx errors per service
	error4xxQuery := fmt.Sprintf(`sum by (destination_service_name) (
		increase(istio_requests_total{
			reporter="destination",
			destination_service_namespace="%s",
			response_code=~"4.."
		}[1h])
	)`, namespace)

	error4xxResult, err := c.Query(ctx, error4xxQuery)
	if err != nil {
		c.logger.Warn("Failed to query 4xx errors", zap.Error(err))
	}

	// Total requests per service for error rate calculation
	totalQuery := fmt.Sprintf(`sum by (destination_service_name) (
		increase(istio_requests_total{
			reporter="destination",
			destination_service_namespace="%s"
		}[1h])
	)`, namespace)

	totalResult, err := c.Query(ctx, totalQuery)
	if err != nil {
		c.logger.Warn("Failed to query total requests", zap.Error(err))
	}

	// Build service map
	serviceMap := make(map[string]*ServiceErrorSummary)

	// Process 5xx errors
	for _, r := range error5xxResult.Data.Result {
		svc := r.Metric["destination_service_name"]
		if svc == "" {
			continue
		}
		if _, exists := serviceMap[svc]; !exists {
			serviceMap[svc] = &ServiceErrorSummary{ServiceName: svc}
		}
		serviceMap[svc].Error5xxCount = c.extractValue(r.Value)
		serviceMap[svc].TotalErrors += serviceMap[svc].Error5xxCount
	}

	// Process 4xx errors
	if error4xxResult != nil {
		for _, r := range error4xxResult.Data.Result {
			svc := r.Metric["destination_service_name"]
			if svc == "" {
				continue
			}
			if _, exists := serviceMap[svc]; !exists {
				serviceMap[svc] = &ServiceErrorSummary{ServiceName: svc}
			}
			serviceMap[svc].Error4xxCount = c.extractValue(r.Value)
			serviceMap[svc].TotalErrors += serviceMap[svc].Error4xxCount
		}
	}

	// Calculate error rates
	if totalResult != nil {
		for _, r := range totalResult.Data.Result {
			svc := r.Metric["destination_service_name"]
			total := c.extractValue(r.Value)
			if summary, exists := serviceMap[svc]; exists && total > 0 {
				summary.ErrorRate = (summary.TotalErrors / total) * 100
			}
		}
	}

	// Convert map to slice
	var summaries []ServiceErrorSummary
	for _, s := range serviceMap {
		summaries = append(summaries, *s)
	}

	return summaries, nil
}

// GetErrorOverview returns overall error statistics for the dashboard
func (c *PrometheusClient) GetErrorOverview(ctx context.Context, namespace string) (*ErrorOverview, error) {
	overview := &ErrorOverview{}

	// Total errors in the last hour
	totalErrorsQuery := fmt.Sprintf(`sum(increase(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s",
		response_code=~"[45].."
	}[1h]))`, namespace)

	if result, err := c.Query(ctx, totalErrorsQuery); err == nil && len(result.Data.Result) > 0 {
		overview.TotalErrors = c.extractValue(result.Data.Result[0].Value)
	}

	// Error rate
	errorRateQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s",
		response_code=~"5.."
	}[5m])) / sum(rate(istio_requests_total{
		reporter="destination",
		destination_service_namespace="%s"
	}[5m])) * 100`, namespace, namespace)

	if result, err := c.Query(ctx, errorRateQuery); err == nil && len(result.Data.Result) > 0 {
		overview.ErrorRate = c.extractValue(result.Data.Result[0].Value)
	}

	// Service with most errors
	mostErrorQuery := fmt.Sprintf(`topk(1, sum by (destination_service_name) (
		increase(istio_requests_total{
			reporter="destination",
			destination_service_namespace="%s",
			response_code=~"5.."
		}[1h])
	))`, namespace)

	if result, err := c.Query(ctx, mostErrorQuery); err == nil && len(result.Data.Result) > 0 {
		overview.MostErrorService = result.Data.Result[0].Metric["destination_service_name"]
		overview.MostErrorCount = c.extractValue(result.Data.Result[0].Value)
	}

	return overview, nil
}

// extractValueFromRange extracts value from range query result
func (c *PrometheusClient) extractValueFromRange(value []interface{}) float64 {
	if len(value) < 2 {
		return 0
	}
	switch v := value[1].(type) {
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		if f != f { // NaN check
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

// =============================================================================
// SLO Dashboard Types and Methods
// =============================================================================

// SLOTarget defines SLO target configuration
type SLOTarget struct {
	Availability float64 // e.g., 99.9 for 99.9%
	LatencyP50   float64 // milliseconds
	LatencyP99   float64 // milliseconds
}

// DefaultSLOTarget returns default SLO targets
func DefaultSLOTarget() SLOTarget {
	return SLOTarget{
		Availability: 99.9,
		LatencyP50:   100,  // 100ms
		LatencyP99:   500,  // 500ms
	}
}

// ServiceSLO represents SLO metrics for a single service
type ServiceSLO struct {
	ServiceName       string  `json:"serviceName"`
	Availability      float64 `json:"availability"`      // current availability %
	AvailabilityTarget float64 `json:"availabilityTarget"` // target %
	AvailabilityMet   bool    `json:"availabilityMet"`
	LatencyP50        float64 `json:"latencyP50"`        // current P50 latency ms
	LatencyP99        float64 `json:"latencyP99"`        // current P99 latency ms
	LatencyTarget     float64 `json:"latencyTarget"`     // target P99 latency ms
	LatencyMet        bool    `json:"latencyMet"`
	ErrorBudgetRemaining float64 `json:"errorBudgetRemaining"` // percentage
	ErrorBudgetConsumed  float64 `json:"errorBudgetConsumed"`  // percentage
}

// SLOOverview represents overall SLO status
type SLOOverview struct {
	Services       []ServiceSLO `json:"services"`
	OverallHealth  string       `json:"overallHealth"`  // healthy, degraded, critical
	ServicesAtRisk int          `json:"servicesAtRisk"`
	TotalServices  int          `json:"totalServices"`
}

// BurnRate represents error budget burn rate
type BurnRate struct {
	ServiceName string  `json:"serviceName"`
	Rate1h      float64 `json:"rate1h"`
	Rate6h      float64 `json:"rate6h"`
	Rate24h     float64 `json:"rate24h"`
	Alerting    bool    `json:"alerting"` // true if burn rate is critical
}

// GetSLOOverview returns SLO metrics for all services
func (c *PrometheusClient) GetSLOOverview(ctx context.Context, namespace string) (*SLOOverview, error) {
	services := []string{
		"auth-service",
		"user-service",
		"board-service",
		"chat-service",
		"noti-service",
		"storage-service",
	}

	target := DefaultSLOTarget()
	overview := &SLOOverview{
		Services:      make([]ServiceSLO, 0),
		TotalServices: len(services),
	}

	for _, svc := range services {
		slo, err := c.getServiceSLO(ctx, svc, namespace, target)
		if err != nil {
			c.logger.Warn("Failed to get SLO for service",
				zap.String("service", svc),
				zap.Error(err))
			slo = &ServiceSLO{
				ServiceName:        svc,
				AvailabilityTarget: target.Availability,
				LatencyTarget:      target.LatencyP99,
			}
		}
		overview.Services = append(overview.Services, *slo)

		// Count services at risk (error budget < 20% or burn rate high)
		if slo.ErrorBudgetRemaining < 20 {
			overview.ServicesAtRisk++
		}
	}

	// Determine overall health
	if overview.ServicesAtRisk == 0 {
		overview.OverallHealth = "healthy"
	} else if overview.ServicesAtRisk <= len(services)/3 {
		overview.OverallHealth = "degraded"
	} else {
		overview.OverallHealth = "critical"
	}

	return overview, nil
}

func (c *PrometheusClient) getServiceSLO(ctx context.Context, serviceName, namespace string, target SLOTarget) (*ServiceSLO, error) {
	slo := &ServiceSLO{
		ServiceName:        serviceName,
		AvailabilityTarget: target.Availability,
		LatencyTarget:      target.LatencyP99,
	}

	// Calculate availability (30 day window for SLO)
	availQuery := fmt.Sprintf(`100 * (1 - sum(rate(istio_requests_total{
		destination_service_name="%s",
		reporter="destination",
		response_code=~"5.."
	}[24h])) / sum(rate(istio_requests_total{
		destination_service_name="%s",
		reporter="destination"
	}[24h])))`, serviceName, serviceName)

	if result, err := c.Query(ctx, availQuery); err == nil && len(result.Data.Result) > 0 {
		slo.Availability = c.extractValue(result.Data.Result[0].Value)
		if slo.Availability == 0 || slo.Availability != slo.Availability {
			slo.Availability = 100 // Default to 100% if no data
		}
	} else {
		slo.Availability = 100
	}

	slo.AvailabilityMet = slo.Availability >= target.Availability

	// P50 latency
	p50Query := fmt.Sprintf(`histogram_quantile(0.50, sum(rate(istio_request_duration_milliseconds_bucket{
		destination_service_name="%s",
		reporter="destination"
	}[5m])) by (le))`, serviceName)

	if result, err := c.Query(ctx, p50Query); err == nil && len(result.Data.Result) > 0 {
		slo.LatencyP50 = c.extractValue(result.Data.Result[0].Value)
	}

	// P99 latency
	p99Query := fmt.Sprintf(`histogram_quantile(0.99, sum(rate(istio_request_duration_milliseconds_bucket{
		destination_service_name="%s",
		reporter="destination"
	}[5m])) by (le))`, serviceName)

	if result, err := c.Query(ctx, p99Query); err == nil && len(result.Data.Result) > 0 {
		slo.LatencyP99 = c.extractValue(result.Data.Result[0].Value)
	}

	slo.LatencyMet = slo.LatencyP99 <= target.LatencyP99 || slo.LatencyP99 == 0

	// Error budget calculation
	// Error budget = (100 - target) / 100
	// e.g., for 99.9% target, budget = 0.1% = 0.001
	errorBudget := (100 - target.Availability) / 100
	actualErrorRate := (100 - slo.Availability) / 100

	if errorBudget > 0 {
		slo.ErrorBudgetConsumed = (actualErrorRate / errorBudget) * 100
		slo.ErrorBudgetRemaining = 100 - slo.ErrorBudgetConsumed
		if slo.ErrorBudgetRemaining < 0 {
			slo.ErrorBudgetRemaining = 0
		}
	} else {
		slo.ErrorBudgetRemaining = 100
	}

	return slo, nil
}

// GetBurnRates returns error budget burn rates for all services
func (c *PrometheusClient) GetBurnRates(ctx context.Context, namespace string) ([]BurnRate, error) {
	services := []string{
		"auth-service",
		"user-service",
		"board-service",
		"chat-service",
		"noti-service",
		"storage-service",
	}

	target := DefaultSLOTarget()
	errorBudget := (100 - target.Availability) / 100 // 0.001 for 99.9%

	var burnRates []BurnRate
	for _, svc := range services {
		br := BurnRate{ServiceName: svc}

		// 1h burn rate
		rate1hQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination",
			response_code=~"5.."
		}[1h])) / sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination"
		}[1h]))`, svc, svc)

		if result, err := c.Query(ctx, rate1hQuery); err == nil && len(result.Data.Result) > 0 {
			errRate := c.extractValue(result.Data.Result[0].Value)
			if errorBudget > 0 && errRate == errRate { // NaN check
				br.Rate1h = errRate / errorBudget
			}
		}

		// 6h burn rate
		rate6hQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination",
			response_code=~"5.."
		}[6h])) / sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination"
		}[6h]))`, svc, svc)

		if result, err := c.Query(ctx, rate6hQuery); err == nil && len(result.Data.Result) > 0 {
			errRate := c.extractValue(result.Data.Result[0].Value)
			if errorBudget > 0 && errRate == errRate {
				br.Rate6h = errRate / errorBudget
			}
		}

		// 24h burn rate
		rate24hQuery := fmt.Sprintf(`sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination",
			response_code=~"5.."
		}[24h])) / sum(rate(istio_requests_total{
			destination_service_name="%s",
			reporter="destination"
		}[24h]))`, svc, svc)

		if result, err := c.Query(ctx, rate24hQuery); err == nil && len(result.Data.Result) > 0 {
			errRate := c.extractValue(result.Data.Result[0].Value)
			if errorBudget > 0 && errRate == errRate {
				br.Rate24h = errRate / errorBudget
			}
		}

		// Alert if 1h burn rate > 14.4 (Google SRE book recommendation)
		// This would consume 100% error budget in ~7 hours
		br.Alerting = br.Rate1h > 14.4 || br.Rate6h > 6

		burnRates = append(burnRates, br)
	}

	return burnRates, nil
}
