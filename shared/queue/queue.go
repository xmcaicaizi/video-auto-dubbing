package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vedio/shared/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "task_exchange"
	exchangeType = "topic"
)

// Connection wraps the RabbitMQ connection.
type Connection struct {
	*amqp.Connection
}

// NewConnection creates a new RabbitMQ connection.
func NewConnection(cfg config.RabbitMQConfig) (*Connection, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &Connection{conn}, nil
}

// Close closes the RabbitMQ connection.
func (c *Connection) Close() error {
	return c.Connection.Close()
}

// Publisher handles publishing messages to RabbitMQ.
type Publisher struct {
	conn *Connection
}

// NewPublisher creates a new publisher.
func NewPublisher(conn *Connection) *Publisher {
	return &Publisher{conn: conn}
}

// Publish publishes a message to the queue.
func (p *Publisher) Publish(ctx context.Context, routingKey string, message interface{}) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := ch.PublishWithContext(
		ctx,
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Conn returns the underlying connection.
func (p *Publisher) Conn() *Connection {
	return p.conn
}
