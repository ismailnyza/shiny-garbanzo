package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	appjwt "github.com/ismael/qr-restaurant/pkg/jwt"

	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

// Auth returns a middleware that validates JWT tokens and injects claims into context.
// It reads the JWT secret from the config stored in gin context.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			resp.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			resp.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Get JWT secret from config stored in context
		secret, exists := c.Get("jwt_secret")
		if !exists {
			resp.InternalError(c)
			c.Abort()
			return
		}

		claims, err := appjwt.Validate(secret.(string), tokenString)
		if err != nil {
			resp.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Inject claims into context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		// Store full claims for downstream use
		c.Set("jwt_claims", map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		})

		c.Next()
	}
}

// GetClaims extracts JWT claims from gin context.
func GetClaims(c *gin.Context) (jwt.MapClaims, bool) {
	claims, exists := c.Get("jwt_claims")
	if !exists {
		return nil, false
	}
	return claims.(jwt.MapClaims), true
}
