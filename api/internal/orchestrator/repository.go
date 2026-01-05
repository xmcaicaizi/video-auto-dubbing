package orchestrator

import (
	"context"
	"time"

	"github.com/google/uuid"

	"vedio/api/internal/database"
	"vedio/api/internal/models"
)

// DBTaskRepository persists task state transitions using the primary database.
type DBTaskRepository struct {
	db *database.DB
}

// NewDBTaskRepository constructs a task repository backed by the SQL database.
func NewDBTaskRepository(db *database.DB) *DBTaskRepository {
	return &DBTaskRepository{db: db}
}

// UpdateStatus updates the task status and timestamp.
func (r *DBTaskRepository) UpdateStatus(ctx context.Context, taskID uuid.UUID, status models.TaskStatus, updatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3", status, updatedAt, taskID)
	return err
}
