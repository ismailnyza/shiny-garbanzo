package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ismael/qr-restaurant/internal/shared/metrics"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimiter returns a middleware that rate-limits requests per IP.
func RateLimiter(rps int) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  int64(rps),
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	middleware := mgin.NewMiddleware(instance,
		mgin.WithKeyGetter(func(c *gin.Context) string {
			return c.ClientIP()
		}),
		mgin.WithErrorHandler(func(c *gin.Context, err error) {
			metrics.Default.IncRateLimited()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please try again later.",
				},
			})
		}),
	)

	return middleware
}
