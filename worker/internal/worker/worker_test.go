package worker

import (
	"context"
	"fmt"
	"testing"

	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/models"
	"vedio/worker/internal/queue"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubProcessor struct {
	name   string
	err    error
	called bool
}

func (p *stubProcessor) Name() string {
	return p.name
}

func (p *stubProcessor) Process(ctx context.Context, taskID uuid.UUID, msg models.TaskMessage) error {
	p.called = true
	return p.err
}

type mockPublisher struct {
	lastRoutingKey string
	lastMessage    interface{}
	publishCount   int
}

func (m *mockPublisher) Publish(ctx context.Context, routingKey string, message interface{}) error {
	m.lastRoutingKey = routingKey
	m.lastMessage = message
	m.publishCount++
	return nil
}

func (m *mockPublisher) Conn() *queue.Connection {
	return nil
}

func TestRunStepWithStatusSuccess(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	pub := &mockPublisher{}

	w := &Worker{
		db:        db,
		publisher: pub,
		config:    &config.Config{},
		logger:    zap.NewNop(),
	}

	processor := &stubProcessor{name: "extract_audio"}
	taskID := uuid.New()
	taskMsg := models.TaskMessage{TaskID: taskID.String(), Attempt: 1, TraceID: "trace"}

	mock.ExpectQuery(`SELECT status FROM task_steps`).
		WithArgs(taskID, processor.Name()).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM task_steps WHERE task_id = \$1 AND step = \$2 AND attempt = \$3\)`).
		WithArgs(taskID, processor.Name(), taskMsg.Attempt).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO task_steps`).
		WithArgs(taskID, processor.Name(), "running", taskMsg.Attempt, sqlmock.AnyArg(), nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM task_steps WHERE task_id = \$1 AND step = \$2 AND attempt = \$3\)`).
		WithArgs(taskID, processor.Name(), taskMsg.Attempt).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE task_steps\s+SET status = \$1, error = \$2, ended_at = \$3, metrics_json = \$4, updated_at = \$5 WHERE task_id = \$6 AND step = \$7 AND attempt = \$8`).
		WithArgs("succeeded", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), taskID, processor.Name(), taskMsg.Attempt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := w.runStepWithStatus(context.Background(), processor, taskID, taskMsg); err != nil {
		t.Fatalf("runStepWithStatus returned error: %v", err)
	}

	if !processor.called {
		t.Fatalf("processor was not invoked")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestRunStepWithStatusRetry(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	pub := &mockPublisher{}

	w := &Worker{
		db:        db,
		publisher: pub,
		config:    &config.Config{},
		logger:    zap.NewNop(),
	}

	processor := &stubProcessor{name: "asr", err: fmt.Errorf("step failed")}
	taskID := uuid.New()
	taskMsg := models.TaskMessage{TaskID: taskID.String(), Attempt: 1, TraceID: "trace"}

	mock.ExpectQuery(`SELECT status FROM task_steps`).
		WithArgs(taskID, processor.Name()).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM task_steps WHERE task_id = \$1 AND step = \$2 AND attempt = \$3\)`).
		WithArgs(taskID, processor.Name(), taskMsg.Attempt).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO task_steps`).
		WithArgs(taskID, processor.Name(), "running", taskMsg.Attempt, sqlmock.AnyArg(), nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM task_steps WHERE task_id = \$1 AND step = \$2 AND attempt = \$3\)`).
		WithArgs(taskID, processor.Name(), taskMsg.Attempt).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE task_steps\s+SET status = \$1, error = \$2, ended_at = \$3, metrics_json = \$4, updated_at = \$5 WHERE task_id = \$6 AND step = \$7 AND attempt = \$8`).
		WithArgs("failed", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg(), taskID, processor.Name(), taskMsg.Attempt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := w.runStepWithStatus(context.Background(), processor, taskID, taskMsg); err != nil {
		t.Fatalf("runStepWithStatus returned error: %v", err)
	}

	if !processor.called {
		t.Fatalf("processor was not invoked")
	}

	if pub.publishCount != 1 {
		t.Fatalf("expected retry publish to be called once, got %d", pub.publishCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestRegistryNames(t *testing.T) {
	reg := NewProcessorRegistry()
	reg.Register(&stubProcessor{name: "b"})
	reg.Register(&stubProcessor{name: "a"})

	names := reg.Names()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("unexpected names order: %v", names)
	}
}
