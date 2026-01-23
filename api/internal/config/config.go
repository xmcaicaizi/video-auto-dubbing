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
type GLMConfig = sharedconfig.GLMConfig

// Config holds all configuration for the application.
type Config struct {
	sharedconfig.BaseConfig
	Server ServerConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host string
	Port int
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	loader := sharedconfig.NewLoader(
		sharedconfig.WithMinIOPublicFallback(),
	)

	v := loader.Viper()
	v.SetDefault("API_HOST", "0.0.0.0")
	v.SetDefault("API_PORT", 8080)

	baseCfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	cfg := &Config{
		BaseConfig: *baseCfg,
		Server: ServerConfig{
			Host: v.GetString("API_HOST"),
			Port: v.GetInt("API_PORT"),
		},
	}

	return cfg, nil
}
