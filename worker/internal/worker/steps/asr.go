package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vedio/worker/internal/asr"
	"vedio/worker/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ASRProcessor struct {
	deps Deps
}

func NewASRProcessor(deps Deps) *ASRProcessor {
	return &ASRProcessor{deps: deps}
}

func (p *ASRProcessor) Name() string {
	return "asr"
}

func (p *ASRProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.ASRPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	p.deps.Logger.Info("Processing ASR",
		zap.String("task_id", taskID.String()),
		zap.String("audio_key", payload.AudioKey),
		zap.String("language", payload.Language),
	)

	// Get effective configuration for this task
	effectiveConfig, err := p.deps.ConfigManager.GetEffectiveConfig(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get effective config: %w", err)
	}

	// Validate ASR configuration
	if err := effectiveConfig.ValidateForASR(); err != nil {
		return fmt.Errorf("ASR configuration validation failed: %w", err)
	}

	// Create ASR client with effective configuration
	asrClient := asr.NewClient(effectiveConfig.External.VolcengineASR, p.deps.Logger)

	// Generate presigned URL for audio (ASR service needs to download it)
	audioURL, err := p.deps.Storage.PresignedGetURL(ctx, payload.AudioKey, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate presigned URL for audio: %w", err)
	}

	p.deps.Logger.Info("Generated presigned audio URL for ASR",
		zap.String("task_id", taskID.String()),
		zap.String("audio_url", audioURL),
	)

	// Call ASR service (Volcengine)
	asrResult, err := asrClient.Recognize(ctx, audioURL, payload.Language)
	if err != nil {
		return fmt.Errorf("ASR service call failed: %w", err)
	}

	p.deps.Logger.Info("ASR completed",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_count", len(asrResult.Segments)),
		zap.Int("duration_ms", asrResult.DurationMs),
	)

	// Save ASR result to MinIO
	resultJSON, _ := json.Marshal(asrResult)
	resultReader := bytes.NewReader(resultJSON)
	if err := p.deps.Storage.PutObject(ctx, payload.OutputKey, resultReader, int64(len(resultJSON)), "application/json"); err != nil {
		return fmt.Errorf("failed to save ASR result: %w", err)
	}

	// Save segments to database (including speaker_id, emotion, gender)
	for _, seg := range asrResult.Segments {
		// 🔥 设置默认speaker_id以启用音色克隆
		speakerID := seg.SpeakerID
		if speakerID == "" {
			speakerID = "speaker_1" // 默认说话人，将触发音色克隆
		}

		query := `
			INSERT INTO segments (task_id, idx, start_ms, end_ms, duration_ms, src_text,
			                      speaker_id, emotion, gender, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (task_id, idx) DO UPDATE
			SET start_ms = EXCLUDED.start_ms, end_ms = EXCLUDED.end_ms,
			    duration_ms = EXCLUDED.duration_ms, src_text = EXCLUDED.src_text,
			    speaker_id = EXCLUDED.speaker_id, emotion = EXCLUDED.emotion,
			    gender = EXCLUDED.gender, updated_at = EXCLUDED.updated_at
		`
		now := time.Now()
		_, err := p.deps.DB.ExecContext(ctx, query,
			taskID, seg.Idx, seg.StartMs, seg.EndMs, seg.EndMs-seg.StartMs,
			seg.Text, speakerID, seg.Emotion, seg.Gender, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to save segment: %w", err)
		}
	}

	// Get task info to get target language
	var sourceLang, targetLang string
	query := `SELECT source_language, target_language FROM tasks WHERE id = $1`
	if err := p.deps.DB.QueryRowContext(ctx, query, taskID).Scan(&sourceLang, &targetLang); err != nil {
		return fmt.Errorf("failed to get task languages: %w", err)
	}

	// Publish translate task
	// Get all segment IDs
	var segmentIDs []string
	rows, err := p.deps.DB.QueryContext(ctx, "SELECT idx FROM segments WHERE task_id = $1 ORDER BY idx", taskID)
	if err != nil {
		return fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var idx int
		if err := rows.Scan(&idx); err != nil {
			continue
		}
		segmentIDs = append(segmentIDs, fmt.Sprintf("seg-%d", idx))
	}

	translateMsg := map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "translate",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"task_id":         taskID.String(),
			"segment_ids":     segmentIDs,
			"source_language": sourceLang,
			"target_language": targetLang,
		},
	}

	if err := p.deps.Publisher.Publish(ctx, "task.translate", translateMsg); err != nil {
		return fmt.Errorf("failed to publish translate task: %w", err)
	}

	return nil
}
