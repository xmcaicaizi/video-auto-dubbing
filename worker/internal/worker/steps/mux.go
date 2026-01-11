package steps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

	// Optional: burn subtitles onto video if available.
	subtitleKey := fmt.Sprintf("subs/%s/subtitles.vtt", taskID.String())
	subtitleSRTPath := ""
	if exists, err := p.deps.Storage.ObjectExists(ctx, subtitleKey); err != nil {
		return fmt.Errorf("failed to check subtitles: %w", err)
	} else if exists {
		subReader, err := p.deps.Storage.GetObject(ctx, subtitleKey)
		if err != nil {
			return fmt.Errorf("failed to get subtitles: %w", err)
		}
		vttBytes, err := io.ReadAll(subReader)
		subReader.Close()
		if err != nil {
			return fmt.Errorf("failed to read subtitles: %w", err)
		}

		srtBytes, err := convertVTTToSRT(vttBytes)
		if err != nil {
			return fmt.Errorf("failed to convert subtitles: %w", err)
		}

		subtitleSRTPath = fmt.Sprintf("/tmp/%s_subtitles.srt", taskID)
		if err := os.WriteFile(subtitleSRTPath, srtBytes, 0644); err != nil {
			return fmt.Errorf("failed to write subtitles: %w", err)
		}
		defer os.Remove(subtitleSRTPath)
	}

	// Use ffmpeg to replace audio track (and burn subtitles when present).
	outputPath := fmt.Sprintf("/tmp/%s_final.mp4", taskID)
	defer os.Remove(outputPath)

	ffmpegArgs := []string{
		"-i", videoPath,
		"-i", audioPath,
	}

	if subtitleSRTPath != "" {
		// Burning subtitles requires video re-encoding.
		// Note: /tmp paths have no spaces, so escaping is minimal.
		subFilter := fmt.Sprintf(
			"subtitles=%s:force_style='FontName=Arial,FontSize=24,Outline=2,Shadow=1'",
			subtitleSRTPath,
		)
		ffmpegArgs = append(ffmpegArgs,
			"-vf", subFilter,
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-crf", "23",
			"-pix_fmt", "yuv420p",
		)
	} else {
		ffmpegArgs = append(ffmpegArgs, "-c:v", "copy") // Copy video codec
	}

	ffmpegArgs = append(ffmpegArgs, "-c:a", "aac") // Encode audio as AAC
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

var vttTimestampRe = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2}\.\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}\.\d{3})`)

func convertVTTToSRT(vtt []byte) ([]byte, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(vtt)))
	type cue struct {
		start string
		end   string
		lines []string
	}
	var cues []cue

	var pendingText []string
	start := ""
	end := ""

	flush := func() {
		if start == "" || end == "" {
			pendingText = nil
			start, end = "", ""
			return
		}
		if len(pendingText) == 0 {
			start, end = "", ""
			return
		}
		cues = append(cues, cue{start: start, end: end, lines: append([]string(nil), pendingText...)})
		pendingText = nil
		start, end = "", ""
	}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		if strings.EqualFold(trimmed, "WEBVTT") {
			continue
		}
		if strings.HasPrefix(trimmed, "NOTE") {
			continue
		}

		if m := vttTimestampRe.FindStringSubmatch(trimmed); m != nil {
			flush()
			start = strings.ReplaceAll(m[1], ".", ",")
			end = strings.ReplaceAll(m[2], ".", ",")
			continue
		}

		// Ignore cue identifier line (a line before timestamp) by only collecting text after timestamp.
		if start == "" || end == "" {
			continue
		}
		pendingText = append(pendingText, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()

	var b strings.Builder
	for i, c := range cues {
		fmt.Fprintf(&b, "%d\n", i+1)
		fmt.Fprintf(&b, "%s --> %s\n", c.start, c.end)
		for _, l := range c.lines {
			b.WriteString(l)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}
