package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/models"
	"vedio/worker/internal/translate"
	"vedio/worker/internal/tts"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// processExtractAudio processes the extract_audio step.
func (w *Worker) processExtractAudio(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.ExtractAudioPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	w.logger.Info("Extracting audio",
		zap.String("task_id", taskID.String()),
		zap.String("source_video_key", payload.SourceVideoKey),
		zap.String("output_audio_key", payload.OutputAudioKey),
	)

	// Download video from MinIO
	videoReader, err := w.storage.GetObject(ctx, payload.SourceVideoKey)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}
	defer videoReader.Close()

	// Create temporary video file
	videoPath := fmt.Sprintf("/tmp/%s_video.mp4", taskID)
	videoFile, err := os.Create(videoPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(videoPath)
	defer videoFile.Close()

	if _, err := io.Copy(videoFile, videoReader); err != nil {
		return fmt.Errorf("failed to write video: %w", err)
	}
	videoFile.Close()

	// Extract audio using ffmpeg
	audioPath := fmt.Sprintf("/tmp/%s_audio.wav", taskID)
	cmd := exec.CommandContext(ctx, w.config.FFmpeg.Path,
		"-i", videoPath,
		"-vn",           // No video
		"-acodec", "pcm_s16le", // PCM 16-bit
		"-ar", "16000",  // Sample rate (ASR recommended)
		"-ac", "1",      // Mono
		"-y",            // Overwrite
		audioPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}
	defer os.Remove(audioPath)

	// Upload audio to MinIO
	audioFile, err := os.Open(audioPath)
	if err != nil {
		return fmt.Errorf("failed to open audio: %w", err)
	}
	defer audioFile.Close()

	stat, err := audioFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat audio: %w", err)
	}

	if err := w.storage.PutObject(ctx, payload.OutputAudioKey, audioFile, stat.Size(), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Get task info to get source language
	var sourceLang string
	query := `SELECT source_language FROM tasks WHERE id = $1`
	if err := w.db.QueryRowContext(ctx, query, taskID).Scan(&sourceLang); err != nil {
		return fmt.Errorf("failed to get task source language: %w", err)
	}

	// Publish next step (ASR)
	asrMsg := map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "asr",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"audio_key":  payload.OutputAudioKey,
			"language":   sourceLang,
			"output_key": fmt.Sprintf("asr/%s/asr.json", taskID),
		},
	}

	if err := w.publisher.Publish(ctx, "task.asr", asrMsg); err != nil {
		return fmt.Errorf("failed to publish asr task: %w", err)
	}

	return nil
}

// processASR processes the asr step.
func (w *Worker) processASR(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.ASRPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	w.logger.Info("Processing ASR",
		zap.String("task_id", taskID.String()),
		zap.String("audio_key", payload.AudioKey),
		zap.String("language", payload.Language),
	)

	// Get audio from MinIO
	audioReader, err := w.storage.GetObject(ctx, payload.AudioKey)
	if err != nil {
		return fmt.Errorf("failed to get audio: %w", err)
	}
	defer audioReader.Close()

	// Load per-task external credentials (optional)
	var asrAppID, asrToken, asrCluster string
	queryTask := `SELECT asr_appid, asr_token, asr_cluster FROM tasks WHERE id = $1`
	_ = w.db.QueryRowContext(ctx, queryTask, taskID).Scan(&asrAppID, &asrToken, &asrCluster)
	w.logger.Debug("Loaded ASR credentials (per-task)",
		zap.String("task_id", taskID.String()),
		zap.Bool("has_asr_appid", asrAppID != ""),
		zap.Bool("has_asr_token", asrToken != ""),
		zap.Bool("has_asr_cluster", asrCluster != ""),
	)

	// Call ASR API
	asrResult, err := w.asrClient.Recognize(ctx, audioReader, payload.Language)
	if err != nil {
		return fmt.Errorf("ASR API call failed: %w", err)
	}

	w.logger.Info("ASR completed",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_count", len(asrResult.Segments)),
		zap.Int("duration_ms", asrResult.DurationMs),
	)

	// Save ASR result to MinIO
	resultJSON, _ := json.Marshal(asrResult)
	resultReader := bytes.NewReader(resultJSON)
	if err := w.storage.PutObject(ctx, payload.OutputKey, resultReader, int64(len(resultJSON)), "application/json"); err != nil {
		return fmt.Errorf("failed to save ASR result: %w", err)
	}

	// Save segments to database
	for _, seg := range asrResult.Segments {
		query := `
			INSERT INTO segments (task_id, idx, start_ms, end_ms, duration_ms, src_text, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (task_id, idx) DO UPDATE
			SET start_ms = EXCLUDED.start_ms, end_ms = EXCLUDED.end_ms,
			    duration_ms = EXCLUDED.duration_ms, src_text = EXCLUDED.src_text,
			    updated_at = EXCLUDED.updated_at
		`
		now := time.Now()
		_, err := w.db.ExecContext(ctx, query,
			taskID, seg.Idx, seg.StartMs, seg.EndMs, seg.EndMs-seg.StartMs,
			seg.Text, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to save segment: %w", err)
		}
	}

	// Get task info to get target language
	var sourceLang, targetLang string
	query := `SELECT source_language, target_language FROM tasks WHERE id = $1`
	if err := w.db.QueryRowContext(ctx, query, taskID).Scan(&sourceLang, &targetLang); err != nil {
		return fmt.Errorf("failed to get task languages: %w", err)
	}

	// Publish translate task
	// Get all segment IDs
	var segmentIDs []string
	rows, err := w.db.QueryContext(ctx, "SELECT idx FROM segments WHERE task_id = $1 ORDER BY idx", taskID)
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

	if err := w.publisher.Publish(ctx, "task.translate", translateMsg); err != nil {
		return fmt.Errorf("failed to publish translate task: %w", err)
	}

	return nil
}

// processTranslate processes the translate step.
func (w *Worker) processTranslate(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.TranslatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	w.logger.Info("Processing translation",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_count", len(payload.SegmentIDs)),
		zap.String("source_language", payload.SourceLanguage),
		zap.String("target_language", payload.TargetLanguage),
	)

	// Get segments from database
	query := `SELECT idx, src_text FROM segments WHERE task_id = $1 ORDER BY idx`
	rows, err := w.db.QueryContext(ctx, query, taskID)
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
	_ = w.db.QueryRowContext(ctx, q, taskID).Scan(&glmAPIKey, &glmAPIURL, &glmModel)
	if glmAPIKey == "" {
		glmAPIKey = w.config.External.GLM.APIKey
	}
	if glmAPIURL == "" {
		glmAPIURL = w.config.External.GLM.APIURL
	}
	if glmModel == "" {
		glmModel = w.config.External.GLM.Model
	}
	transClient := translate.NewClient(config.GLMConfig{APIKey: glmAPIKey, APIURL: glmAPIURL, Model: glmModel}, w.logger)

	// Call translation API
	translations, err := transClient.Translate(ctx, texts, payload.SourceLanguage, payload.TargetLanguage)
	if err != nil {
		return fmt.Errorf("translation API call failed: %w", err)
	}

	if len(translations) != len(segments) {
		return fmt.Errorf("translation count mismatch: expected %d, got %d", len(segments), len(translations))
	}

	w.logger.Info("Translation completed",
		zap.String("task_id", taskID.String()),
		zap.Int("translated_count", len(translations)),
	)

	// Update segments with translations
	for i, seg := range segments {
		translatedText := translations[i]
		updateQuery := `UPDATE segments SET mt_text = $1, updated_at = $2 WHERE task_id = $3 AND idx = $4`
		if _, err := w.db.ExecContext(ctx, updateQuery, translatedText, time.Now(), taskID, seg.idx); err != nil {
			return fmt.Errorf("failed to update segment: %w", err)
		}
	}

	// Get segment durations for TTS target_duration_ms
	segDurations := make(map[int]int)
	durQuery := `SELECT idx, duration_ms FROM segments WHERE task_id = $1 ORDER BY idx`
	durRows, err := w.db.QueryContext(ctx, durQuery, taskID)
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
				"task_id":           taskID.String(),
				"segment_id":        fmt.Sprintf("seg-%d", seg.idx),
				"segment_idx":        seg.idx,
				"text":               translatedText,
				"target_duration_ms": targetDur,
				"speaker_id":         "default",
			},
		}

		if err := w.publisher.Publish(ctx, "task.tts", ttsMsg); err != nil {
			w.logger.Error("Failed to publish TTS task", zap.Error(err), zap.Int("segment_idx", seg.idx))
			// Continue with other segments
		}
	}

	return nil
}

// processTTS processes the tts step.
func (w *Worker) processTTS(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.TTSPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	w.logger.Info("Processing TTS",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", payload.SegmentIdx),
		zap.String("text", payload.Text),
		zap.Int("target_duration_ms", payload.TargetDurationMs),
	)

	// Get task info to get target language
	var targetLang string
	var modelscopeToken string
	query := `SELECT target_language, modelscope_token FROM tasks WHERE id = $1`
	if err := w.db.QueryRowContext(ctx, query, taskID).Scan(&targetLang, &modelscopeToken); err != nil {
		return fmt.Errorf("failed to get task target language: %w", err)
	}

	// Prepare TTS request
	ttsReq := tts.SynthesisRequest{
		Text:             payload.Text,
		SpeakerID:      payload.SpeakerID,
		TargetDurationMs: payload.TargetDurationMs,
		Language:         targetLang,
		ProsodyControl:  payload.ProsodyControl,
		OutputFormat:     "wav",
		SampleRate:       22050,
	}

	// Call TTS service
	audioReader, err := w.ttsClient.Synthesize(ctx, ttsReq, modelscopeToken)
	if err != nil {
		return fmt.Errorf("TTS API call failed: %w", err)
	}
	defer audioReader.Close()

	// Save audio to MinIO
	audioKey := fmt.Sprintf("tts/%s/segment_%d.wav", taskID, payload.SegmentIdx)
	
	// Read audio data to get size
	audioData, err := io.ReadAll(audioReader)
	if err != nil {
		return fmt.Errorf("failed to read audio: %w", err)
	}

	// Upload to MinIO
	audioBytesReader := bytes.NewReader(audioData)
	if err := w.storage.PutObject(ctx, audioKey, audioBytesReader, int64(len(audioData)), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload TTS audio: %w", err)
	}

	// Prepare TTS params JSON
	ttsParams := map[string]interface{}{
		"speaker_id":          payload.SpeakerID,
		"target_duration_ms":  payload.TargetDurationMs,
		"prosody_control":     payload.ProsodyControl,
	}
	ttsParamsJSON, _ := json.Marshal(ttsParams)
	ttsParamsStr := string(ttsParamsJSON)

	// Update segment with TTS audio key and params
	updateQuery := `UPDATE segments SET tts_audio_key = $1, tts_params_json = $2, updated_at = $3 WHERE task_id = $4 AND idx = $5`
	if _, err := w.db.ExecContext(ctx, updateQuery, audioKey, ttsParamsStr, time.Now(), taskID, payload.SegmentIdx); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	w.logger.Info("TTS completed",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", payload.SegmentIdx),
		zap.String("audio_key", audioKey),
	)

	// Check if all segments have TTS audio
	var count int
	if err := w.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM segments WHERE task_id = $1 AND tts_audio_key IS NULL",
		taskID,
	).Scan(&count); err != nil {
		return fmt.Errorf("failed to check segments: %w", err)
	}

	// If all segments are done, merge audio and publish mux_video task
	if count == 0 {
		// Merge all segment audios
		if err := w.mergeSegmentAudios(ctx, taskID); err != nil {
			return fmt.Errorf("failed to merge segment audios: %w", err)
		}

		// Get task info
		var sourceVideoKey string
		if err := w.db.QueryRowContext(ctx,
			"SELECT source_video_key FROM tasks WHERE id = $1",
			taskID,
		).Scan(&sourceVideoKey); err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		muxMsg := map[string]interface{}{
			"task_id":    taskID.String(),
			"step":       "mux_video",
			"attempt":    1,
			"trace_id":   uuid.New().String(),
			"created_at": time.Now().Format(time.RFC3339),
			"payload": map[string]interface{}{
				"task_id":         taskID.String(),
				"source_video_key": sourceVideoKey,
				"tts_audio_key":    fmt.Sprintf("tts/%s/dub.wav", taskID),
				"output_video_key": fmt.Sprintf("outputs/%s/final.mp4", taskID),
			},
		}

		if err := w.publisher.Publish(ctx, "task.mux_video", muxMsg); err != nil {
			return fmt.Errorf("failed to publish mux_video task: %w", err)
		}
	}

	return nil
}

// mergeSegmentAudios merges all segment audio files into a single dub.wav file.
func (w *Worker) mergeSegmentAudios(ctx context.Context, taskID uuid.UUID) error {
	w.logger.Info("Merging segment audios", zap.String("task_id", taskID.String()))

	// Get all segments ordered by idx
	query := `SELECT idx, start_ms, end_ms, tts_audio_key FROM segments WHERE task_id = $1 ORDER BY idx`
	rows, err := w.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	type segmentInfo struct {
		idx          int
		startMs      int
		endMs        int
		ttsAudioKey  string
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

	// Find total duration
	var totalDurationMs int
	if len(segments) > 0 {
		lastSeg := segments[len(segments)-1]
		totalDurationMs = lastSeg.endMs
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
		audioReader, err := w.storage.GetObject(ctx, seg.ttsAudioKey)
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
		fmt.Fprintf(concatF, "file '%s'\n", segFile)
	}
	concatF.Close()

	// Use ffmpeg to concatenate audio files
	outputPath := fmt.Sprintf("/tmp/%s_dub.wav", taskID)
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, w.config.FFmpeg.Path,
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
	if err := w.storage.PutObject(ctx, dubKey, mergedFile, stat.Size(), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload merged audio: %w", err)
	}

	w.logger.Info("Segment audios merged successfully",
		zap.String("task_id", taskID.String()),
		zap.String("dub_key", dubKey),
		zap.Int("segment_count", len(segments)),
		zap.Int64("file_size", stat.Size()),
	)

	return nil
}

// processMuxVideo processes the mux_video step.
func (w *Worker) processMuxVideo(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.MuxVideoPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	w.logger.Info("Processing video muxing",
		zap.String("task_id", taskID.String()),
		zap.String("source_video_key", payload.SourceVideoKey),
		zap.String("tts_audio_key", payload.TTSAudioKey),
	)

	// Download video from MinIO
	videoReader, err := w.storage.GetObject(ctx, payload.SourceVideoKey)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}
	defer videoReader.Close()

	// Create temporary video file
	videoPath := fmt.Sprintf("/tmp/%s_source.mp4", taskID)
	videoFile, err := os.Create(videoPath)
	if err != nil {
		return fmt.Errorf("failed to create temp video file: %w", err)
	}
	defer os.Remove(videoPath)
	defer videoFile.Close()

	if _, err := io.Copy(videoFile, videoReader); err != nil {
		return fmt.Errorf("failed to write video: %w", err)
	}
	videoFile.Close()

	// Download TTS audio from MinIO
	audioReader, err := w.storage.GetObject(ctx, payload.TTSAudioKey)
	if err != nil {
		return fmt.Errorf("failed to get TTS audio: %w", err)
	}
	defer audioReader.Close()

	// Create temporary audio file
	audioPath := fmt.Sprintf("/tmp/%s_dub.wav", taskID)
	audioFile, err := os.Create(audioPath)
	if err != nil {
		return fmt.Errorf("failed to create temp audio file: %w", err)
	}
	defer os.Remove(audioPath)
	defer audioFile.Close()

	if _, err := io.Copy(audioFile, audioReader); err != nil {
		return fmt.Errorf("failed to write audio: %w", err)
	}
	audioFile.Close()

	// Use ffmpeg to replace audio track
	outputPath := fmt.Sprintf("/tmp/%s_final.mp4", taskID)
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, w.config.FFmpeg.Path,
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",        // Copy video codec
		"-c:a", "aac",         // Encode audio as AAC
		"-map", "0:v:0",       // Use video from first input
		"-map", "1:a:0",       // Use audio from second input
		"-shortest",           // Finish encoding when the shortest input stream ends
		"-y",                  // Overwrite
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg muxing failed: %w", err)
	}

	// Check if output file exists
	stat, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat output video: %w", err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("output video is empty")
	}

	// Upload final video to MinIO
	finalFile, err := os.Open(outputPath)
	if err != nil {
		return fmt.Errorf("failed to open final video: %w", err)
	}
	defer finalFile.Close()

	if err := w.storage.PutObject(ctx, payload.OutputVideoKey, finalFile, stat.Size(), "video/mp4"); err != nil {
		return fmt.Errorf("failed to upload final video: %w", err)
	}

	w.logger.Info("Video muxing completed",
		zap.String("task_id", taskID.String()),
		zap.String("output_video_key", payload.OutputVideoKey),
		zap.Int64("file_size", stat.Size()),
	)

	// Update task status to done
	if err := w.updateTaskStatus(ctx, taskID, "done", nil); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Update output video key
	updateQuery := `UPDATE tasks SET output_video_key = $1, updated_at = $2 WHERE id = $3`
	if _, err := w.db.ExecContext(ctx, updateQuery, payload.OutputVideoKey, time.Now(), taskID); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

