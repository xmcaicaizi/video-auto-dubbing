package config

import (
	"context"
	"database/sql"
	"fmt"

	"vedio/worker/internal/settings"

	"github.com/google/uuid"
	sharedconfig "vedio/shared/config"
)

// Manager provides runtime configuration management.
// It merges environment config, database settings, and per-task overrides.
type Manager struct {
	baseConfig     *Config
	settingsLoader *settings.Loader
}

// NewManager creates a new configuration manager.
func NewManager(baseConfig *Config, db *sql.DB) *Manager {
	return &Manager{
		baseConfig:     baseConfig,
		settingsLoader: settings.NewLoader(db),
	}
}

// GetEffectiveConfig returns the effective configuration for a task.
// Priority: task-level overrides > database settings > environment config
func (m *Manager) GetEffectiveConfig(ctx context.Context, taskID uuid.UUID) (*EffectiveConfig, error) {
	// Start with base environment configuration
	effective := &EffectiveConfig{
		Config: *m.baseConfig,
	}

	// Load database settings and merge
	dbSettings, err := m.settingsLoader.Load(ctx)
	if err == nil {
		dbSettings.MergeIntoConfig(&effective.BaseConfig)
	}

	// Load task-level overrides and merge
	taskOverrides, err := m.loadTaskOverrides(ctx, taskID)
	if err == nil {
		taskOverrides.MergeIntoConfig(&effective.BaseConfig)
	}

	return effective, nil
}

// EffectiveConfig represents the final merged configuration for a task.
type EffectiveConfig struct {
	Config
}

// ValidateForASR validates that ASR configuration is complete.
func (c *EffectiveConfig) ValidateForASR() error {
	return ValidateVolcengineASR(&c.BaseConfig)
}

// ValidateForTTS validates that TTS configuration is complete.
func (c *EffectiveConfig) ValidateForTTS() error {
	return ValidateTTSConfig(&c.BaseConfig)
}

// ValidateForTranslate validates that translation configuration is complete.
func (c *EffectiveConfig) ValidateForTranslate() error {
	return ValidateGLMConfig(&c.BaseConfig)
}

// TaskOverrides represents per-task configuration overrides.
type TaskOverrides struct {
	ASR       *TaskASRConfig
	TTS       *TaskTTSConfig
	Translate *TaskTranslateConfig
}

// TaskASRConfig holds task-level ASR overrides.
type TaskASRConfig struct {
	AppID     *string
	Token     *string
	Cluster   *string
	APIKey    *string
}

// TaskTTSConfig holds task-level TTS overrides.
type TaskTTSConfig struct {
	Backend   *string
	GradioURL *string
}

// TaskTranslateConfig holds task-level translate overrides.
type TaskTranslateConfig struct {
	GLMAPIKey *string
	GLMURL    *string
	GLMModel  *string
}

// MergeIntoConfig merges task overrides into the shared config.
func (t *TaskOverrides) MergeIntoConfig(cfg *sharedconfig.BaseConfig) {
	if t.ASR != nil {
		if t.ASR.AppID != nil {
			cfg.External.VolcengineASR.AppKey = *t.ASR.AppID
		}
		if t.ASR.Token != nil {
			cfg.External.VolcengineASR.AccessKey = *t.ASR.Token
		}
		if t.ASR.Cluster != nil {
			cfg.External.VolcengineASR.ResourceID = *t.ASR.Cluster
		}
		if t.ASR.APIKey != nil {
			// Note: The current schema has both asr_appid and asr_api_key.
			// We'll prioritize asr_api_key if set, otherwise use asr_appid
			cfg.External.VolcengineASR.AppKey = *t.ASR.APIKey
		}
	}

	if t.TTS != nil {
		if t.TTS.Backend != nil {
			cfg.TTS.Backend = *t.TTS.Backend
		}
		if t.TTS.GradioURL != nil {
			cfg.TTS.URL = *t.TTS.GradioURL
		}
	}

	if t.Translate != nil {
		if t.Translate.GLMAPIKey != nil {
			cfg.External.GLM.APIKey = *t.Translate.GLMAPIKey
		}
		if t.Translate.GLMURL != nil {
			cfg.External.GLM.APIURL = *t.Translate.GLMURL
		}
		if t.Translate.GLMModel != nil {
			cfg.External.GLM.Model = *t.Translate.GLMModel
		}
	}
}

// loadTaskOverrides loads per-task configuration from the database.
func (m *Manager) loadTaskOverrides(ctx context.Context, taskID uuid.UUID) (*TaskOverrides, error) {
	query := `
		SELECT asr_appid, asr_token, asr_cluster, asr_api_key,
		       glm_api_key, glm_api_url, glm_model,
		       tts_backend, indextts_gradio_url
		FROM tasks
		WHERE id = $1
	`

	var overrides TaskOverrides
	var asrAppID, asrToken, asrCluster, asrAPIKey sql.NullString
	var glmAPIKey, glmURL, glmModel sql.NullString
	var ttsBackend, ttsGradioURL sql.NullString

	err := m.settingsLoader.DB().QueryRowContext(ctx, query, taskID).Scan(
		&asrAppID, &asrToken, &asrCluster, &asrAPIKey,
		&glmAPIKey, &glmURL, &glmModel,
		&ttsBackend, &ttsGradioURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &overrides, nil // No overrides
		}
		return nil, fmt.Errorf("failed to load task overrides: %w", err)
	}

	// Build overrides structure
	if asrAppID.Valid || asrToken.Valid || asrCluster.Valid || asrAPIKey.Valid {
		overrides.ASR = &TaskASRConfig{}
		if asrAppID.Valid {
			overrides.ASR.AppID = &asrAppID.String
		}
		if asrToken.Valid {
			overrides.ASR.Token = &asrToken.String
		}
		if asrCluster.Valid {
			overrides.ASR.Cluster = &asrCluster.String
		}
		if asrAPIKey.Valid {
			overrides.ASR.APIKey = &asrAPIKey.String
		}
	}

	if ttsBackend.Valid || ttsGradioURL.Valid {
		overrides.TTS = &TaskTTSConfig{}
		if ttsBackend.Valid {
			overrides.TTS.Backend = &ttsBackend.String
		}
		if ttsGradioURL.Valid {
			overrides.TTS.GradioURL = &ttsGradioURL.String
		}
	}

	if glmAPIKey.Valid || glmURL.Valid || glmModel.Valid {
		overrides.Translate = &TaskTranslateConfig{}
		if glmAPIKey.Valid {
			overrides.Translate.GLMAPIKey = &glmAPIKey.String
		}
		if glmURL.Valid {
			overrides.Translate.GLMURL = &glmURL.String
		}
		if glmModel.Valid {
			overrides.Translate.GLMModel = &glmModel.String
		}
	}

	return &overrides, nil
}