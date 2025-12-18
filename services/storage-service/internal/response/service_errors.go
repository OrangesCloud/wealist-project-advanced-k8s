package response

import (
	"errors"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// Service-level sentinel errors for storage-service
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
