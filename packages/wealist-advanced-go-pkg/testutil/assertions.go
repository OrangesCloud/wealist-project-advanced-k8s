package testutil

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v. %s", expected, actual, formatMessage(msgAndArgs...))
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected values to be different, got %v. %s", actual, formatMessage(msgAndArgs...))
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isNil(value) {
		t.Errorf("Expected nil, got %v. %s", value, formatMessage(msgAndArgs...))
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if isNil(value) {
		t.Errorf("Expected non-nil value. %s", formatMessage(msgAndArgs...))
	}
}

// AssertTrue checks if a value is true
func AssertTrue(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !value {
		t.Errorf("Expected true, got false. %s", formatMessage(msgAndArgs...))
	}
}

// AssertFalse checks if a value is false
func AssertFalse(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if value {
		t.Errorf("Expected false, got true. %s", formatMessage(msgAndArgs...))
	}
}

// AssertNoError checks that an error is nil
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got: %v. %s", err, formatMessage(msgAndArgs...))
	}
}

// AssertError checks that an error is not nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected an error, got nil. %s", formatMessage(msgAndArgs...))
	}
}

// AssertErrorContains checks that an error message contains the expected substring
func AssertErrorContains(t *testing.T, err error, substring string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected an error containing '%s', got nil. %s", substring, formatMessage(msgAndArgs...))
		return
	}
	if !strings.Contains(err.Error(), substring) {
		t.Errorf("Expected error containing '%s', got '%s'. %s", substring, err.Error(), formatMessage(msgAndArgs...))
	}
}

// AssertAppError checks that an error is an AppError with the expected code
func AssertAppError(t *testing.T, err error, expectedCode string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected AppError with code '%s', got nil. %s", expectedCode, formatMessage(msgAndArgs...))
		return
	}
	appErr := apperrors.AsAppError(err)
	if appErr == nil {
		t.Errorf("Expected AppError, got %T: %v. %s", err, err, formatMessage(msgAndArgs...))
		return
	}
	if appErr.Code != expectedCode {
		t.Errorf("Expected error code '%s', got '%s'. %s", expectedCode, appErr.Code, formatMessage(msgAndArgs...))
	}
}

// AssertHTTPStatus checks that the HTTP response has the expected status code
func AssertHTTPStatus(t *testing.T, expected, actual int, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected HTTP status %d, got %d. %s", expected, actual, formatMessage(msgAndArgs...))
	}
}

// AssertJSONContains checks that the JSON response contains the expected key-value pairs
func AssertJSONContains(t *testing.T, body []byte, expected map[string]interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	var actual map[string]interface{}
	if err := json.Unmarshal(body, &actual); err != nil {
		t.Errorf("Failed to unmarshal JSON: %v. %s", err, formatMessage(msgAndArgs...))
		return
	}
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("Expected key '%s' not found in response. %s", key, formatMessage(msgAndArgs...))
			continue
		}
		if !reflect.DeepEqual(expectedValue, actualValue) {
			t.Errorf("Key '%s': expected %v, got %v. %s", key, expectedValue, actualValue, formatMessage(msgAndArgs...))
		}
	}
}

// AssertValidUUID checks that a string is a valid UUID
func AssertValidUUID(t *testing.T, value string, msgAndArgs ...interface{}) {
	t.Helper()
	if _, err := uuid.Parse(value); err != nil {
		t.Errorf("Expected valid UUID, got '%s': %v. %s", value, err, formatMessage(msgAndArgs...))
	}
}

// AssertNotEmptyString checks that a string is not empty
func AssertNotEmptyString(t *testing.T, value string, msgAndArgs ...interface{}) {
	t.Helper()
	if value == "" {
		t.Errorf("Expected non-empty string. %s", formatMessage(msgAndArgs...))
	}
}

// AssertLen checks that a slice/map/string has the expected length
func AssertLen(t *testing.T, object interface{}, expectedLen int, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := reflect.ValueOf(object).Len()
	if actualLen != expectedLen {
		t.Errorf("Expected length %d, got %d. %s", expectedLen, actualLen, formatMessage(msgAndArgs...))
	}
}

// AssertContains checks that a slice contains an element
func AssertContains(t *testing.T, slice interface{}, element interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	sliceValue := reflect.ValueOf(slice)
	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			return
		}
	}
	t.Errorf("Expected slice to contain %v. %s", element, formatMessage(msgAndArgs...))
}

// AssertStringContains checks that a string contains a substring
func AssertStringContains(t *testing.T, str, substring string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(str, substring) {
		t.Errorf("Expected string to contain '%s', got '%s'. %s", substring, str, formatMessage(msgAndArgs...))
	}
}

// Helper functions

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		if msg, ok := msgAndArgs[0].(string); ok {
			return msg
		}
		return ""
	}
	if format, ok := msgAndArgs[0].(string); ok {
		return format // Simplified - not doing full Sprintf
	}
	return ""
}
