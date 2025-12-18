// Package response provides centralized HTTP response helpers for noti-service.
//
// This package standardizes API responses across all handlers, ensuring
// consistent error formats and success responses throughout the service.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/OrangesCloud/wealist-advanced-go-pkg/errors"
)

// Error sends an error response based on AppError.
// It maps the error code to appropriate HTTP status.
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

// ValidationError sends a 400 Bad Request response for validation failures.
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

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.Internal(message, ""))
}

// Success sends a success response with a message.
func Success(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// OK sends a 200 OK response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// OKWithPagination sends a 200 OK response with pagination info.
func OKWithPagination(c *gin.Context, data interface{}, total int64, page, limit int) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// CustomError sends a custom error response with specific status code and code.
func CustomError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
