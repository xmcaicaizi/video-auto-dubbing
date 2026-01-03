package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"vedio/api/internal/models"
)

// DefaultTaskOrchestrator implements task state transitions and initial message dispatch.
// It owns the task state machine entrypoint so the API layer can remain focused on validation
// and persistence.
type DefaultTaskOrchestrator struct {
	publisher QueuePublisher
	repo      TaskRepository
}

// NewTaskOrchestrator builds a DefaultTaskOrchestrator.
func NewTaskOrchestrator(publisher QueuePublisher, repo TaskRepository) *DefaultTaskOrchestrator {
	return &DefaultTaskOrchestrator{
		publisher: publisher,
		repo:      repo,
	}
}

// StartTask initializes the task state machine by publishing the first step and updating the status.
func (o *DefaultTaskOrchestrator) StartTask(ctx context.Context, task *models.Task) error {
	now := time.Now()
	extractAudioMsg := map[string]interface{}{
		"task_id":    task.ID.String(),
		"step":       "extract_audio",
		"attempt":    1,
		"trace_id":   uuid.New().String(),
		"created_at": now.Format(time.RFC3339),
		"payload": map[string]interface{}{
			"source_video_key": task.SourceVideoKey,
			"output_audio_key": fmt.Sprintf("audios/%s/source.wav", task.ID),
		},
	}

	if err := o.publisher.Publish(ctx, "task.extract_audio", extractAudioMsg); err != nil {
		return fmt.Errorf("publish initial step: %w", err)
	}

	if err := o.repo.UpdateStatus(ctx, task.ID, models.TaskStatusQueued, now); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	task.Status = models.TaskStatusQueued
	task.UpdatedAt = now
	return nil
}
