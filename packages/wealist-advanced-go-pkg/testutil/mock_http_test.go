// Package testutil은 테스트 유틸리티를 제공합니다.
// 이 파일은 mock_http.go의 HTTP 테스트 헬퍼 함수들을 테스트합니다.
package testutil

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestDefaultHTTPTestConfig는 기본 HTTP 테스트 설정이 올바르게 생성되는지 검증합니다.
func TestDefaultHTTPTestConfig(t *testing.T) {
	config := DefaultHTTPTestConfig()

	// 기본값 검증
	if !config.UseTestMode {
		t.Error("UseTestMode 기본값이 true여야 합니다")
	}
	if config.Logger == nil {
		t.Error("Logger가 nil이 아니어야 합니다")
	}
	if config.Middleware != nil {
		t.Error("Middleware 기본값이 nil이어야 합니다")
	}
}

// TestSetupTestRouter는 테스트 라우터가 올바르게 설정되는지 검증합니다.
func TestSetupTestRouter(t *testing.T) {
	tests := []struct {
		name   string                            // 테스트 케이스 이름
		config *HTTPTestConfig                   // 테스트 설정
		check  func(t *testing.T, r *gin.Engine) // 검증 함수
	}{
		{
			name:   "nil config로 기본 설정 사용",
			config: nil,
			check: func(t *testing.T, r *gin.Engine) {
				if r == nil {
					t.Error("라우터가 nil이 아니어야 합니다")
				}
			},
		},
		{
			name:   "커스텀 config로 설정",
			config: DefaultHTTPTestConfig(),
			check: func(t *testing.T, r *gin.Engine) {
				if r == nil {
					t.Error("라우터가 nil이 아니어야 합니다")
				}
			},
		},
		{
			name: "미들웨어가 적용되는지 확인",
			config: &HTTPTestConfig{
				UseTestMode: true,
				Middleware: []gin.HandlerFunc{
					func(c *gin.Context) {
						c.Set("test_key", "test_value")
						c.Next()
					},
				},
			},
			check: func(t *testing.T, r *gin.Engine) {
				if r == nil {
					t.Error("라우터가 nil이 아니어야 합니다")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupTestRouter(tt.config)
			tt.check(t, router)
		})
	}
}

// TestHTTPResponse는 HTTPResponse 타입의 메서드들을 검증합니다.
func TestHTTPResponse(t *testing.T) {
	t.Run("JSON 언마샬링 성공", func(t *testing.T) {
		resp := &HTTPResponse{
			Code: 200,
			Body: []byte(`{"name":"test","value":123}`),
		}

		var result map[string]interface{}
		err := resp.JSON(&result)
		if err != nil {
			t.Errorf("JSON 언마샬링 실패: %v", err)
		}
		if result["name"] != "test" {
			t.Errorf("name 필드가 'test'여야 하는데 '%v'입니다", result["name"])
		}
	})

	t.Run("JSON 언마샬링 실패", func(t *testing.T) {
		resp := &HTTPResponse{
			Code: 200,
			Body: []byte(`invalid json`),
		}

		var result map[string]interface{}
		err := resp.JSON(&result)
		if err == nil {
			t.Error("잘못된 JSON에서 에러가 발생해야 합니다")
		}
	})

	t.Run("BodyString 반환", func(t *testing.T) {
		expected := "test body content"
		resp := &HTTPResponse{
			Code: 200,
			Body: []byte(expected),
		}

		if resp.BodyString() != expected {
			t.Errorf("BodyString()이 '%s'를 반환해야 하는데 '%s'를 반환했습니다", expected, resp.BodyString())
		}
	})
}

// TestPerformRequest는 HTTP 요청 수행 함수를 검증합니다.
func TestPerformRequest(t *testing.T) {
	router := SetupTestRouter(nil)

	// 테스트용 핸들러 등록
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	router.POST("/echo", func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, body)
	})
	router.GET("/header-check", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		c.JSON(200, gin.H{"auth": auth})
	})
	router.GET("/query-check", func(c *gin.Context) {
		name := c.Query("name")
		c.JSON(200, gin.H{"name": name})
	})

	tests := []struct {
		name       string      // 테스트 케이스 이름
		req        HTTPRequest // 요청 설정
		wantStatus int         // 예상 HTTP 상태 코드
	}{
		{
			name: "GET 요청 성공",
			req: HTTPRequest{
				Method: http.MethodGet,
				Path:   "/test",
			},
			wantStatus: 200,
		},
		{
			name: "POST 요청 - JSON body 포함",
			req: HTTPRequest{
				Method: http.MethodPost,
				Path:   "/echo",
				Body:   map[string]string{"key": "value"},
			},
			wantStatus: 200,
		},
		{
			name: "GET 요청 - 커스텀 헤더 포함",
			req: HTTPRequest{
				Method:  http.MethodGet,
				Path:    "/header-check",
				Headers: map[string]string{"Authorization": "Bearer test-token"},
			},
			wantStatus: 200,
		},
		{
			name: "GET 요청 - 쿼리 파라미터 포함",
			req: HTTPRequest{
				Method: http.MethodGet,
				Path:   "/query-check",
				Query:  map[string]string{"name": "test-name"},
			},
			wantStatus: 200,
		},
		{
			name: "존재하지 않는 경로",
			req: HTTPRequest{
				Method: http.MethodGet,
				Path:   "/not-found",
			},
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := PerformRequest(t, router, tt.req)
			if resp.Code != tt.wantStatus {
				t.Errorf("상태 코드가 %d여야 하는데 %d입니다", tt.wantStatus, resp.Code)
			}
		})
	}
}

// TestHTTPMethodHelpers는 GET, POST, PUT, PATCH, DELETE 헬퍼 함수들을 검증합니다.
func TestHTTPMethodHelpers(t *testing.T) {
	router := SetupTestRouter(nil)

	// 각 메서드별 핸들러 등록
	router.GET("/resource", func(c *gin.Context) { c.JSON(200, gin.H{"method": "GET"}) })
	router.POST("/resource", func(c *gin.Context) { c.JSON(201, gin.H{"method": "POST"}) })
	router.PUT("/resource", func(c *gin.Context) { c.JSON(200, gin.H{"method": "PUT"}) })
	router.PATCH("/resource", func(c *gin.Context) { c.JSON(200, gin.H{"method": "PATCH"}) })
	router.DELETE("/resource", func(c *gin.Context) { c.JSON(204, gin.H{"method": "DELETE"}) })

	t.Run("GET 헬퍼", func(t *testing.T) {
		resp := GET(t, router, "/resource", nil)
		if resp.Code != 200 {
			t.Errorf("GET 응답 코드가 200이어야 하는데 %d입니다", resp.Code)
		}
	})

	t.Run("POST 헬퍼", func(t *testing.T) {
		resp := POST(t, router, "/resource", map[string]string{"test": "data"}, nil)
		if resp.Code != 201 {
			t.Errorf("POST 응답 코드가 201이어야 하는데 %d입니다", resp.Code)
		}
	})

	t.Run("PUT 헬퍼", func(t *testing.T) {
		resp := PUT(t, router, "/resource", map[string]string{"test": "data"}, nil)
		if resp.Code != 200 {
			t.Errorf("PUT 응답 코드가 200이어야 하는데 %d입니다", resp.Code)
		}
	})

	t.Run("PATCH 헬퍼", func(t *testing.T) {
		resp := PATCH(t, router, "/resource", map[string]string{"test": "data"}, nil)
		if resp.Code != 200 {
			t.Errorf("PATCH 응답 코드가 200이어야 하는데 %d입니다", resp.Code)
		}
	})

	t.Run("DELETE 헬퍼", func(t *testing.T) {
		resp := DELETE(t, router, "/resource", nil)
		if resp.Code != 204 {
			t.Errorf("DELETE 응답 코드가 204이어야 하는데 %d입니다", resp.Code)
		}
	})
}

// TestWithAuthHeader는 Authorization 헤더 생성 함수를 검증합니다.
func TestWithAuthHeader(t *testing.T) {
	token := "test-jwt-token"
	headers := WithAuthHeader(token)

	expected := "Bearer " + token
	if headers["Authorization"] != expected {
		t.Errorf("Authorization 헤더가 '%s'여야 하는데 '%s'입니다", expected, headers["Authorization"])
	}
}

// TestMergeHeaders는 헤더 병합 함수를 검증합니다.
func TestMergeHeaders(t *testing.T) {
	tests := []struct {
		name     string              // 테스트 케이스 이름
		headers  []map[string]string // 병합할 헤더들
		expected map[string]string   // 예상 결과
	}{
		{
			name:     "빈 헤더 병합",
			headers:  []map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "단일 헤더",
			headers: []map[string]string{
				{"A": "1"},
			},
			expected: map[string]string{"A": "1"},
		},
		{
			name: "여러 헤더 병합",
			headers: []map[string]string{
				{"A": "1"},
				{"B": "2"},
			},
			expected: map[string]string{"A": "1", "B": "2"},
		},
		{
			name: "중복 키는 나중 값으로 덮어씀",
			headers: []map[string]string{
				{"A": "1"},
				{"A": "2"},
			},
			expected: map[string]string{"A": "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeHeaders(tt.headers...)
			if len(result) != len(tt.expected) {
				t.Errorf("결과 길이가 %d여야 하는데 %d입니다", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("키 '%s'의 값이 '%s'여야 하는데 '%s'입니다", k, v, result[k])
				}
			}
		})
	}
}

// TestMockHTTPServer는 mock HTTP 서버 생성 함수를 검증합니다.
func TestMockHTTPServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock response"))
	})

	server, cleanup := MockHTTPServer(t, handler)
	defer cleanup()

	// 서버가 실행 중인지 확인
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("서버 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("응답 코드가 200이어야 하는데 %d입니다", resp.StatusCode)
	}
}

// TestMockJSONHandler는 JSON 응답 핸들러 생성 함수를 검증합니다.
func TestMockJSONHandler(t *testing.T) {
	response := map[string]string{"status": "success"}
	handler := MockJSONHandler(http.StatusCreated, response)

	server, cleanup := MockHTTPServer(t, handler)
	defer cleanup()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("서버 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 상태 코드 검증
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("응답 코드가 201이어야 하는데 %d입니다", resp.StatusCode)
	}

	// Content-Type 검증
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type이 'application/json'이어야 하는데 '%s'입니다", contentType)
	}
}

// TestMockErrorHandler는 에러 응답 핸들러 생성 함수를 검증합니다.
func TestMockErrorHandler(t *testing.T) {
	errorMsg := "not found"
	handler := MockErrorHandler(http.StatusNotFound, errorMsg)

	server, cleanup := MockHTTPServer(t, handler)
	defer cleanup()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("서버 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 상태 코드 검증
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("응답 코드가 404이어야 하는데 %d입니다", resp.StatusCode)
	}

	// Content-Type 검증
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type이 'application/json'이어야 하는데 '%s'입니다", contentType)
	}
}
