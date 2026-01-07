package steps

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"vedio/worker/internal/models"
	"vedio/worker/internal/tts"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TTSProcessor struct {
	deps Deps
}

func NewTTSProcessor(deps Deps) *TTSProcessor {
	return &TTSProcessor{deps: deps}
}

func (p *TTSProcessor) Name() string {
	return "tts"
}

func (p *TTSProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.TTSPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	p.deps.Logger.Info("Processing TTS",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", payload.SegmentIdx),
		zap.String("text", payload.Text),
		zap.Int("target_duration_ms", payload.TargetDurationMs),
		zap.Int("batch_size", p.resolveBatchSize(payload)),
		zap.Int("max_concurrency", p.resolveConcurrency(payload)),
		zap.Int("max_retries", p.resolveMaxRetries(payload)),
	)

	sourceAudioPath, err := p.ensureSourceAudio(ctx, taskID)
	if err != nil {
		return err
	}

	promptInfo, err := p.ensurePromptAudio(ctx, taskID, payload.SpeakerID, sourceAudioPath)
	if err != nil {
		return err
	}

	targetLang, modelscopeToken, err := p.loadTaskLanguage(ctx, taskID)
	if err != nil {
		return err
	}

	pendingSegments, err := p.loadPendingSegments(ctx, taskID, payload)
	if err != nil {
		return err
	}
	if len(pendingSegments) == 0 {
		p.deps.Logger.Info("No pending TTS segments", zap.String("task_id", taskID.String()))
		return p.tryFinalizeTTS(ctx, taskID)
	}

	maxRetries := p.resolveMaxRetries(payload)
	retryDelay := p.resolveRetryDelay(payload)

	concurrency := p.resolveConcurrency(payload)
	if concurrency <= 0 {
		concurrency = 1
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var segmentErr error

	for _, seg := range pendingSegments {
		if seg.ttsAudioKey != "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		segment := seg
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := p.processSegmentWithRetry(ctx, taskID, segment, targetLang, modelscopeToken, promptInfo, maxRetries, retryDelay); err != nil {
				p.deps.Logger.Error("TTS segment failed after retries", zap.String("task_id", taskID.String()), zap.Int("segment_idx", segment.idx), zap.Error(err))
				_ = p.publishCompensation(ctx, taskID, segment.idx, err)
				mu.Lock()
				if segmentErr == nil {
					segmentErr = err
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if segmentErr != nil {
		return segmentErr
	}

	pendingCount, err := p.remainingPendingSegments(ctx, taskID)
	if err != nil {
		return err
	}

	if pendingCount > 0 {
		p.deps.Logger.Info("Pending segments remain, requeueing tts", zap.String("task_id", taskID.String()), zap.Int("remaining", pendingCount))
		return p.deps.Publisher.Publish(ctx, "task.tts", map[string]interface{}{
			"task_id":    taskID.String(),
			"step":       "tts",
			"attempt":    1,
			"trace_id":   uuid.New().String(),
			"created_at": time.Now().Format(time.RFC3339),
			"payload": map[string]interface{}{
				"task_id":         taskID.String(),
				"batch_size":      p.resolveBatchSize(payload),
				"max_concurrency": concurrency,
				"max_retries":     maxRetries,
				"retry_delay_sec": retryDelay.Seconds(),
				"speaker_id":      payload.SpeakerID,
			},
		})
	}

	return p.tryFinalizeTTS(ctx, taskID)
}

func (p *TTSProcessor) loadTaskLanguage(ctx context.Context, taskID uuid.UUID) (string, *string, error) {
	var targetLang string
	var modelscopeToken *string
	query := `SELECT target_language, modelscope_token FROM tasks WHERE id = $1`
	if err := p.deps.DB.QueryRowContext(ctx, query, taskID).Scan(&targetLang, &modelscopeToken); err != nil {
		return "", nil, fmt.Errorf("failed to get task target language: %w", err)
	}
	return targetLang, modelscopeToken, nil
}

type ttsSegment struct {
	idx         int
	text        string
	targetDurMs int
	ttsAudioKey string
	prosody     map[string]interface{}
	speakerID   string
}

func (p *TTSProcessor) loadPendingSegments(ctx context.Context, taskID uuid.UUID, payload models.TTSPayload) ([]ttsSegment, error) {
	batchSize := p.resolveBatchSize(payload)
	if batchSize <= 0 {
		batchSize = 20
	}
	query := `SELECT idx, COALESCE(mt_text, src_text, ''), duration_ms, COALESCE(tts_audio_key, '') FROM segments WHERE task_id = $1 AND (tts_audio_key IS NULL OR tts_audio_key = '') ORDER BY idx LIMIT $2`
	rows, err := p.deps.DB.QueryContext(ctx, query, taskID, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending segments: %w", err)
	}
	defer rows.Close()

	var segments []ttsSegment
	for rows.Next() {
		var seg ttsSegment
		if err := rows.Scan(&seg.idx, &seg.text, &seg.targetDurMs, &seg.ttsAudioKey); err != nil {
			continue
		}
		if seg.targetDurMs == 0 {
			seg.targetDurMs = 1500
		}
		seg.speakerID = payload.SpeakerID
		if seg.speakerID == "" {
			seg.speakerID = "default"
		}
		seg.prosody = payload.ProsodyControl
		segments = append(segments, seg)
	}

	if len(segments) == 0 && payload.SegmentIdx > 0 {
		var seg ttsSegment
		singleQuery := `SELECT idx, COALESCE(mt_text, src_text, ''), duration_ms, COALESCE(tts_audio_key, '') FROM segments WHERE task_id = $1 AND idx = $2`
		if err := p.deps.DB.QueryRowContext(ctx, singleQuery, taskID, payload.SegmentIdx).Scan(&seg.idx, &seg.text, &seg.targetDurMs, &seg.ttsAudioKey); err == nil {
			if seg.targetDurMs == 0 {
				seg.targetDurMs = 1500
			}
			seg.speakerID = payload.SpeakerID
			if seg.speakerID == "" {
				seg.speakerID = "default"
			}
			seg.prosody = payload.ProsodyControl
			segments = append(segments, seg)
		}
	}

	if len(segments) == 0 && payload.Text != "" {
		targetDur := payload.TargetDurationMs
		if targetDur == 0 {
			targetDur = 1500
		}
		speaker := payload.SpeakerID
		if speaker == "" {
			speaker = "default"
		}
		segments = append(segments, ttsSegment{
			idx:         payload.SegmentIdx,
			text:        payload.Text,
			targetDurMs: targetDur,
			speakerID:   speaker,
			prosody:     payload.ProsodyControl,
		})
	}

	return segments, nil
}

func (p *TTSProcessor) resolveBatchSize(payload models.TTSPayload) int {
	if payload.BatchSize > 0 {
		return payload.BatchSize
	}
	if p.deps.Config.Processing.TTS.BatchSize > 0 {
		return p.deps.Config.Processing.TTS.BatchSize
	}
	return 20
}

func (p *TTSProcessor) resolveConcurrency(payload models.TTSPayload) int {
	if payload.MaxConcurrency > 0 {
		return payload.MaxConcurrency
	}
	if p.deps.Config.Processing.TTS.MaxConcurrency > 0 {
		return p.deps.Config.Processing.TTS.MaxConcurrency
	}
	return 4
}

func (p *TTSProcessor) resolveMaxRetries(payload models.TTSPayload) int {
	if payload.MaxRetries > 0 {
		return payload.MaxRetries
	}
	if p.deps.Config.Processing.TTS.MaxRetries > 0 {
		return p.deps.Config.Processing.TTS.MaxRetries
	}
	return 3
}

func (p *TTSProcessor) resolveRetryDelay(payload models.TTSPayload) time.Duration {
	if payload.RetryDelaySec > 0 {
		return time.Duration(payload.RetryDelaySec * float64(time.Second))
	}
	if p.deps.Config.Processing.TTS.RetryDelay > 0 {
		return p.deps.Config.Processing.TTS.RetryDelay
	}
	return 2 * time.Second
}

func (p *TTSProcessor) processSegmentWithRetry(ctx context.Context, taskID uuid.UUID, segment ttsSegment, targetLang string, modelscopeToken *string, promptInfo promptInfo, maxRetries int, retryDelay time.Duration) error {
	existingKey, err := p.getExistingSegmentAudio(ctx, taskID, segment.idx)
	if err == nil && existingKey != "" {
		p.deps.Logger.Info("Segment already synthesized, skipping", zap.String("task_id", taskID.String()), zap.Int("segment_idx", segment.idx), zap.String("tts_audio_key", existingKey))
		return nil
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}
		if err := p.processSingleSegment(ctx, taskID, segment, targetLang, modelscopeToken, promptInfo); err != nil {
			lastErr = err
			p.deps.Logger.Warn("Segment TTS attempt failed", zap.String("task_id", taskID.String()), zap.Int("segment_idx", segment.idx), zap.Int("attempt", attempt+1), zap.Error(err))
			continue
		}
		return nil
	}
	return lastErr
}

func (p *TTSProcessor) processSingleSegment(ctx context.Context, taskID uuid.UUID, segment ttsSegment, targetLang string, modelscopeToken *string, promptInfo promptInfo) error {
	if strings.TrimSpace(segment.text) == "" {
		return fmt.Errorf("segment %d has empty text", segment.idx)
	}

	ttsReq := tts.SynthesisRequest{
		Text:             segment.text,
		SpeakerID:        segment.speakerID,
		PromptAudioURL:   promptInfo.url,
		TargetDurationMs: segment.targetDurMs,
		Language:         targetLang,
		ProsodyControl:   segment.prosody,
		OutputFormat:     "wav",
		SampleRate:       22050,
	}

	token := ""
	if modelscopeToken != nil {
		token = *modelscopeToken
	}
	audioReader, err := p.deps.TTSClient.Synthesize(ctx, ttsReq, token)
	if err != nil {
		return fmt.Errorf("TTS API call failed: %w", err)
	}
	defer audioReader.Close()

	audioKey := fmt.Sprintf("tts/%s/segment_%d.wav", taskID, segment.idx)

	audioData, err := io.ReadAll(audioReader)
	if err != nil {
		return fmt.Errorf("failed to read audio: %w", err)
	}

	audioBytesReader := bytes.NewReader(audioData)
	if err := p.deps.Storage.PutObject(ctx, audioKey, audioBytesReader, int64(len(audioData)), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload TTS audio: %w", err)
	}

	ttsParams := map[string]interface{}{
		"speaker_id":         segment.speakerID,
		"target_duration_ms": segment.targetDurMs,
		"prosody_control":    segment.prosody,
		"prompt_speaker_id":  segment.speakerID,
		"prompt_key":         promptInfo.key,
		"prompt_url":         promptInfo.url,
		"prompt_segment_idx": promptInfo.segmentIdx,
		"prompt_duration_ms": promptInfo.durationMs,
	}
	ttsParamsJSON, _ := json.Marshal(ttsParams)
	ttsParamsStr := string(ttsParamsJSON)

	updateQuery := `UPDATE segments SET tts_audio_key = $1, tts_params_json = $2, updated_at = $3 WHERE task_id = $4 AND idx = $5`
	if _, err := p.deps.DB.ExecContext(ctx, updateQuery, audioKey, ttsParamsStr, time.Now(), taskID, segment.idx); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	p.deps.Logger.Info("TTS completed",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", segment.idx),
		zap.String("audio_key", audioKey),
	)

	return nil
}

func (p *TTSProcessor) getExistingSegmentAudio(ctx context.Context, taskID uuid.UUID, segmentIdx int) (string, error) {
	var audioKey sql.NullString
	if err := p.deps.DB.QueryRowContext(ctx, `SELECT tts_audio_key FROM segments WHERE task_id = $1 AND idx = $2`, taskID, segmentIdx).Scan(&audioKey); err != nil {
		return "", err
	}
	if audioKey.Valid {
		return audioKey.String, nil
	}
	return "", nil
}

func (p *TTSProcessor) remainingPendingSegments(ctx context.Context, taskID uuid.UUID) (int, error) {
	var count int
	if err := p.deps.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM segments WHERE task_id = $1 AND (tts_audio_key IS NULL OR tts_audio_key = '')", taskID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to check pending segments: %w", err)
	}
	return count, nil
}

func (p *TTSProcessor) tryFinalizeTTS(ctx context.Context, taskID uuid.UUID) error {
	count, err := p.remainingPendingSegments(ctx, taskID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("pending segments remain: %d", count)
	}

	if err := p.mergeSegmentAudios(ctx, taskID); err != nil {
		return fmt.Errorf("failed to merge segment audios: %w", err)
	}

	var sourceVideoKey string
	if err := p.deps.DB.QueryRowContext(ctx, "SELECT source_video_key FROM tasks WHERE id = $1", taskID).Scan(&sourceVideoKey); err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	muxMsg := map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "mux_video",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"task_id":          taskID.String(),
			"source_video_key": sourceVideoKey,
			"tts_audio_key":    fmt.Sprintf("tts/%s/dub.wav", taskID),
			"output_video_key": fmt.Sprintf("outputs/%s/final.mp4", taskID),
		},
	}

	if err := p.deps.Publisher.Publish(ctx, "task.mux_video", muxMsg); err != nil {
		return fmt.Errorf("failed to publish mux_video task: %w", err)
	}
	return nil
}

func (p *TTSProcessor) publishCompensation(ctx context.Context, taskID uuid.UUID, segmentIdx int, err error) error {
	payload := map[string]interface{}{
		"task_id":     taskID.String(),
		"segment_idx": segmentIdx,
		"error":       err.Error(),
		"created_at":  time.Now().Format(time.RFC3339),
		"routing_key": "task.tts",
	}
	return p.deps.Publisher.Publish(ctx, "task.tts_compensation", map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "tts",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload":    payload,
	})
}

type promptInfo struct {
	key        string
	url        string
	segmentIdx int
	durationMs int
}

const (
	minPromptDurationMs          = 3_000
	preferredMaxPromptDurationMs = 8_000
	hardMaxPromptDurationMs      = 10_000
)

var speakerIDSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (p *TTSProcessor) ensurePromptAudio(ctx context.Context, taskID uuid.UUID, speakerID, sourceAudioPath string) (promptInfo, error) {
	promptKey := promptKeyForTask(taskID, speakerID)

	if info, found, err := p.findExistingPrompt(ctx, taskID, speakerID, promptKey); err == nil && found {
		return info, nil
	} else if err != nil {
		return promptInfo{}, err
	}

	segment, err := p.selectPromptSegment(ctx, taskID)
	if err != nil {
		return promptInfo{}, err
	}

	p.deps.Logger.Info("Selected prompt segment",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", segment.idx),
		zap.Int("start_ms", segment.startMs),
		zap.Int("end_ms", segment.endMs),
		zap.String("prompt_key", promptKey),
		zap.String("speaker_id", speakerID),
	)

	cutDurationMs := segment.endMs - segment.startMs
	if cutDurationMs > hardMaxPromptDurationMs {
		cutDurationMs = hardMaxPromptDurationMs
	}

	promptPath := fmt.Sprintf("/tmp/%s_prompt_%s.wav", taskID, sanitizeSpeakerID(speakerID))
	defer os.Remove(promptPath)

	if err := p.cutPrompt(ctx, sourceAudioPath, promptPath, segment.startMs, cutDurationMs); err != nil {
		return promptInfo{}, err
	}

	promptFile, err := os.Open(promptPath)
	if err != nil {
		return promptInfo{}, fmt.Errorf("failed to open prompt audio: %w", err)
	}
	defer promptFile.Close()
	promptStat, err := promptFile.Stat()
	if err != nil {
		return promptInfo{}, fmt.Errorf("failed to stat prompt audio: %w", err)
	}

	if err := p.deps.Storage.PutObject(ctx, promptKey, promptFile, promptStat.Size(), "audio/wav"); err != nil {
		return promptInfo{}, fmt.Errorf("failed to upload prompt audio: %w", err)
	}

	promptURL, err := p.deps.Storage.PresignedGetURL(ctx, promptKey, 24*time.Hour)
	if err != nil {
		return promptInfo{}, fmt.Errorf("failed to sign prompt audio URL: %w", err)
	}

	return promptInfo{
		key:        promptKey,
		url:        promptURL,
		segmentIdx: segment.idx,
		durationMs: cutDurationMs,
	}, nil
}

func (p *TTSProcessor) ensureSourceAudio(ctx context.Context, taskID uuid.UUID) (string, error) {
	sourceAudioKey := fmt.Sprintf("audios/%s/source.wav", taskID)
	sourceAudioPath := fmt.Sprintf("/tmp/%s_source.wav", taskID)
	if _, err := os.Stat(sourceAudioPath); err == nil {
		return sourceAudioPath, nil
	}

	audioReader, err := p.deps.Storage.GetObject(ctx, sourceAudioKey)
	if err != nil {
		return "", fmt.Errorf("failed to get source audio: %w", err)
	}
	defer audioReader.Close()

	sourceFile, err := os.Create(sourceAudioPath)
	if err != nil {
		return "", fmt.Errorf("failed to create source audio file: %w", err)
	}
	if _, err := io.Copy(sourceFile, audioReader); err != nil {
		sourceFile.Close()
		return "", fmt.Errorf("failed to write source audio: %w", err)
	}
	sourceFile.Close()

	return sourceAudioPath, nil
}

type promptSegment struct {
	idx     int
	startMs int
	endMs   int
}

func (p *TTSProcessor) selectPromptSegment(ctx context.Context, taskID uuid.UUID) (promptSegment, error) {
	query := `SELECT idx, start_ms, end_ms FROM segments WHERE task_id = $1 ORDER BY duration_ms DESC`
	rows, err := p.deps.DB.QueryContext(ctx, query, taskID)
	if err != nil {
		return promptSegment{}, fmt.Errorf("failed to list segments for prompt: %w", err)
	}
	defer rows.Close()

	var preferred *promptSegment
	var withinHardMax *promptSegment
	var longestValid *promptSegment

	for rows.Next() {
		var seg promptSegment
		if err := rows.Scan(&seg.idx, &seg.startMs, &seg.endMs); err != nil {
			continue
		}
		duration := seg.endMs - seg.startMs
		if duration <= 0 {
			continue
		}

		if duration < minPromptDurationMs {
			continue
		}

		if longestValid == nil || duration > (longestValid.endMs-longestValid.startMs) {
			temp := seg
			longestValid = &temp
		}

		if duration <= preferredMaxPromptDurationMs {
			if preferred == nil || duration > (preferred.endMs-preferred.startMs) {
				temp := seg
				preferred = &temp
			}
			continue
		}

		if duration <= hardMaxPromptDurationMs {
			if withinHardMax == nil || duration > (withinHardMax.endMs-withinHardMax.startMs) {
				temp := seg
				withinHardMax = &temp
			}
		}
	}

	if preferred != nil {
		return *preferred, nil
	}
	if withinHardMax != nil {
		return *withinHardMax, nil
	}
	if longestValid != nil {
		return *longestValid, nil
	}

	return promptSegment{}, fmt.Errorf("no valid segment found to build prompt")
}

func (p *TTSProcessor) cutPrompt(ctx context.Context, sourcePath, promptPath string, startMs, durationMs int) error {
	startSec := fmt.Sprintf("%.3f", float64(startMs)/1000.0)
	durSec := fmt.Sprintf("%.3f", float64(durationMs)/1000.0)
	cmd := exec.CommandContext(ctx, p.deps.Config.FFmpeg.Path,
		"-ss", startSec,
		"-t", durSec,
		"-i", sourcePath,
		"-ac", "1",
		"-ar", "16000",
		"-y",
		promptPath,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg prompt cut failed: %w", err)
	}
	return nil
}

func (p *TTSProcessor) findExistingPrompt(ctx context.Context, taskID uuid.UUID, speakerID, preferredKey string) (promptInfo, bool, error) {
	candidateInfos := make([]promptInfo, 0, 3)

	if dbPrompt, found, err := p.findPromptFromDB(ctx, taskID, speakerID); err == nil && found {
		candidateInfos = append(candidateInfos, dbPrompt)
	} else if err != nil {
		return promptInfo{}, false, err
	}

	candidateInfos = append(candidateInfos, promptInfo{key: preferredKey})
	if speakerID != "" {
		candidateInfos = append(candidateInfos, promptInfo{key: promptKeyForTask(taskID, "")})
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidateInfos {
		if candidate.key == "" {
			continue
		}
		if _, ok := seen[candidate.key]; ok {
			continue
		}
		seen[candidate.key] = struct{}{}

		exists, err := p.deps.Storage.ObjectExists(ctx, candidate.key)
		if err != nil {
			return promptInfo{}, false, err
		}
		if !exists {
			continue
		}

		// Always re-sign to avoid expired URLs.
		url, err := p.deps.Storage.PresignedGetURL(ctx, candidate.key, 24*time.Hour)
		if err != nil {
			return promptInfo{}, false, fmt.Errorf("failed to sign existing prompt URL: %w", err)
		}
		candidate.url = url

		return promptInfo{
			key: candidate.key,
			url: candidate.url,
		}, true, nil
	}

	return promptInfo{}, false, nil
}

func (p *TTSProcessor) findPromptFromDB(ctx context.Context, taskID uuid.UUID, speakerID string) (promptInfo, bool, error) {
	var promptKey sql.NullString
	var promptURL sql.NullString
	var query string
	var args []interface{}

	if speakerID != "" {
		query = `SELECT tts_params_json->>'prompt_key', tts_params_json->>'prompt_url'
FROM segments
WHERE task_id = $1
  AND tts_params_json->>'prompt_key' IS NOT NULL
  AND (tts_params_json->>'prompt_speaker_id' = $2 OR tts_params_json->>'speaker_id' = $2)
LIMIT 1`
		args = []interface{}{taskID, speakerID}
	} else {
		query = `SELECT tts_params_json->>'prompt_key', tts_params_json->>'prompt_url'
FROM segments
WHERE task_id = $1
  AND tts_params_json->>'prompt_key' IS NOT NULL
LIMIT 1`
		args = []interface{}{taskID}
	}

	if err := p.deps.DB.QueryRowContext(ctx, query, args...).Scan(&promptKey, &promptURL); err != nil {
		if err == sql.ErrNoRows {
			return promptInfo{}, false, nil
		}
		return promptInfo{}, false, fmt.Errorf("failed to check existing prompt key: %w", err)
	}

	if promptKey.Valid {
		info := promptInfo{key: promptKey.String}
		if promptURL.Valid {
			info.url = promptURL.String
		}
		return info, true, nil
	}

	return promptInfo{}, false, nil
}

func sanitizeSpeakerID(speakerID string) string {
	cleaned := speakerIDSanitizer.ReplaceAllString(strings.TrimSpace(speakerID), "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return "default"
	}
	return cleaned
}

func promptKeyForTask(taskID uuid.UUID, speakerID string) string {
	if speakerID == "" || speakerID == "default" {
		return fmt.Sprintf("tts/%s/prompt.wav", taskID)
	}
	return fmt.Sprintf("tts/%s/speakers/%s/prompt.wav", taskID, sanitizeSpeakerID(speakerID))
}

// mergeSegmentAudios merges all segment audio files into a single dub.wav file.
func (p *TTSProcessor) mergeSegmentAudios(ctx context.Context, taskID uuid.UUID) error {
	p.deps.Logger.Info("Merging segment audios", zap.String("task_id", taskID.String()))

	// Get all segments ordered by idx
	query := `SELECT idx, start_ms, end_ms, tts_audio_key FROM segments WHERE task_id = $1 ORDER BY idx`
	rows, err := p.deps.DB.QueryContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	type segmentInfo struct {
		idx         int
		startMs     int
		endMs       int
		ttsAudioKey string
	}
	var segments []segmentInfo

	for rows.Next() {
		var s segmentInfo
		if err := rows.Scan(&s.idx, &s.startMs, &s.endMs, &s.ttsAudioKey); err != nil {
			continue
		}
		if s.ttsAudioKey == "" {
			return fmt.Errorf("segment %d has no TTS audio", s.idx)
		}
		segments = append(segments, s)
	}

	if len(segments) == 0 {
		return fmt.Errorf("no segments found")
	}

	// Download all segment audio files to temp directory
	tempDir := fmt.Sprintf("/tmp/%s_merge", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	segmentFiles := make([]string, 0, len(segments))
	for _, seg := range segments {
		// Download segment audio
		audioReader, err := p.deps.Storage.GetObject(ctx, seg.ttsAudioKey)
		if err != nil {
			return fmt.Errorf("failed to get segment audio %d: %w", seg.idx, err)
		}

		segmentPath := fmt.Sprintf("%s/segment_%d.wav", tempDir, seg.idx)
		segmentFile, err := os.Create(segmentPath)
		if err != nil {
			audioReader.Close()
			return fmt.Errorf("failed to create segment file: %w", err)
		}

		if _, err := io.Copy(segmentFile, audioReader); err != nil {
			audioReader.Close()
			segmentFile.Close()
			return fmt.Errorf("failed to copy segment audio: %w", err)
		}

		audioReader.Close()
		segmentFile.Close()
		segmentFiles = append(segmentFiles, segmentPath)
	}

	// Create concat file for ffmpeg
	concatFile := fmt.Sprintf("%s/concat.txt", tempDir)
	concatF, err := os.Create(concatFile)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}

	for _, segFile := range segmentFiles {
		fmt.Fprintf(concatF, "file '%s'\\n", segFile)
	}
	concatF.Close()

	// Use ffmpeg to concatenate audio files
	outputPath := fmt.Sprintf("/tmp/%s_dub.wav", taskID)
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, p.deps.Config.FFmpeg.Path,
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		// Re-encode to ensure a valid WAV container after concatenation
		"-c:a", "pcm_s16le",
		"-ar", "22050",
		"-ac", "1",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w", err)
	}

	// Check if output file exists and has reasonable size
	stat, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat merged audio: %w", err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("merged audio is empty")
	}

	// Upload merged audio to MinIO
	mergedFile, err := os.Open(outputPath)
	if err != nil {
		return fmt.Errorf("failed to open merged audio: %w", err)
	}
	defer mergedFile.Close()

	dubKey := fmt.Sprintf("tts/%s/dub.wav", taskID)
	if err := p.deps.Storage.PutObject(ctx, dubKey, mergedFile, stat.Size(), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload merged audio: %w", err)
	}

	p.deps.Logger.Info("Segment audios merged successfully",
		zap.String("task_id", taskID.String()),
		zap.String("dub_key", dubKey),
		zap.Int("segment_count", len(segments)),
		zap.Int64("file_size", stat.Size()),
	)

	return nil
}
