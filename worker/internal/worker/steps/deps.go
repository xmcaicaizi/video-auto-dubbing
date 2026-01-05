package steps

import (
	"context"

	"vedio/worker/internal/asr"
	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/storage"
	"vedio/worker/internal/tts"

	"go.uber.org/zap"
)

// Publisher defines the minimal behaviour for publishing next-step messages.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, message interface{}) error
}

// Deps groups common dependencies shared across step processors.
type Deps struct {
	DB        *database.DB
	Storage   *storage.Service
	Publisher Publisher
	Config    *config.Config
	Logger    *zap.Logger
	ASRClient *asr.Client
	TTSClient *tts.Client
}
