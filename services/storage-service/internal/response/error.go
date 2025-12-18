// Package response provides HTTP response utilities for the storage-service.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// Error sends an error response using the common AppError type.
func Error(c *gin.Context, err *apperrors.AppError) {
	status := apperrors.GetHTTPStatus(err.Code)
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// ErrorWithDetails sends an error response with additional details.
func ErrorWithDetails(c *gin.Context, err *apperrors.AppError) {
	status := apperrors.GetHTTPStatus(err.Code)
	response := gin.H{
		"code":    err.Code,
		"message": err.Message,
	}
	if err.Details != "" {
		response["details"] = err.Details
	}
	c.JSON(status, gin.H{"error": response})
}

// HandleError converts a generic error to an appropriate HTTP response.
func HandleError(c *gin.Context, err error) {
	if appErr := apperrors.AsAppError(err); appErr != nil {
		Error(c, appErr)
		return
	}
	Error(c, apperrors.Internal("An unexpected error occurred", err.Error()))
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	Error(c, apperrors.BadRequest(message, ""))
}

// BadRequestWithDetails sends a 400 Bad Request response with details.
func BadRequestWithDetails(c *gin.Context, message, details string) {
	Error(c, apperrors.BadRequest(message, details))
}

// ValidationError sends a 400 response for validation errors.
func ValidationError(c *gin.Context, message string) {
	Error(c, apperrors.Validation(message, ""))
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	Error(c, apperrors.Unauthorized(message, ""))
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	Error(c, apperrors.Forbidden(message, ""))
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	Error(c, apperrors.NotFound(message, ""))
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	Error(c, apperrors.Conflict(message, ""))
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.Internal(message, ""))
}

// InternalErrorWithDetails sends a 500 response with error details.
func InternalErrorWithDetails(c *gin.Context, message string, err error) {
	details := ""
	if err != nil {
		details = err.Error()
	}
	Error(c, apperrors.Internal(message, details))
}

// Success sends a success response with a message.
func Success(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// OK sends a 200 OK response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
