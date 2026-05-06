package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
	"gorm.io/gorm"
)

// RestaurantAccess returns a middleware that validates the calling user has access
// to the restaurant specified by :restaurantId in the URL path.
// OWNER: verifies restaurants.owner_id = claims.user_id
// STAFF: verifies row exists in restaurant_staff
// Injects the restaurant into context on success.
func RestaurantAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, exists := c.Get("db")
		if !exists {
			resp.InternalError(c)
			c.Abort()
			return
		}

		restaurantIDStr := c.Param("restaurantId")
		if restaurantIDStr == "" {
			resp.NotFound(c, "Restaurant ID is required")
			c.Abort()
			return
		}

		restaurantID, err := uuid.Parse(restaurantIDStr)
		if err != nil {
			resp.NotFound(c, "Invalid restaurant ID")
			c.Abort()
			return
		}

		userIDStr, exists := c.Get("user_id")
		if !exists {
			resp.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}
		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			resp.InternalError(c)
			c.Abort()
			return
		}

		role, _ := c.Get("role")
		userRole := role.(string)

		gormDB := db.(*gorm.DB)

		var restaurant models.Restaurant
		if err := gormDB.First(&restaurant, "id = ?", restaurantID).Error; err != nil {
			resp.NotFound(c, "Restaurant not found")
			c.Abort()
			return
		}

		if userRole == "OWNER" {
			if restaurant.OwnerID != userID {
				resp.Forbidden(c, "You do not have access to this restaurant")
				c.Abort()
				return
			}
		} else if userRole == "STAFF" {
			var count int64
			if err := gormDB.Model(&models.RestaurantStaff{}).
				Where("restaurant_id = ? AND user_id = ?", restaurantID, userID).
				Count(&count).Error; err != nil || count == 0 {
				resp.Forbidden(c, "You do not have access to this restaurant")
				c.Abort()
				return
			}
		}

		c.Set("restaurant", &restaurant)
		c.Set("restaurant_id", restaurantID)
		c.Next()
	}
}
