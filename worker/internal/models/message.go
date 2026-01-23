package models

// TaskMessage represents a task message from RabbitMQ.
type TaskMessage struct {
	TaskID    string                 `json:"task_id"`
	Step      string                 `json:"step"`
	Attempt   int                    `json:"attempt"`
	TraceID   string                 `json:"trace_id"`
	CreatedAt string                 `json:"created_at"`
	Payload   map[string]interface{} `json:"payload"`
}

// ExtractAudioPayload represents the payload for extract_audio step.
type ExtractAudioPayload struct {
	SourceVideoKey string `json:"source_video_key"`
	OutputAudioKey string `json:"output_audio_key"`
}

// ASRPayload represents the payload for asr step.
type ASRPayload struct {
	AudioKey  string `json:"audio_key"`
	Language  string `json:"language"`
	OutputKey string `json:"output_key"`
}

// TranslatePayload represents the payload for translate step.
type TranslatePayload struct {
	TaskID         string   `json:"task_id"`
	SegmentIDs     []string `json:"segment_ids"`
	SourceLanguage string   `json:"source_language"`
	TargetLanguage string   `json:"target_language"`
	BatchSize      int      `json:"batch_size,omitempty"`
}

// TTSPayload represents the payload for tts step.
type TTSPayload struct {
	TaskID           string                 `json:"task_id"`
	SegmentID        string                 `json:"segment_id"`
	SegmentIdx       int                    `json:"segment_idx"`
	Text             string                 `json:"text"`
	TargetDurationMs int                    `json:"target_duration_ms"`
	SpeakerID        string                 `json:"speaker_id"`
	ProsodyControl   map[string]interface{} `json:"prosody_control"`
	BatchSize        int                    `json:"batch_size,omitempty"`
	MaxConcurrency   int                    `json:"max_concurrency,omitempty"`
	MaxRetries       int                    `json:"max_retries,omitempty"`
	RetryDelaySec    float64                `json:"retry_delay_sec,omitempty"`
}

// MuxVideoPayload represents the payload for mux_video step.
type MuxVideoPayload struct {
	TaskID         string `json:"task_id"`
	SourceVideoKey string `json:"source_video_key"`
	TTSAudioKey    string `json:"tts_audio_key"`
	OutputVideoKey string `json:"output_video_key"`
}

// ASRResult represents the ASR recognition result.
type ASRResult struct {
	Segments   []ASRSegment `json:"segments"`
	Language   string       `json:"language"`
	DurationMs int          `json:"duration_ms"`
}

// ASRSegment represents a single ASR segment.
type ASRSegment struct {
	Idx       int    `json:"idx"`
	StartMs   int    `json:"start_ms"`
	EndMs     int    `json:"end_ms"`
	Text      string `json:"text"`
	SpeakerID string `json:"speaker_id,omitempty"` // 说话人标识
	Emotion   string `json:"emotion,omitempty"`    // 情绪: angry, happy, neutral, sad, surprise
	Gender    string `json:"gender,omitempty"`     // 性别: male, female
}
