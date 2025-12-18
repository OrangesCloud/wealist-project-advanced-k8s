// Package response provides HTTP response utilities for the chat-service.
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
		"success": false,
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	Error(c, apperrors.BadRequest(message, ""))
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

// Success sends a 200 OK response with success wrapper and message.
func Success(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// OK sends a 200 OK response with success wrapper and data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
