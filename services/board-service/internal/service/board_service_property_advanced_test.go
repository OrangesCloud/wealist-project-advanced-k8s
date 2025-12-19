package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"project-board-api/internal/domain"
	"project-board-api/internal/dto"
)

// **Feature: board-creation-with-participants, Property 4: Backward compatibility**
// **Validates: Requirements 2.3**
// For any CreateBoardRequest with an empty or nil participants array,
// the Board Service should create the board successfully with an empty participantIds array in the response
func TestProperty_BackwardCompatibility(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Board creation succeeds with empty or nil participants", prop.ForAll(
		func(useNil bool) bool {
			projectID := uuid.New()
			userID := uuid.New()
			boardID := uuid.New()

			// Setup mocks
			mockProjectRepo := &MockProjectRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel: domain.BaseModel{ID: projectID},
					}, nil
				},
			}

			mockBoardRepo := &MockBoardRepository{
				CreateFunc: func(ctx context.Context, board *domain.Board) error {
					board.ID = boardID
					return nil
				},
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					// Return board without participants
					return &domain.Board{
						BaseModel:    domain.BaseModel{ID: boardID},
						ProjectID:    projectID,
						Title:        "Test Board",
						Participants: []domain.Participant{}, // Empty participants
					}, nil
				},
			}

			mockParticipantRepo := &MockParticipantRepository{}
			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			logger, _ := zap.NewDevelopment()

			service := NewBoardService(
				mockBoardRepo,
				mockProjectRepo,
				mockFieldOptionRepo,
				mockParticipantRepo,
				&MockAttachmentRepository{},
				nil, // s3Client
				mockConverter,
				nil,
				logger,
			)

			ctx := context.WithValue(context.Background(), "user_id", userID)

			// Create request with either nil or empty participants array
			var participants []uuid.UUID
			if !useNil {
				participants = []uuid.UUID{} // Empty array
			}
			// If useNil is true, participants remains nil

			req := &dto.CreateBoardRequest{
				ProjectID:    projectID,
				Title:        "Test Board",
				Content:      "Test Content",
				Participants: participants,
			}

			// Execute
			response, err := service.CreateBoard(ctx, req)

			// Verify: board creation should succeed
			if err != nil {
				t.Logf("Board creation failed: %v", err)
				return false
			}

			if response == nil {
				t.Log("Response is nil")
				return false
			}

			// Verify board was created
			if response.ID == uuid.Nil {
				t.Log("Board ID is nil")
				return false
			}

			// Verify participantIds is not nil (should be empty array, not null)
			if response.ParticipantIDs == nil {
				t.Log("ParticipantIDs is nil, should be empty array")
				return false
			}

			// Verify participantIds is empty
			if len(response.ParticipantIDs) != 0 {
				t.Logf("Expected 0 participants, got %d", len(response.ParticipantIDs))
				return false
			}

			return true
		},
		gen.Bool(), // Test with both nil and empty array
	))

	properties.TestingRun(t)
}

// **Feature: board-creation-with-participants, Property 5: Assignee as participant**
// **Validates: Requirements 1.5**
// For any CreateBoardRequest where the assigneeId is included in the participants array,
// the Board Service should ensure the assignee appears in the response participantIds
func TestProperty_AssigneeAsParticipant(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Assignee included in participants appears in response participantIds", prop.ForAll(
		func(additionalParticipantCount int, includeAssigneeInParticipants bool) bool {
			projectID := uuid.New()
			userID := uuid.New()
			boardID := uuid.New()
			assigneeID := uuid.New()

			// Generate additional participants (not including assignee)
			additionalParticipants := make([]uuid.UUID, additionalParticipantCount)
			for i := 0; i < additionalParticipantCount; i++ {
				additionalParticipants[i] = uuid.New()
			}

			// Build participants array
			var participants []uuid.UUID
			if includeAssigneeInParticipants {
				// Include assignee in participants
				participants = append(participants, assigneeID)
			}
			participants = append(participants, additionalParticipants...)

			// Track created participants
			createdParticipants := make(map[uuid.UUID]bool)

			// Setup mocks
			mockProjectRepo := &MockProjectRepository{
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
					return &domain.Project{
						BaseModel: domain.BaseModel{ID: projectID},
					}, nil
				},
			}

			mockBoardRepo := &MockBoardRepository{
				CreateFunc: func(ctx context.Context, board *domain.Board) error {
					board.ID = boardID
					return nil
				},
				FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
					// Return board with participants that were created
					participantList := make([]domain.Participant, 0)
					for userID := range createdParticipants {
						participantList = append(participantList, domain.Participant{
							BoardID: boardID,
							UserID:  userID,
						})
					}
					return &domain.Board{
						BaseModel:    domain.BaseModel{ID: boardID},
						ProjectID:    projectID,
						Title:        "Test Board",
						AssigneeID:   &assigneeID,
						Participants: participantList,
					}, nil
				},
			}

			mockParticipantRepo := &MockParticipantRepository{
				FindByBoardAndUserFunc: func(ctx context.Context, bID, uID uuid.UUID) (*domain.Participant, error) {
					if createdParticipants[uID] {
						return &domain.Participant{BoardID: bID, UserID: uID}, nil
					}
					return nil, gorm.ErrRecordNotFound
				},
				CreateFunc: func(ctx context.Context, participant *domain.Participant) error {
					createdParticipants[participant.UserID] = true
					return nil
				},
			}

			mockFieldOptionRepo := &MockFieldOptionRepository{}
			mockConverter := &MockFieldOptionConverter{}
			logger, _ := zap.NewDevelopment()

			service := NewBoardService(
				mockBoardRepo,
				mockProjectRepo,
				mockFieldOptionRepo,
				mockParticipantRepo,
				&MockAttachmentRepository{},
				nil, // s3Client
				mockConverter,
				nil,
				logger,
			)

			ctx := context.WithValue(context.Background(), "user_id", userID)

			req := &dto.CreateBoardRequest{
				ProjectID:    projectID,
				Title:        "Test Board",
				Content:      "Test Content",
				AssigneeID:   &assigneeID,
				Participants: participants,
			}

			// Execute
			response, err := service.CreateBoard(ctx, req)

			// Verify
			if err != nil {
				t.Logf("Unexpected error: %v", err)
				return false
			}

			if response == nil {
				t.Log("Response is nil")
				return false
			}

			// If assignee was included in participants, verify it appears in response
			if includeAssigneeInParticipants {
				assigneeFound := false
				for _, id := range response.ParticipantIDs {
					if id == assigneeID {
						assigneeFound = true
						break
					}
				}
				if !assigneeFound {
					t.Logf("Assignee %s was in participants but not found in response participantIds", assigneeID)
					return false
				}
			}

			// Verify all additional participants are in response
			responseMap := make(map[uuid.UUID]bool)
			for _, id := range response.ParticipantIDs {
				responseMap[id] = true
			}

			for _, expectedID := range additionalParticipants {
				if !responseMap[expectedID] {
					t.Logf("Expected participant %s not found in response", expectedID)
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 10), // Generate 0-10 additional participants
		gen.Bool(),          // Whether to include assignee in participants
	))

	properties.TestingRun(t)
}
