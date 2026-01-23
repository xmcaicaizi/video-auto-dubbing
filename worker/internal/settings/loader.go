// Package settings provides database-backed configuration loading for worker.
package settings

import (
	"context"
	"database/sql"
	"fmt"

	sharedconfig "vedio/shared/config"
)

// Loader loads settings from the database and merges with environment config.
type Loader struct {
	db *sql.DB
}

// DB returns the database connection (for config manager access).
func (l *Loader) DB() *sql.DB {
	return l.db
}

// NewLoader creates a new settings loader.
func NewLoader(db *sql.DB) *Loader {
	return &Loader{db: db}
}

// Settings represents the loaded settings from database.
type Settings struct {
	ASR       ASRSettings
	TTS       TTSSettings
	Translate TranslateSettings
	Storage   StorageSettings
}

// StorageSettings holds object storage configuration from database.
type StorageSettings struct {
	Backend           string
	OSSEndpoint       string
	OSSBucket         string
	OSSAccessKeyID    string
	OSSAccessKeySecret string
	OSSPublicDomain   string
	OSSPrefix         string
	OSSUseSSL         bool
}

// ASRSettings holds ASR configuration from database.
type ASRSettings struct {
	VolcengineAppKey     string
	VolcengineAccessKey  string
	VolcengineResourceID string
	EnableSpeakerInfo    bool
	EnableEmotion        bool
	EnableGender         bool
	EnablePunc           bool
	EnableITN            bool
}

// TTSSettings holds TTS configuration from database.
type TTSSettings struct {
	ServiceURL string
	APIKey     string
	Backend    string
}

// TranslateSettings holds translation configuration from database.
type TranslateSettings struct {
	GLMAPIKey string
	GLMAPIURL string
	GLMModel  string
}

// Load reads all settings from the database.
func (l *Loader) Load(ctx context.Context) (*Settings, error) {
	query := `SELECT category, key, value FROM settings`
	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := &Settings{
		ASR: ASRSettings{
			VolcengineResourceID: "volc.bigasr.auc",
			EnableSpeakerInfo:    true,
			EnableEmotion:        true,
			EnableGender:         true,
			EnablePunc:           true,
			EnableITN:            true,
		},
		TTS: TTSSettings{
			Backend: "vllm",
		},
		Translate: TranslateSettings{
			GLMAPIURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions",
			GLMModel:  "glm-4-flash",
		},
		Storage: StorageSettings{
			Backend:   "minio",
			OSSUseSSL: true,
		},
	}

	for rows.Next() {
		var category, key, value string
		if err := rows.Scan(&category, &key, &value); err != nil {
			continue
		}
		applyValue(settings, category, key, value)
	}

	return settings, nil
}

// applyValue applies a single setting value.
func applyValue(s *Settings, category, key, value string) {
	switch category {
	case "storage":
		switch key {
		case "backend":
			s.Storage.Backend = value
		case "oss_endpoint":
			s.Storage.OSSEndpoint = value
		case "oss_bucket":
			s.Storage.OSSBucket = value
		case "oss_access_key_id":
			s.Storage.OSSAccessKeyID = value
		case "oss_access_key_secret":
			s.Storage.OSSAccessKeySecret = value
		case "oss_public_domain":
			s.Storage.OSSPublicDomain = value
		case "oss_prefix":
			s.Storage.OSSPrefix = value
		case "oss_use_ssl":
			s.Storage.OSSUseSSL = value == "true"
		}
	case "asr":
		switch key {
		case "volcengine_app_key":
			s.ASR.VolcengineAppKey = value
		case "volcengine_access_key":
			s.ASR.VolcengineAccessKey = value
		case "volcengine_resource_id":
			s.ASR.VolcengineResourceID = value
		case "enable_speaker_info":
			s.ASR.EnableSpeakerInfo = value == "true"
		case "enable_emotion":
			s.ASR.EnableEmotion = value == "true"
		case "enable_gender":
			s.ASR.EnableGender = value == "true"
		case "enable_punc":
			s.ASR.EnablePunc = value == "true"
		case "enable_itn":
			s.ASR.EnableITN = value == "true"
		}
	case "tts":
		switch key {
		case "service_url":
			s.TTS.ServiceURL = value
		case "api_key":
			s.TTS.APIKey = value
		case "backend":
			s.TTS.Backend = value
		}
	case "translate":
		switch key {
		case "glm_api_key":
			s.Translate.GLMAPIKey = value
		case "glm_api_url":
			s.Translate.GLMAPIURL = value
		case "glm_model":
			s.Translate.GLMModel = value
		}
	}
}

// MergeIntoConfig merges database settings into the shared config.
// Database settings take precedence over environment variables.
func (s *Settings) MergeIntoConfig(cfg *sharedconfig.BaseConfig) {
	// Merge Storage settings
	if s.Storage.Backend != "" {
		cfg.Storage.Backend = s.Storage.Backend
	}
	// OSS settings
	if s.Storage.OSSEndpoint != "" {
		cfg.OSS.Endpoint = s.Storage.OSSEndpoint
	}
	if s.Storage.OSSBucket != "" {
		cfg.OSS.Bucket = s.Storage.OSSBucket
	}
	if s.Storage.OSSAccessKeyID != "" {
		cfg.OSS.AccessKeyID = s.Storage.OSSAccessKeyID
	}
	if s.Storage.OSSAccessKeySecret != "" {
		cfg.OSS.AccessKeySecret = s.Storage.OSSAccessKeySecret
	}
	if s.Storage.OSSPublicDomain != "" {
		cfg.OSS.PublicDomain = s.Storage.OSSPublicDomain
	}
	if s.Storage.OSSPrefix != "" {
		cfg.OSS.Prefix = s.Storage.OSSPrefix
	}
	cfg.OSS.UseSSL = s.Storage.OSSUseSSL

	// Merge ASR settings (only if set in database)
	if s.ASR.VolcengineAppKey != "" {
		cfg.External.VolcengineASR.AppKey = s.ASR.VolcengineAppKey
	}
	if s.ASR.VolcengineAccessKey != "" {
		cfg.External.VolcengineASR.AccessKey = s.ASR.VolcengineAccessKey
	}
	if s.ASR.VolcengineResourceID != "" {
		cfg.External.VolcengineASR.ResourceID = s.ASR.VolcengineResourceID
	}
	cfg.External.VolcengineASR.EnableSpeakerInfo = s.ASR.EnableSpeakerInfo
	cfg.External.VolcengineASR.EnableEmotionDetect = s.ASR.EnableEmotion
	cfg.External.VolcengineASR.EnableGenderDetect = s.ASR.EnableGender
	cfg.External.VolcengineASR.EnablePunc = s.ASR.EnablePunc
	cfg.External.VolcengineASR.EnableITN = s.ASR.EnableITN

	// Merge TTS settings
	if s.TTS.ServiceURL != "" {
		cfg.TTS.URL = s.TTS.ServiceURL
	}
	if s.TTS.APIKey != "" {
		cfg.TTS.APIKey = s.TTS.APIKey
	}
	if s.TTS.Backend != "" {
		cfg.TTS.Backend = s.TTS.Backend
	}

	// Merge Translate settings
	if s.Translate.GLMAPIKey != "" {
		cfg.External.GLM.APIKey = s.Translate.GLMAPIKey
	}
	if s.Translate.GLMAPIURL != "" {
		cfg.External.GLM.APIURL = s.Translate.GLMAPIURL
	}
	if s.Translate.GLMModel != "" {
		cfg.External.GLM.Model = s.Translate.GLMModel
	}
}

// HasValidASRConfig returns true if ASR configuration is complete.
func (s *Settings) HasValidASRConfig() bool {
	return s.ASR.VolcengineAppKey != "" && s.ASR.VolcengineAccessKey != ""
}

// HasValidTTSConfig returns true if TTS configuration is complete.
func (s *Settings) HasValidTTSConfig() bool {
	return s.TTS.ServiceURL != ""
}

// HasValidTranslateConfig returns true if translation configuration is complete.
func (s *Settings) HasValidTranslateConfig() bool {
	return s.Translate.GLMAPIKey != ""
}

// HasValidStorageConfig returns true if storage config is complete.
func (s *Settings) HasValidStorageConfig() bool {
	if s.Storage.Backend == "oss" {
		return s.Storage.OSSEndpoint != "" && s.Storage.OSSBucket != "" && s.Storage.OSSPublicDomain != "" && s.Storage.OSSAccessKeyID != "" && s.Storage.OSSAccessKeySecret != ""
	}
	return true
}
