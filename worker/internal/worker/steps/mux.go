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

type MuxVideoProcessor struct {
	deps Deps
}

func NewMuxVideoProcessor(deps Deps) *MuxVideoProcessor {
	return &MuxVideoProcessor{deps: deps}
}

func (p *MuxVideoProcessor) Name() string {
	return "mux_video"
}

func (p *MuxVideoProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	// Parse payload
	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload models.MuxVideoPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	p.deps.Logger.Info("Processing video muxing",
		zap.String("task_id", taskID.String()),
		zap.String("source_video_key", payload.SourceVideoKey),
		zap.String("tts_audio_key", payload.TTSAudioKey),
	)

	// Download video from MinIO
	videoReader, err := p.deps.Storage.GetObject(ctx, payload.SourceVideoKey)
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
	audioReader, err := p.deps.Storage.GetObject(ctx, payload.TTSAudioKey)
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

	ffmpegArgs := []string{
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy", // Copy video codec
		"-c:a", "aac", // Encode audio as AAC
	}
	if p.deps.Config.FFmpeg.DenoiseFilter != "" {
		ffmpegArgs = append(ffmpegArgs, "-af", p.deps.Config.FFmpeg.DenoiseFilter)
	}
	ffmpegArgs = append(ffmpegArgs,
		"-map", "0:v:0", // Use video from first input
		"-map", "1:a:0", // Use audio from second input
		"-shortest", // Finish encoding when the shortest input stream ends
		"-y",        // Overwrite
		outputPath,
	)

	cmd := exec.CommandContext(ctx, p.deps.Config.FFmpeg.Path, ffmpegArgs...)

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

	if err := p.deps.Storage.PutObject(ctx, payload.OutputVideoKey, finalFile, stat.Size(), "video/mp4"); err != nil {
		return fmt.Errorf("failed to upload final video: %w", err)
	}

	p.deps.Logger.Info("Video muxing completed",
		zap.String("task_id", taskID.String()),
		zap.String("output_video_key", payload.OutputVideoKey),
		zap.Int64("file_size", stat.Size()),
	)

	// Update task status to done
	if err := p.updateTaskStatus(ctx, taskID, "done", nil); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Update output video key
	updateQuery := `UPDATE tasks SET output_video_key = $1, updated_at = $2 WHERE id = $3`
	if _, err := p.deps.DB.ExecContext(ctx, updateQuery, payload.OutputVideoKey, time.Now(), taskID); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (p *MuxVideoProcessor) updateTaskStatus(ctx context.Context, taskID uuid.UUID, status string, errorMsg *string) error {
	query := `UPDATE tasks SET status = $1, error = $2, updated_at = $3 WHERE id = $4`
	_, err := p.deps.DB.ExecContext(ctx, query, status, errorMsg, time.Now(), taskID)
	return err
}
