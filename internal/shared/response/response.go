package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the standard API response wrapper.
type Envelope struct {
	Success   bool        `json:"success"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorBody  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Meta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Envelope{
		Success:   true,
		RequestID: requestID(c),
		Data:      data,
	})
}

func SuccessWithMeta(c *gin.Context, status int, data interface{}, meta *Meta) {
	c.JSON(status, Envelope{
		Success:   true,
		RequestID: requestID(c),
		Data:      data,
		Meta:      meta,
	})
}

func Created(c *gin.Context, data interface{}) {
	Success(c, http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, Envelope{Success: true, RequestID: requestID(c)})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Envelope{
		Success:   false,
		RequestID: requestID(c),
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func ErrorWithDetails(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, Envelope{
		Success:   false,
		RequestID: requestID(c),
		Error: &ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func ValidationError(c *gin.Context, message string, details interface{}) {
	ErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", message, details)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}

func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString("request_id")
}
