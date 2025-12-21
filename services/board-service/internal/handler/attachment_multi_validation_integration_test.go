package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"project-board-api/internal/client"
	"project-board-api/internal/converter"
	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/metrics"
	"project-board-api/internal/repository"
	"project-board-api/internal/service"
)

// TestIntegration_MultipleAttachmentsFlow tests creating a board with multiple attachments
// **Validates: Requirements 8.1, 8.2, 8.3**
func TestIntegrationMultipleAttachmentsFlow(t *testing.T) {
	db := setupFullFlowTestDB(t)

	// Use MockS3Client instead of real S3 client
	s3Client := client.NewMockS3Client()

	// Initialize repositories and services
	boardRepo := repository.NewBoardRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	fieldOptionRepo := repository.NewFieldOptionRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

	logger := zap.NewNop()
	// Create a new registry for each test to avoid duplicate metric registration
	registry := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(registry, logger)
	fieldOptionConverter := converter.NewFieldOptionConverter(fieldOptionRepo)
	boardService := service.NewBoardService(
		boardRepo,
		projectRepo,
		fieldOptionRepo,
		participantRepo,
		attachmentRepo,
		s3Client,
		fieldOptionConverter,
		m,
		logger,
	)

	router := setupFullFlowRouter(db, s3Client, boardService)

	workspaceID := uuid.New()
	userID := uuid.New()

	// Create test project
	project := &domain.Project{
		WorkspaceID: workspaceID,
		Name:        "Test Project",
		OwnerID:     userID,
	}
	errProject := projectRepo.Create(context.Background(), project)
	require.NoError(t, errProject)

	// Create multiple temporary attachments
	attachmentIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		now := time.Now()
		expiresAt := now.Add(1 * time.Hour)
		attachment := &domain.Attachment{
			EntityType:  domain.EntityTypeBoard,
			EntityID:    nil,
			Status:      domain.AttachmentStatusTemp,
			FileName:    "file-" + string(rune('A'+i)) + ".pdf",
			FileURL:     "https://test-bucket.s3.us-east-1.amazonaws.com/board/boards/" + workspaceID.String() + "/2024/01/file_" + string(rune('A'+i)) + ".pdf",
			FileSize:    int64((i + 1) * 100000),
			ContentType: "application/pdf",
			UploadedBy:  userID,
			ExpiresAt:   &expiresAt,
		}
		err := attachmentRepo.Create(context.Background(), attachment)
		require.NoError(t, err)
		attachmentIDs[i] = attachment.ID
	}

	// Create board with multiple attachments
	createBoardReq := dto.CreateBoardRequest{
		ProjectID:     project.ID,
		Title:         "Board with Multiple Attachments",
		Content:       "Testing multiple attachments",
		AttachmentIDs: attachmentIDs,
	}

	body, err := json.Marshal(createBoardReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Board creation should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok)

	idStr, _ := data["boardId"].(string)
	boardID, err := uuid.Parse(idStr)
	require.NoError(t, err)

	// Verify all attachments are confirmed
	for _, attachmentID := range attachmentIDs {
		attachment, err := attachmentRepo.FindByID(context.Background(), attachmentID)
		require.NoError(t, err)
		assert.Equal(t, domain.AttachmentStatusConfirmed, attachment.Status)
		assert.NotNil(t, attachment.EntityID)
		assert.Equal(t, boardID, *attachment.EntityID)
	}

	// Verify attachments can be retrieved
	attachments, err := attachmentRepo.FindByEntityID(context.Background(), domain.EntityTypeBoard, boardID)
	require.NoError(t, err)
	assert.Len(t, attachments, 3, "Should have 3 attachments")
}

// TestIntegration_AttachmentValidationErrors tests error scenarios
// **Validates: Requirements 8.2**
func TestIntegrationAttachmentValidationErrors(t *testing.T) {
	db := setupFullFlowTestDB(t)

	// Use MockS3Client instead of real S3 client
	s3Client := client.NewMockS3Client()

	// Initialize repositories and services
	boardRepo := repository.NewBoardRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	fieldOptionRepo := repository.NewFieldOptionRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

	logger := zap.NewNop()
	// Create a new registry for each test to avoid duplicate metric registration
	registry := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(registry, logger)
	fieldOptionConverter := converter.NewFieldOptionConverter(fieldOptionRepo)
	boardService := service.NewBoardService(
		boardRepo,
		projectRepo,
		fieldOptionRepo,
		participantRepo,
		attachmentRepo,
		s3Client,
		fieldOptionConverter,
		m,
		logger,
	)

	router := setupFullFlowRouter(db, s3Client, boardService)

	workspaceID := uuid.New()
	userID := uuid.New()

	// Create test project
	project := &domain.Project{
		WorkspaceID: workspaceID,
		Name:        "Test Project",
		OwnerID:     userID,
	}
	errProject := projectRepo.Create(context.Background(), project)
	require.NoError(t, errProject)

	t.Run("Reject non-existent attachment ID", func(t *testing.T) {
		nonExistentID := uuid.New()

		createBoardReq := dto.CreateBoardRequest{
			ProjectID:     project.ID,
			Title:         "Board with Invalid Attachment",
			Content:       "Testing invalid attachment",
			AttachmentIDs: []uuid.UUID{nonExistentID},
		}

		body, err := json.Marshal(createBoardReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject non-existent attachment")
	})

	t.Run("Reject already confirmed attachment", func(t *testing.T) {
		// Create a confirmed attachment
		boardID := uuid.New()
		confirmedAttachment := &domain.Attachment{
			EntityType:  domain.EntityTypeBoard,
			EntityID:    &boardID,
			Status:      domain.AttachmentStatusConfirmed,
			FileName:    "confirmed-file.pdf",
			FileURL:     "https://test-bucket.s3.us-east-1.amazonaws.com/file.pdf",
			FileSize:    100000,
			ContentType: "application/pdf",
			UploadedBy:  userID,
		}
		err := attachmentRepo.Create(context.Background(), confirmedAttachment)
		require.NoError(t, err)

		createBoardReq := dto.CreateBoardRequest{
			ProjectID:     project.ID,
			Title:         "Board with Confirmed Attachment",
			Content:       "Testing confirmed attachment reuse",
			AttachmentIDs: []uuid.UUID{confirmedAttachment.ID},
		}

		body, err := json.Marshal(createBoardReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/boards", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject already confirmed attachment")
	})
}
