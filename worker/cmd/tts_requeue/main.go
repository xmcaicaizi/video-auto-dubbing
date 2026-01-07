package main

import (
	"context"
	"flag"
	"log"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/database"
	"vedio/worker/internal/queue"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	limit := flag.Int("limit", 100, "maximum number of tasks to scan for missing TTS segments")
	batchSize := flag.Int("batch-size", 0, "optional override for TTS batch size when requeueing")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}
	defer db.Close()

	conn, err := queue.NewConnection(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	pub := queue.NewPublisher(conn)
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT task_id, COUNT(*) as pending
		FROM segments
		WHERE tts_audio_key IS NULL OR tts_audio_key = ''
		GROUP BY task_id
		ORDER BY MAX(updated_at) DESC
		LIMIT $1
	`, *limit)
	if err != nil {
		log.Fatalf("failed to query pending segments: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var taskID uuid.UUID
		var pending int
		if err := rows.Scan(&taskID, &pending); err != nil {
			continue
		}

		payload := map[string]interface{}{
			"task_id":         taskID.String(),
			"batch_size":      chooseBatchSize(cfg, *batchSize),
			"max_concurrency": cfg.Processing.TTS.MaxConcurrency,
			"max_retries":     cfg.Processing.TTS.MaxRetries,
			"retry_delay_sec": cfg.Processing.TTS.RetryDelay.Seconds(),
			"speaker_id":      "default",
		}

		msg := map[string]interface{}{
			"task_id":    taskID.String(),
			"step":       "tts",
			"attempt":    1,
			"trace_id":   uuid.New().String(),
			"created_at": time.Now().Format(time.RFC3339),
			"payload":    payload,
		}

		if err := pub.Publish(ctx, "task.tts", msg); err != nil {
			logger.Error("failed to requeue tts task", zap.String("task_id", taskID.String()), zap.Error(err))
			continue
		}
		logger.Info("requeued tts batch", zap.String("task_id", taskID.String()), zap.Int("pending_segments", pending))
		count++
	}

	log.Printf("requeued %d TTS batches\n", count)
}

func chooseBatchSize(cfg *config.Config, override int) int {
	if override > 0 {
		return override
	}
	if cfg.Processing.TTS.BatchSize > 0 {
		return cfg.Processing.TTS.BatchSize
	}
	return 20
}
