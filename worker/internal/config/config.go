package config

import (
	"fmt"

	sharedconfig "vedio/shared/config"
)

// Aliases for shared configuration structures to keep existing references intact.
type DatabaseConfig = sharedconfig.DatabaseConfig
type MinIOConfig = sharedconfig.MinIOConfig
type RabbitMQConfig = sharedconfig.RabbitMQConfig
type TTSConfig = sharedconfig.TTSConfig
type ExternalConfig = sharedconfig.ExternalConfig
type ASRConfig = sharedconfig.ASRConfig
type GLMConfig = sharedconfig.GLMConfig

// Config holds all configuration for the worker.
type Config struct {
	sharedconfig.BaseConfig
	FFmpeg FFmpegConfig
}

// FFmpegConfig holds FFmpeg configuration.
type FFmpegConfig struct {
	Path string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	loader := sharedconfig.NewLoader(
		sharedconfig.WithDefaults(map[string]interface{}{
			"ASR_SERVICE_URL": "http://localhost:8002",
		}),
		sharedconfig.WithValidator(sharedconfig.RequireASRURL),
		sharedconfig.WithMinIOPublicFallback(),
	)

	v := loader.Viper()
	v.SetDefault("FFMPEG_PATH", "/usr/bin/ffmpeg")

	baseCfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	cfg := &Config{
		BaseConfig: *baseCfg,
		FFmpeg: FFmpegConfig{
			Path: v.GetString("FFMPEG_PATH"),
		},
	}

	return cfg, nil
}
