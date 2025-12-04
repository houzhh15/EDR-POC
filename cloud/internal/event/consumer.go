// Package event provides Kafka consumer implementation for EDR cloud services.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer is the interface for Kafka consumers.
type Consumer interface {
	// Start starts the consumer and begins processing messages.
	Start(ctx context.Context) error
	// Messages returns a channel for receiving consumed messages.
	Messages() <-chan *EventMessage
	// Errors returns a channel for receiving consumer errors.
	Errors() <-chan error
	// CommitMessages commits the offsets for the given messages.
	CommitMessages(ctx context.Context, msgs ...*EventMessage) error
	// Close stops the consumer and releases resources.
	Close() error
}

// KafkaConsumerConfig configuration for KafkaConsumer.
type KafkaConsumerConfig struct {
	Brokers           []string      `yaml:"brokers"`
	Topic             string        `yaml:"topic"`
	GroupID           string        `yaml:"group_id"`
	MinBytes          int           `yaml:"min_bytes"`
	MaxBytes          int           `yaml:"max_bytes"`
	MaxWait           time.Duration `yaml:"max_wait"`
	CommitInterval    time.Duration `yaml:"commit_interval"`
	StartOffset       int64         `yaml:"start_offset"` // kafka.FirstOffset or kafka.LastOffset
	Concurrency       int           `yaml:"concurrency"`
	SessionTimeout    time.Duration `yaml:"session_timeout"`
	RebalanceTimeout  time.Duration `yaml:"rebalance_timeout"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	MessageBufferSize int           `yaml:"message_buffer_size"`
	ErrorBufferSize   int           `yaml:"error_buffer_size"`
}

// DefaultConsumerConfig returns default consumer configuration.
func DefaultConsumerConfig() *KafkaConsumerConfig {
	return &KafkaConsumerConfig{
		Brokers:           []string{"localhost:19092"},
		MinBytes:          1024,             // 1KB
		MaxBytes:          10 * 1024 * 1024, // 10MB
		MaxWait:           500 * time.Millisecond,
		CommitInterval:    time.Second,
		StartOffset:       kafka.LastOffset,
		Concurrency:       1,
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		MessageBufferSize: 1000,
		ErrorBufferSize:   100,
	}
}

// KafkaConsumer implements Consumer interface using segmentio/kafka-go.
type KafkaConsumer struct {
	reader *kafka.Reader
	config *KafkaConsumerConfig
	logger *zap.Logger

	messages chan *EventMessage
	errors   chan error
	done     chan struct{}

	wg      sync.WaitGroup
	started atomic.Bool
	closed  atomic.Bool
}

// NewKafkaConsumer creates a new Kafka consumer.
func NewKafkaConsumer(cfg *KafkaConsumerConfig, logger *zap.Logger) (*KafkaConsumer, error) {
	if cfg == nil {
		cfg = DefaultConsumerConfig()
	}
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("group_id cannot be empty")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Apply defaults for zero values
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 1024
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10 * 1024 * 1024
	}
	if cfg.MaxWait == 0 {
		cfg.MaxWait = 500 * time.Millisecond
	}
	if cfg.CommitInterval == 0 {
		cfg.CommitInterval = time.Second
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
	}
	if cfg.MessageBufferSize == 0 {
		cfg.MessageBufferSize = 1000
	}
	if cfg.ErrorBufferSize == 0 {
		cfg.ErrorBufferSize = 100
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		Topic:             cfg.Topic,
		GroupID:           cfg.GroupID,
		MinBytes:          cfg.MinBytes,
		MaxBytes:          cfg.MaxBytes,
		MaxWait:           cfg.MaxWait,
		CommitInterval:    cfg.CommitInterval,
		StartOffset:       cfg.StartOffset,
		SessionTimeout:    cfg.SessionTimeout,
		RebalanceTimeout:  cfg.RebalanceTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Debug(fmt.Sprintf(msg, args...), zap.String("component", "kafka-consumer"))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error(fmt.Sprintf(msg, args...), zap.String("component", "kafka-consumer"))
		}),
	})

	return &KafkaConsumer{
		reader:   reader,
		config:   cfg,
		logger:   logger,
		messages: make(chan *EventMessage, cfg.MessageBufferSize),
		errors:   make(chan error, cfg.ErrorBufferSize),
		done:     make(chan struct{}),
	}, nil
}

// Start starts the consumer goroutines.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	if !c.started.CompareAndSwap(false, true) {
		return fmt.Errorf("consumer already started")
	}
	if c.closed.Load() {
		return fmt.Errorf("consumer is closed")
	}

	c.logger.Info("starting kafka consumer",
		zap.String("topic", c.config.Topic),
		zap.String("group_id", c.config.GroupID),
		zap.Int("concurrency", c.config.Concurrency),
	)

	for i := 0; i < c.config.Concurrency; i++ {
		c.wg.Add(1)
		go c.consumeLoop(ctx, i)
	}

	return nil
}

// consumeLoop is the main consume loop for a worker.
func (c *KafkaConsumer) consumeLoop(ctx context.Context, workerID int) {
	defer c.wg.Done()

	c.logger.Debug("consumer worker started", zap.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("consumer worker stopped (context canceled)", zap.Int("worker_id", workerID))
			return
		case <-c.done:
			c.logger.Debug("consumer worker stopped (done signal)", zap.Int("worker_id", workerID))
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// Context canceled, normal shutdown
					return
				}
				c.logger.Error("failed to fetch message",
					zap.Error(err),
					zap.Int("worker_id", workerID),
				)
				c.sendError(fmt.Errorf("fetch message: %w", err))
				continue
			}

			// Deserialize message
			event, err := c.deserializeMessage(msg)
			if err != nil {
				c.logger.Error("failed to deserialize message",
					zap.Error(err),
					zap.Int("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
					zap.Int("worker_id", workerID),
				)
				c.sendError(fmt.Errorf("deserialize: %w", err))
				// Still commit to avoid blocking on bad messages
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("failed to commit bad message", zap.Error(commitErr))
				}
				continue
			}

			// Send message to channel
			select {
			case c.messages <- event:
				c.logger.Debug("message received",
					zap.String("agent_id", event.AgentID),
					zap.Int("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
					zap.Int("worker_id", workerID),
				)
			case <-ctx.Done():
				return
			case <-c.done:
				return
			}
		}
	}
}

// deserializeMessage deserializes a Kafka message into EventMessage.
func (c *KafkaConsumer) deserializeMessage(msg kafka.Message) (*EventMessage, error) {
	var event EventMessage
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	// Store kafka message reference for commit
	event.kafkaMsg = &msg
	event.partition = msg.Partition
	event.offset = msg.Offset

	// Parse headers
	event.Headers = ParseHeaders(msg.Headers)

	return &event, nil
}

// sendError sends an error to the errors channel (non-blocking).
func (c *KafkaConsumer) sendError(err error) {
	select {
	case c.errors <- err:
	default:
		c.logger.Warn("error channel full, dropping error", zap.Error(err))
	}
}

// Messages returns the messages channel.
func (c *KafkaConsumer) Messages() <-chan *EventMessage {
	return c.messages
}

// Errors returns the errors channel.
func (c *KafkaConsumer) Errors() <-chan error {
	return c.errors
}

// CommitMessages commits offsets for the given messages.
func (c *KafkaConsumer) CommitMessages(ctx context.Context, msgs ...*EventMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	kafkaMsgs := make([]kafka.Message, 0, len(msgs))
	for _, msg := range msgs {
		if msg.kafkaMsg != nil {
			kafkaMsgs = append(kafkaMsgs, *msg.kafkaMsg)
		}
	}

	if len(kafkaMsgs) == 0 {
		return nil
	}

	if err := c.reader.CommitMessages(ctx, kafkaMsgs...); err != nil {
		return fmt.Errorf("commit messages: %w", err)
	}

	c.logger.Debug("messages committed", zap.Int("count", len(kafkaMsgs)))
	return nil
}

// Close stops the consumer and releases resources.
func (c *KafkaConsumer) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	c.logger.Info("closing kafka consumer")

	// Signal workers to stop
	close(c.done)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Debug("all consumer workers stopped")
	case <-time.After(10 * time.Second):
		c.logger.Warn("timeout waiting for consumer workers to stop")
	}

	// Close reader
	if err := c.reader.Close(); err != nil {
		c.logger.Error("failed to close reader", zap.Error(err))
		return fmt.Errorf("close reader: %w", err)
	}

	// Close channels
	close(c.messages)
	close(c.errors)

	c.logger.Info("kafka consumer closed")
	return nil
}

// Stats returns consumer statistics.
func (c *KafkaConsumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// Lag returns the current consumer lag.
func (c *KafkaConsumer) Lag() int64 {
	return c.reader.Stats().Lag
}

// MessageHandler is a function type for handling messages.
type MessageHandler func(ctx context.Context, msg *EventMessage) error

// ConsumeWithHandler consumes messages and calls the handler for each.
// It automatically commits messages after successful processing.
func (c *KafkaConsumer) ConsumeWithHandler(ctx context.Context, handler MessageHandler) error {
	if err := c.Start(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-c.messages:
			if !ok {
				return nil // Channel closed
			}
			if err := handler(ctx, msg); err != nil {
				c.logger.Error("handler failed",
					zap.Error(err),
					zap.String("agent_id", msg.AgentID),
				)
				c.sendError(fmt.Errorf("handler: %w", err))
				// Don't commit failed messages
				continue
			}
			// Commit after successful processing
			if err := c.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit message", zap.Error(err))
			}
		case err, ok := <-c.errors:
			if !ok {
				return nil // Channel closed
			}
			c.logger.Error("consumer error", zap.Error(err))
		}
	}
}
