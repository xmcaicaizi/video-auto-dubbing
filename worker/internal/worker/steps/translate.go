package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/models"
	"vedio/worker/internal/translate"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TranslateProcessor struct {
	deps Deps
}

func NewTranslateProcessor(deps Deps) *TranslateProcessor {
	return &TranslateProcessor{deps: deps}
}

func (p *TranslateProcessor) Name() string {
	return "translate"
}

func (p *TranslateProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.TranslatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	p.deps.Logger.Info("Processing translation",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_count", len(payload.SegmentIDs)),
		zap.String("source_language", payload.SourceLanguage),
		zap.String("target_language", payload.TargetLanguage),
	)

	// Get segments from database
	query := `SELECT idx, src_text FROM segments WHERE task_id = $1 ORDER BY idx`
	rows, err := p.deps.DB.QueryContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	type segment struct {
		idx     int
		srcText string
	}
	var segments []segment

	for rows.Next() {
		var s segment
		if err := rows.Scan(&s.idx, &s.srcText); err != nil {
			continue
		}
		segments = append(segments, s)
	}

	if len(segments) == 0 {
		return fmt.Errorf("no segments found for task")
	}

	// Prepare texts for batch translation
	texts := make([]string, len(segments))
	for i, seg := range segments {
		texts[i] = seg.srcText
	}

	// Load per-task GLM config; fallback to worker config if not set.
	var glmAPIKey, glmAPIURL, glmModel string
	q := `SELECT glm_api_key, glm_api_url, glm_model FROM tasks WHERE id = $1`
	_ = p.deps.DB.QueryRowContext(ctx, q, taskID).Scan(&glmAPIKey, &glmAPIURL, &glmModel)
	if glmAPIKey == "" {
		glmAPIKey = p.deps.Config.External.GLM.APIKey
	}
	if glmAPIURL == "" {
		glmAPIURL = p.deps.Config.External.GLM.APIURL
	}
	if glmModel == "" {
		glmModel = p.deps.Config.External.GLM.Model
	}
	transClient := translate.NewClient(config.GLMConfig{APIKey: glmAPIKey, APIURL: glmAPIURL, Model: glmModel}, p.deps.Logger)

	// Call translation API
	translations, err := transClient.Translate(ctx, texts, payload.SourceLanguage, payload.TargetLanguage)
	if err != nil {
		return fmt.Errorf("translation API call failed: %w", err)
	}

	if len(translations) != len(segments) {
		return fmt.Errorf("translation count mismatch: expected %d, got %d", len(segments), len(translations))
	}

	p.deps.Logger.Info("Translation completed",
		zap.String("task_id", taskID.String()),
		zap.Int("translated_count", len(translations)),
	)

	// Update segments with translations
	for i, seg := range segments {
		translatedText := translations[i]
		updateQuery := `UPDATE segments SET mt_text = $1, updated_at = $2 WHERE task_id = $3 AND idx = $4`
		if _, err := p.deps.DB.ExecContext(ctx, updateQuery, translatedText, time.Now(), taskID, seg.idx); err != nil {
			return fmt.Errorf("failed to update segment: %w", err)
		}
	}

	// Get segment durations for TTS target_duration_ms
	segDurations := make(map[int]int)
	durQuery := `SELECT idx, duration_ms FROM segments WHERE task_id = $1 ORDER BY idx`
	durRows, err := p.deps.DB.QueryContext(ctx, durQuery, taskID)
	if err == nil {
		defer durRows.Close()
		for durRows.Next() {
			var idx, dur int
			if err := durRows.Scan(&idx, &dur); err == nil {
				segDurations[idx] = dur
			}
		}
	}

	// Publish TTS tasks for each segment
	for i, seg := range segments {
		translatedText := translations[i]
		targetDur := segDurations[seg.idx]
		if targetDur == 0 {
			targetDur = 1500 // Default fallback
		}

		ttsMsg := map[string]interface{}{
			"task_id":    taskID.String(),
			"step":       "tts",
			"attempt":    1,
			"trace_id":   uuid.New().String(),
			"created_at": time.Now().Format(time.RFC3339),
			"payload": map[string]interface{}{
				"task_id":            taskID.String(),
				"segment_id":         fmt.Sprintf("seg-%d", seg.idx),
				"segment_idx":        seg.idx,
				"text":               translatedText,
				"target_duration_ms": targetDur,
				"speaker_id":         "default",
			},
		}

		if err := p.deps.Publisher.Publish(ctx, "task.tts", ttsMsg); err != nil {
			p.deps.Logger.Error("Failed to publish TTS task", zap.Error(err), zap.Int("segment_idx", seg.idx))
			// Continue with other segments
		}
	}

	return nil
}
