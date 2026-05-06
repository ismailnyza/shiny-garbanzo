package middleware

import (
	"github.com/gin-gonic/gin"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

// RequireRole returns a middleware that checks the user's role against allowed roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			resp.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowed := range roles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		resp.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}
