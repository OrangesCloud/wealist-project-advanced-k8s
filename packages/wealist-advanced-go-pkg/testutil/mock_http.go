package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HTTPTestConfig holds configuration for HTTP test setup
type HTTPTestConfig struct {
	// UseTestMode sets gin to test mode
	UseTestMode bool
	// Logger to inject into context
	Logger *zap.Logger
	// Middleware to apply to the router
	Middleware []gin.HandlerFunc
}

// DefaultHTTPTestConfig returns default HTTP test configuration
func DefaultHTTPTestConfig() *HTTPTestConfig {
	return &HTTPTestConfig{
		UseTestMode: true,
		Logger:      zap.NewNop(),
		Middleware:  nil,
	}
}

// SetupTestRouter creates a Gin router for testing with common middleware.
func SetupTestRouter(config *HTTPTestConfig) *gin.Engine {
	if config == nil {
		config = DefaultHTTPTestConfig()
	}

	if config.UseTestMode {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()

	// Inject logger into context
	if config.Logger != nil {
		r.Use(func(c *gin.Context) {
			c.Set("logger", config.Logger)
			c.Next()
		})
	}

	// Apply additional middleware
	for _, mw := range config.Middleware {
		r.Use(mw)
	}

	return r
}

// HTTPRequest represents an HTTP request for testing
type HTTPRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	Query   map[string]string
}

// HTTPResponse represents the response from an HTTP test request
type HTTPResponse struct {
	Code int
	Body []byte
}

// JSON unmarshals the response body into the given interface
func (r *HTTPResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// BodyString returns the response body as a string
func (r *HTTPResponse) BodyString() string {
	return string(r.Body)
}

// PerformRequest performs an HTTP request against a test router.
func PerformRequest(t *testing.T, router *gin.Engine, req HTTPRequest) *HTTPResponse {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set default content type for JSON
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set query parameters
	if len(req.Query) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.Query {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return &HTTPResponse{
		Code: w.Code,
		Body: w.Body.Bytes(),
	}
}

// GET performs a GET request
func GET(t *testing.T, router *gin.Engine, path string, headers map[string]string) *HTTPResponse {
	return PerformRequest(t, router, HTTPRequest{
		Method:  http.MethodGet,
		Path:    path,
		Headers: headers,
	})
}

// POST performs a POST request with JSON body
func POST(t *testing.T, router *gin.Engine, path string, body interface{}, headers map[string]string) *HTTPResponse {
	return PerformRequest(t, router, HTTPRequest{
		Method:  http.MethodPost,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// PUT performs a PUT request with JSON body
func PUT(t *testing.T, router *gin.Engine, path string, body interface{}, headers map[string]string) *HTTPResponse {
	return PerformRequest(t, router, HTTPRequest{
		Method:  http.MethodPut,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// PATCH performs a PATCH request with JSON body
func PATCH(t *testing.T, router *gin.Engine, path string, body interface{}, headers map[string]string) *HTTPResponse {
	return PerformRequest(t, router, HTTPRequest{
		Method:  http.MethodPatch,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// DELETE performs a DELETE request
func DELETE(t *testing.T, router *gin.Engine, path string, headers map[string]string) *HTTPResponse {
	return PerformRequest(t, router, HTTPRequest{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: headers,
	})
}

// WithAuthHeader adds an Authorization header with Bearer token
func WithAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// MergeHeaders merges multiple header maps into one
func MergeHeaders(headers ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		for k, v := range h {
			result[k] = v
		}
	}
	return result
}

// MockHTTPServer creates a mock HTTP server for testing external API calls.
// Returns the server and a cleanup function.
func MockHTTPServer(t *testing.T, handler http.Handler) (*httptest.Server, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	cleanup := func() {
		server.Close()
	}
	return server, cleanup
}

// MockJSONHandler creates an http.HandlerFunc that returns JSON response
func MockJSONHandler(statusCode int, response interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// MockErrorHandler creates an http.HandlerFunc that returns an error response
func MockErrorHandler(statusCode int, errorMessage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errorMessage})
	}
}
