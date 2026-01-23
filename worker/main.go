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
	"vedio/worker/internal/queue"
	"vedio/worker/internal/settings"
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

	// Load settings from database and merge into config
	settingsLoader := settings.NewLoader(db.DB)
	dbSettings, err := settingsLoader.Load(context.Background())
	if err != nil {
		logger.Warn("Failed to load settings from database, using environment config only", zap.Error(err))
	} else {
		dbSettings.MergeIntoConfig(&cfg.BaseConfig)
		logger.Info("Database settings loaded and merged",
			zap.Bool("asr_configured", dbSettings.HasValidASRConfig()),
			zap.Bool("tts_configured", dbSettings.HasValidTTSConfig()),
			zap.Bool("translate_configured", dbSettings.HasValidTranslateConfig()),
			zap.Bool("storage_configured", dbSettings.HasValidStorageConfig()),
		)
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
