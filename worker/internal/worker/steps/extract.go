package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"vedio/worker/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ExtractAudioProcessor struct {
	deps Deps
}

func NewExtractAudioProcessor(deps Deps) *ExtractAudioProcessor {
	return &ExtractAudioProcessor{deps: deps}
}

func (p *ExtractAudioProcessor) Name() string {
	return "extract_audio"
}

func (p *ExtractAudioProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.ExtractAudioPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	p.deps.Logger.Info("Extracting audio",
		zap.String("task_id", taskID.String()),
		zap.String("source_video_key", payload.SourceVideoKey),
		zap.String("output_audio_key", payload.OutputAudioKey),
	)

	// Download video from MinIO
	videoReader, err := p.deps.Storage.GetObject(ctx, payload.SourceVideoKey)
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
	cmd := exec.CommandContext(ctx, p.deps.Config.FFmpeg.Path,
		"-i", videoPath,
		"-vn",                  // No video
		"-acodec", "pcm_s16le", // PCM 16-bit
		"-ar", "16000", // Sample rate (ASR recommended)
		"-ac", "1", // Mono
		"-y", // Overwrite
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

	if err := p.deps.Storage.PutObject(ctx, payload.OutputAudioKey, audioFile, stat.Size(), "audio/wav"); err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Get task info to get source language
	var sourceLang string
	query := `SELECT source_language FROM tasks WHERE id = $1`
	if err := p.deps.DB.QueryRowContext(ctx, query, taskID).Scan(&sourceLang); err != nil {
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

	if err := p.deps.Publisher.Publish(ctx, "task.asr", asrMsg); err != nil {
		return fmt.Errorf("failed to publish asr task: %w", err)
	}

	return nil
}
