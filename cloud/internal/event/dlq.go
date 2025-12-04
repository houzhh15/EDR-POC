// Package event provides Dead Letter Queue implementation for failed messages.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// DeadLetterMessage represents a message routed to the Dead Letter Queue.
type DeadLetterMessage struct {
	OriginalTopic   string            `json:"original_topic"`
	OriginalKey     string            `json:"original_key"`
	OriginalValue   json.RawMessage   `json:"original_value"`
	OriginalHeaders map[string]string `json:"original_headers"`
	Error           string            `json:"error"`
	ErrorType       string            `json:"error_type"`
	RetryCount      int               `json:"retry_count"`
	FirstFailedAt   time.Time         `json:"first_failed_at"`
	LastFailedAt    time.Time         `json:"last_failed_at"`
	Source          string            `json:"source"` // "producer" or "consumer"
	AgentID         string            `json:"agent_id,omitempty"`
	TenantID        string            `json:"tenant_id,omitempty"`
}

// DeadLetterQueueConfig configuration for DeadLetterQueue.
type DeadLetterQueueConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Topic        string        `yaml:"topic"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryBackoff time.Duration `yaml:"retry_backoff"`
}

// DefaultDLQConfig returns default DLQ configuration.
func DefaultDLQConfig() *DeadLetterQueueConfig {
	return &DeadLetterQueueConfig{
		Enabled:      true,
		Topic:        "edr.dlq",
		MaxRetries:   3,
		RetryBackoff: time.Second,
	}
}

// DeadLetterQueue handles routing failed messages to DLQ.
type DeadLetterQueue struct {
	producer Producer
	config   *DeadLetterQueueConfig
	logger   *zap.Logger
	metrics  *DLQMetrics
}

// NewDeadLetterQueue creates a new DeadLetterQueue.
func NewDeadLetterQueue(producer Producer, cfg *DeadLetterQueueConfig, logger *zap.Logger) (*DeadLetterQueue, error) {
	if producer == nil {
		return nil, fmt.Errorf("producer cannot be nil")
	}
	if cfg == nil {
		cfg = DefaultDLQConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	if cfg.Topic == "" {
		cfg.Topic = "edr.dlq"
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = time.Second
	}

	return &DeadLetterQueue{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

// SetMetrics sets the DLQ metrics collector.
func (d *DeadLetterQueue) SetMetrics(metrics *DLQMetrics) {
	d.metrics = metrics
}

// Route routes a failed message to the DLQ.
func (d *DeadLetterQueue) Route(ctx context.Context, msg *DeadLetterMessage) error {
	if !d.config.Enabled {
		d.logger.Debug("DLQ disabled, dropping message",
			zap.String("original_topic", msg.OriginalTopic),
			zap.String("error", msg.Error),
		)
		return nil
	}

	start := time.Now()

	// Serialize DLQ message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	// Create EventMessage wrapper for producer
	dlqEvent := &EventMessage{
		AgentID:    msg.AgentID,
		TenantID:   msg.TenantID,
		BatchID:    fmt.Sprintf("dlq-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
		Events: []*SecurityEvent{
			{
				EventID:   fmt.Sprintf("dlq-%s-%d", msg.OriginalTopic, time.Now().UnixNano()),
				EventType: "dead_letter",
				Timestamp: time.Now(),
				Severity:  3, // Warning
				RawData:   msgBytes,
			},
		},
	}

	// Produce to DLQ topic
	if err := d.producer.ProduceBatch(ctx, []*EventMessage{dlqEvent}); err != nil {
		if d.metrics != nil {
			d.metrics.RecordRouteError(msg.OriginalTopic, "produce_failed")
		}
		return fmt.Errorf("failed to produce to DLQ: %w", err)
	}

	latency := time.Since(start).Seconds()
	if d.metrics != nil {
		d.metrics.RecordMessageRouted(msg.OriginalTopic, msg.ErrorType)
		d.metrics.RecordRouteLatency(msg.OriginalTopic, latency)
	}

	d.logger.Info("message routed to DLQ",
		zap.String("original_topic", msg.OriginalTopic),
		zap.String("error", msg.Error),
		zap.String("error_type", msg.ErrorType),
		zap.Int("retry_count", msg.RetryCount),
		zap.Duration("latency", time.Since(start)),
	)

	return nil
}

// RouteWithRetry routes a message to DLQ with retry logic.
func (d *DeadLetterQueue) RouteWithRetry(ctx context.Context, msg *DeadLetterMessage) error {
	var lastErr error

	for attempt := 0; attempt <= d.config.MaxRetries; attempt++ {
		if d.metrics != nil && attempt > 0 {
			d.metrics.RecordRetryAttempt(msg.OriginalTopic)
		}

		err := d.Route(ctx, msg)
		if err == nil {
			return nil
		}

		lastErr = err
		d.logger.Warn("DLQ route failed, retrying",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", d.config.MaxRetries),
		)

		if attempt < d.config.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d.config.RetryBackoff * time.Duration(attempt+1)):
				// Exponential backoff
			}
		}
	}

	if d.metrics != nil {
		d.metrics.RecordRouteError(msg.OriginalTopic, "max_retries_exceeded")
	}

	return fmt.Errorf("failed to route to DLQ after %d retries: %w", d.config.MaxRetries, lastErr)
}

// CreateDeadLetterMessage creates a DeadLetterMessage from an EventMessage.
func CreateDeadLetterMessage(
	originalTopic string,
	originalKey string,
	event *EventMessage,
	err error,
	errorType string,
	source string,
	retryCount int,
) *DeadLetterMessage {
	var originalValue json.RawMessage
	if event != nil {
		originalValue, _ = json.Marshal(event)
	}

	now := time.Now()
	msg := &DeadLetterMessage{
		OriginalTopic:   originalTopic,
		OriginalKey:     originalKey,
		OriginalValue:   originalValue,
		OriginalHeaders: make(map[string]string),
		Error:           err.Error(),
		ErrorType:       errorType,
		RetryCount:      retryCount,
		FirstFailedAt:   now,
		LastFailedAt:    now,
		Source:          source,
	}

	if event != nil {
		msg.AgentID = event.AgentID
		msg.TenantID = event.TenantID
		if event.Headers != nil {
			msg.OriginalHeaders["tenant_id"] = event.Headers.TenantID
			msg.OriginalHeaders["schema_version"] = event.Headers.SchemaVersion
			msg.OriginalHeaders["content_type"] = event.Headers.ContentType
		}
	}

	return msg
}

// Enabled returns whether DLQ is enabled.
func (d *DeadLetterQueue) Enabled() bool {
	return d.config.Enabled
}

// Topic returns the DLQ topic name.
func (d *DeadLetterQueue) Topic() string {
	return d.config.Topic
}

// Close closes the DLQ (no-op as producer is managed externally).
func (d *DeadLetterQueue) Close() error {
	return nil
}
