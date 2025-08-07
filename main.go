package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"creative-studio-server/config"
	"creative-studio-server/middleware"
	"creative-studio-server/pkg/cache"
	"creative-studio-server/pkg/database"
	"creative-studio-server/pkg/logger"
	"creative-studio-server/pkg/queue"
	"creative-studio-server/routes"
)

// @title Creative Studio Server API
// @version 1.0
// @description This is the API documentation for Creative Studio Server
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg := config.AppConfig

	// Initialize logger
	logger.InitLogger(cfg)
	logger.Info("Starting Creative Studio Server...")

	// Initialize database
	if err := database.InitDatabase(cfg); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Redis cache
	if err := cache.InitRedis(cfg); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Initialize RabbitMQ
	if err := queue.InitRabbitMQ(cfg); err != nil {
		logger.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}

	// Start background workers
	startBackgroundWorkers()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create Gin router
	r := gin.New()

	// Add global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.APIRateLimit())

	// Setup routes
	routes.SetupRoutes(r)

	// Create HTTP server
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	// Close connections
	cleanup()

	logger.Info("Server stopped")
}

func startBackgroundWorkers() {
	logger.Info("Starting background workers...")

	// Start video processing workers
	go func() {
		if err := queue.Queue.ConsumeTask("video_processing", queue.VideoProcessingHandler, 2); err != nil {
			logger.Errorf("Failed to start video processing workers: %v", err)
		}
	}()

	// Start smart composition workers
	go func() {
		if err := queue.Queue.ConsumeTask("smart_composition", queue.SmartCompositionHandler, 1); err != nil {
			logger.Errorf("Failed to start smart composition workers: %v", err)
		}
	}()

	// Start render task workers
	go func() {
		if err := queue.Queue.ConsumeTask("render_tasks", queue.RenderTaskHandler, 3); err != nil {
			logger.Errorf("Failed to start render task workers: %v", err)
		}
	}()

	// Start analysis task workers
	go func() {
		if err := queue.Queue.ConsumeTask("analysis_tasks", queue.AnalysisTaskHandler, 2); err != nil {
			logger.Errorf("Failed to start analysis task workers: %v", err)
		}
	}()

	// Start thumbnail generation workers
	go func() {
		if err := queue.Queue.ConsumeTask("thumbnail_generation", queue.ThumbnailTaskHandler, 4); err != nil {
			logger.Errorf("Failed to start thumbnail generation workers: %v", err)
		}
	}()

	logger.Info("Background workers started")
}

func cleanup() {
	logger.Info("Cleaning up resources...")

	// Close RabbitMQ connection
	if err := queue.Queue.Close(); err != nil {
		logger.Errorf("Failed to close RabbitMQ connection: %v", err)
	}

	// Close Redis connection
	if err := cache.Cache.Close(); err != nil {
		logger.Errorf("Failed to close Redis connection: %v", err)
	}

	// Close database connections would be handled by GORM automatically
	logger.Info("Cleanup completed")
}