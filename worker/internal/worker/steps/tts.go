package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	)

	// Build prompt audio from original segment to preserve voice
	var startMs, endMs int
	segQuery := `SELECT start_ms, end_ms FROM segments WHERE task_id = $1 AND idx = $2`
	if err := p.deps.DB.QueryRowContext(ctx, segQuery, taskID, payload.SegmentIdx).Scan(&startMs, &endMs); err != nil {
		return fmt.Errorf("failed to get segment timing: %w", err)
	}
	if endMs <= startMs {
		return fmt.Errorf("invalid segment duration: start=%d end=%d", startMs, endMs)
	}

	sourceAudioKey := fmt.Sprintf("audios/%s/source.wav", taskID)
	sourceAudioPath := fmt.Sprintf("/tmp/%s_source.wav", taskID)
	if _, err := os.Stat(sourceAudioPath); err != nil {
		audioReader, err := p.deps.Storage.GetObject(ctx, sourceAudioKey)
		if err != nil {
			return fmt.Errorf("failed to get source audio: %w", err)
		}
		defer audioReader.Close()

		sourceFile, err := os.Create(sourceAudioPath)
		if err != nil {
			return fmt.Errorf("failed to create source audio file: %w", err)
		}
		if _, err := io.Copy(sourceFile, audioReader); err != nil {
			sourceFile.Close()
			return fmt.Errorf("failed to write source audio: %w", err)
		}
		sourceFile.Close()
	}

	promptPath := fmt.Sprintf("/tmp/%s_prompt_%d.wav", taskID, payload.SegmentIdx)
	defer os.Remove(promptPath)
	startSec := fmt.Sprintf("%.3f", float64(startMs)/1000.0)
	durSec := fmt.Sprintf("%.3f", float64(endMs-startMs)/1000.0)
	cmd := exec.CommandContext(ctx, p.deps.Config.FFmpeg.Path,
		"-ss", startSec,
		"-t", durSec,
		"-i", sourceAudioPath,
		"-ac", "1",
		"-ar", "16000",
		"-y",
		promptPath,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg prompt cut failed: %w", err)
	}

	promptFile, err := os.Open(promptPath)
	if err != nil {
		return fmt.Errorf("failed to open prompt audio: %w", err)
	}
	defer promptFile.Close()
	promptStat, err := promptFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat prompt audio: %w", err)
	}
	promptKey := fmt.Sprintf("tts/%s/prompt_%d.wav", taskID, payload.SegmentIdx)
	if err := p.deps.Storage.PutObject(ctx, promptKey, promptFile, promptStat.Size(), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload prompt audio: %w", err)
	}
	promptURL, err := p.deps.Storage.PresignedGetURL(ctx, promptKey, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to sign prompt audio URL: %w", err)
	}

	// Get task info to get target language
	var targetLang string
	var modelscopeToken *string
	query := `SELECT target_language, modelscope_token FROM tasks WHERE id = $1`
	if err := p.deps.DB.QueryRowContext(ctx, query, taskID).Scan(&targetLang, &modelscopeToken); err != nil {
		return fmt.Errorf("failed to get task target language: %w", err)
	}

	// Prepare TTS request
	ttsReq := tts.SynthesisRequest{
		Text:             payload.Text,
		SpeakerID:        payload.SpeakerID,
		PromptAudioURL:   promptURL,
		TargetDurationMs: payload.TargetDurationMs,
		Language:         targetLang,
		ProsodyControl:   payload.ProsodyControl,
		OutputFormat:     "wav",
		SampleRate:       22050,
	}

	// Call TTS service
	token := ""
	if modelscopeToken != nil {
		token = *modelscopeToken
	}
	audioReader, err := p.deps.TTSClient.Synthesize(ctx, ttsReq, token)
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
	if err := p.deps.Storage.PutObject(ctx, audioKey, audioBytesReader, int64(len(audioData)), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload TTS audio: %w", err)
	}

	// Prepare TTS params JSON
	ttsParams := map[string]interface{}{
		"speaker_id":         payload.SpeakerID,
		"target_duration_ms": payload.TargetDurationMs,
		"prosody_control":    payload.ProsodyControl,
	}
	ttsParamsJSON, _ := json.Marshal(ttsParams)
	ttsParamsStr := string(ttsParamsJSON)

	// Update segment with TTS audio key and params
	updateQuery := `UPDATE segments SET tts_audio_key = $1, tts_params_json = $2, updated_at = $3 WHERE task_id = $4 AND idx = $5`
	if _, err := p.deps.DB.ExecContext(ctx, updateQuery, audioKey, ttsParamsStr, time.Now(), taskID, payload.SegmentIdx); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	p.deps.Logger.Info("TTS completed",
		zap.String("task_id", taskID.String()),
		zap.Int("segment_idx", payload.SegmentIdx),
		zap.String("audio_key", audioKey),
	)

	// Check if all segments have TTS audio
	var count int
	if err := p.deps.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM segments WHERE task_id = $1 AND tts_audio_key IS NULL",
		taskID,
	).Scan(&count); err != nil {
		return fmt.Errorf("failed to check segments: %w", err)
	}

	// If all segments are done, merge audio and publish mux_video task
	if count == 0 {
		// Merge all segment audios
		if err := p.mergeSegmentAudios(ctx, taskID); err != nil {
			return fmt.Errorf("failed to merge segment audios: %w", err)
		}

		// Get task info
		var sourceVideoKey string
		if err := p.deps.DB.QueryRowContext(ctx,
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
				"task_id":          taskID.String(),
				"source_video_key": sourceVideoKey,
				"tts_audio_key":    fmt.Sprintf("tts/%s/dub.wav", taskID),
				"output_video_key": fmt.Sprintf("outputs/%s/final.mp4", taskID),
			},
		}

		if err := p.deps.Publisher.Publish(ctx, "task.mux_video", muxMsg); err != nil {
			return fmt.Errorf("failed to publish mux_video task: %w", err)
		}
	}

	return nil
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
		fmt.Fprintf(concatF, "file '%s'\n", segFile)
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
