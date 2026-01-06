package handler

import (
	"net/http"

	"chat-service/internal/client"
	"chat-service/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commnotel "github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
)

// 최대 파일 크기: 50MB
const MaxFileSize = 50 * 1024 * 1024

// 허용된 이미지 Content-Type
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// PresignedURLRequest는 presigned URL 요청 구조체입니다.
type PresignedURLRequest struct {
	WorkspaceID string `json:"workspaceId" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required,min=1"`
}

// PresignedURLResponse는 presigned URL 응답 구조체입니다.
type PresignedURLResponse struct {
	UploadURL   string `json:"uploadUrl"`
	DownloadURL string `json:"downloadUrl"` // 업로드 후 파일 접근 URL
	FileKey     string `json:"fileKey"`
	ExpiresIn   int    `json:"expiresIn"` // 초 단위
}

// FileHandler는 파일 업로드 관련 핸들러입니다.
type FileHandler struct {
	s3Client client.S3ClientInterface
	logger   *zap.Logger
}

// NewFileHandler는 새로운 FileHandler를 생성합니다.
func NewFileHandler(s3Client client.S3ClientInterface, logger *zap.Logger) *FileHandler {
	return &FileHandler{
		s3Client: s3Client,
		logger:   logger,
	}
}

// log returns a trace-context aware logger
func (h *FileHandler) log(c *gin.Context) *zap.Logger {
	return commnotel.WithTraceContext(c.Request.Context(), h.logger)
}

// GeneratePresignedURL은 S3 presigned URL을 생성합니다.
// @Summary 채팅 파일 업로드용 Presigned URL 생성
// @Description 채팅 이미지 업로드를 위한 S3 presigned URL을 생성합니다.
// @Tags files
// @Accept json
// @Produce json
// @Param request body PresignedURLRequest true "Presigned URL 요청"
// @Success 200 {object} PresignedURLResponse
// @Failure 400 {object} response.ErrorResponse "잘못된 요청"
// @Failure 401 {object} response.ErrorResponse "인증 필요"
// @Failure 500 {object} response.ErrorResponse "내부 서버 오류"
// @Router /api/chats/files/presigned-url [post]
func (h *FileHandler) GeneratePresignedURL(c *gin.Context) {
	log := h.log(c)
	log.Debug("GeneratePresignedURL started")

	// 사용자 인증 확인
	userIDValue, exists := c.Get("user_id")
	if !exists {
		log.Warn("GeneratePresignedURL user not authenticated")
		response.SendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		// 문자열인 경우 파싱 시도
		userIDStr, ok := userIDValue.(string)
		if !ok {
			response.SendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user ID format")
			return
		}
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			response.SendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user ID format")
			return
		}
	}

	// 요청 바인딩
	var req PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("GeneratePresignedURL validation failed", zap.Error(err))
		response.SendError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	// 파일 크기 검증
	if req.FileSize > MaxFileSize {
		log.Warn("GeneratePresignedURL file too large",
			zap.Int64("fileSize", req.FileSize),
			zap.Int64("maxSize", MaxFileSize))
		response.SendError(c, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 50MB limit")
		return
	}

	// 이미지 타입 검증
	if !allowedImageTypes[req.ContentType] {
		log.Warn("GeneratePresignedURL invalid content type",
			zap.String("contentType", req.ContentType))
		response.SendError(c, http.StatusBadRequest, "INVALID_FILE_TYPE", "Only image files are allowed (jpeg, png, gif, webp)")
		return
	}

	// 워크스페이스 ID 검증
	if _, err := uuid.Parse(req.WorkspaceID); err != nil {
		log.Warn("GeneratePresignedURL invalid workspace ID")
		response.SendError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace ID")
		return
	}

	log.Debug("GeneratePresignedURL generating URL",
		zap.String("enduser.id", userID.String()),
		zap.String("workspace.id", req.WorkspaceID),
		zap.String("fileName", req.FileName),
		zap.String("contentType", req.ContentType))

	// Presigned URL 생성
	uploadURL, fileKey, err := h.s3Client.GeneratePresignedURL(
		c.Request.Context(),
		req.WorkspaceID,
		req.FileName,
		req.ContentType,
	)
	if err != nil {
		log.Error("GeneratePresignedURL failed to generate URL", zap.Error(err))
		response.SendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate presigned URL")
		return
	}

	// 업로드 후 접근 가능한 URL 생성
	downloadURL := h.s3Client.GetFileURL(fileKey)

	// 응답 반환
	resp := PresignedURLResponse{
		UploadURL:   uploadURL,
		DownloadURL: downloadURL,
		FileKey:     fileKey,
		ExpiresIn:   300, // 5분
	}

	log.Info("GeneratePresignedURL completed",
		zap.String("enduser.id", userID.String()),
		zap.String("fileKey", fileKey))

	response.SendSuccess(c, http.StatusOK, resp)
}
