package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// BaseConfig holds the shared configuration used by API and Worker services.
type BaseConfig struct {
	Database DatabaseConfig
	MinIO    MinIOConfig
	RabbitMQ RabbitMQConfig
	TTS      TTSConfig
	External ExternalConfig
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
	PublicEndpoint string
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
	ASR ASRConfig
	GLM GLMConfig
}

// ASRConfig holds ASR service configuration.
type ASRConfig struct {
	URL string
}

// GLMConfig holds GLM API configuration.
type GLMConfig struct {
	APIKey string
	APIURL string
	Model  string
	RPS    float64
}

// Option customizes the Loader behaviour.
type Option func(*loader)

// Loader loads and validates shared configuration with optional overrides.
type loader struct {
	v          *viper.Viper
	defaults   map[string]interface{}
	validators []func(*BaseConfig) error
	postLoad   []func(*BaseConfig)
}

// NewLoader creates a new loader with shared defaults and optional overrides.
func NewLoader(opts ...Option) *loader {
	baseDefaults := map[string]interface{}{
		"DB_HOST":               "localhost",
		"DB_PORT":               5432,
		"DB_NAME":               "dubbing",
		"DB_USER":               "dubbing",
		"DB_PASSWORD":           "dubbing123",
		"DB_SSLMODE":            "disable",
		"MINIO_ENDPOINT":        "localhost:9000",
		"MINIO_PUBLIC_ENDPOINT": "",
		"MINIO_ACCESS_KEY":      "minioadmin",
		"MINIO_SECRET_KEY":      "minioadmin123",
		"MINIO_USE_SSL":         false,
		"MINIO_BUCKET":          "videos",
		"RABBITMQ_URL":          "amqp://rabbitmq:rabbitmq123@localhost:5672/",
		"TTS_SERVICE_URL":       "http://localhost:8000",
		"ASR_SERVICE_URL":       "",
		"GLM_API_URL":           "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		"GLM_MODEL":             "glm-4.5",
		"GLM_RPS":               5.0,
	}

	l := &loader{
		v:          viper.New(),
		defaults:   baseDefaults,
		validators: []func(*BaseConfig) error{validateBase},
	}

	l.v.SetEnvPrefix("")
	l.v.AutomaticEnv()

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// WithDefaults overrides or adds default values before loading configuration.
func WithDefaults(overrides map[string]interface{}) Option {
	return func(l *loader) {
		for k, v := range overrides {
			l.defaults[k] = v
		}
	}
}

// WithValidator adds a custom validator to the loader.
func WithValidator(validator func(*BaseConfig) error) Option {
	return func(l *loader) {
		l.validators = append(l.validators, validator)
	}
}

// WithPostLoad appends a hook executed after the configuration is loaded.
func WithPostLoad(hook func(*BaseConfig)) Option {
	return func(l *loader) {
		l.postLoad = append(l.postLoad, hook)
	}
}

// WithMinIOPublicFallback sets PublicEndpoint to Endpoint when left empty.
func WithMinIOPublicFallback() Option {
	return WithPostLoad(func(cfg *BaseConfig) {
		if cfg.MinIO.PublicEndpoint == "" {
			cfg.MinIO.PublicEndpoint = cfg.MinIO.Endpoint
		}
	})
}

// Viper returns the underlying viper instance for additional module-specific defaults.
func (l *loader) Viper() *viper.Viper {
	return l.v
}

// Load reads configuration values, applies defaults, post-load hooks, and validators.
func (l *loader) Load() (*BaseConfig, error) {
	for k, v := range l.defaults {
		l.v.SetDefault(k, v)
	}

	cfg := &BaseConfig{
		Database: DatabaseConfig{
			Host:     l.v.GetString("DB_HOST"),
			Port:     l.v.GetInt("DB_PORT"),
			Name:     l.v.GetString("DB_NAME"),
			User:     l.v.GetString("DB_USER"),
			Password: l.v.GetString("DB_PASSWORD"),
			SSLMode:  l.v.GetString("DB_SSLMODE"),
		},
		MinIO: MinIOConfig{
			Endpoint:       l.v.GetString("MINIO_ENDPOINT"),
			PublicEndpoint: l.v.GetString("MINIO_PUBLIC_ENDPOINT"),
			AccessKey:      l.v.GetString("MINIO_ACCESS_KEY"),
			SecretKey:      l.v.GetString("MINIO_SECRET_KEY"),
			UseSSL:         l.v.GetBool("MINIO_USE_SSL"),
			Bucket:         l.v.GetString("MINIO_BUCKET"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: l.v.GetString("RABBITMQ_URL"),
		},
		TTS: TTSConfig{
			URL: l.v.GetString("TTS_SERVICE_URL"),
		},
		External: ExternalConfig{
			ASR: ASRConfig{URL: l.v.GetString("ASR_SERVICE_URL")},
			GLM: GLMConfig{
				APIKey: l.v.GetString("GLM_API_KEY"),
				APIURL: l.v.GetString("GLM_API_URL"),
				Model:  l.v.GetString("GLM_MODEL"),
				RPS:    l.v.GetFloat64("GLM_RPS"),
			},
		},
	}

	for _, hook := range l.postLoad {
		hook(cfg)
	}

	for _, validator := range l.validators {
		if err := validator(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// validateBase validates required shared configuration fields.
func validateBase(cfg *BaseConfig) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if cfg.MinIO.Endpoint == "" {
		return fmt.Errorf("MINIO_ENDPOINT is required")
	}
	if cfg.MinIO.AccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if cfg.MinIO.SecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if cfg.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}

	return nil
}

// RequireASRURL ensures ASR service URL is provided.
func RequireASRURL(cfg *BaseConfig) error {
	if cfg.External.ASR.URL == "" {
		return fmt.Errorf("ASR_SERVICE_URL is required")
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
