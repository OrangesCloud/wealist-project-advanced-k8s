package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"user-service/internal/domain"
	"user-service/internal/middleware"
	"user-service/internal/response"
	"user-service/internal/service"
)

// ProfileHandler handles user profile HTTP requests
type ProfileHandler struct {
	profileService    *service.ProfileService
	attachmentService *service.AttachmentService
}

// NewProfileHandler creates a new ProfileHandler
func NewProfileHandler(profileService *service.ProfileService, attachmentService *service.AttachmentService) *ProfileHandler {
	return &ProfileHandler{
		profileService:    profileService,
		attachmentService: attachmentService,
	}
}

// CreateProfile godoc
// @Summary Create user profile
// @Tags Profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.CreateProfileRequest true "Create profile request"
// @Success 201 {object} domain.UserProfileResponse
// @Failure 400 {object} ErrorResponse
// @Router /profiles [post]
func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req domain.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	profile, err := h.profileService.CreateProfile(userID, req)
	if err != nil {
		// 서비스 에러(Forbidden, AlreadyExists 등)를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	response.Created(c, profile.ToResponse())
}

// GetMyProfile godoc
// @Summary Get my profile for workspace
// @Tags Profiles
// @Produce json
// @Security BearerAuth
// @Param X-Workspace-Id header string true "Workspace ID"
// @Success 200 {object} domain.UserProfileResponse
// @Failure 404 {object} ErrorResponse
// @Router /profiles/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	workspaceIDStr := c.GetHeader("X-Workspace-Id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Valid X-Workspace-Id header required")
		return
	}

	profile, err := h.profileService.GetMyProfile(userID, workspaceID)
	if err != nil {
		response.NotFound(c, "Profile not found")
		return
	}

	response.OK(c, profile.ToResponse())
}

// GetAllMyProfiles godoc
// @Summary Get all my profiles
// @Tags Profiles
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.UserProfileResponse
// @Router /profiles/all/me [get]
func (h *ProfileHandler) GetAllMyProfiles(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	profiles, err := h.profileService.GetAllMyProfiles(userID)
	if err != nil {
		response.InternalErrorWithDetails(c, "Failed to fetch profiles", err)
		return
	}

	responses := make([]domain.UserProfileResponse, len(profiles))
	for i, p := range profiles {
		responses[i] = p.ToResponse()
	}

	response.OK(c, responses)
}

// GetUserProfile godoc
// @Summary Get user profile by workspace and user ID
// @Tags Profiles
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 200 {object} domain.UserProfileResponse
// @Failure 404 {object} ErrorResponse
// @Router /profiles/workspace/{workspaceId}/user/{userId} [get]
func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	viewerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	profile, err := h.profileService.GetUserProfile(viewerID, userID, workspaceID)
	if err != nil {
		// 서비스 에러(Forbidden, NotFound 등)를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	response.OK(c, profile.ToResponse())
}

// UpdateProfile godoc
// @Summary Update my profile
// @Tags Profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Workspace-Id header string true "Workspace ID"
// @Param request body domain.UpdateProfileRequest true "Update profile request"
// @Success 200 {object} domain.UserProfileResponse
// @Failure 400 {object} ErrorResponse
// @Router /profiles/me [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	workspaceIDStr := c.GetHeader("X-Workspace-Id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Valid X-Workspace-Id header required")
		return
	}

	var req domain.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	profile, err := h.profileService.UpdateProfile(userID, workspaceID, req)
	if err != nil {
		// 서비스 에러를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	response.OK(c, profile.ToResponse())
}

// DeleteProfile godoc
// @Summary Delete my profile for workspace
// @Tags Profiles
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} SuccessResponse
// @Router /profiles/workspace/{workspaceId} [delete]
func (h *ProfileHandler) DeleteProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid workspace ID")
		return
	}

	if err := h.profileService.DeleteProfile(userID, workspaceID); err != nil {
		response.InternalErrorWithDetails(c, "Failed to delete profile", err)
		return
	}

	response.Success(c, "Profile deleted successfully")
}

// GeneratePresignedURL godoc
// @Summary Generate presigned URL for profile image upload
// @Tags Profile Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.PresignedURLRequest true "Presigned URL request"
// @Success 200 {object} domain.PresignedURLResponse
// @Failure 400 {object} ErrorResponse
// @Router /profiles/me/image/presigned-url [post]
func (h *ProfileHandler) GeneratePresignedURL(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req domain.PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	resp, err := h.attachmentService.GeneratePresignedURL(c.Request.Context(), userID, req)
	if err != nil {
		// 서비스 에러(Validation 등)를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	response.OK(c, resp)
}

// SaveAttachment godoc
// @Summary Save attachment metadata after S3 upload
// @Tags Profile Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.SaveAttachmentRequest true "Save attachment request"
// @Success 201 {object} domain.AttachmentResponse
// @Failure 400 {object} ErrorResponse
// @Router /profiles/me/image/attachment [post]
func (h *ProfileHandler) SaveAttachment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req domain.SaveAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	attachment, err := h.attachmentService.SaveAttachment(c.Request.Context(), userID, req)
	if err != nil {
		// 서비스 에러(Validation 등)를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	response.Created(c, attachment.ToResponse())
}

// ConfirmProfileImage godoc
// @Summary Confirm profile image (link attachment to profile)
// @Tags Profile Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Workspace-Id header string true "Workspace ID"
// @Param request body domain.ConfirmAttachmentRequest true "Confirm attachment request"
// @Success 200 {object} domain.UserProfileResponse
// @Failure 400 {object} ErrorResponse
// @Router /profiles/me/image [put]
func (h *ProfileHandler) ConfirmProfileImage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	workspaceIDStr := c.GetHeader("X-Workspace-Id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, "Valid X-Workspace-Id header required")
		return
	}

	var req domain.ConfirmAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Get attachment
	attachment, err := h.attachmentService.GetAttachment(req.AttachmentID)
	if err != nil {
		response.NotFound(c, "Attachment not found")
		return
	}

	// Get or create profile - try to get default nickname from existing profiles
	defaultNickName := "User"
	existingProfiles, _ := h.profileService.GetAllMyProfiles(userID)
	if len(existingProfiles) > 0 {
		defaultNickName = existingProfiles[0].NickName
	}

	profile, err := h.profileService.GetOrCreateProfile(userID, workspaceID, defaultNickName)
	if err != nil {
		response.InternalError(c, "Failed to get or create profile")
		return
	}

	// Confirm attachment
	// 첨부파일 확정: 프로필에 연결
	_, err = h.attachmentService.ConfirmAttachment(c.Request.Context(), userID, req.AttachmentID, profile.ID)
	if err != nil {
		// 서비스 에러(Forbidden, Conflict 등)를 자동으로 적절한 HTTP 상태 코드로 변환
		response.HandleError(c, err)
		return
	}

	// Update profile with image URL (use profile.WorkspaceID which was resolved from default UUID)
	updatedProfile, err := h.profileService.UpdateProfileImage(userID, profile.WorkspaceID, attachment.FileURL)
	if err != nil {
		response.InternalErrorWithDetails(c, "Failed to update profile image", err)
		return
	}

	response.OK(c, updatedProfile.ToResponse())
}
