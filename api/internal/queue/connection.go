package queue

import (
	"vedio/api/internal/config"
	sharedqueue "vedio/shared/queue"
)

// Connection is an alias to the shared RabbitMQ connection.
type Connection = sharedqueue.Connection

// NewConnection creates a new RabbitMQ connection using the shared implementation.
func NewConnection(cfg config.RabbitMQConfig) (*Connection, error) {
	return sharedqueue.NewConnection(cfg)
}
