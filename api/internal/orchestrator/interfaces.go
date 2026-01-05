package orchestrator

import (
	"context"
	"time"

	"github.com/google/uuid"

	"vedio/api/internal/models"
)

// QueuePublisher describes the minimal queue publisher behavior orchestrator depends on.
// It intentionally matches the signature of queue.Publisher to enable easy swapping
// with other implementations (e.g., an RPC client in a standalone orchestrator process).
type QueuePublisher interface {
	Publish(ctx context.Context, routingKey string, message interface{}) error
}

// TaskRepository abstracts task persistence mutations required by the orchestrator.
// This allows the orchestration logic to move into a dedicated service without dragging
// database-specific details along.
type TaskRepository interface {
	UpdateStatus(ctx context.Context, taskID uuid.UUID, status models.TaskStatus, updatedAt time.Time) error
}

// TaskOrchestrator exposes the orchestration operations used by the API layer.
// Keeping this minimal makes it easy to inject mocks in tests and split the service later on.
type TaskOrchestrator interface {
	StartTask(ctx context.Context, task *models.Task) error
}
