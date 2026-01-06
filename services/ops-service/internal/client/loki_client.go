package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// LokiClient is a client for Loki API
type LokiClient struct {
	baseURL   string
	namespace string
	client    *http.Client
	logger    *zap.Logger
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Service     string    `json:"service"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	TraceID     string    `json:"traceId,omitempty"`
	SpanID      string    `json:"spanId,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// LogQueryParams represents query parameters for log search
type LogQueryParams struct {
	Service   string    `json:"service,omitempty"`
	Level     string    `json:"level,omitempty"`
	Query     string    `json:"query,omitempty"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Limit     int       `json:"limit"`
}

// LogQueryResult represents the result of a log query
type LogQueryResult struct {
	Entries    []LogEntry `json:"entries"`
	TotalCount int        `json:"totalCount"`
}

// ServiceInfo represents basic service information
type ServiceInfo struct {
	Name string `json:"name"`
}

// LokiQueryResponse represents the Loki API response
type LokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"` // [timestamp_ns, log_line]
		} `json:"result"`
	} `json:"data"`
}

// NewLokiClient creates a new Loki client
func NewLokiClient(baseURL, namespace string, timeout time.Duration, logger *zap.Logger) *LokiClient {
	return &LokiClient{
		baseURL:   baseURL,
		namespace: namespace,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// GetLogs queries logs from Loki
func (c *LokiClient) GetLogs(params LogQueryParams) (*LogQueryResult, error) {
	// Build LogQL query
	logQL := c.buildLogQLQuery(params)

	// Build URL with query parameters
	queryURL := fmt.Sprintf("%s/loki/api/v1/query_range", c.baseURL)
	u, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("query", logQL)
	q.Set("start", strconv.FormatInt(params.Start.UnixNano(), 10))
	q.Set("end", strconv.FormatInt(params.End.UnixNano(), 10))

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("direction", "backward") // Most recent first

	u.RawQuery = q.Encode()

	c.logger.Debug("Querying Loki",
		zap.String("url", u.String()),
		zap.String("query", logQL))

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Loki returned status %d", resp.StatusCode)
	}

	var lokiResp LokiQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Loki response: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("Loki query failed with status: %s", lokiResp.Status)
	}

	// Parse results
	entries := c.parseLogEntries(lokiResp)

	return &LogQueryResult{
		Entries:    entries,
		TotalCount: len(entries),
	}, nil
}

// GetServices returns list of available services
func (c *LokiClient) GetServices() ([]ServiceInfo, error) {
	// Query Loki for unique app labels
	queryURL := fmt.Sprintf("%s/loki/api/v1/label/app/values", c.baseURL)
	u, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Query last 24 hours for available services
	q := u.Query()
	q.Set("start", strconv.FormatInt(time.Now().Add(-24*time.Hour).UnixNano(), 10))
	q.Set("end", strconv.FormatInt(time.Now().UnixNano(), 10))
	u.RawQuery = q.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query Loki labels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Loki returned status %d", resp.StatusCode)
	}

	var labelsResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&labelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode Loki response: %w", err)
	}

	var services []ServiceInfo
	for _, app := range labelsResp.Data {
		services = append(services, ServiceInfo{Name: app})
	}

	return services, nil
}

// buildLogQLQuery builds a LogQL query string
func (c *LokiClient) buildLogQLQuery(params LogQueryParams) string {
	// Start with namespace filter
	query := fmt.Sprintf(`{namespace="%s"`, c.namespace)

	// Add service filter
	if params.Service != "" {
		query += fmt.Sprintf(`, app="%s"`, params.Service)
	}

	query += "}"

	// Add JSON parsing
	query += " | json"

	// Add level filter
	if params.Level != "" {
		query += fmt.Sprintf(` | level=~"(?i)%s"`, params.Level)
	}

	// Add text search filter
	if params.Query != "" {
		query += fmt.Sprintf(` |~ "(?i)%s"`, params.Query)
	}

	return query
}

// parseLogEntries parses Loki response into LogEntry slice
func (c *LokiClient) parseLogEntries(resp LokiQueryResponse) []LogEntry {
	var entries []LogEntry

	for _, stream := range resp.Data.Result {
		labels := stream.Stream

		for _, value := range stream.Values {
			if len(value) < 2 {
				continue
			}

			// Parse timestamp (nanoseconds)
			tsNano, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				c.logger.Warn("Failed to parse timestamp", zap.String("ts", value[0]))
				continue
			}
			ts := time.Unix(0, tsNano)

			// Parse log line
			logLine := value[1]
			entry := LogEntry{
				Timestamp: ts,
				Service:   labels["app"],
				Labels:    labels,
			}

			// Try to parse as JSON to extract structured fields
			var logJSON map[string]interface{}
			if err := json.Unmarshal([]byte(logLine), &logJSON); err == nil {
				// Extract level
				if level, ok := logJSON["level"].(string); ok {
					entry.Level = level
				}
				// Extract message
				if msg, ok := logJSON["msg"].(string); ok {
					entry.Message = msg
				} else if msg, ok := logJSON["message"].(string); ok {
					entry.Message = msg
				}
				// Extract trace ID
				if traceID, ok := logJSON["trace_id"].(string); ok {
					entry.TraceID = traceID
				} else if traceID, ok := logJSON["traceId"].(string); ok {
					entry.TraceID = traceID
				}
				// Extract span ID
				if spanID, ok := logJSON["span_id"].(string); ok {
					entry.SpanID = spanID
				} else if spanID, ok := logJSON["spanId"].(string); ok {
					entry.SpanID = spanID
				}
			} else {
				// Plain text log
				entry.Message = logLine
				entry.Level = "info" // Default level
			}

			// Fallback to stream labels if not found in JSON
			if entry.Level == "" {
				if level, ok := labels["level"]; ok {
					entry.Level = level
				}
			}

			entries = append(entries, entry)
		}
	}

	return entries
}
