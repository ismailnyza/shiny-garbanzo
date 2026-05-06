package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ismael/qr-restaurant/internal/app"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	"github.com/ismael/qr-restaurant/internal/shared/database"
	"github.com/ismael/qr-restaurant/pkg/storage"
	"github.com/rs/zerolog"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Connect to database
	dsn := cfg.DatabaseDSN()
	db, err := database.NewGORM(dsn)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Run migrations
	if err := database.RunMigrations(cfg.MigrationDatabaseURL(), cfg.MigratePath); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize R2 storage client (optional)
	var storageClient *storage.Client
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" {
		sc, err := storage.NewClient(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2BucketName, cfg.R2PublicURL)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize R2 storage, uploads will be unavailable")
		} else {
			storageClient = sc
		}
	} else {
		logger.Warn().Msg("R2 credentials not configured, uploads will be unavailable")
	}

	r := app.NewRouter(cfg, db, storageClient)

	// Start server with graceful shutdown
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info().Str("port", cfg.Port).Msg("Starting server")

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exiting")
}
