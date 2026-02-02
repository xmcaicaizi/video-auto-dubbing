package config

import (
	"fmt"
	"time"

	sharedconfig "vedio/shared/config"
)

// Aliases for shared configuration structures to keep existing references intact.
type DatabaseConfig = sharedconfig.DatabaseConfig
type MinIOConfig = sharedconfig.MinIOConfig
type RabbitMQConfig = sharedconfig.RabbitMQConfig
type TTSConfig = sharedconfig.TTSConfig
type ExternalConfig = sharedconfig.ExternalConfig
type VolcengineASRConfig = sharedconfig.VolcengineASRConfig
type AliyunASRConfig = sharedconfig.AliyunASRConfig
type GLMConfig = sharedconfig.GLMConfig

// Config holds all configuration for the worker.
type Config struct {
	sharedconfig.BaseConfig
	FFmpeg     FFmpegConfig
	Processing ProcessingConfig
	Timeouts   StepTimeouts
	ASR        ASRConfig
}

// ASRConfig holds ASR service configuration.
type ASRConfig struct {
	Backend string // ASR backend: "volcengine" (default), "aliyun"
}

// FFmpegConfig holds FFmpeg configuration.
type FFmpegConfig struct {
	Path          string
	DenoiseFilter string
}

// ProcessingConfig tunes batching, concurrency and retries for heavy steps.
type ProcessingConfig struct {
	Translate TranslateConfig
	TTS       TTSProcessingConfig
}

// TranslateConfig controls translate step batching and retries.
type TranslateConfig struct {
	BatchSize      int
	ItemMaxRetries int
	MaxTextLength  int
}

// TTSProcessingConfig controls TTS step batching, concurrency and retries.
type TTSProcessingConfig struct {
	BatchSize      int
	MaxConcurrency int
	MaxRetries     int
	RetryDelay     time.Duration
}

// StepTimeouts contains per-step timeout configuration.
type StepTimeouts struct {
	ExtractAudio time.Duration
	ASR          time.Duration
	TTS          time.Duration
	Mux          time.Duration
}

// Load loads configuration from environment variables.
// Note: ASR/TTS/GLM settings can also be loaded from database at runtime.
func Load() (*Config, error) {
	loader := sharedconfig.NewLoader(
		// No longer require Volcengine ASR at startup - can be loaded from DB
		sharedconfig.WithMinIOPublicFallback(),
	)

	v := loader.Viper()
	v.SetDefault("FFMPEG_PATH", "/usr/bin/ffmpeg")
	v.SetDefault("FFMPEG_DENOISE_FILTER", "afftdn=nr=12:nf=-25")
	v.SetDefault("ASR_BACKEND", "volcengine")
	v.SetDefault("TRANSLATE_BATCH_SIZE", 20)
	v.SetDefault("TRANSLATE_ITEM_MAX_RETRIES", 2)
	v.SetDefault("TRANSLATE_MAX_TEXT_LENGTH", 4000)
	v.SetDefault("TTS_BATCH_SIZE", 20)
	v.SetDefault("TTS_MAX_CONCURRENCY", 4)
	v.SetDefault("TTS_MAX_RETRIES", 3)
	v.SetDefault("TTS_RETRY_DELAY_SECONDS", 2.0)
	v.SetDefault("TIMEOUT_EXTRACT_AUDIO_SECONDS", 600)
	v.SetDefault("TIMEOUT_ASR_SECONDS", 900)
	v.SetDefault("TIMEOUT_TTS_SECONDS", 900)
	v.SetDefault("TIMEOUT_MUX_SECONDS", 900)

	baseCfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	cfg := &Config{
		BaseConfig: *baseCfg,
		FFmpeg: FFmpegConfig{
			Path:          v.GetString("FFMPEG_PATH"),
			DenoiseFilter: v.GetString("FFMPEG_DENOISE_FILTER"),
		},
		ASR: ASRConfig{
			Backend: v.GetString("ASR_BACKEND"),
		},
		Processing: ProcessingConfig{
			Translate: TranslateConfig{
				BatchSize:      v.GetInt("TRANSLATE_BATCH_SIZE"),
				ItemMaxRetries: v.GetInt("TRANSLATE_ITEM_MAX_RETRIES"),
				MaxTextLength:  v.GetInt("TRANSLATE_MAX_TEXT_LENGTH"),
			},
			TTS: TTSProcessingConfig{
				BatchSize:      v.GetInt("TTS_BATCH_SIZE"),
				MaxConcurrency: v.GetInt("TTS_MAX_CONCURRENCY"),
				MaxRetries:     v.GetInt("TTS_MAX_RETRIES"),
				RetryDelay:     time.Duration(v.GetFloat64("TTS_RETRY_DELAY_SECONDS") * float64(time.Second)),
			},
		},
		Timeouts: StepTimeouts{
			ExtractAudio: time.Duration(v.GetInt("TIMEOUT_EXTRACT_AUDIO_SECONDS")) * time.Second,
			ASR:          time.Duration(v.GetInt("TIMEOUT_ASR_SECONDS")) * time.Second,
			TTS:          time.Duration(v.GetInt("TIMEOUT_TTS_SECONDS")) * time.Second,
			Mux:          time.Duration(v.GetInt("TIMEOUT_MUX_SECONDS")) * time.Second,
		},
	}

	return cfg, nil
}

// ValidateVolcengineASR validates that Volcengine ASR credentials are configured.
// This is used at runtime when ASR is needed, not at startup.
func ValidateVolcengineASR(cfg *sharedconfig.BaseConfig) error {
	if cfg.External.VolcengineASR.AppKey == "" {
		return fmt.Errorf("VOLCENGINE_ASR_APP_KEY is required (configure via settings or environment)")
	}
	if cfg.External.VolcengineASR.AccessKey == "" {
		return fmt.Errorf("VOLCENGINE_ASR_ACCESS_KEY is required (configure via settings or environment)")
	}
	return nil
}

// ValidateAliyunASR validates that Aliyun ASR credentials are configured.
// This is used at runtime when ASR is needed, not at startup.
func ValidateAliyunASR(cfg *sharedconfig.BaseConfig) error {
	if cfg.External.AliyunASR.APIKey == "" {
		return fmt.Errorf("ALIYUN_ASR_API_KEY is required (configure via settings or environment)")
	}
	return nil
}

// ValidateASRBackend validates ASR configuration based on the selected backend.
func ValidateASRBackend(cfg *Config) error {
	switch cfg.ASR.Backend {
	case "volcengine":
		return ValidateVolcengineASR(&cfg.BaseConfig)
	case "aliyun":
		return ValidateAliyunASR(&cfg.BaseConfig)
	default:
		return fmt.Errorf("unsupported ASR backend: %s (supported: volcengine, aliyun)", cfg.ASR.Backend)
	}
}

// ValidateTTSConfig validates that TTS service is configured.
func ValidateTTSConfig(cfg *sharedconfig.BaseConfig) error {
	if cfg.TTS.URL == "" {
		return fmt.Errorf("TTS_SERVICE_URL is required (configure via settings or environment)")
	}
	return nil
}

// ValidateGLMConfig validates that GLM API is configured.
func ValidateGLMConfig(cfg *sharedconfig.BaseConfig) error {
	if cfg.External.GLM.APIKey == "" {
		return fmt.Errorf("GLM_API_KEY is required (configure via settings or environment)")
	}
	return nil
}

// ValidateDashScopeConfig validates that DashScope LLM API is configured.
func ValidateDashScopeConfig(cfg *sharedconfig.BaseConfig) error {
	if cfg.External.DashScope.APIKey == "" {
		return fmt.Errorf("DASHSCOPE_LLM_API_KEY is required (configure via settings or environment)")
	}
	return nil
}
