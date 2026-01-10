package database

import (
	"database/sql"
	"fmt"
)

// Migrate runs database migrations.
func Migrate(db *sql.DB) error {
	migrations := []string{
		createExtensions,
		createTasksTable,
		alterTasksTableAddExternalKeys,
		alterTasksTableAddTTSConfig,
		createTaskStepsTable,
		createSegmentsTable,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

const createExtensions = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
`

const createTasksTable = `
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(20) NOT NULL DEFAULT 'created',
    progress INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    source_video_key VARCHAR(255) NOT NULL,
    source_language VARCHAR(10) NOT NULL DEFAULT 'zh',
    target_language VARCHAR(10) NOT NULL DEFAULT 'en',
    -- Per-task external credentials (MVP; consider encrypting at rest in production)
    asr_appid TEXT,
    asr_token TEXT,
    asr_cluster TEXT,
    glm_api_key TEXT,
    glm_api_url TEXT,
    glm_model TEXT,
    -- Per-task TTS configuration (optional override)
    tts_backend TEXT,
    indextts_gradio_url TEXT,
    output_video_key VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
`

const alterTasksTableAddExternalKeys = `
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS asr_appid TEXT,
    ADD COLUMN IF NOT EXISTS asr_token TEXT,
    ADD COLUMN IF NOT EXISTS asr_cluster TEXT,
    ADD COLUMN IF NOT EXISTS asr_api_key TEXT,
    ADD COLUMN IF NOT EXISTS glm_api_key TEXT,
    ADD COLUMN IF NOT EXISTS glm_api_url TEXT,
    ADD COLUMN IF NOT EXISTS glm_model TEXT;
`

const alterTasksTableAddTTSConfig = `
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS tts_backend TEXT,
    ADD COLUMN IF NOT EXISTS indextts_gradio_url TEXT;
`

const createTaskStepsTable = `
CREATE TABLE IF NOT EXISTS task_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    step VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempt INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    error TEXT,
    metrics_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id, step, attempt)
);

CREATE INDEX IF NOT EXISTS idx_task_steps_task_id ON task_steps(task_id);
CREATE INDEX IF NOT EXISTS idx_task_steps_status ON task_steps(status);
CREATE INDEX IF NOT EXISTS idx_task_steps_step ON task_steps(step);
`

const createSegmentsTable = `
CREATE TABLE IF NOT EXISTS segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    idx INTEGER NOT NULL,
    start_ms INTEGER NOT NULL,
    end_ms INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    src_text TEXT NOT NULL,
    mt_text TEXT,
    tts_params_json JSONB,
    tts_audio_key VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id, idx)
);

CREATE INDEX IF NOT EXISTS idx_segments_task_id ON segments(task_id);
CREATE INDEX IF NOT EXISTS idx_segments_task_id_idx ON segments(task_id, idx);
`

