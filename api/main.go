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

	"vedio/api/internal/config"
	"vedio/api/internal/database"
	"vedio/api/internal/queue"
	"vedio/api/internal/router"
	"vedio/api/internal/storage"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting API service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connected successfully")
	if err := database.Migrate(db.DB); err != nil {
		logger.Fatal("Failed to migrate database schema", zap.Error(err))
	}

	// Initialize object storage (minio or oss)
	storageService, err := storage.NewFromConfig(&cfg.BaseConfig)
	if err != nil {
		logger.Fatal("Failed to initialize object storage", zap.Error(err))
	}
	logger.Info("Object storage initialized successfully", zap.String("backend", cfg.Storage.Backend))

	// Initialize RabbitMQ connection
	queueConn, err := queue.NewConnection(cfg.RabbitMQ)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer queueConn.Close()

	logger.Info("RabbitMQ connected successfully")

	// Initialize publisher
	publisher := queue.NewPublisher(queueConn)

	// Initialize router
	r := router.New(db, storageService, publisher, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
