package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"project-board-api/internal/client"
	"project-board-api/internal/database"
	"project-board-api/internal/dto"
	"project-board-api/internal/response"
	"project-board-api/internal/service"
)

type BoardHandler struct {
	boardService service.BoardService
	notiClient   client.NotiClient
}

func NewBoardHandler(boardService service.BoardService, notiClient client.NotiClient) *BoardHandler {
	return &BoardHandler{
		boardService: boardService,
		notiClient:   notiClient,
	}
}

// CreateBoard godoc
// @Summary      Board 생성
// @Description  새로운 Board를 생성합니다
// @Description  customFields는 value 기반 인터페이스를 사용합니다 (UUID 아님)
// @Description  유효한 필드 타입: stage, role, importance
// @Description  예시 값: stage="in_progress", role="developer", importance="high"
// @Description  잘못된 field value 제공 시 400 에러 반환
// @Description  assigneeId가 제공되지 않으면 자동으로 authorId로 설정됩니다
// @Description  startDate와 dueDate는 선택 사항이며, startDate는 dueDate보다 이전이어야 합니다
// @Description  participants는 선택 사항이며, Board 생성 시 참여자를 함께 추가할 수 있습니다
// @Description  participants는 최대 50개의 UUID 배열이며, 중복된 ID는 자동으로 제거됩니다
// @Description  participants 추가가 실패해도 Board 생성은 성공하며, 성공한 참여자만 응답에 포함됩니다
// @Description  응답의 participantIds 필드에 생성된 참여자 ID 목록이 포함됩니다
// @Description  예시: {"participants": ["550e8400-e29b-41d4-a716-446655440001", "550e8400-e29b-41d4-a716-446655440002"]}
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBoardRequest true "Board 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.BoardResponse} "Board 생성 성공 (participantIds 포함)"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청 또는 유효하지 않은 field value"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards [post]
func (h *BoardHandler) CreateBoard(c *gin.Context) {
	log := getLogger(c)
	log.Debug("CreateBoard started")

	var req dto.CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("CreateBoard validation failed", zap.Error(err))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	ctx := c.Request.Context()
	if userID, exists := c.Get("user_id"); exists {
		ctx = context.WithValue(ctx, "user_id", userID)
	}

	log.Debug("CreateBoard calling service",
		zap.String("project.id", req.ProjectID.String()),
		zap.String("board.title", req.Title))

	board, err := h.boardService.CreateBoard(ctx, &req)
	if err != nil {
		log.Error("CreateBoard service error", zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Info("Board created",
		zap.String("board.id", board.ID.String()),
		zap.String("project.id", req.ProjectID.String()))

	// 💡 [수정] 응답을 먼저 보낸 후 브로드캐스트
	response.SendSuccess(c, http.StatusCreated, board)

	// 💡 [추가] 브로드캐스트
	event := WSEvent{
		Type:    "BOARD_CREATED",
		BoardID: board.ID.String(),
		Payload: board,
	}
	BroadcastEvent(req.ProjectID.String(), event)
}

// GetBoard godoc
// @Summary      Board 상세 조회
// @Description  Board ID로 상세 정보를 조회합니다 (참여자, 댓글, 첨부파일 포함)
// @Description  응답의 customFields는 value 기반 (UUID가 아닌 문자열 값)
// @Description  예시: {"importance": "high", "role": "developer", "stage": "in_progress"}
// @Description  participantIds는 보드에 참여하는 사용자 ID 배열입니다
// @Description  attachments는 보드에 첨부된 파일 메타데이터 배열입니다
// @Description  startDate와 dueDate는 설정된 경우에만 포함됩니다
// @Tags         boards
// @Produce      json
// @Param        boardId path string true "Board ID (UUID)"
// @Success      200 {object} response.SuccessResponse{data=dto.BoardDetailResponse} "Board 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Board ID"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards/{boardId} [get]
func (h *BoardHandler) GetBoard(c *gin.Context) {
	log := getLogger(c)

	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		log.Warn("GetBoard invalid board ID", zap.String("board.id", boardIDStr))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	log.Debug("GetBoard started", zap.String("board.id", boardID.String()))

	board, err := h.boardService.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		log.Error("GetBoard service error", zap.String("board.id", boardID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Debug("GetBoard completed", zap.String("board.id", boardID.String()))
	response.SendSuccess(c, http.StatusOK, board)
}

// GetBoardsByProject godoc
// @Summary      Project의 Board 목록 조회
// @Description  특정 Project에 속한 모든 Board를 조회합니다. customFields 파라미터로 필터링 가능 (JSON 형식)
// @Description  응답의 customFields는 value 기반 (UUID가 아닌 문자열 값)
// @Description  예시: {"importance": "high", "role": "developer", "stage": "in_progress"}
// @Description  각 보드는 participantIds (참여자 ID 배열)와 attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Description  startDate와 dueDate는 설정된 경우에만 포함됩니다
// @Tags         boards
// @Produce      json
// @Param        projectId    path      string  true   "Project ID (UUID)"
// @Param        customFields query     string  false  "Custom Fields 필터 JSON 객체. 예시: {\"importance\":\"high\",\"stage\":\"in_progress\"}"
// @Success      200 {object} response.SuccessResponse{data=[]dto.BoardResponse} "Board 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID 또는 필터 파라미터"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards/project/{projectId} [get]
func (h *BoardHandler) GetBoardsByProject(c *gin.Context) {
	log := getLogger(c)

	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		log.Warn("GetBoardsByProject invalid project ID", zap.String("project.id", projectIDStr))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	log.Debug("GetBoardsByProject started", zap.String("project.id", projectID.String()))

	filters := &dto.BoardFilters{}
	customFieldsStr := c.Query("customFields")

	if customFieldsStr != "" {
		var customFields map[string]interface{}
		if err := json.Unmarshal([]byte(customFieldsStr), &customFields); err != nil {
			log.Warn("GetBoardsByProject invalid customFields", zap.String("customFields", customFieldsStr))
			response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid customFields format: must be valid JSON")
			return
		}
		filters.CustomFields = customFields
	}

	boards, err := h.boardService.GetBoardsByProject(c.Request.Context(), projectID, filters)
	if err != nil {
		log.Error("GetBoardsByProject service error", zap.String("project.id", projectID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Debug("GetBoardsByProject completed",
		zap.String("project.id", projectID.String()),
		zap.Int("board.count", len(boards)))
	response.SendSuccess(c, http.StatusOK, boards)
}

// GetBoardsByProjectQuery godoc
// @Summary      Project의 Board 목록 조회 (쿼리 파라미터 방식)
// @Description  특정 Project에 속한 모든 Board를 조회합니다. 프론트엔드 호환용 엔드포인트
// @Description  응답의 customFields는 value 기반 (UUID가 아닌 문자열 값)
// @Description  예시: {"importance": "high", "role": "developer", "stage": "in_progress"}
// @Description  각 보드는 participantIds (참여자 ID 배열)와 attachments (첨부파일 메타데이터 배열)를 포함합니다
// @Description  startDate와 dueDate는 설정된 경우에만 포함됩니다
// @Tags         boards
// @Produce      json
// @Param        projectId    query     string  true   "Project ID (UUID)"
// @Param        customFields query     string  false  "Custom Fields 필터 JSON 객체. 예시: {\"importance\":\"high\",\"stage\":\"in_progress\"}"
// @Success      200 {object} response.SuccessResponse{data=[]dto.BoardResponse} "Board 목록 조회 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Project ID 또는 필터 파라미터"
// @Failure      404 {object} response.ErrorResponse "Project를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards [get]
func (h *BoardHandler) GetBoardsByProjectQuery(c *gin.Context) {
	log := getLogger(c)

	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		log.Warn("GetBoardsByProjectQuery missing project ID")
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Project ID is required")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		log.Warn("GetBoardsByProjectQuery invalid project ID", zap.String("project.id", projectIDStr))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	log.Debug("GetBoardsByProjectQuery started", zap.String("project.id", projectID.String()))

	filters := &dto.BoardFilters{}
	customFieldsStr := c.Query("customFields")

	if customFieldsStr != "" {
		var customFields map[string]interface{}
		if err := json.Unmarshal([]byte(customFieldsStr), &customFields); err != nil {
			log.Warn("GetBoardsByProjectQuery invalid customFields", zap.String("customFields", customFieldsStr))
			response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid customFields format: must be valid JSON")
			return
		}
		filters.CustomFields = customFields
	}

	boards, err := h.boardService.GetBoardsByProject(c.Request.Context(), projectID, filters)
	if err != nil {
		log.Error("GetBoardsByProjectQuery service error", zap.String("project.id", projectID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Debug("GetBoardsByProjectQuery completed",
		zap.String("project.id", projectID.String()),
		zap.Int("board.count", len(boards)))
	response.SendSuccess(c, http.StatusOK, boards)
}

// UpdateBoard godoc
// @Summary      Board 수정
// @Description  Board 정보를 수정합니다 (제목, 내용, 단계, 중요도, 역할, 담당자, 날짜)
// @Description  customFields는 value 기반 인터페이스를 사용합니다 (UUID 아님)
// @Description  유효한 필드 타입: stage, role, importance
// @Description  예시 값: stage="completed", role="designer", importance="medium"
// @Description  잘못된 field value 제공 시 400 에러 반환
// @Description  startDate와 dueDate를 수정할 수 있으며, startDate는 dueDate보다 이전이어야 합니다
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        boardId path string true "Board ID (UUID)"
// @Param        request body dto.UpdateBoardRequest true "Board 수정 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.BoardResponse} "Board 수정 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청 또는 유효하지 않은 field value"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards/{boardId} [put]
func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	log := getLogger(c)

	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		log.Warn("UpdateBoard invalid board ID", zap.String("board.id", boardIDStr))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	var req dto.UpdateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("UpdateBoard validation failed", zap.String("board.id", boardID.String()), zap.Error(err))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	// 🔥 알림 전송을 위해 기존 board 정보 가져오기
	oldBoard, _ := h.boardService.GetBoard(c.Request.Context(), boardID)
	var oldAssigneeID *uuid.UUID
	if oldBoard != nil && oldBoard.AssigneeID != nil {
		oldAssigneeID = oldBoard.AssigneeID
	}

	log.Debug("UpdateBoard started", zap.String("board.id", boardID.String()))

	board, err := h.boardService.UpdateBoard(c.Request.Context(), boardID, &req)
	if err != nil {
		log.Error("UpdateBoard service error", zap.String("board.id", boardID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Info("Board updated", zap.String("board.id", boardID.String()))

	// 💡 [수정] 응답을 먼저 보낸 후 브로드캐스트
	response.SendSuccess(c, http.StatusOK, board)

	// 💡 [추가] 브로드캐스트
	event := WSEvent{
		Type:    "BOARD_UPDATED",
		BoardID: boardID.String(),
		Payload: board,
	}
	BroadcastEvent(board.ProjectID.String(), event)

	// 🔥 알림 전송 (비동기 - 실패해도 응답에 영향 없음)
	go h.sendBoardNotifications(c.Request.Context(), log, oldBoard, board, oldAssigneeID, req.AssigneeID)
}

// sendBoardNotifications sends notifications for board updates
func (h *BoardHandler) sendBoardNotifications(ctx context.Context, log *zap.Logger, oldBoard, newBoard *dto.BoardResponse, oldAssigneeID, newAssigneeID *uuid.UUID) {
	if h.notiClient == nil {
		return
	}

	// Get actor ID from context (current user)
	actorIDValue, exists := ctx.Value("user_id").(uuid.UUID)
	if !exists {
		// Try string format
		if actorIDStr, ok := ctx.Value("user_id").(string); ok {
			actorIDValue, _ = uuid.Parse(actorIDStr)
		}
	}

	// 1. 작업자가 새로 할당된 경우 (TASK_ASSIGNED)
	if newAssigneeID != nil && *newAssigneeID != uuid.Nil {
		// 기존에 할당자가 없었거나, 다른 사람으로 변경된 경우
		if oldAssigneeID == nil || *oldAssigneeID != *newAssigneeID {
			// 자기 자신에게 할당한 경우는 알림 제외
			if actorIDValue != *newAssigneeID {
				notification := client.NewTaskAssignedNotification(
					actorIDValue,
					*newAssigneeID,
					newBoard.WorkspaceID,
					newBoard.ID,
					newBoard.Title,
				)
				if err := h.notiClient.SendNotification(ctx, notification); err != nil {
					log.Warn("Failed to send task assignment notification",
						zap.String("board.id", newBoard.ID.String()),
						zap.String("targetUserId", newAssigneeID.String()),
						zap.Error(err))
				} else {
					log.Info("Task assignment notification sent",
						zap.String("board.id", newBoard.ID.String()),
						zap.String("targetUserId", newAssigneeID.String()))
				}
			}
		}
	}

	// 2. 작업자가 있는 보드가 업데이트된 경우 (TASK_STATUS_CHANGED)
	// 단, 작업자 변경이 아닌 다른 변경사항이 있을 때만
	if oldBoard != nil && oldBoard.AssigneeID != nil && *oldBoard.AssigneeID != uuid.Nil {
		// 작업자 변경이 아닌 경우에만 알림
		if newAssigneeID == nil || (oldAssigneeID != nil && *oldAssigneeID == *newAssigneeID) {
			// 자기 자신의 보드인 경우는 알림 제외
			if actorIDValue != *oldBoard.AssigneeID {
				notification := client.NewTaskUpdatedNotification(
					actorIDValue,
					*oldBoard.AssigneeID,
					newBoard.WorkspaceID,
					newBoard.ID,
					newBoard.Title,
					"updated",
				)
				if err := h.notiClient.SendNotification(ctx, notification); err != nil {
					log.Warn("Failed to send task update notification",
						zap.String("board.id", newBoard.ID.String()),
						zap.Error(err))
				}
			}
		}
	}
}

// DeleteBoard godoc
// @Summary      Board 삭제
// @Description  Board를 소프트 삭제합니다
// @Tags         boards
// @Produce      json
// @Param        boardId path string true "Board ID (UUID)"
// @Success      200 {object} response.SuccessResponse "Board 삭제 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 Board ID"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards/{boardId} [delete]
func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	log := getLogger(c)

	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		log.Warn("DeleteBoard invalid board ID", zap.String("board.id", boardIDStr))
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	log.Debug("DeleteBoard started", zap.String("board.id", boardID.String()))

	// 💡 [수정] 삭제 전에 보드 정보 가져오기 (projectId 필요)
	board, err := h.boardService.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		log.Error("DeleteBoard get board error", zap.String("board.id", boardID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	err = h.boardService.DeleteBoard(c.Request.Context(), boardID)
	if err != nil {
		log.Error("DeleteBoard service error", zap.String("board.id", boardID.String()), zap.Error(err))
		handleServiceError(c, err)
		return
	}

	log.Info("Board deleted",
		zap.String("board.id", boardID.String()),
		zap.String("project.id", board.ProjectID.String()))

	// 💡 [수정] 응답을 먼저 보낸 후 브로드캐스트
	response.SendSuccess(c, http.StatusOK, nil)

	// 💡 [추가] 브로드캐스트
	event := WSEvent{
		Type:    "BOARD_DELETED",
		BoardID: boardID.String(),
		Payload: map[string]string{
			"boardId": boardID.String(),
		},
	}
	BroadcastEvent(board.ProjectID.String(), event)
}

// MoveBoard godoc
// @Summary      Board 이동 (실시간 동기화)
// @Description  Board를 다른 컬럼으로 이동합니다. WebSocket을 통해 실시간으로 다른 클라이언트에게 전파됩니다
// @Description  groupByFieldName에 해당하는 필드의 값을 newFieldValue로 변경합니다
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        boardId path string true "Board ID (UUID)"
// @Param        request body dto.MoveBoardRequest true "Board 이동 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.MoveBoardResponse} "Board 이동 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 요청"
// @Failure      404 {object} response.ErrorResponse "Board를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse "서버 에러"
// @Router       /boards/{boardId}/move [put]
func (h *BoardHandler) MoveBoard(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid board ID")
		return
	}

	var req dto.MoveBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid request body")
		return
	}

	ctx := c.Request.Context()

	// 🔥 [수정] req.ProjectID를 UUID로 파싱
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "Invalid project ID")
		return
	}

	// 🔥 [수정] newFieldValue가 nil이면 에러
	if req.NewFieldValue == nil {
		response.SendError(c, http.StatusBadRequest, response.ErrCodeValidation, "newFieldValue is required")
		return
	}

	newFieldValue := *req.NewFieldValue // 🔥 포인터 역참조

	// 1. 기존 보드 가져오기
	board, err := h.boardService.GetBoard(ctx, boardID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// 현재 그룹 값 추출
	oldGroupValue := ""
	if board.CustomFields != nil {
		if val, ok := board.CustomFields[req.GroupByFieldName]; ok {
			oldGroupValue = val.(string)
		}
	}

	// 🔥 [핵심 수정] 기존 customFields를 유지하면서 해당 필드만 업데이트
	existingCustomFields := make(map[string]interface{})
	if board.CustomFields != nil {
		// 기존 필드들을 모두 복사
		for k, v := range board.CustomFields {
			existingCustomFields[k] = v
		}
	}
	// 변경할 필드만 업데이트
	existingCustomFields[req.GroupByFieldName] = newFieldValue

	// 2. 필드 값 업데이트
	updateReq := &dto.UpdateBoardRequest{
		CustomFields: &existingCustomFields,
	}
	_, err = h.boardService.UpdateBoard(ctx, boardID, updateReq)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// 3. Redis 순서 업데이트
	redisClient := database.GetRedis()
	if redisClient != nil {
		oldKey := fmt.Sprintf("kanban:project:%s:group:%s", projectID.String(), oldGroupValue)
		newKey := fmt.Sprintf("kanban:project:%s:group:%s", projectID.String(), newFieldValue)

		redisClient.ZRem(context.Background(), oldKey, boardID.String())
		redisClient.ZAdd(context.Background(), newKey, redis.Z{
			Score:  float64(time.Now().UnixMilli()),
			Member: boardID.String(),
		})
	}

	// 4. 실시간 브로드캐스트
	event := WSEvent{
		Type:    "BOARD_MOVED",
		BoardID: boardID.String(),
		Payload: map[string]string{
			"from": oldGroupValue,
			"to":   newFieldValue,
		},
	}

	log := getLogger(c)
	log.Info("Broadcasting BOARD_MOVED event",
		zap.String("projectId", projectID.String()),
		zap.String("boardId", boardID.String()),
		zap.String("from", oldGroupValue),
		zap.String("to", newFieldValue))

	BroadcastEvent(projectID.String(), event)

	// 응답
	response.SendSuccess(c, http.StatusOK, dto.MoveBoardResponse{
		BoardID:       boardID.String(),
		NewFieldValue: newFieldValue,
		Message:       "Board moved successfully",
	})
}
