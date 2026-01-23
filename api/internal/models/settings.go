package models

import "time"

// SettingRecord represents a single setting record in the database.
type SettingRecord struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	IsEncrypted bool      `json:"is_encrypted"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Settings represents all application settings grouped by category.
type Settings struct {
	ASR       ASRSettings       `json:"asr"`
	TTS       TTSSettings       `json:"tts"`
	Translate TranslateSettings `json:"translate"`
	Storage   StorageSettings   `json:"storage"`
}

// StorageSettings holds object storage configuration.
type StorageSettings struct {
	Backend string      `json:"backend"` // "oss" or "minio"
	OSS     OSSSettings `json:"oss"`
}

// OSSSettings holds Aliyun OSS configuration.
type OSSSettings struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	PublicDomain    string `json:"public_domain"` // CNAME domain
	Prefix          string `json:"prefix"`
	UseSSL          bool   `json:"use_ssl"`
}

// ASRSettings holds Volcengine ASR configuration.
type ASRSettings struct {
	VolcengineAppKey     string `json:"volcengine_app_key"`
	VolcengineAccessKey  string `json:"volcengine_access_key"`
	VolcengineResourceID string `json:"volcengine_resource_id"`
	EnableSpeakerInfo    bool   `json:"enable_speaker_info"`
	EnableEmotion        bool   `json:"enable_emotion"`
	EnableGender         bool   `json:"enable_gender"`
	EnablePunc           bool   `json:"enable_punc"`
	EnableITN            bool   `json:"enable_itn"`
}

// TTSSettings holds TTS service configuration.
type TTSSettings struct {
	ServiceURL string `json:"service_url"`
	APIKey     string `json:"api_key"`
	Backend    string `json:"backend"` // "vllm" or "legacy"
}

// TranslateSettings holds translation (GLM) configuration.
type TranslateSettings struct {
	GLMAPIKey string `json:"glm_api_key"`
	GLMAPIURL string `json:"glm_api_url"`
	GLMModel  string `json:"glm_model"`
}

// SettingsUpdateRequest represents a request to update settings.
type SettingsUpdateRequest struct {
	ASR       *ASRSettings       `json:"asr,omitempty"`
	TTS       *TTSSettings       `json:"tts,omitempty"`
	Translate *TranslateSettings `json:"translate,omitempty"`
	Storage   *StorageSettings   `json:"storage,omitempty"`
}

// TestConnectionRequest represents a request to test a service connection.
type TestConnectionRequest struct {
	Type string `json:"type" binding:"required"` // "asr", "tts", or "translate"
}

// TestConnectionResponse represents the result of a connection test.
type TestConnectionResponse struct {
	Status    string `json:"status"`     // "connected", "failed"
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// MaskSensitive masks sensitive values for display.
func (s *Settings) MaskSensitive() *Settings {
	masked := &Settings{
		ASR: ASRSettings{
			VolcengineAppKey:     maskValue(s.ASR.VolcengineAppKey),
			VolcengineAccessKey:  maskValue(s.ASR.VolcengineAccessKey),
			VolcengineResourceID: s.ASR.VolcengineResourceID,
			EnableSpeakerInfo:    s.ASR.EnableSpeakerInfo,
			EnableEmotion:        s.ASR.EnableEmotion,
			EnableGender:         s.ASR.EnableGender,
			EnablePunc:           s.ASR.EnablePunc,
			EnableITN:            s.ASR.EnableITN,
		},
		TTS: TTSSettings{
			ServiceURL: s.TTS.ServiceURL,
			APIKey:     maskValue(s.TTS.APIKey),
			Backend:    s.TTS.Backend,
		},
		Translate: TranslateSettings{
			GLMAPIKey: maskValue(s.Translate.GLMAPIKey),
			GLMAPIURL: s.Translate.GLMAPIURL,
			GLMModel:  s.Translate.GLMModel,
		},
		Storage: StorageSettings{
			Backend: s.Storage.Backend,
			OSS: OSSSettings{
				Endpoint:        s.Storage.OSS.Endpoint,
				Bucket:          s.Storage.OSS.Bucket,
				AccessKeyID:     maskValue(s.Storage.OSS.AccessKeyID),
				AccessKeySecret: maskValue(s.Storage.OSS.AccessKeySecret),
				PublicDomain:    s.Storage.OSS.PublicDomain,
				Prefix:          s.Storage.OSS.Prefix,
				UseSSL:          s.Storage.OSS.UseSSL,
			},
		},
	}
	return masked
}

// maskValue masks a sensitive value, showing only first and last 3 characters.
func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "***"
	}
	return value[:3] + "***" + value[len(value)-3:]
}

// HasASRConfig returns true if ASR settings are configured.
func (s *Settings) HasASRConfig() bool {
	return s.ASR.VolcengineAppKey != "" && s.ASR.VolcengineAccessKey != ""
}

// HasTTSConfig returns true if TTS settings are configured.
func (s *Settings) HasTTSConfig() bool {
	return s.TTS.ServiceURL != ""
}

// HasTranslateConfig returns true if translation settings are configured.
func (s *Settings) HasTranslateConfig() bool {
	return s.Translate.GLMAPIKey != ""
}

// HasStorageConfig returns true if storage settings are configured.
func (s *Settings) HasStorageConfig() bool {
	if s.Storage.Backend == "oss" {
		return s.Storage.OSS.Endpoint != "" &&
			s.Storage.OSS.Bucket != "" &&
			s.Storage.OSS.PublicDomain != "" &&
			s.Storage.OSS.AccessKeyID != "" &&
			s.Storage.OSS.AccessKeySecret != ""
	}
	return true
}
