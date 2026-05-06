package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ismael/qr-restaurant/internal/audit"
	"github.com/ismael/qr-restaurant/internal/auth"
	"github.com/ismael/qr-restaurant/internal/menu"
	"github.com/ismael/qr-restaurant/internal/middleware"
	"github.com/ismael/qr-restaurant/internal/order"
	"github.com/ismael/qr-restaurant/internal/restaurant"
	"github.com/ismael/qr-restaurant/internal/session"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	"github.com/ismael/qr-restaurant/internal/shared/metrics"
	"github.com/ismael/qr-restaurant/internal/staff"
	"github.com/ismael/qr-restaurant/internal/table"
	"github.com/ismael/qr-restaurant/internal/theme"
	"github.com/ismael/qr-restaurant/internal/upload"
	"github.com/ismael/qr-restaurant/internal/websocket"
	"github.com/ismael/qr-restaurant/pkg/storage"
	"gorm.io/gorm"
)

func NewRouter(cfg *config.Config, db *gorm.DB, storageClient *storage.Client) *gin.Engine {
	audit.Configure(db)

	authRepo := auth.NewRepository(db)
	restaurantRepo := restaurant.NewRepository(db)
	staffRepo := staff.NewRepository(db)
	themeRepo := theme.NewRepository(db)
	tableRepo := table.NewRepository(db)
	sessionOrderRepo := order.NewRepository(db)
	sessionRepo := session.NewRepository(db)
	menuRepo := menu.NewRepository(db)

	authSvc := auth.NewService(authRepo, cfg.BCryptCost)
	restaurantSvc := restaurant.NewService(restaurantRepo, authRepo, staffRepo, themeRepo, cfg.BCryptCost)
	staffSvc := staff.NewService(staffRepo, authRepo)
	themeSvc := theme.NewService(themeRepo)
	tableSvc := table.NewService(tableRepo, restaurantRepo)
	menuSvc := menu.NewService(menuRepo)

	wsHub := websocket.NewHub()
	orderBroadcaster := &orderBroadcaster{hub: wsHub}
	sessionSvc := session.NewService(sessionRepo, tableRepo, restaurantRepo, sessionOrderRepo, cfg.SessionTTLHours)
	orderSvc := order.NewService(sessionOrderRepo, sessionRepo, menuRepo, orderBroadcaster)
	wsHandler := websocket.NewWSHandler(wsHub, db, cfg.FrontendBaseURL)

	authHandler := auth.NewHandler(authSvc, cfg)
	restaurantHandler := restaurant.NewHandler(restaurantSvc)
	staffHandler := staff.NewHandler(staffSvc, cfg)
	themeHandler := theme.NewHandler(themeSvc)
	tableHandler := table.NewHandler(tableSvc, cfg)
	sessionHandler := session.NewHandler(sessionSvc, cfg, db)
	menuHandler := menu.NewHandler(menuSvc, sessionRepo, restaurantRepo)
	orderHandler := order.NewHandler(orderSvc, wsHub)
	uploadHandler := upload.NewHandler(storageClient, cfg)

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(metrics.Middleware())
	r.Use(middleware.Logger())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{cfg.FrontendBaseURL}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Request-ID"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "X-Request-ID"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour
	r.Use(cors.New(corsConfig))

	r.Use(func(c *gin.Context) {
		c.Set("jwt_secret", cfg.JWTSecret)
		c.Set("db", db)
		c.Next()
	})

	r.GET("/api/v1/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unavailable"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "ok"})
	})
	r.GET("/metrics", func(c *gin.Context) {
		sqlDB, _ := db.DB()
		metrics.Handler(sqlDB)(c)
	})

	r.GET("/ws/sessions/:sessionToken", wsHandler.ServeWS)

	v1 := r.Group("/api/v1")
	authG := v1.Group("/auth", middleware.RateLimiter(cfg.RateLimitPublic))
	{
		authG.POST("/register", authHandler.Register)
		authG.POST("/login", authHandler.Login)
		authG.GET("/me", middleware.Auth(), authHandler.Me)
	}

	admin := v1.Group("/restaurants", middleware.RateLimiter(cfg.RateLimitAdmin), middleware.Auth())
	{
		admin.POST("", middleware.RequireRole("OWNER"), restaurantHandler.Create)
		admin.GET("", middleware.RequireRole("OWNER"), restaurantHandler.ListOwned)

		rGroup := admin.Group("/:restaurantId", middleware.RestaurantAccess())
		{
			rGroup.GET("", restaurantHandler.Get)
			rGroup.PATCH("", middleware.RequireRole("OWNER"), restaurantHandler.Update)
			rGroup.DELETE("", middleware.RequireRole("OWNER"), restaurantHandler.Delete)

			rGroup.POST("/staff", middleware.RequireRole("OWNER"), staffHandler.Create)
			rGroup.GET("/staff", middleware.RequireRole("OWNER"), staffHandler.List)
			rGroup.DELETE("/staff/:sid", middleware.RequireRole("OWNER"), staffHandler.Remove)

			rGroup.GET("/theme", themeHandler.Get)
			rGroup.PUT("/theme", middleware.RequireRole("OWNER"), themeHandler.Upsert)

			rGroup.POST("/tables", middleware.RequireRole("OWNER"), tableHandler.Create)
			rGroup.GET("/tables", tableHandler.List)
			rGroup.PATCH("/tables/:tableId", middleware.RequireRole("OWNER"), tableHandler.Update)
			rGroup.DELETE("/tables/:tableId", middleware.RequireRole("OWNER"), tableHandler.Delete)
			rGroup.POST("/tables/:tableId/regenerate-qr", middleware.RequireRole("OWNER"), tableHandler.RegenerateQR)

			rGroup.POST("/menu/categories", middleware.RequireRole("OWNER"), menuHandler.CreateCategory)
			rGroup.GET("/menu/categories", menuHandler.ListCategories)
			rGroup.PATCH("/menu/categories/:cid", middleware.RequireRole("OWNER"), menuHandler.UpdateCategory)
			rGroup.DELETE("/menu/categories/:cid", middleware.RequireRole("OWNER"), menuHandler.DeleteCategory)
			rGroup.PUT("/menu/categories/reorder", middleware.RequireRole("OWNER"), menuHandler.ReorderCategories)

			rGroup.POST("/menu/categories/:cid/items", middleware.RequireRole("OWNER"), menuHandler.CreateItem)
			rGroup.GET("/menu/categories/:cid/items", menuHandler.ListItemsByCategory)
			rGroup.GET("/menu/items", menuHandler.ListAllItems)
			rGroup.PATCH("/menu/items/:itemId", middleware.RequireRole("OWNER"), menuHandler.UpdateItem)
			rGroup.DELETE("/menu/items/:itemId", middleware.RequireRole("OWNER"), menuHandler.DeleteItem)
			rGroup.PUT("/menu/categories/:cid/items/reorder", middleware.RequireRole("OWNER"), menuHandler.ReorderItems)

			rGroup.GET("/sessions", sessionHandler.ListAdmin)
			rGroup.GET("/sessions/:sid", sessionHandler.GetAdmin)
			rGroup.POST("/sessions/:sid/close", sessionHandler.Close)
			rGroup.PATCH("/sessions/:sid/payment", sessionHandler.UpdatePayment)

			rGroup.GET("/orders", orderHandler.ListAdmin)
			rGroup.GET("/orders/:oid", orderHandler.GetAdmin)
			rGroup.PATCH("/orders/:oid/status", orderHandler.UpdateStatus)
		}
	}

	v1.POST("/upload/image", middleware.RateLimiter(cfg.RateLimitAdmin), middleware.Auth(), middleware.RequireRole("OWNER"), uploadHandler.UploadImage)

	pub := v1.Group("/public", middleware.RateLimiter(cfg.RateLimitPublic))
	{
		pub.GET("/scan", sessionHandler.Scan)
		pub.GET("/sessions/:sessionToken", sessionHandler.GetPublic)
		pub.GET("/sessions/:sessionToken/menu", menuHandler.GetPublicMenu)
		pub.POST("/sessions/:sessionToken/orders", orderHandler.PlaceOrder)
		pub.GET("/sessions/:sessionToken/orders", orderHandler.ListCustomerOrders)
	}

	return r
}

type orderBroadcaster struct {
	hub *websocket.Hub
}

func (b *orderBroadcaster) BroadcastOrderUpdate(sessionToken string, orderResp *order.OrderResponse) {
	if b.hub == nil {
		return
	}
	data, _ := json.Marshal(orderResp)
	b.hub.BroadcastToSession(sessionToken, &websocket.Message{
		Type: "order_updated",
		Data: json.RawMessage(data),
	})
}
