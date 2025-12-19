// Package testutil은 테스트 유틸리티를 제공합니다.
// 이 파일은 assertions.go의 테스트 assertion 함수들을 테스트합니다.
//
// 참고: assertion 함수들은 *testing.T를 받으므로,
// 성공 케이스만 직접 테스트하고 실패 케이스는 로직 검증으로 확인합니다.
package testutil

import (
	"errors"
	"testing"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// TestAssertEqual_Success는 동등한 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertEqual_Success(t *testing.T) {
	// 동일한 정수
	AssertEqual(t, 42, 42, "정수 동등성 검사")

	// 동일한 문자열
	AssertEqual(t, "hello", "hello", "문자열 동등성 검사")

	// 동일한 슬라이스
	AssertEqual(t, []int{1, 2, 3}, []int{1, 2, 3}, "슬라이스 동등성 검사")

	// 동일한 맵
	AssertEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1}, "맵 동등성 검사")
}

// TestAssertNotEqual_Success는 다른 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertNotEqual_Success(t *testing.T) {
	// 다른 정수
	AssertNotEqual(t, 42, 43, "정수 비동등성 검사")

	// 다른 문자열
	AssertNotEqual(t, "hello", "world", "문자열 비동등성 검사")
}

// TestAssertNil_Success는 nil 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertNil_Success(t *testing.T) {
	// 명시적 nil
	AssertNil(t, nil, "명시적 nil 검사")

	// nil 포인터
	var ptr *string
	AssertNil(t, ptr, "nil 포인터 검사")

	// nil 슬라이스
	var slice []int
	AssertNil(t, slice, "nil 슬라이스 검사")

	// nil 맵
	var m map[string]int
	AssertNil(t, m, "nil 맵 검사")

	// nil 채널
	var ch chan int
	AssertNil(t, ch, "nil 채널 검사")

	// nil 함수
	var fn func()
	AssertNil(t, fn, "nil 함수 검사")
}

// TestAssertNotNil_Success는 non-nil 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertNotNil_Success(t *testing.T) {
	// 문자열 값
	AssertNotNil(t, "not nil", "문자열 값 검사")

	// 정수 값
	AssertNotNil(t, 42, "정수 값 검사")

	// 빈 슬라이스 (nil이 아님)
	AssertNotNil(t, []int{}, "빈 슬라이스 검사")

	// 빈 맵 (nil이 아님)
	AssertNotNil(t, map[string]int{}, "빈 맵 검사")
}

// TestAssertTrue_Success는 true 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertTrue_Success(t *testing.T) {
	AssertTrue(t, true, "true 값 검사")
	AssertTrue(t, 1 == 1, "표현식 true 검사")
}

// TestAssertFalse_Success는 false 값에 대해 테스트가 통과하는지 검증합니다.
func TestAssertFalse_Success(t *testing.T) {
	AssertFalse(t, false, "false 값 검사")
	AssertFalse(t, 1 == 2, "표현식 false 검사")
}

// TestAssertNoError_Success는 nil 에러에 대해 테스트가 통과하는지 검증합니다.
func TestAssertNoError_Success(t *testing.T) {
	AssertNoError(t, nil, "nil 에러 검사")

	// 실제 에러가 없는 함수 결과
	_, err := noErrorFunc()
	AssertNoError(t, err, "에러 없는 함수 결과 검사")
}

// 테스트용 헬퍼 함수
func noErrorFunc() (string, error) {
	return "success", nil
}

// TestAssertError_Success는 non-nil 에러에 대해 테스트가 통과하는지 검증합니다.
func TestAssertError_Success(t *testing.T) {
	err := errors.New("sample error")
	AssertError(t, err, "에러 존재 검사")
}

// TestAssertErrorContains_Success는 에러 메시지에 특정 문자열이 포함된 경우 테스트가 통과하는지 검증합니다.
func TestAssertErrorContains_Success(t *testing.T) {
	err := errors.New("this is a test error message")

	// 포함된 문자열 검사
	AssertErrorContains(t, err, "test", "에러 메시지 포함 검사")
	AssertErrorContains(t, err, "error", "다른 포함된 문자열 검사")
}

// TestAssertAppError_Success는 올바른 AppError 코드에 대해 테스트가 통과하는지 검증합니다.
func TestAssertAppError_Success(t *testing.T) {
	// NotFound 에러
	notFoundErr := apperrors.NotFound("user not found", "details")
	AssertAppError(t, notFoundErr, apperrors.ErrCodeNotFound, "NotFound 에러 검사")

	// Validation 에러
	validationErr := apperrors.Validation("validation failed", "invalid value")
	AssertAppError(t, validationErr, apperrors.ErrCodeValidation, "Validation 에러 검사")

	// AlreadyExists 에러
	existsErr := apperrors.AlreadyExists("resource exists", "already exists")
	AssertAppError(t, existsErr, apperrors.ErrCodeAlreadyExists, "AlreadyExists 에러 검사")
}

// TestAssertHTTPStatus_Success는 동일한 HTTP 상태 코드에 대해 테스트가 통과하는지 검증합니다.
func TestAssertHTTPStatus_Success(t *testing.T) {
	AssertHTTPStatus(t, 200, 200, "200 상태 코드 검사")
	AssertHTTPStatus(t, 404, 404, "404 상태 코드 검사")
	AssertHTTPStatus(t, 500, 500, "500 상태 코드 검사")
}

// TestAssertJSONContains_Success는 JSON에 예상 키-값이 포함된 경우 테스트가 통과하는지 검증합니다.
func TestAssertJSONContains_Success(t *testing.T) {
	// 단일 키-값 검사
	body := []byte(`{"name":"test","value":123}`)
	AssertJSONContains(t, body, map[string]interface{}{"name": "test"}, "단일 키-값 검사")

	// 여러 키-값 검사
	AssertJSONContains(t, body, map[string]interface{}{
		"name": "test",
	}, "여러 키-값 검사")

	// 중첩된 JSON
	nestedBody := []byte(`{"data":{"id":"123"}}`)
	AssertJSONContains(t, nestedBody, map[string]interface{}{
		"data": map[string]interface{}{"id": "123"},
	}, "중첩된 JSON 검사")
}

// TestAssertValidUUID_Success는 유효한 UUID에 대해 테스트가 통과하는지 검증합니다.
func TestAssertValidUUID_Success(t *testing.T) {
	// 표준 UUID v4 형식
	AssertValidUUID(t, "550e8400-e29b-41d4-a716-446655440000", "표준 UUID 검사")

	// 다른 유효한 UUID
	AssertValidUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "다른 UUID 검사")
}

// TestAssertNotEmptyString_Success는 비어있지 않은 문자열에 대해 테스트가 통과하는지 검증합니다.
func TestAssertNotEmptyString_Success(t *testing.T) {
	AssertNotEmptyString(t, "hello", "일반 문자열 검사")
	AssertNotEmptyString(t, " ", "공백 문자열 검사") // 공백도 비어있지 않음
}

// TestAssertLen_Success는 올바른 길이에 대해 테스트가 통과하는지 검증합니다.
func TestAssertLen_Success(t *testing.T) {
	// 슬라이스 길이
	AssertLen(t, []int{1, 2, 3}, 3, "슬라이스 길이 검사")

	// 빈 슬라이스
	AssertLen(t, []int{}, 0, "빈 슬라이스 길이 검사")

	// 문자열 길이
	AssertLen(t, "hello", 5, "문자열 길이 검사")

	// 맵 길이
	AssertLen(t, map[string]int{"a": 1, "b": 2}, 2, "맵 길이 검사")
}

// TestAssertContains_Success는 슬라이스에 요소가 포함된 경우 테스트가 통과하는지 검증합니다.
func TestAssertContains_Success(t *testing.T) {
	// 정수 슬라이스
	AssertContains(t, []int{1, 2, 3}, 2, "정수 슬라이스 포함 검사")

	// 문자열 슬라이스
	AssertContains(t, []string{"a", "b", "c"}, "b", "문자열 슬라이스 포함 검사")

	// 첫 번째 요소
	AssertContains(t, []int{1, 2, 3}, 1, "첫 번째 요소 포함 검사")

	// 마지막 요소
	AssertContains(t, []int{1, 2, 3}, 3, "마지막 요소 포함 검사")
}

// TestAssertStringContains_Success는 문자열에 부분 문자열이 포함된 경우 테스트가 통과하는지 검증합니다.
func TestAssertStringContains_Success(t *testing.T) {
	AssertStringContains(t, "hello world", "world", "부분 문자열 포함 검사")
	AssertStringContains(t, "hello world", "hello", "시작 문자열 포함 검사")
	AssertStringContains(t, "hello world", "lo wo", "중간 문자열 포함 검사")
}

// TestIsNilHelperBehavior는 isNil 헬퍼 함수의 동작을 간접적으로 검증합니다.
// 다양한 nil 타입에 대해 AssertNil이 올바르게 동작하는지 확인합니다.
func TestIsNilHelperBehavior(t *testing.T) {
	// interface nil
	var iface interface{}
	AssertNil(t, iface, "interface nil 검사")

	// 다양한 nil 타입들
	var ptr *int
	var slice []string
	var mapVal map[string]int
	var ch chan bool
	var fn func()

	AssertNil(t, ptr, "포인터 nil 검사")
	AssertNil(t, slice, "슬라이스 nil 검사")
	AssertNil(t, mapVal, "맵 nil 검사")
	AssertNil(t, ch, "채널 nil 검사")
	AssertNil(t, fn, "함수 nil 검사")
}

// TestFormatMessageBehavior는 메시지 포맷팅이 올바르게 처리되는지 검증합니다.
// 메시지 인자 유무에 관계없이 assertion이 정상 동작하는지 확인합니다.
func TestFormatMessageBehavior(t *testing.T) {
	// 메시지 없이 호출
	AssertEqual(t, 1, 1)

	// 단일 문자열 메시지와 함께 호출
	AssertEqual(t, 1, 1, "단일 메시지")

	// 여러 인자와 함께 호출
	AssertEqual(t, 1, 1, "format: %d", 42)
}

// TestAppErrorHelperFunctions는 AppError 관련 헬퍼 함수들이 올바르게 동작하는지 검증합니다.
func TestAppErrorHelperFunctions(t *testing.T) {
	// 다양한 AppError 타입 테스트
	tests := []struct {
		name string // 테스트 케이스 이름
		err  error  // 테스트할 에러
		code string // 예상 코드
	}{
		{
			name: "NotFound 에러",
			err:  apperrors.NotFound("resource not found", "details"),
			code: apperrors.ErrCodeNotFound,
		},
		{
			name: "Validation 에러",
			err:  apperrors.Validation("validation failed", "invalid"),
			code: apperrors.ErrCodeValidation,
		},
		{
			name: "AlreadyExists 에러",
			err:  apperrors.AlreadyExists("item exists", "exists"),
			code: apperrors.ErrCodeAlreadyExists,
		},
		{
			name: "Unauthorized 에러",
			err:  apperrors.Unauthorized("auth failed", "unauthorized"),
			code: apperrors.ErrCodeUnauthorized,
		},
		{
			name: "Forbidden 에러",
			err:  apperrors.Forbidden("access denied", "forbidden"),
			code: apperrors.ErrCodeForbidden,
		},
		{
			name: "Internal 에러",
			err:  apperrors.Internal("system error", "internal error"),
			code: apperrors.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertAppError(t, tt.err, tt.code)
		})
	}
}
