// Package queue provides message queue functionality using RabbitMQ.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/guidiju-50/pandora/CONTROL/internal/config"
	"go.uber.org/zap"
)

// RabbitMQ provides RabbitMQ connection and channel management.
type RabbitMQ struct {
	config     config.RabbitMQConfig
	conn       *amqp.Connection
	channel    *amqp.Channel
	logger     *zap.Logger
	mu         sync.RWMutex
	connected  bool
	closeChan  chan struct{}
}

// NewRabbitMQ creates a new RabbitMQ client.
func NewRabbitMQ(cfg config.RabbitMQConfig, logger *zap.Logger) *RabbitMQ {
	return &RabbitMQ{
		config:    cfg,
		logger:    logger,
		closeChan: make(chan struct{}),
	}
}

// Connect establishes connection to RabbitMQ.
func (r *RabbitMQ) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("connecting to RabbitMQ", zap.String("url", r.config.URL))

	conn, err := amqp.Dial(r.config.URL)
	if err != nil {
		return fmt.Errorf("connecting to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("opening channel: %w", err)
	}

	r.conn = conn
	r.channel = channel
	r.connected = true

	// Declare queues
	if err := r.declareQueues(); err != nil {
		return err
	}

	// Start connection monitor
	go r.monitorConnection()

	r.logger.Info("connected to RabbitMQ successfully")
	return nil
}

// declareQueues declares all required queues.
func (r *RabbitMQ) declareQueues() error {
	queues := []string{
		r.config.Queues.Processing,
		r.config.Queues.Analysis,
		r.config.Queues.Notifications,
		r.config.Queues.Processing + ".dlq",
		r.config.Queues.Analysis + ".dlq",
	}

	for _, queue := range queues {
		_, err := r.channel.QueueDeclare(
			queue,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			amqp.Table{
				"x-dead-letter-exchange":    "",
				"x-dead-letter-routing-key": queue + ".dlq",
			},
		)
		if err != nil {
			return fmt.Errorf("declaring queue %s: %w", queue, err)
		}
		r.logger.Debug("declared queue", zap.String("queue", queue))
	}

	return nil
}

// monitorConnection monitors the connection and reconnects if needed.
func (r *RabbitMQ) monitorConnection() {
	for {
		select {
		case <-r.closeChan:
			return
		case err := <-r.conn.NotifyClose(make(chan *amqp.Error)):
			if err != nil {
				r.logger.Error("RabbitMQ connection closed", zap.Error(err))
				r.mu.Lock()
				r.connected = false
				r.mu.Unlock()

				// Attempt to reconnect
				for i := 0; i < 5; i++ {
					time.Sleep(time.Duration(i+1) * time.Second)
					if err := r.Connect(); err == nil {
						break
					}
					r.logger.Warn("failed to reconnect to RabbitMQ", zap.Int("attempt", i+1))
				}
			}
		}
	}
}

// Close closes the RabbitMQ connection.
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(r.closeChan)

	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Message represents a queue message.
type Message struct {
	Type      string         `json:"type"`
	JobID     string         `json:"job_id"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

// Publish publishes a message to a queue.
func (r *RabbitMQ) Publish(ctx context.Context, queue string, msg *Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.connected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	msg.Timestamp = time.Now()

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	err = r.channel.PublishWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publishing message: %w", err)
	}

	r.logger.Debug("published message",
		zap.String("queue", queue),
		zap.String("type", msg.Type),
		zap.String("job_id", msg.JobID),
	)

	return nil
}

// Consume starts consuming messages from a queue.
func (r *RabbitMQ) Consume(queue string, handler func(*Message) error) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.connected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	msgs, err := r.channel.Consume(
		queue,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consuming from queue: %w", err)
	}

	go func() {
		for d := range msgs {
			var msg Message
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				r.logger.Error("failed to unmarshal message", zap.Error(err))
				d.Nack(false, false) // Don't requeue invalid messages
				continue
			}

			if err := handler(&msg); err != nil {
				r.logger.Error("failed to process message",
					zap.String("job_id", msg.JobID),
					zap.Error(err),
				)
				d.Nack(false, true) // Requeue on error
				continue
			}

			d.Ack(false)
		}
	}()

	r.logger.Info("started consuming from queue", zap.String("queue", queue))
	return nil
}

// PublishProcessingJob publishes a job to the processing queue.
func (r *RabbitMQ) PublishProcessingJob(ctx context.Context, jobID string, payload map[string]any) error {
	return r.Publish(ctx, r.config.Queues.Processing, &Message{
		Type:    "processing",
		JobID:   jobID,
		Payload: payload,
	})
}

// PublishAnalysisJob publishes a job to the analysis queue.
func (r *RabbitMQ) PublishAnalysisJob(ctx context.Context, jobID string, payload map[string]any) error {
	return r.Publish(ctx, r.config.Queues.Analysis, &Message{
		Type:    "analysis",
		JobID:   jobID,
		Payload: payload,
	})
}

// PublishNotification publishes a notification.
func (r *RabbitMQ) PublishNotification(ctx context.Context, userID string, payload map[string]any) error {
	return r.Publish(ctx, r.config.Queues.Notifications, &Message{
		Type:    "notification",
		JobID:   userID,
		Payload: payload,
	})
}

// IsConnected returns the connection status.
func (r *RabbitMQ) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}
