// Package testutil은 테스트 유틸리티를 제공합니다.
// 이 파일은 외부 의존성(DB, Redis 등)을 mock하기 위한 인터페이스와 구현체를 제공합니다.
package testutil

import (
	"context"
	"errors"
)

// Pinger는 연결 상태를 확인하기 위한 인터페이스입니다.
// DB, Redis 등의 헬스체크에서 사용됩니다.
// health 패키지에서 이 인터페이스를 사용하여 의존성을 mock할 수 있습니다.
type Pinger interface {
	// Ping은 연결 상태를 확인합니다.
	// 연결이 정상이면 nil을 반환하고, 그렇지 않으면 에러를 반환합니다.
	Ping(ctx context.Context) error
}

// MockPinger는 테스트용 Pinger 구현체입니다.
// PingErr 필드를 설정하여 Ping 호출 시 반환할 에러를 지정할 수 있습니다.
type MockPinger struct {
	// PingErr는 Ping 호출 시 반환할 에러입니다.
	// nil이면 성공으로 처리됩니다.
	PingErr error

	// PingCount는 Ping이 호출된 횟수입니다.
	// 테스트에서 Ping 호출 여부를 확인할 때 사용합니다.
	PingCount int
}

// Ping은 Pinger 인터페이스를 구현합니다.
// PingErr가 설정되어 있으면 해당 에러를 반환하고, 그렇지 않으면 nil을 반환합니다.
func (m *MockPinger) Ping(ctx context.Context) error {
	m.PingCount++
	return m.PingErr
}

// NewMockPinger는 성공하는 MockPinger를 생성합니다.
func NewMockPinger() *MockPinger {
	return &MockPinger{}
}

// NewFailingMockPinger는 항상 실패하는 MockPinger를 생성합니다.
func NewFailingMockPinger(errMsg string) *MockPinger {
	return &MockPinger{
		PingErr: errors.New(errMsg),
	}
}

// Reset은 MockPinger의 상태를 초기화합니다.
func (m *MockPinger) Reset() {
	m.PingCount = 0
}

// SQLExecer는 SQL 실행을 위한 인터페이스입니다.
// DB 작업을 mock할 때 사용합니다.
type SQLExecer interface {
	// Exec는 SQL 쿼리를 실행합니다.
	Exec(ctx context.Context, query string, args ...interface{}) error
}

// MockSQLExecer는 테스트용 SQLExecer 구현체입니다.
type MockSQLExecer struct {
	// ExecErr는 Exec 호출 시 반환할 에러입니다.
	ExecErr error

	// ExecCount는 Exec이 호출된 횟수입니다.
	ExecCount int

	// LastQuery는 마지막으로 실행된 쿼리입니다.
	LastQuery string

	// LastArgs는 마지막 쿼리의 인자들입니다.
	LastArgs []interface{}
}

// Exec는 SQLExecer 인터페이스를 구현합니다.
func (m *MockSQLExecer) Exec(ctx context.Context, query string, args ...interface{}) error {
	m.ExecCount++
	m.LastQuery = query
	m.LastArgs = args
	return m.ExecErr
}

// NewMockSQLExecer는 성공하는 MockSQLExecer를 생성합니다.
func NewMockSQLExecer() *MockSQLExecer {
	return &MockSQLExecer{}
}

// KeyValueStore는 키-값 저장소 인터페이스입니다.
// Redis와 같은 캐시를 mock할 때 사용합니다.
type KeyValueStore interface {
	// Get은 키에 해당하는 값을 반환합니다.
	Get(ctx context.Context, key string) (string, error)

	// Set은 키-값 쌍을 저장합니다.
	Set(ctx context.Context, key, value string) error

	// Delete는 키를 삭제합니다.
	Delete(ctx context.Context, key string) error
}

// MockKeyValueStore는 테스트용 KeyValueStore 구현체입니다.
type MockKeyValueStore struct {
	// Data는 저장된 키-값 쌍입니다.
	Data map[string]string

	// GetErr는 Get 호출 시 반환할 에러입니다.
	GetErr error

	// SetErr는 Set 호출 시 반환할 에러입니다.
	SetErr error

	// DeleteErr는 Delete 호출 시 반환할 에러입니다.
	DeleteErr error
}

// NewMockKeyValueStore는 빈 MockKeyValueStore를 생성합니다.
func NewMockKeyValueStore() *MockKeyValueStore {
	return &MockKeyValueStore{
		Data: make(map[string]string),
	}
}

// Get은 KeyValueStore 인터페이스를 구현합니다.
func (m *MockKeyValueStore) Get(ctx context.Context, key string) (string, error) {
	if m.GetErr != nil {
		return "", m.GetErr
	}
	value, exists := m.Data[key]
	if !exists {
		return "", errors.New("key not found")
	}
	return value, nil
}

// Set은 KeyValueStore 인터페이스를 구현합니다.
func (m *MockKeyValueStore) Set(ctx context.Context, key, value string) error {
	if m.SetErr != nil {
		return m.SetErr
	}
	m.Data[key] = value
	return nil
}

// Delete는 KeyValueStore 인터페이스를 구현합니다.
func (m *MockKeyValueStore) Delete(ctx context.Context, key string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	delete(m.Data, key)
	return nil
}

// Reset은 MockKeyValueStore의 상태를 초기화합니다.
func (m *MockKeyValueStore) Reset() {
	m.Data = make(map[string]string)
	m.GetErr = nil
	m.SetErr = nil
	m.DeleteErr = nil
}

// HTTPClient는 HTTP 클라이언트 인터페이스입니다.
// 외부 API 호출을 mock할 때 사용합니다.
type HTTPClient interface {
	// Do는 HTTP 요청을 실행합니다.
	Do(ctx context.Context, method, url string, body interface{}) ([]byte, int, error)
}

// MockHTTPClient는 테스트용 HTTPClient 구현체입니다.
type MockHTTPClient struct {
	// ResponseBody는 반환할 응답 본문입니다.
	ResponseBody []byte

	// StatusCode는 반환할 HTTP 상태 코드입니다.
	StatusCode int

	// Err는 반환할 에러입니다.
	Err error

	// CallCount는 Do가 호출된 횟수입니다.
	CallCount int

	// LastMethod는 마지막 요청의 HTTP 메서드입니다.
	LastMethod string

	// LastURL는 마지막 요청의 URL입니다.
	LastURL string

	// LastBody는 마지막 요청의 본문입니다.
	LastBody interface{}
}

// NewMockHTTPClient는 성공 응답을 반환하는 MockHTTPClient를 생성합니다.
func NewMockHTTPClient(statusCode int, responseBody []byte) *MockHTTPClient {
	return &MockHTTPClient{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
	}
}

// Do는 HTTPClient 인터페이스를 구현합니다.
func (m *MockHTTPClient) Do(ctx context.Context, method, url string, body interface{}) ([]byte, int, error) {
	m.CallCount++
	m.LastMethod = method
	m.LastURL = url
	m.LastBody = body
	return m.ResponseBody, m.StatusCode, m.Err
}

// Reset은 MockHTTPClient의 상태를 초기화합니다.
func (m *MockHTTPClient) Reset() {
	m.CallCount = 0
	m.LastMethod = ""
	m.LastURL = ""
	m.LastBody = nil
}
