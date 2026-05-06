package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Logger returns a middleware that logs requests using zerolog.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		tenantID := contextString(c, "restaurant_id")
		sessionID := c.GetString("session_id")

		log.Info().
			Str("request_id", GetRequestID(c)).
			Str("method", method).
			Str("route", c.FullPath()).
			Str("path", path).
			Int("status_code", status).
			Dur("latency", latency).
			Int("size", c.Writer.Size()).
			Str("ip", c.ClientIP()).
			Str("tenant_id", tenantID).
			Str("session_id", sessionID).
			Msg("request")
	}
}

func contextString(c *gin.Context, key string) string {
	if value, ok := c.Get(key); ok && value != nil {
		return fmt.Sprint(value)
	}
	return ""
}
