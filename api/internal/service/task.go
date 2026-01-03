package service

import (
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"vedio/api/internal/database"
	"vedio/api/internal/models"
	"vedio/api/internal/queue"
	"vedio/api/internal/storage"

	"github.com/google/uuid"
)

// TaskService handles task business logic.
type TaskService struct {
	db        *database.DB
	storage   *storage.Service
	publisher *queue.Publisher
}

// CreateTaskOptions carries optional per-task external credentials.
// NOTE: For MVP we store them in DB as plain text. In production, encrypt at rest and/or use a secret manager.
type CreateTaskOptions struct {
	ASRAppID        string
	ASRToken        string
	ASRCluster      string
	ASRAPIKey       string
	GLMAPIKey       string
	GLMAPIURL       string
	GLMModel        string
	ModelScopeToken string
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// NewTaskService creates a new task service.
func NewTaskService(db *database.DB, storage *storage.Service, publisher *queue.Publisher) *TaskService {
	return &TaskService{
		db:        db,
		storage:   storage,
		publisher: publisher,
	}
}

// CreateTask creates a new task and uploads the video file.
func (s *TaskService) CreateTask(ctx context.Context, file *multipart.FileHeader, sourceLang, targetLang string, opts CreateTaskOptions) (*models.Task, error) {
	// Generate task ID
	taskID := uuid.New()

	// Generate video key
	videoKey := fmt.Sprintf("videos/%s/source%s", taskID, filepath.Ext(file.Filename))

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Upload to MinIO
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	if err := s.storage.PutObject(ctx, videoKey, src, file.Size, contentType); err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	// Create task record
	task := &models.Task{
		ID:              taskID,
		Status:          models.TaskStatusCreated,
		Progress:        0,
		SourceVideoKey:  videoKey,
		SourceLanguage:  sourceLang,
		TargetLanguage:  targetLang,
		ASRAppID:        nil,
		ASRToken:        nil,
		ASRCluster:      nil,
		GLMAPIKey:       nil,
		GLMAPIURL:       nil,
		GLMModel:        nil,
		ModelScopeToken: nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	query := `
		INSERT INTO tasks (
			id, status, progress,
			source_video_key, source_language, target_language,
			asr_appid, asr_token, asr_cluster, asr_api_key,
			glm_api_key, glm_api_url, glm_model,
			modelscope_token,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13,
			$14,
			$15, $16
		)
	`
	if _, err := s.db.ExecContext(ctx, query,
		task.ID, task.Status, task.Progress, task.SourceVideoKey,
		task.SourceLanguage, task.TargetLanguage,
		toNullString(opts.ASRAppID), toNullString(opts.ASRToken), toNullString(opts.ASRCluster), toNullString(opts.ASRAPIKey),
		toNullString(opts.GLMAPIKey), toNullString(opts.GLMAPIURL), toNullString(opts.GLMModel),
		toNullString(opts.ModelScopeToken),
		task.CreatedAt, task.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Publish extract_audio task
	extractAudioMsg := map[string]interface{}{
		"task_id":    taskID.String(),
		"step":       "extract_audio",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": time.Now().Format(time.RFC3339),
		"payload": map[string]interface{}{
			"source_video_key": videoKey,
			"output_audio_key": fmt.Sprintf("audios/%s/source.wav", taskID),
		},
	}
	if err := s.publisher.Publish(ctx, "task.extract_audio", extractAudioMsg); err != nil {
		return nil, fmt.Errorf("failed to publish extract_audio task: %w", err)
	}

	// Update task status
	task.Status = models.TaskStatusQueued
	if _, err := s.db.ExecContext(ctx, "UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3",
		task.Status, time.Now(), task.ID); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return task, nil
}

// GetTaskWithSteps retrieves a task with its steps.
func (s *TaskService) GetTaskWithSteps(ctx context.Context, taskID uuid.UUID) (*models.Task, []models.TaskStep, error) {
	// Get task
	var task models.Task
	query := `
		SELECT id, status, progress, error, source_video_key, source_language, target_language,
		       asr_appid, asr_token, asr_cluster, asr_api_key,
		       glm_api_key, glm_api_url, glm_model,
		       modelscope_token,
		       output_video_key, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID, &task.Status, &task.Progress, &task.Error,
		&task.SourceVideoKey, &task.SourceLanguage, &task.TargetLanguage,
		&task.ASRAppID, &task.ASRToken, &task.ASRCluster, &task.ASRAPIKey,
		&task.GLMAPIKey, &task.GLMAPIURL, &task.GLMModel,
		&task.ModelScopeToken,
		&task.OutputVideoKey, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrTaskNotFound
		}
		return nil, nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Get steps
	stepsQuery := `
		SELECT id, task_id, step, status, attempt, started_at, ended_at, error, metrics_json, created_at, updated_at
		FROM task_steps WHERE task_id = $1 ORDER BY created_at
	`
	rows, err := s.db.QueryContext(ctx, stepsQuery, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get task steps: %w", err)
	}
	defer rows.Close()

	var steps []models.TaskStep
	for rows.Next() {
		var step models.TaskStep
		if err := rows.Scan(
			&step.ID, &step.TaskID, &step.Step, &step.Status, &step.Attempt,
			&step.StartedAt, &step.EndedAt, &step.Error, &step.MetricsJSON,
			&step.CreatedAt, &step.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan step: %w", err)
		}
		steps = append(steps, step)
	}

	return &task, steps, nil
}

// GetTaskResult retrieves the result of a completed task.
func (s *TaskService) GetTaskResult(ctx context.Context, taskID uuid.UUID) (map[string]interface{}, error) {
	task, _, err := s.GetTaskWithSteps(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != models.TaskStatusDone {
		return nil, ErrTaskNotCompleted
	}

	// Get segments
	segmentsQuery := `
		SELECT id, task_id, idx, start_ms, end_ms, duration_ms, src_text, mt_text, tts_params_json, tts_audio_key, created_at, updated_at
		FROM segments WHERE task_id = $1 ORDER BY idx
	`
	rows, err := s.db.QueryContext(ctx, segmentsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segments: %w", err)
	}
	defer rows.Close()

	var segments []map[string]interface{}
	for rows.Next() {
		var seg models.Segment
		if err := rows.Scan(
			&seg.ID, &seg.TaskID, &seg.Idx, &seg.StartMs, &seg.EndMs, &seg.DurationMs,
			&seg.SrcText, &seg.MtText, &seg.TtsParamsJSON, &seg.TtsAudioKey,
			&seg.CreatedAt, &seg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan segment: %w", err)
		}

		segResp := map[string]interface{}{
			"idx":      seg.Idx,
			"start_ms": seg.StartMs,
			"end_ms":   seg.EndMs,
			"src_text": seg.SrcText,
		}
		if seg.MtText != nil {
			segResp["mt_text"] = *seg.MtText
		}
		if seg.TtsAudioKey != nil {
			segResp["tts_audio_url"] = fmt.Sprintf("http://minio:9000/%s", *seg.TtsAudioKey)
		}
		segments = append(segments, segResp)
	}

	result := map[string]interface{}{
		"task_id":    task.ID.String(),
		"status":     string(task.Status),
		"segments":   segments,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}

	if task.OutputVideoKey != nil {
		result["output_video_url"] = fmt.Sprintf("http://minio:9000/%s", *task.OutputVideoKey)
	}

	return result, nil
}

// GetDownloadURL generates a presigned download URL.
func (s *TaskService) GetDownloadURL(ctx context.Context, taskID uuid.UUID, downloadType string) (string, error) {
	task, _, err := s.GetTaskWithSteps(ctx, taskID)
	if err != nil {
		return "", err
	}

	var key string
	switch downloadType {
	case "video":
		if task.OutputVideoKey == nil {
			return "", ErrTaskNotCompleted
		}
		key = *task.OutputVideoKey
	case "subtitle":
		key = fmt.Sprintf("subs/%s/subtitles.vtt", taskID)
	case "audio":
		key = fmt.Sprintf("tts/%s/dub.wav", taskID)
	default:
		return "", fmt.Errorf("invalid download type: %s", downloadType)
	}

	url, err := s.storage.PresignedGetURL(ctx, key, time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

// ListTasks lists tasks with pagination.
func (s *TaskService) ListTasks(ctx context.Context, status string, page, pageSize int) ([]models.Task, int, error) {
	offset := (page - 1) * pageSize

	var query string
	var countQuery string
	var args []interface{}

	if status != "" {
		query = `SELECT id, status, progress, error, source_video_key, source_language, target_language, output_video_key, created_at, updated_at
		         FROM tasks WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		countQuery = `SELECT COUNT(*) FROM tasks WHERE status = $1`
		args = []interface{}{status, pageSize, offset}
	} else {
		query = `SELECT id, status, progress, error, source_video_key, source_language, target_language, output_video_key, created_at, updated_at
		         FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		countQuery = `SELECT COUNT(*) FROM tasks`
		args = []interface{}{pageSize, offset}
	}

	// Get count
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get tasks
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID, &task.Status, &task.Progress, &task.Error,
			&task.SourceVideoKey, &task.SourceLanguage, &task.TargetLanguage,
			&task.OutputVideoKey, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// DeleteTask deletes a task and its associated data.
func (s *TaskService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	// Check if task exists
	var exists bool
	if err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1)", taskID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if !exists {
		return ErrTaskNotFound
	}

	// Delete task (cascade will delete steps and segments)
	if _, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}
