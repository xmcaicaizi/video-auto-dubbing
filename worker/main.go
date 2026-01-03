package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/minio"
	"vedio/worker/internal/queue"
	"vedio/worker/internal/storage"
	"vedio/worker/internal/worker"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Worker service...")

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

	// Initialize MinIO client
	minioClient, err := minio.New(cfg.MinIO)
	if err != nil {
		logger.Fatal("Failed to initialize MinIO client", zap.Error(err))
	}

	logger.Info("MinIO client initialized successfully")

	// Initialize storage service
	publicEndpoint := cfg.MinIO.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.MinIO.Endpoint
	}
	storageService := storage.New(minioClient, cfg.MinIO.Bucket, publicEndpoint)

	// Initialize RabbitMQ connection
	queueConn, err := queue.NewConnection(cfg.RabbitMQ)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer queueConn.Close()

	logger.Info("RabbitMQ connected successfully")

	// Initialize publisher for next steps
	publisher := queue.NewPublisher(queueConn)

	// Initialize worker
	w := worker.New(db, storageService, publisher, cfg, logger)

	// Start workers for each step
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer for each registered step
	w.StartAllConsumers(ctx)

	logger.Info("All workers started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down workers...")
	cancel()

	// Give workers time to finish
	time.Sleep(5 * time.Second)
	logger.Info("Worker service exited")
}
