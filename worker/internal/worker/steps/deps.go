package steps

import (
	"context"

	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/storage"

	"go.uber.org/zap"
)

// Publisher defines the minimal behaviour for publishing next-step messages.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, message interface{}) error
}

// Deps groups common dependencies shared across step processors.
type Deps struct {
	DB             *database.DB
	Storage        storage.ObjectStorage
	Publisher      Publisher
	ConfigManager  *config.Manager
	ProcessingConfig *config.Config  // Worker processing config (timeouts, batch sizes, etc.)
	Logger         *zap.Logger
}
