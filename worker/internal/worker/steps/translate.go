package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
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
		zap.Int("batch_size", p.translateBatchSize(payload)),
	)

	segments, err := p.loadSegments(ctx, taskID)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return fmt.Errorf("no segments to translate for task")
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

	if err := p.translateBatches(ctx, taskID, payload, segments, transClient); err != nil {
		return err
	}

	ttsMsg := map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "tts",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"task_id":          taskID.String(),
			"batch_size":       p.deps.Config.Processing.TTS.BatchSize,
			"max_concurrency":  p.deps.Config.Processing.TTS.MaxConcurrency,
			"max_retries":      p.deps.Config.Processing.TTS.MaxRetries,
			"retry_delay_sec":  p.deps.Config.Processing.TTS.RetryDelay.Seconds(),
			"source_language":  payload.SourceLanguage,
			"target_language":  payload.TargetLanguage,
			"segment_ids":      payload.SegmentIDs,
			"translated_count": len(segments),
		},
	}

	if err := p.deps.Publisher.Publish(ctx, "task.tts", ttsMsg); err != nil {
		return fmt.Errorf("failed to publish TTS task: %w", err)
	}

	return nil
}

type translateSegment struct {
	idx        int
	srcText    string
	mtText     string
	durationMs int
}

func (p *TranslateProcessor) loadSegments(ctx context.Context, taskID uuid.UUID) ([]translateSegment, error) {
	query := `SELECT idx, src_text, COALESCE(mt_text, ''), duration_ms FROM segments WHERE task_id = $1 ORDER BY idx`
	rows, err := p.deps.DB.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	var segments []translateSegment
	for rows.Next() {
		var s translateSegment
		if err := rows.Scan(&s.idx, &s.srcText, &s.mtText, &s.durationMs); err != nil {
			continue
		}
		if strings.TrimSpace(s.mtText) != "" {
			continue
		}
		segments = append(segments, s)
	}
	return segments, nil
}

func (p *TranslateProcessor) translateBatchSize(payload models.TranslatePayload) int {
	if payload.BatchSize > 0 {
		return payload.BatchSize
	}
	if p.deps.Config.Processing.Translate.BatchSize > 0 {
		return p.deps.Config.Processing.Translate.BatchSize
	}
	return 20
}

func (p *TranslateProcessor) translateBatches(ctx context.Context, taskID uuid.UUID, payload models.TranslatePayload, segments []translateSegment, client *translate.Client) error {
	batchSize := p.translateBatchSize(payload)
	itemRetries := p.deps.Config.Processing.Translate.ItemMaxRetries
	if itemRetries <= 0 {
		itemRetries = 2
	}
	maxTextLength := p.deps.Config.Processing.Translate.MaxTextLength
	if maxTextLength <= 0 {
		maxTextLength = 4000
	}

	for start := 0; start < len(segments); start += batchSize {
		end := int(math.Min(float64(len(segments)), float64(start+batchSize)))
		batch := segments[start:end]

		texts := make([]string, len(batch))
		for i, seg := range batch {
			texts[i] = seg.srcText
		}

		translations, err := client.Translate(ctx, texts, payload.SourceLanguage, payload.TargetLanguage)
		if err != nil || len(translations) != len(batch) {
			p.deps.Logger.Warn("Batch translation failed, falling back to per-segment retry", zap.Error(err), zap.Int("batch_start", start), zap.Int("batch_end", end))
			for i, seg := range batch {
				translation, singleErr := p.translateSingleWithRetry(ctx, client, payload, seg.srcText, itemRetries, maxTextLength)
				if singleErr != nil {
					return fmt.Errorf("failed to translate segment %d: %w", seg.idx, singleErr)
				}
				if err := p.updateSegmentTranslation(ctx, taskID, seg.idx, translation); err != nil {
					return err
				}
				batch[i].mtText = translation
			}
			continue
		}

		for i, seg := range batch {
			translatedText := translations[i]
			if err := p.updateSegmentTranslation(ctx, taskID, seg.idx, translatedText); err != nil {
				return err
			}
			batch[i].mtText = translatedText
		}
	}

	p.deps.Logger.Info("Translation completed", zap.String("task_id", taskID.String()), zap.Int("translated_count", len(segments)))
	return nil
}

func (p *TranslateProcessor) translateSingleWithRetry(ctx context.Context, client *translate.Client, payload models.TranslatePayload, text string, maxRetries, maxLength int) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		translation, err := p.translateSingle(ctx, client, payload, text, maxLength)
		if err == nil {
			return translation, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return "", lastErr
}

func (p *TranslateProcessor) translateSingle(ctx context.Context, client *translate.Client, payload models.TranslatePayload, text string, maxLength int) (string, error) {
	if len([]rune(text)) <= maxLength {
		res, err := client.Translate(ctx, []string{text}, payload.SourceLanguage, payload.TargetLanguage)
		if err != nil {
			return "", err
		}
		if len(res) != 1 {
			return "", fmt.Errorf("unexpected translation response length: %d", len(res))
		}
		return res[0], nil
	}

	chunks := splitText(text, maxLength)
	var translatedChunks []string
	for _, chunk := range chunks {
		res, err := client.Translate(ctx, []string{chunk}, payload.SourceLanguage, payload.TargetLanguage)
		if err != nil {
			return "", err
		}
		if len(res) != 1 {
			return "", fmt.Errorf("unexpected translation response length for chunk: %d", len(res))
		}
		translatedChunks = append(translatedChunks, res[0])
	}
	return strings.Join(translatedChunks, " "), nil
}

func (p *TranslateProcessor) updateSegmentTranslation(ctx context.Context, taskID uuid.UUID, idx int, translation string) error {
	updateQuery := `UPDATE segments SET mt_text = $1, updated_at = $2 WHERE task_id = $3 AND idx = $4`
	if _, err := p.deps.DB.ExecContext(ctx, updateQuery, translation, time.Now(), taskID, idx); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}
	return nil
}

func splitText(text string, maxLength int) []string {
	runes := []rune(text)
	if len(runes) <= maxLength {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(runes) {
		end := start + maxLength
		if end > len(runes) {
			end = len(runes)
		}
		chunkRunes := runes[start:end]
		// try to split at sentence boundary
		for i := end - 1; i > start; i-- {
			if runes[i] == '。' || runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
				chunkRunes = runes[start : i+1]
				end = i + 1
				break
			}
		}
		chunks = append(chunks, string(chunkRunes))
		start = end
	}
	return chunks
}
