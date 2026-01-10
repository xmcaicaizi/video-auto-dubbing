package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusQueued  TaskStatus = "queued"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusFailed  TaskStatus = "failed"
	TaskStatusDone    TaskStatus = "done"
)

// Task represents a video dubbing task.
type Task struct {
	ID             uuid.UUID  `json:"task_id" db:"id"`
	Status         TaskStatus `json:"status" db:"status"`
	Progress       int        `json:"progress" db:"progress"`
	Error          *string    `json:"error,omitempty" db:"error"`
	SourceVideoKey string    `json:"-" db:"source_video_key"`
	SourceLanguage string    `json:"source_language" db:"source_language"`
	TargetLanguage string    `json:"target_language" db:"target_language"`
	// Per-task external credentials (not exposed via JSON)
	ASRAppID        *string `json:"-" db:"asr_appid"`
	ASRToken        *string `json:"-" db:"asr_token"`
	ASRCluster      *string `json:"-" db:"asr_cluster"`
	ASRAPIKey       *string `json:"-" db:"asr_api_key"`
	GLMAPIKey       *string `json:"-" db:"glm_api_key"`
	GLMAPIURL       *string `json:"-" db:"glm_api_url"`
	GLMModel        *string `json:"-" db:"glm_model"`
	// Per-task TTS configuration (not exposed via JSON)
	TTSBackend        *string `json:"-" db:"tts_backend"`
	IndexTTSGradioURL *string `json:"-" db:"indextts_gradio_url"`
	OutputVideoKey *string   `json:"-" db:"output_video_key"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// TaskStepStatus represents the status of a task step.
type TaskStepStatus string

const (
	TaskStepStatusPending   TaskStepStatus = "pending"
	TaskStepStatusRunning   TaskStepStatus = "running"
	TaskStepStatusSucceeded TaskStepStatus = "succeeded"
	TaskStepStatusFailed    TaskStepStatus = "failed"
)

// TaskStep represents a step in a task.
type TaskStep struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	TaskID     uuid.UUID      `json:"task_id" db:"task_id"`
	Step       string         `json:"step" db:"step"`
	Status     TaskStepStatus `json:"status" db:"status"`
	Attempt    int            `json:"attempt" db:"attempt"`
	StartedAt  *time.Time     `json:"started_at,omitempty" db:"started_at"`
	EndedAt    *time.Time     `json:"ended_at,omitempty" db:"ended_at"`
	Error      *string        `json:"error,omitempty" db:"error"`
	MetricsJSON *string       `json:"metrics_json,omitempty" db:"metrics_json"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
}

// Segment represents a video segment.
type Segment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TaskID       uuid.UUID `json:"task_id" db:"task_id"`
	Idx          int       `json:"idx" db:"idx"`
	StartMs      int       `json:"start_ms" db:"start_ms"`
	EndMs        int       `json:"end_ms" db:"end_ms"`
	DurationMs   int       `json:"duration_ms" db:"duration_ms"`
	SrcText      string    `json:"src_text" db:"src_text"`
	MtText       *string   `json:"mt_text,omitempty" db:"mt_text"`
	TtsParamsJSON *string  `json:"tts_params_json,omitempty" db:"tts_params_json"`
	TtsAudioKey  *string   `json:"tts_audio_key,omitempty" db:"tts_audio_key"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

