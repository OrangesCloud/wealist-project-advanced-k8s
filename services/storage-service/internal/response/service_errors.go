package response

import (
	"errors"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// AppError는 공통 에러 패키지의 타입 alias입니다.
type AppError = apperrors.AppError

// Service-level sentinel errors for storage-service
// 기존 sentinel 에러들 (하위 호환성 유지)
var (
	ErrAccessDenied           = errors.New("access denied")
	ErrNotWorkspaceMember     = errors.New("user is not a workspace member")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrCannotRemoveOwner      = errors.New("cannot remove project owner")
	ErrCannotChangeOwnRole    = errors.New("cannot change own role")
	ErrInvalidPermission      = errors.New("invalid permission value")
	ErrProjectNotFound        = errors.New("project not found")
	ErrProjectMemberNotFound  = errors.New("project member not found")
	ErrFileNotFound           = errors.New("file not found")
	ErrFolderNotFound         = errors.New("folder not found")
	ErrMemberAlreadyExists    = errors.New("member already exists")
	ErrShareNotFound          = errors.New("share not found")
)

// ============================================================
// 타입화된 에러 생성 함수들 (Service Layer용)
// Handler에서 HandleError()로 자동 HTTP 상태 매핑
// ============================================================

// NewNotFoundError는 404 NOT_FOUND 에러를 생성합니다.
// 리소스를 찾을 수 없을 때 사용합니다.
func NewNotFoundError(message, details string) *AppError {
	return apperrors.NotFound(message, details)
}

// NewForbiddenError는 403 FORBIDDEN 에러를 생성합니다.
// 권한이 없을 때 사용합니다.
func NewForbiddenError(message, details string) *AppError {
	return apperrors.Forbidden(message, details)
}

// NewValidationError는 400 VALIDATION 에러를 생성합니다.
// 입력값 검증 실패 시 사용합니다.
func NewValidationError(message, details string) *AppError {
	return apperrors.Validation(message, details)
}

// NewConflictError는 409 CONFLICT 에러를 생성합니다.
// 리소스 충돌(중복) 시 사용합니다.
func NewConflictError(message, details string) *AppError {
	return apperrors.Conflict(message, details)
}

// NewAlreadyExistsError는 409 ALREADY_EXISTS 에러를 생성합니다.
// 리소스가 이미 존재할 때 사용합니다.
func NewAlreadyExistsError(message, details string) *AppError {
	return apperrors.AlreadyExists(message, details)
}

// NewBadRequestError는 400 BAD_REQUEST 에러를 생성합니다.
// 잘못된 요청일 때 사용합니다.
func NewBadRequestError(message, details string) *AppError {
	return apperrors.BadRequest(message, details)
}

// NewUnauthorizedError는 401 UNAUTHORIZED 에러를 생성합니다.
// 인증이 필요할 때 사용합니다.
func NewUnauthorizedError(message, details string) *AppError {
	return apperrors.Unauthorized(message, details)
}

// NewInternalError는 500 INTERNAL 에러를 생성합니다.
// 서버 내부 오류 시 사용합니다.
func NewInternalError(message, details string) *AppError {
	return apperrors.Internal(message, details)
}

// HandleServiceError maps service errors to HTTP responses.
// This replaces string-matching error handling with proper error comparison.
func HandleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAccessDenied):
		Error(c, apperrors.Forbidden("Access denied", ""))

	case errors.Is(err, ErrNotWorkspaceMember):
		Error(c, apperrors.Forbidden("User is not a member of this workspace", ""))

	case errors.Is(err, ErrInsufficientPermission):
		Error(c, apperrors.Forbidden("Insufficient permission to perform this action", ""))

	case errors.Is(err, ErrCannotRemoveOwner):
		Error(c, apperrors.BadRequest("Cannot remove the only owner of the project", ""))

	case errors.Is(err, ErrCannotChangeOwnRole):
		Error(c, apperrors.BadRequest("Cannot change your own role", ""))

	case errors.Is(err, ErrInvalidPermission):
		Error(c, apperrors.BadRequest("Invalid permission value", ""))

	case errors.Is(err, ErrProjectNotFound):
		Error(c, apperrors.NotFound("Project not found", ""))

	case errors.Is(err, ErrProjectMemberNotFound):
		Error(c, apperrors.NotFound("Project member not found", ""))

	case errors.Is(err, ErrFileNotFound):
		Error(c, apperrors.NotFound("File not found", ""))

	case errors.Is(err, ErrFolderNotFound):
		Error(c, apperrors.NotFound("Folder not found", ""))

	case errors.Is(err, ErrMemberAlreadyExists):
		Error(c, apperrors.Conflict("User is already a member of this project", ""))

	case errors.Is(err, ErrShareNotFound):
		Error(c, apperrors.NotFound("Share not found", ""))

	default:
		// Handle AppError if present
		if appErr := apperrors.AsAppError(err); appErr != nil {
			Error(c, appErr)
			return
		}
		// Default to internal error
		Error(c, apperrors.Internal("An internal error occurred", ""))
	}
}
