package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
)

func (s *boardServiceImpl) UpdateBoard(ctx context.Context, boardID uuid.UUID, req *dto.UpdateBoardRequest) (*dto.BoardResponse, error) {
	// Extract user_id from context for notification actor
	actorID, _ := ctx.Value("user_id").(uuid.UUID)

	// Fetch existing board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewAppError(response.ErrCodeNotFound, "Board not found", "")
		}
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to fetch board", err.Error())
	}

	// Store original values for change detection
	var originalAssigneeID *uuid.UUID
	if board.AssigneeID != nil {
		id := *board.AssigneeID
		originalAssigneeID = &id
	}

	// Store original participant IDs for change detection
	originalParticipantIDs := make(map[uuid.UUID]bool)
	for _, p := range board.Participants {
		originalParticipantIDs[p.UserID] = true
	}

	// 🔔 Store original values for change tracking
	originalTitle := board.Title
	originalContent := board.Content
	var originalCustomFields map[string]interface{}
	if len(board.CustomFields) > 0 {
		_ = json.Unmarshal(board.CustomFields, &originalCustomFields)
	}
	originalStartDate := board.StartDate
	originalDueDate := board.DueDate

	// Determine the effective start and due dates for validation
	effectiveStartDate := board.StartDate
	effectiveDueDate := board.DueDate

	if req.StartDate != nil {
		effectiveStartDate = req.StartDate
	}
	if req.DueDate != nil {
		effectiveDueDate = req.DueDate
	}

	// Validate date range with effective dates
	if err := validateDateRange(effectiveStartDate, effectiveDueDate); err != nil {
		return nil, err
	}

	// Validate and confirm attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if err := s.validateAndConfirmAttachments(ctx, req.AttachmentIDs, domain.EntityTypeBoard, uuid.Nil); err != nil {
			return nil, err
		}
	}

	// Update fields if provided
	if req.Title != nil {
		board.Title = *req.Title
	}
	if req.Content != nil {
		board.Content = *req.Content
	}
	if req.CustomFields != nil {
		// Convert values to IDs
		convertedFields, err := s.fieldOptionConverter.ConvertValuesToIDs(ctx, board.ProjectID, *req.CustomFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeValidation, "Invalid custom field values", err.Error())
		}

		// Convert CustomFields to datatypes.JSON
		jsonBytes, err := json.Marshal(convertedFields)
		if err != nil {
			return nil, response.NewAppError(response.ErrCodeInternal, "Failed to marshal custom fields", err.Error())
		}
		board.CustomFields = jsonBytes
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == uuid.Nil {
			board.AssigneeID = nil
		} else {
			board.AssigneeID = req.AssigneeID
		}
	}
	if req.StartDate != nil {
		board.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		board.DueDate = req.DueDate
	}

	// Update board first
	if err := s.boardRepo.Update(ctx, board); err != nil {
		return nil, response.NewAppError(response.ErrCodeInternal, "Failed to update board", err.Error())
	}

	// Attachments 처리 로직 개선 및 Confirm
	if len(req.AttachmentIDs) > 0 {
		// 에러 발생 시 업데이트 실패 처리
		if err := s.attachmentRepo.ConfirmAttachments(ctx, req.AttachmentIDs, board.ID); err != nil {
			s.logger.Error("Failed to confirm attachments during board update",
				zap.String("board_id", board.ID.String()),
				zap.Strings("attachment_ids", func() []string {
					ids := make([]string, len(req.AttachmentIDs))
					for i, id := range req.AttachmentIDs {
						ids[i] = id.String()
					}
					return ids
				}()),
				zap.Error(err))

			return nil, response.NewAppError(response.ErrCodeInternal,
				"Failed to confirm attachments: "+err.Error(),
				"Please ensure all attachment IDs are valid and not already used")
		}
	}

	// ✅ [수정] Participants 업데이트 로직 - board 업데이트 후 처리
	if req.Participants != nil {
		// 1. 기존 참여자 모두 조회
		existingParticipants, err := s.participantRepo.FindByBoardID(ctx, boardID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("Failed to fetch existing participants for update",
				zap.String("board_id", boardID.String()),
				zap.Error(err))
		}

		// 2. 기존 참여자 모두 삭제
		if len(existingParticipants) > 0 {
			s.logger.Info("Deleting existing participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(existingParticipants)))

			for _, p := range existingParticipants {
				if err := s.participantRepo.Delete(ctx, boardID, p.UserID); err != nil {
					s.logger.Warn("Failed to delete existing participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", p.UserID.String()),
						zap.Error(err))
				}
			}
		}

		// 3. 새로운 참여자 추가
		if len(req.Participants) > 0 {
			s.logger.Info("Adding new participants",
				zap.String("board_id", boardID.String()),
				zap.Int("count", len(req.Participants)))

			uniqueUserIDs := removeDuplicateUUIDs(req.Participants)
			for _, userID := range uniqueUserIDs {
				participant := &domain.Participant{
					BoardID: boardID,
					UserID:  userID,
				}
				if err := s.participantRepo.Create(ctx, participant); err != nil {
					s.logger.Warn("Failed to add new participant",
						zap.String("board_id", boardID.String()),
						zap.String("user_id", userID.String()),
						zap.Error(err))
				}
			}
		}
	}

	// board와 연결된 모든 Attachments를 다시 조회합니다. (타입 변환 적용)
	allAttachments, err := s.attachmentRepo.FindByEntityID(ctx, domain.EntityTypeBoard, board.ID)
	if err != nil {
		s.logger.Warn("Failed to fetch all confirmed attachments after update", zap.Error(err))
	} else {
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// ✅ [수정] 업데이트된 participants를 다시 로드
	reloadedBoard, err := s.boardRepo.FindByID(ctx, board.ID)
	if err != nil {
		s.logger.Warn("Failed to reload board with participants after update",
			zap.String("board_id", board.ID.String()),
			zap.Error(err))
	} else {
		board = reloadedBoard
		// Attachments는 위에서 이미 로드했으므로 다시 할당
		board.Attachments = toDomainAttachments(allAttachments)
	}

	// 🔔 Build list of changes for notification
	changes := make([]BoardChange, 0)

	if req.Title != nil && originalTitle != board.Title {
		changes = append(changes, BoardChange{Field: "title", OldValue: originalTitle, NewValue: board.Title})
	}
	if req.Content != nil && originalContent != board.Content {
		changes = append(changes, BoardChange{Field: "content", OldValue: "(내용 변경)", NewValue: "(내용 변경)"})
	}
	if req.StartDate != nil && !datesEqual(originalStartDate, board.StartDate) {
		changes = append(changes, BoardChange{Field: "startDate", OldValue: formatDatePtr(originalStartDate), NewValue: formatDatePtr(board.StartDate)})
	}
	if req.DueDate != nil && !datesEqual(originalDueDate, board.DueDate) {
		changes = append(changes, BoardChange{Field: "dueDate", OldValue: formatDatePtr(originalDueDate), NewValue: formatDatePtr(board.DueDate)})
	}
	if s.isAssigneeChanged(originalAssigneeID, board.AssigneeID) {
		changes = append(changes, BoardChange{Field: "assignee", OldValue: formatUUIDPtr(originalAssigneeID), NewValue: formatUUIDPtr(board.AssigneeID)})
	}

	// Check customFields changes (stage, role, importance, etc.)
	if req.CustomFields != nil {
		var newCustomFields map[string]interface{}
		if len(board.CustomFields) > 0 {
			_ = json.Unmarshal(board.CustomFields, &newCustomFields)
		}
		for key, newVal := range newCustomFields {
			oldVal, existed := originalCustomFields[key]
			if !existed || oldVal != newVal {
				changes = append(changes, BoardChange{
					Field:    key,
					OldValue: formatInterface(oldVal),
					NewValue: formatInterface(newVal),
				})
			}
		}
	}

	// Send notifications for board update

	// 1. Notify new assignee if assignee changed
	if s.isAssigneeChanged(originalAssigneeID, board.AssigneeID) && board.AssigneeID != nil {
		s.sendAssigneeNotification(ctx, board, actorID)
	}

	// 2. Notify new participants (those who weren't participants before)
	if req.Participants != nil && len(req.Participants) > 0 {
		var newParticipantIDs []uuid.UUID
		for _, pid := range req.Participants {
			if !originalParticipantIDs[pid] {
				newParticipantIDs = append(newParticipantIDs, pid)
			}
		}
		if len(newParticipantIDs) > 0 {
			s.sendParticipantAddedNotifications(ctx, board, newParticipantIDs, actorID)
		}
	}

	// 3. Notify all assignee + participants about the update (excluding actor) with changes
	if len(changes) > 0 {
		s.sendBoardUpdateNotifications(ctx, board, actorID, changes)
	}

	// Convert to response DTO
	return s.toBoardResponseWithWorkspace(ctx, board), nil
}

// DeleteBoard soft deletes a board and its associated attachments
