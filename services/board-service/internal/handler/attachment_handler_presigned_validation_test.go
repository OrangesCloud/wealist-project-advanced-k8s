package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratePresignedURL_FileSizeExceeded tests file size validation
func TestGeneratePresignedURL_FileSizeExceeded(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "large-file.jpg",
		FileSize:    51 * 1024 * 1024, // 51MB (exceeds 50MB limit)
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "FILE_TOO_LARGE", code)
}

// TestGeneratePresignedURL_InvalidFileType tests file type validation
func TestGeneratePresignedURL_InvalidFileType(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	tests := []struct {
		name        string
		fileName    string
		contentType string
	}{
		{
			name:        "Audio file (mp3)",
			fileName:    "audio.mp3",
			contentType: "audio/mpeg",
		},
		{
			name:        "Video file (mp4)",
			fileName:    "video.mp4",
			contentType: "video/mp4",
		},
		{
			name:        "Unsupported document type",
			fileName:    "diagram.exe",
			contentType: "application/x-msdownload",
		},
		{
			name:        "Mismatched extension and content type",
			fileName:    "image.jpg",
			contentType: "application/pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  "BOARD",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    tt.fileName,
				FileSize:    1024000,
				ContentType: tt.contentType,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorData, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "Response should contain error field")

			code, ok := errorData["code"].(string)
			assert.True(t, ok)
			assert.Equal(t, "INVALID_FILE_TYPE", code)
		})
	}
}

// TestGeneratePresignedURL_InvalidEntityType tests entity type validation
func TestGeneratePresignedURL_InvalidEntityType(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	tests := []struct {
		name       string
		entityType string
	}{
		{
			name:       "Invalid entity type - USER",
			entityType: "USER",
		},
		{
			name:       "Invalid entity type - WORKSPACE",
			entityType: "WORKSPACE",
		},
		{
			name:       "Empty entity type",
			entityType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := PresignedURLRequest{
				EntityType:  tt.entityType,
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
				FileName:    "test.jpg",
				FileSize:    1024000,
				ContentType: "image/jpeg",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorData, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "Response should contain error field")

			code, ok := errorData["code"].(string)
			assert.True(t, ok)
			assert.Equal(t, "VALIDATION_ERROR", code)
		})
	}
}

// TestGeneratePresignedURL_InvalidWorkspaceID tests workspace ID validation
func TestGeneratePresignedURL_InvalidWorkspaceID(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "invalid-uuid",
		FileName:    "test.jpg",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// TestGeneratePresignedURL_ZeroFileSize tests zero file size validation
func TestGeneratePresignedURL_ZeroFileSize(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "test.jpg",
		FileSize:    0,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// TestGeneratePresignedURL_MissingExtension tests file without extension
func TestGeneratePresignedURL_MissingExtension(t *testing.T) {
	_, router := setupAttachmentHandler(t)

	reqBody := PresignedURLRequest{
		EntityType:  "BOARD",
		WorkspaceID: "550e8400-e29b-41d4-a716-446655440000",
		FileName:    "testfile",
		FileSize:    1024000,
		ContentType: "image/jpeg",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/attachments/presigned-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData, ok := response["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain error field")

	code, ok := errorData["code"].(string)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_FILE_TYPE", code)
}

// TestValidateEntityType tests the validateEntityType function
func TestValidateEntityType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"Valid BOARD", "BOARD", false},
		{"Valid board (lowercase)", "board", false},
		{"Valid COMMENT", "COMMENT", false},
		{"Valid PROJECT", "PROJECT", false},
		{"Invalid USER", "USER", true},
		{"Invalid empty", "", true},
		{"Invalid random", "RANDOM", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateEntityType(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFileType tests the validateFileType function
func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		contentType string
		expectError bool
	}{
		{"Valid JPEG", "test.jpg", "image/jpeg", false},
		{"Valid PNG", "test.png", "image/png", false},
		{"Valid PDF", "doc.pdf", "application/pdf", false},
		{"Valid DOCX", "report.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", false},
		{"Invalid MP3", "audio.mp3", "audio/mpeg", true},
		{"Invalid MP4", "video.mp4", "video/mp4", true},
		{"Mismatched type", "image.jpg", "application/pdf", true},
		{"No extension", "file", "image/jpeg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileType(tt.fileName, tt.contentType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
