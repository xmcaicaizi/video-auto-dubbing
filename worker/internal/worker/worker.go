package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vedio/worker/internal/asr"
	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/models"
	"vedio/worker/internal/queue"
	"vedio/worker/internal/storage"
	"vedio/worker/internal/translate"
	"vedio/worker/internal/tts"
	"vedio/worker/internal/worker/steps"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	exchangeName = "task_exchange"
	exchangeType = "topic"
	maxRetries   = 3
)

// Publisher describes the minimal publishing behaviour Worker needs.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, message interface{}) error
	Conn() *queue.Connection
}

// Worker handles task processing.
type Worker struct {
	db          *database.DB
	storage     *storage.Service
	publisher   Publisher
	config      *config.Config
	logger      *zap.Logger
	asrClient   *asr.Client
	transClient *translate.Client
	ttsClient   *tts.Client
	registry    *ProcessorRegistry
}

// New creates a new worker.
func New(db *database.DB, storage *storage.Service, publisher Publisher, cfg *config.Config, logger *zap.Logger) *Worker {
	asrClient := asr.NewClient(cfg.External.ASR, logger)
	transClient := translate.NewClient(cfg.External.GLM, logger)
	ttsClient := tts.NewClient(cfg.TTS, logger)

	w := &Worker{
		db:          db,
		storage:     storage,
		publisher:   publisher,
		config:      cfg,
		logger:      logger,
		asrClient:   asrClient,
		transClient: transClient,
		ttsClient:   ttsClient,
	}

	w.registry = NewProcessorRegistry()
	w.registerDefaultProcessors()

	return w
}

func (w *Worker) registerDefaultProcessors() {
	deps := w.buildDeps()
	w.registry.Register(steps.NewExtractAudioProcessor(deps))
	w.registry.Register(steps.NewASRProcessor(deps))
	w.registry.Register(steps.NewTranslateProcessor(deps))
	w.registry.Register(steps.NewTTSProcessor(deps))
	w.registry.Register(steps.NewMuxVideoProcessor(deps))
}

func (w *Worker) buildDeps() steps.Deps {
	return steps.Deps{
		DB:        w.db,
		Storage:   w.storage,
		Publisher: w.publisher,
		Config:    w.config,
		Logger:    w.logger,

		ASRClient: w.asrClient,
		TTSClient: w.ttsClient,
	}
}

// StartConsumer starts consuming messages for a specific registered step.
func (w *Worker) StartConsumer(ctx context.Context, step string) error {
	processor, ok := w.registry.Get(step)
	if !ok {
		return fmt.Errorf("no processor registered for step: %s", step)
	}

	conn := w.publisher.Conn()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange
	if err := ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queueName := fmt.Sprintf("task.%s", step)
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	routingKey := fmt.Sprintf("task.%s", step)
	if err := ch.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Set QoS - only process one message at a time
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		q.Name,
		"",
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	w.logger.Info("Started consumer", zap.String("step", step), zap.String("queue", q.Name))

	// Process messages
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping consumer", zap.String("step", step))
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}

			if err := w.processMessage(ctx, processor, msg); err != nil {
				w.logger.Error("Failed to process message",
					zap.String("step", step),
					zap.Error(err),
					zap.String("message_id", msg.MessageId),
				)
				// Nack and don't requeue - will be handled by retry logic
				_ = msg.Nack(false, false)
			} else {
				// Ack on success
				_ = msg.Ack(false)
			}
		}
	}
}

// StartAllConsumers starts consumers for all registered processors.
func (w *Worker) StartAllConsumers(ctx context.Context) {
	for _, step := range w.registry.Names() {
		go func(stepName string) {
			if err := w.StartConsumer(ctx, stepName); err != nil {
				w.logger.Error("Consumer failed", zap.String("step", stepName), zap.Error(err))
			}
		}(step)
	}
}

// processMessage processes a single message using the registered processor.
func (w *Worker) processMessage(ctx context.Context, processor StepProcessor, msg amqp.Delivery) error {
	taskMsg, taskID, err := decodeTaskMessage(msg.Body)
	if err != nil {
		return err
	}

	return w.runStepWithStatus(ctx, processor, taskID, taskMsg)
}

func decodeTaskMessage(body []byte) (models.TaskMessage, uuid.UUID, error) {
	var taskMsg models.TaskMessage
	if err := json.Unmarshal(body, &taskMsg); err != nil {
		return models.TaskMessage{}, uuid.Nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	taskID, err := uuid.Parse(taskMsg.TaskID)
	if err != nil {
		return models.TaskMessage{}, uuid.Nil, fmt.Errorf("invalid task_id: %w", err)
	}

	return taskMsg, taskID, nil
}

func (w *Worker) runStepWithStatus(ctx context.Context, processor StepProcessor, taskID uuid.UUID, taskMsg models.TaskMessage) error {
	step := processor.Name()

	stepCtx, cancel := w.withStepTimeout(ctx, step)
	defer cancel()

	w.logger.Info("Processing message",
		zap.String("step", step),
		zap.String("task_id", taskID.String()),
		zap.Int("attempt", taskMsg.Attempt),
		zap.String("trace_id", taskMsg.TraceID),
		zap.Duration("timeout", w.stepTimeout(step)),
	)

	// Check if step is already completed (idempotency)
	stepStatus, err := w.getStepStatus(ctx, taskID, step)
	if err == nil && stepStatus == "succeeded" {
		w.logger.Info("Step already succeeded, skipping",
			zap.String("step", step),
			zap.String("task_id", taskID.String()),
		)
		return nil
	}

	// Update step status to running
	if err := w.updateStepStatus(ctx, taskID, step, taskMsg.Attempt, "running", nil); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// Process the step
	startTime := time.Now()
	processErr := processor.Process(stepCtx, taskID, taskMsg)
	duration := time.Since(startTime)

	if processErr != nil {
		// Update step status to failed
		errMsg := processErr.Error()
		if err := w.updateStepStatus(ctx, taskID, step, taskMsg.Attempt, "failed", &errMsg); err != nil {
			w.logger.Error("Failed to update step status", zap.Error(err))
		}

		// Retry logic
		if taskMsg.Attempt < maxRetries {
			return w.retryMessage(ctx, taskMsg, step)
		}

		// Max retries reached, update task status
		if err := w.updateTaskStatus(ctx, taskID, "failed", &errMsg); err != nil {
			w.logger.Error("Failed to update task status", zap.Error(err))
		}

		return fmt.Errorf("step failed after %d attempts: %w", taskMsg.Attempt, processErr)
	}

	// Update step status to succeeded with metrics
	metrics := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
		"task_id":     taskID.String(),
		"step":        step,
		"trace_id":    taskMsg.TraceID,
	}
	metricsJSON, _ := json.Marshal(metrics)
	metricsStr := string(metricsJSON)
	if err := w.updateStepStatusWithMetrics(ctx, taskID, step, taskMsg.Attempt, "succeeded", nil, &metricsStr); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	w.logger.Info("Step completed successfully",
		zap.String("step", step),
		zap.String("task_id", taskID.String()),
		zap.Duration("duration", duration),
	)

	return nil
}

// getStepStatus gets the status of a task step.
func (w *Worker) getStepStatus(ctx context.Context, taskID uuid.UUID, step string) (string, error) {
	query := `SELECT status FROM task_steps WHERE task_id = $1 AND step = $2 ORDER BY attempt DESC LIMIT 1`
	var status string
	err := w.db.QueryRowContext(ctx, query, taskID, step).Scan(&status)
	return status, err
}

// updateStepStatus updates the status of a task step.
func (w *Worker) updateStepStatus(ctx context.Context, taskID uuid.UUID, step string, attempt int, status string, errorMsg *string) error {
	return w.updateStepStatusWithMetrics(ctx, taskID, step, attempt, status, errorMsg, nil)
}

// updateStepStatusWithMetrics updates the status of a task step with metrics.
func (w *Worker) updateStepStatusWithMetrics(ctx context.Context, taskID uuid.UUID, step string, attempt int, status string, errorMsg *string, metricsJSON *string) error {
	now := time.Now()

	// Check if step record exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM task_steps WHERE task_id = $1 AND step = $2 AND attempt = $3)`
	if err := w.db.QueryRowContext(ctx, checkQuery, taskID, step, attempt).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check step existence: %w", err)
	}

	if !exists {
		// Insert new step record
		insertQuery := `
			INSERT INTO task_steps (task_id, step, status, attempt, started_at, error, metrics_json, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err := w.db.ExecContext(ctx, insertQuery,
			taskID, step, status, attempt, now, errorMsg, metricsJSON, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert step: %w", err)
		}
	} else {
		// Update existing step record
		updateQuery := `
			UPDATE task_steps
			SET status = $1, error = $2, metrics_json = $3, updated_at = $4
			WHERE task_id = $5 AND step = $6 AND attempt = $7
		`
		if status == "succeeded" || status == "failed" {
			updateQuery = `
				UPDATE task_steps
				SET status = $1, error = $2, ended_at = $3, metrics_json = $4, updated_at = $5
				WHERE task_id = $6 AND step = $7 AND attempt = $8
			`
			_, err := w.db.ExecContext(ctx, updateQuery,
				status, errorMsg, now, metricsJSON, now, taskID, step, attempt,
			)
			if err != nil {
				return fmt.Errorf("failed to update step: %w", err)
			}
		} else {
			_, err := w.db.ExecContext(ctx, updateQuery,
				status, errorMsg, metricsJSON, now, taskID, step, attempt,
			)
			if err != nil {
				return fmt.Errorf("failed to update step: %w", err)
			}
		}
	}

	return nil
}

// updateTaskStatus updates the task status.
func (w *Worker) updateTaskStatus(ctx context.Context, taskID uuid.UUID, status string, errorMsg *string) error {
	query := `UPDATE tasks SET status = $1, error = $2, updated_at = $3 WHERE id = $4`
	_, err := w.db.ExecContext(ctx, query, status, errorMsg, time.Now(), taskID)
	return err
}

// retryMessage retries a message with exponential backoff.
func (w *Worker) retryMessage(ctx context.Context, msg models.TaskMessage, step string) error {
	msg.Attempt++
	delay := time.Duration(1<<uint(msg.Attempt-1)) * time.Second // Exponential backoff

	w.logger.Info("Retrying message",
		zap.String("step", step),
		zap.String("task_id", msg.TaskID),
		zap.Int("attempt", msg.Attempt),
		zap.Duration("delay", delay),
	)

	// Wait before retrying
	time.Sleep(delay)

	// Publish retry message
	routingKey := fmt.Sprintf("task.%s", step)
	return w.publisher.Publish(ctx, routingKey, msg)
}

func (w *Worker) stepTimeout(step string) time.Duration {
	switch step {
	case "extract_audio":
		return w.config.Timeouts.ExtractAudio
	case "asr":
		return w.config.Timeouts.ASR
	case "tts":
		return w.config.Timeouts.TTS
	case "mux_video":
		return w.config.Timeouts.Mux
	default:
		return 0
	}
}

func (w *Worker) withStepTimeout(ctx context.Context, step string) (context.Context, context.CancelFunc) {
	timeout := w.stepTimeout(step)
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
