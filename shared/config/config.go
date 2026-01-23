package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// BaseConfig holds the shared configuration used by API and Worker services.
type BaseConfig struct {
	Database DatabaseConfig
	// Storage selects which object storage backend to use.
	Storage StorageConfig
	// MinIO configuration is kept for the "minio" backend.
	MinIO    MinIOConfig
	// OSS configuration is used for the "oss" backend.
	OSS      OSSConfig
	RabbitMQ RabbitMQConfig
	TTS      TTSConfig
	External ExternalConfig
}

// StorageConfig selects the object storage backend.
// Supported values: "minio" (default), "oss".
type StorageConfig struct {
	Backend string
}

// OSSConfig holds Aliyun OSS configuration.
// Endpoint example: oss-cn-beijing.aliyuncs.com
// PublicDomain example (CNAME): vedio-auto-tran.cn-beijing.taihangcda.cn
type OSSConfig struct {
	Endpoint         string
	Bucket           string
	AccessKeyID      string
	AccessKeySecret  string
	PublicDomain     string
	Prefix           string
	UseSSL           bool
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
	URL     string
	APIKey  string // Optional API key for remote TTS service
	Backend string // TTS backend: "vllm" (default), "legacy"
}

// ExternalConfig holds external API configuration.
type ExternalConfig struct {
	VolcengineASR VolcengineASRConfig
	GLM          GLMConfig
}

// VolcengineASRConfig holds Volcengine ASR API configuration.
type VolcengineASRConfig struct {
	AppKey     string // X-Api-App-Key (火山控制台 APP ID)
	AccessKey  string // X-Api-Access-Key (Access Token)
	ResourceID string // X-Api-Resource-Id (volc.bigasr.auc 或 volc.seedasr.auc)

	// Feature toggles
	EnableSpeakerInfo   bool // 说话人分离 (10人以内)
	EnableEmotionDetect bool // 情绪检测
	EnableGenderDetect  bool // 性别检测
	EnablePunc          bool // 标点符号
	EnableITN           bool // 文本规范化

	// Polling configuration
	PollIntervalSeconds int // 轮询间隔 (秒)
	PollTimeoutSeconds  int // 轮询超时 (秒)
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
		"STORAGE_BACKEND":       "minio",
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
		// Aliyun OSS defaults
		"OSS_ENDPOINT":          "",
		"OSS_BUCKET":            "",
		"OSS_ACCESS_KEY_ID":     "",
		"OSS_ACCESS_KEY_SECRET": "",
		"OSS_PUBLIC_DOMAIN":     "",
		"OSS_PREFIX":            "",
		"OSS_USE_SSL":           true,
		"RABBITMQ_URL":          "amqp://rabbitmq:rabbitmq123@localhost:5672/",
		"TTS_SERVICE_URL":       "http://localhost:8000",
		"TTS_API_KEY":           "",
		"TTS_BACKEND":           "vllm",
		"GLM_API_URL":           "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		"GLM_MODEL":             "glm-4-flash",
		"GLM_RPS":               5.0,
		// Volcengine ASR defaults
		"VOLCENGINE_ASR_APP_KEY":              "",
		"VOLCENGINE_ASR_ACCESS_KEY":           "",
		"VOLCENGINE_ASR_RESOURCE_ID":          "volc.bigasr.auc",
		"VOLCENGINE_ASR_ENABLE_SPEAKER_INFO":  true,
		"VOLCENGINE_ASR_ENABLE_EMOTION":       true,
		"VOLCENGINE_ASR_ENABLE_GENDER":        true,
		"VOLCENGINE_ASR_ENABLE_PUNC":          true,
		"VOLCENGINE_ASR_ENABLE_ITN":           true,
		"VOLCENGINE_ASR_POLL_INTERVAL_SECONDS": 2,
		"VOLCENGINE_ASR_POLL_TIMEOUT_SECONDS":  900,
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
		Storage: StorageConfig{
			Backend: l.v.GetString("STORAGE_BACKEND"),
		},
		MinIO: MinIOConfig{
			Endpoint:       l.v.GetString("MINIO_ENDPOINT"),
			PublicEndpoint: l.v.GetString("MINIO_PUBLIC_ENDPOINT"),
			AccessKey:      l.v.GetString("MINIO_ACCESS_KEY"),
			SecretKey:      l.v.GetString("MINIO_SECRET_KEY"),
			UseSSL:         l.v.GetBool("MINIO_USE_SSL"),
			Bucket:         l.v.GetString("MINIO_BUCKET"),
		},
		OSS: OSSConfig{
			Endpoint:        l.v.GetString("OSS_ENDPOINT"),
			Bucket:          l.v.GetString("OSS_BUCKET"),
			AccessKeyID:     l.v.GetString("OSS_ACCESS_KEY_ID"),
			AccessKeySecret: l.v.GetString("OSS_ACCESS_KEY_SECRET"),
			PublicDomain:    l.v.GetString("OSS_PUBLIC_DOMAIN"),
			Prefix:          l.v.GetString("OSS_PREFIX"),
			UseSSL:          l.v.GetBool("OSS_USE_SSL"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: l.v.GetString("RABBITMQ_URL"),
		},
		TTS: TTSConfig{
			URL:     l.v.GetString("TTS_SERVICE_URL"),
			APIKey:  l.v.GetString("TTS_API_KEY"),
			Backend: l.v.GetString("TTS_BACKEND"),
		},
		External: ExternalConfig{
			VolcengineASR: VolcengineASRConfig{
				AppKey:              l.v.GetString("VOLCENGINE_ASR_APP_KEY"),
				AccessKey:           l.v.GetString("VOLCENGINE_ASR_ACCESS_KEY"),
				ResourceID:          l.v.GetString("VOLCENGINE_ASR_RESOURCE_ID"),
				EnableSpeakerInfo:   l.v.GetBool("VOLCENGINE_ASR_ENABLE_SPEAKER_INFO"),
				EnableEmotionDetect: l.v.GetBool("VOLCENGINE_ASR_ENABLE_EMOTION"),
				EnableGenderDetect:  l.v.GetBool("VOLCENGINE_ASR_ENABLE_GENDER"),
				EnablePunc:          l.v.GetBool("VOLCENGINE_ASR_ENABLE_PUNC"),
				EnableITN:           l.v.GetBool("VOLCENGINE_ASR_ENABLE_ITN"),
				PollIntervalSeconds: l.v.GetInt("VOLCENGINE_ASR_POLL_INTERVAL_SECONDS"),
				PollTimeoutSeconds:  l.v.GetInt("VOLCENGINE_ASR_POLL_TIMEOUT_SECONDS"),
			},
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
	backend := cfg.Storage.Backend
	if backend == "" {
		backend = "minio"
	}
	// IMPORTANT: do not hard-require OSS credentials at startup.
	// OSS credentials can be configured later via DB settings UI.
	// The storage factory will attempt OSS init and can fall back to MinIO.
	switch backend {
	case "minio":
		if cfg.MinIO.Endpoint == "" {
			return fmt.Errorf("MINIO_ENDPOINT is required")
		}
		if cfg.MinIO.AccessKey == "" {
			return fmt.Errorf("MINIO_ACCESS_KEY is required")
		}
		if cfg.MinIO.SecretKey == "" {
			return fmt.Errorf("MINIO_SECRET_KEY is required")
		}
	case "oss":
		// defer validation to runtime
	default:
		return fmt.Errorf("unsupported STORAGE_BACKEND: %s", backend)
	}
	if cfg.RabbitMQ.URL == "" {
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
