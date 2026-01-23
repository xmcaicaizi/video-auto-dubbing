package asr

import (
	"context"

	"vedio/shared/config"
	"vedio/worker/internal/models"

	"go.uber.org/zap"
)

// Client defines the interface for ASR services.
type Client interface {
	Recognize(ctx context.Context, audioURL string, language string) (*models.ASRResult, error)
}

// NewClient creates the appropriate ASR client based on configuration.
// Currently, it always returns a Volcengine client.
func NewClient(cfg config.VolcengineASRConfig, logger *zap.Logger) Client {
	return NewVolcengineClient(cfg, logger)
}
