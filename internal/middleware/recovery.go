package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismael/qr-restaurant/internal/shared/metrics"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		metrics.Default.IncPanic()
		resp.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	})
}
