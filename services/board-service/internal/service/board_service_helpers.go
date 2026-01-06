package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/client"
	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

// BoardChange represents a single field change in a board update
type BoardChange struct {
	Field    string `json:"field"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// datesEqual compares two time pointers for equality
func datesEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

// formatDatePtr formats a time pointer to string
func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// formatUUIDPtr formats a UUID pointer to string
func formatUUIDPtr(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

// formatInterface formats an interface value to string
func formatInterface(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (s *boardServiceImpl) convertBoardCustomFieldsToValues(ctx context.Context, board *domain.Board) error {
	if board.CustomFields == nil || len(board.CustomFields) == 0 {
		return nil
	}

	var customFields map[string]interface{}
	if err := json.Unmarshal(board.CustomFields, &customFields); err != nil {
		return err
	}

	convertedFields, err := s.fieldOptionConverter.ConvertIDsToValues(ctx, customFields)
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(convertedFields)
	if err != nil {
		return err
	}

	board.CustomFields = jsonBytes
	return nil
}

// toBoardResponse converts domain.Board to dto.BoardResponse
func (s *boardServiceImpl) toBoardResponse(board *domain.Board) *dto.BoardResponse {
	return s.toBoardResponseWithWorkspace(context.Background(), board)
}

// toBoardResponseWithWorkspace converts domain.Board to dto.BoardResponse with context for workspace lookup
func (s *boardServiceImpl) toBoardResponseWithWorkspace(ctx context.Context, board *domain.Board) *dto.BoardResponse {
	// Convert datatypes.JSON to map[string]interface{}
	var customFields map[string]interface{}
	if len(board.CustomFields) > 0 {
		_ = json.Unmarshal(board.CustomFields, &customFields)
	}

	// Extract participant IDs from board participants
	participantIDs := make([]uuid.UUID, 0, len(board.Participants))
	for _, p := range board.Participants {
		participantIDs = append(participantIDs, p.UserID)
	}

	// Convert attachments to response DTOs with s3Client.GetFileURL
	attachments := make([]dto.AttachmentResponse, 0, len(board.Attachments))
	for _, a := range board.Attachments {
		// s3Client.GetFileURL을 사용하여 FileURL 필드 채우기 (DB의 FileURL은 S3 Key)
		fileURL := s.s3Client.GetFileURL(a.FileURL)

		attachments = append(attachments, dto.AttachmentResponse{
			ID:          a.ID,
			FileName:    a.FileName,
			FileURL:     fileURL, // full URL 반환
			FileSize:    a.FileSize,
			ContentType: a.ContentType,
			UploadedBy:  a.UploadedBy,
			UploadedAt:  a.CreatedAt,
		})
	}

	// Get WorkspaceID from project
	var workspaceID uuid.UUID
	if board.Project.ID != uuid.Nil {
		// Project was preloaded
		workspaceID = board.Project.WorkspaceID
	} else {
		// Fetch project to get WorkspaceID
		project, err := s.projectRepo.FindByID(ctx, board.ProjectID)
		if err == nil && project != nil {
			workspaceID = project.WorkspaceID
		}
	}

	return &dto.BoardResponse{
		ID:             board.ID,
		ProjectID:      board.ProjectID,
		WorkspaceID:    workspaceID,
		AuthorID:       board.AuthorID,
		AssigneeID:     board.AssigneeID,
		Title:          board.Title,
		Content:        board.Content,
		CustomFields:   customFields,
		StartDate:      board.StartDate,
		DueDate:        board.DueDate,
		ParticipantIDs: participantIDs,
		Attachments:    attachments,
		CreatedAt:      board.CreatedAt,
		UpdatedAt:      board.UpdatedAt,
	}
}

// toBoardDetailResponse converts domain.Board to dto.BoardDetailResponse
func (s *boardServiceImpl) toBoardDetailResponse(ctx context.Context, board *domain.Board) *dto.BoardDetailResponse {
	// Convert participants
	participants := make([]dto.ParticipantResponse, len(board.Participants))
	for i, p := range board.Participants {
		participants[i] = dto.ParticipantResponse{
			ID:        p.ID,
			BoardID:   p.BoardID,
			UserID:    p.UserID,
			CreatedAt: p.CreatedAt,
		}
	}

	// Convert comments
	comments := make([]dto.CommentResponse, len(board.Comments))
	for i, c := range board.Comments {
		comments[i] = dto.CommentResponse{
			CommentID: c.ID,
			BoardID:   c.BoardID,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}

	return &dto.BoardDetailResponse{
		BoardResponse: *s.toBoardResponseWithWorkspace(ctx, board),
		Participants:  participants,
		Comments:      comments,
	}
}

// addParticipantsInternal is an internal helper to add participants during board creation
// It does not verify board existence (assumes board was just created)
// Returns the number of successfully added participants and any errors
func (s *boardServiceImpl) addParticipantsInternal(ctx context.Context, boardID uuid.UUID, userIDs []uuid.UUID) (int, error) {
	// Remove duplicates from the user IDs
	uniqueUserIDs := removeDuplicateUUIDs(userIDs)

	successCount := 0
	var failedUserIDs []uuid.UUID

	// Process each participant individually
	for _, userID := range uniqueUserIDs {
		// Check if participant already exists
		existing, err := s.participantRepo.FindByBoardAndUser(ctx, boardID, userID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("Failed to check existing participant",
				zap.String("board_id", boardID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err))
			failedUserIDs = append(failedUserIDs, userID)
			continue
		}
		if existing != nil {
			// Participant already exists, skip
			continue
		}

		// Create domain model
		participant := &domain.Participant{
			BoardID: boardID,
			UserID:  userID,
		}

		// Save to repository
		if err := s.participantRepo.Create(ctx, participant); err != nil {
			// Check for unique constraint violation
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				// Participant already exists, skip
				continue
			}
			s.logger.Warn("Failed to add participant",
				zap.String("board_id", boardID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err))
			failedUserIDs = append(failedUserIDs, userID)
			continue
		}

		successCount++
	}

	// Log summary if there were failures
	if len(failedUserIDs) > 0 {
		s.logger.Warn("Some participants failed to be added during board creation",
			zap.String("board_id", boardID.String()),
			zap.Int("success_count", successCount),
			zap.Int("failed_count", len(failedUserIDs)),
			zap.Any("failed_user_ids", failedUserIDs))
	}

	return successCount, nil
}

// validateAndConfirmAttachments validates that attachments exist and are in TEMP status
func (s *boardServiceImpl) validateAndConfirmAttachments(ctx context.Context, attachmentIDs []uuid.UUID, entityType domain.EntityType, entityID uuid.UUID) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	// Fetch attachments by IDs
	attachments, err := s.attachmentRepo.FindByIDs(ctx, attachmentIDs)
	if err != nil {
		return response.NewAppError(response.ErrCodeInternal, "Failed to fetch attachments", err.Error())
	}

	// Check if all attachments exist
	if len(attachments) != len(attachmentIDs) {
		return response.NewAppError(response.ErrCodeValidation, "One or more attachments not found", "")
	}

	// Validate each attachment
	for _, attachment := range attachments {
		// Check if attachment is in TEMP status
		if attachment.Status != domain.AttachmentStatusTemp {
			return response.NewAppError(response.ErrCodeValidation, "Attachment is not in temporary status and cannot be reused", "")
		}

		// Check if attachment entity type matches
		if attachment.EntityType != entityType {
			return response.NewAppError(response.ErrCodeValidation, "Attachment entity type does not match", "")
		}
	}

	return nil
}

// deleteAttachmentsWithS3 deletes attachments from both S3 and database
func (s *boardServiceImpl) deleteAttachmentsWithS3(ctx context.Context, attachments []*domain.Attachment) {
	attachmentIDs := make([]uuid.UUID, 0, len(attachments))

	// Delete files from S3
	for _, attachment := range attachments {
		// Extract S3 key from FileURL
		fileKey := extractS3KeyFromURL(attachment.FileURL)
		if fileKey == "" {
			s.logger.Warn("Failed to extract S3 key from URL",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_url", attachment.FileURL))
			continue
		}

		// Delete from S3
		if err := s.s3Client.DeleteFile(ctx, fileKey); err != nil {
			s.logger.Warn("Failed to delete file from S3",
				zap.String("attachment_id", attachment.ID.String()),
				zap.String("file_key", fileKey),
				zap.Error(err))
			// Continue even if S3 deletion fails
		}

		attachmentIDs = append(attachmentIDs, attachment.ID)
	}

	// Delete from database
	if len(attachmentIDs) > 0 {
		if err := s.attachmentRepo.DeleteBatch(ctx, attachmentIDs); err != nil {
			s.logger.Warn("Failed to delete attachments from database",
				zap.Int("count", len(attachmentIDs)),
				zap.Error(err))
		}
	}
}

// isAssigneeChanged checks if the assignee was changed
func (s *boardServiceImpl) isAssigneeChanged(original, current *uuid.UUID) bool {
	// Both nil - no change
	if original == nil && current == nil {
		return false
	}
	// One nil, one not - changed
	if original == nil || current == nil {
		return true
	}
	// Both non-nil - compare values
	return *original != *current
}

// sendAssigneeNotification sends a BOARD_ASSIGNED notification to the assignee
// This is called asynchronously (in a goroutine) so notification failures don't affect the main business logic
func (s *boardServiceImpl) sendAssigneeNotification(ctx context.Context, board *domain.Board, actorID uuid.UUID) {
	if s.notiClient == nil || board.AssigneeID == nil {
		return
	}

	// Get project info for workspace ID
	project, err := s.projectRepo.FindByID(ctx, board.ProjectID)
	if err != nil {
		s.logger.Warn("Failed to get project for notification",
			zap.String("board.id", board.ID.String()),
			zap.Error(err))
		return
	}

	event := &client.NotificationEvent{
		Type:         client.NotificationTypeBoardAssigned,
		ActorID:      actorID,
		TargetUserID: *board.AssigneeID,
		WorkspaceID:  project.WorkspaceID,
		ResourceType: client.ResourceTypeBoard,
		ResourceID:   board.ID,
		ResourceName: &board.Title,
		Metadata: map[string]interface{}{
			"projectId":   board.ProjectID.String(),
			"projectName": project.Name,
		},
	}

	// Send notification asynchronously
	go func() {
		// Use background context to avoid cancellation when request completes
		if err := s.notiClient.SendNotification(context.Background(), event); err != nil {
			s.logger.Warn("Failed to send board assigned notification",
				zap.String("board.id", board.ID.String()),
				zap.String("assignee.id", board.AssigneeID.String()),
				zap.Error(err))
		}
	}()
}

// sendParticipantAddedNotifications sends BOARD_PARTICIPANT_ADDED notifications to new participants
func (s *boardServiceImpl) sendParticipantAddedNotifications(ctx context.Context, board *domain.Board, participantIDs []uuid.UUID, actorID uuid.UUID) {
	if s.notiClient == nil || len(participantIDs) == 0 {
		return
	}

	// Get project info for workspace ID
	project, err := s.projectRepo.FindByID(ctx, board.ProjectID)
	if err != nil {
		s.logger.Warn("Failed to get project for participant notification",
			zap.String("board.id", board.ID.String()),
			zap.Error(err))
		return
	}

	// Send notification to each participant asynchronously
	go func() {
		for _, participantID := range participantIDs {
			event := &client.NotificationEvent{
				Type:         client.NotificationTypeBoardParticipantAdded,
				ActorID:      actorID,
				TargetUserID: participantID,
				WorkspaceID:  project.WorkspaceID,
				ResourceType: client.ResourceTypeBoard,
				ResourceID:   board.ID,
				ResourceName: &board.Title,
				Metadata: map[string]interface{}{
					"projectId":   board.ProjectID.String(),
					"projectName": project.Name,
				},
			}

			if err := s.notiClient.SendNotification(context.Background(), event); err != nil {
				s.logger.Warn("Failed to send participant added notification",
					zap.String("board.id", board.ID.String()),
					zap.String("participant.id", participantID.String()),
					zap.Error(err))
			}
		}
	}()
}

// sendBoardUpdateNotifications sends BOARD_UPDATED notifications to assignee and all participants
// Excludes the actor (the person who made the update)
// Includes the list of changes made to the board
func (s *boardServiceImpl) sendBoardUpdateNotifications(ctx context.Context, board *domain.Board, actorID uuid.UUID, changes []BoardChange) {
	if s.notiClient == nil {
		return
	}

	// Get project info for workspace ID
	project, err := s.projectRepo.FindByID(ctx, board.ProjectID)
	if err != nil {
		s.logger.Warn("Failed to get project for update notification",
			zap.String("board.id", board.ID.String()),
			zap.Error(err))
		return
	}

	// Collect all users to notify (assignee + participants), excluding actor
	notifyUserIDs := make(map[uuid.UUID]bool)

	// Add assignee if exists and not the actor
	if board.AssigneeID != nil && *board.AssigneeID != actorID {
		notifyUserIDs[*board.AssigneeID] = true
	}

	// Add participants if not the actor
	for _, p := range board.Participants {
		if p.UserID != actorID {
			notifyUserIDs[p.UserID] = true
		}
	}

	if len(notifyUserIDs) == 0 {
		return
	}

	// Convert changes to []interface{} for JSON serialization
	changesData := make([]interface{}, len(changes))
	for i, c := range changes {
		changesData[i] = map[string]interface{}{
			"field":    c.Field,
			"oldValue": c.OldValue,
			"newValue": c.NewValue,
		}
	}

	// Send notifications asynchronously
	go func() {
		for userID := range notifyUserIDs {
			event := &client.NotificationEvent{
				Type:         client.NotificationTypeBoardUpdated,
				ActorID:      actorID,
				TargetUserID: userID,
				WorkspaceID:  project.WorkspaceID,
				ResourceType: client.ResourceTypeBoard,
				ResourceID:   board.ID,
				ResourceName: &board.Title,
				Metadata: map[string]interface{}{
					"projectId":   board.ProjectID.String(),
					"projectName": project.Name,
					"changes":     changesData,
				},
			}

			if err := s.notiClient.SendNotification(context.Background(), event); err != nil {
				s.logger.Warn("Failed to send board update notification",
					zap.String("board.id", board.ID.String()),
					zap.String("target.user.id", userID.String()),
					zap.Error(err))
			}
		}
	}()
}

// sendCommentAddedNotifications sends BOARD_COMMENT_ADDED notifications to assignee and all participants
// Excludes the actor (the person who added the comment)
func (s *boardServiceImpl) sendCommentAddedNotifications(ctx context.Context, board *domain.Board, commentID uuid.UUID, commentPreview string, actorID uuid.UUID) {
	if s.notiClient == nil {
		return
	}

	// Get project info for workspace ID
	project, err := s.projectRepo.FindByID(ctx, board.ProjectID)
	if err != nil {
		s.logger.Warn("Failed to get project for comment notification",
			zap.String("board.id", board.ID.String()),
			zap.Error(err))
		return
	}

	// Collect all users to notify (assignee + participants), excluding actor
	notifyUserIDs := make(map[uuid.UUID]bool)

	// Add assignee if exists and not the actor
	if board.AssigneeID != nil && *board.AssigneeID != actorID {
		notifyUserIDs[*board.AssigneeID] = true
	}

	// Add participants if not the actor
	for _, p := range board.Participants {
		if p.UserID != actorID {
			notifyUserIDs[p.UserID] = true
		}
	}

	if len(notifyUserIDs) == 0 {
		return
	}

	// Truncate comment content for notification preview (max 100 chars)
	if len(commentPreview) > 100 {
		commentPreview = commentPreview[:100] + "..."
	}

	// Send notifications asynchronously
	go func() {
		for userID := range notifyUserIDs {
			event := &client.NotificationEvent{
				Type:         client.NotificationTypeBoardCommentAdded,
				ActorID:      actorID,
				TargetUserID: userID,
				WorkspaceID:  project.WorkspaceID,
				ResourceType: client.ResourceTypeBoard,
				ResourceID:   board.ID,
				ResourceName: &board.Title,
				Metadata: map[string]interface{}{
					"projectId":      board.ProjectID.String(),
					"projectName":    project.Name,
					"commentId":      commentID.String(),
					"commentPreview": commentPreview,
				},
			}

			if err := s.notiClient.SendNotification(context.Background(), event); err != nil {
				s.logger.Warn("Failed to send comment notification",
					zap.String("board.id", board.ID.String()),
					zap.String("target.user.id", userID.String()),
					zap.Error(err))
			}
		}
	}()
}
