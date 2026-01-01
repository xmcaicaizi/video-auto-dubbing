package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	MinIO    MinIOConfig
	RabbitMQ RabbitMQConfig
	TTS      TTSConfig
	External ExternalConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// MinIOConfig holds MinIO configuration.
type MinIOConfig struct {
	Endpoint       string
	PublicEndpoint string // 用于生成浏览器可达的预签名 URL
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	Bucket         string
}

// RabbitMQConfig holds RabbitMQ configuration.
type RabbitMQConfig struct {
	URL string
}

// TTSConfig holds TTS service configuration.
type TTSConfig struct {
	URL string
}

// ExternalConfig holds external API configuration.
type ExternalConfig struct {
	VolcEngineASR VolcEngineASRConfig
	GLM           GLMConfig
}

// VolcEngineASRConfig holds VolcEngine ASR API configuration.
type VolcEngineASRConfig struct {
	AccessKey string
	SecretKey string
}

// GLMConfig holds GLM API configuration.
type GLMConfig struct {
	APIKey string
	APIURL string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("API_HOST", "0.0.0.0")
	viper.SetDefault("API_PORT", 8080)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_NAME", "dubbing")
	viper.SetDefault("DB_USER", "dubbing")
	viper.SetDefault("DB_PASSWORD", "dubbing123")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	viper.SetDefault("MINIO_PUBLIC_ENDPOINT", "localhost:9000") // 默认与内部端点相同
	viper.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO_SECRET_KEY", "minioadmin123")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("MINIO_BUCKET", "videos")
	viper.SetDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq123@localhost:5672/")
	viper.SetDefault("TTS_SERVICE_URL", "http://localhost:8000")
	viper.SetDefault("GLM_API_URL", "https://api.example.com/glm")

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("API_HOST"),
			Port: viper.GetInt("API_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			Name:     viper.GetString("DB_NAME"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		MinIO: MinIOConfig{
			Endpoint:       viper.GetString("MINIO_ENDPOINT"),
			PublicEndpoint: viper.GetString("MINIO_PUBLIC_ENDPOINT"),
			AccessKey:      viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey:      viper.GetString("MINIO_SECRET_KEY"),
			UseSSL:         viper.GetBool("MINIO_USE_SSL"),
			Bucket:         viper.GetString("MINIO_BUCKET"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: viper.GetString("RABBITMQ_URL"),
		},
		TTS: TTSConfig{
			URL: viper.GetString("TTS_SERVICE_URL"),
		},
		External: ExternalConfig{
			VolcEngineASR: VolcEngineASRConfig{
				AccessKey: viper.GetString("VOLCENGINE_ASR_ACCESS_KEY"),
				SecretKey: viper.GetString("VOLCENGINE_ASR_SECRET_KEY"),
			},
			GLM: GLMConfig{
				APIKey: viper.GetString("GLM_API_KEY"),
				APIURL: viper.GetString("GLM_API_URL"),
			},
		},
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// validate validates the configuration.
func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.MinIO.Endpoint == "" {
		return fmt.Errorf("MINIO_ENDPOINT is required")
	}
	if c.MinIO.AccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if c.MinIO.SecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	return nil
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

