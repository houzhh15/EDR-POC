package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestDefaultDLQConfig(t *testing.T) {
	cfg := DefaultDLQConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "edr.dlq", cfg.Topic)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryBackoff)
}

type mockProducer struct {
	produceFunc func(ctx context.Context, events []*EventMessage) error
	closed      bool
}

func (m *mockProducer) ProduceBatch(ctx context.Context, events []*EventMessage) error {
	if m.produceFunc != nil {
		return m.produceFunc(ctx, events)
	}
	return nil
}

func (m *mockProducer) Close() error {
	m.closed = true
	return nil
}

func TestNewDeadLetterQueue(t *testing.T) {
	tests := []struct {
		name     string
		producer Producer
		config   *DeadLetterQueueConfig
		logger   *zap.Logger
		wantErr  bool
	}{
		{
			name:     "valid producer and config",
			producer: &mockProducer{},
			config:   DefaultDLQConfig(),
			logger:   zap.NewNop(),
			wantErr:  false,
		},
		{
			name:     "nil producer",
			producer: nil,
			config:   DefaultDLQConfig(),
			logger:   zap.NewNop(),
			wantErr:  true,
		},
		{
			name:     "nil config uses defaults",
			producer: &mockProducer{},
			config:   nil,
			logger:   zap.NewNop(),
			wantErr:  false,
		},
		{
			name:     "nil logger uses nop",
			producer: &mockProducer{},
			config:   DefaultDLQConfig(),
			logger:   nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dlq, err := NewDeadLetterQueue(tt.producer, tt.config, tt.logger)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, dlq)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dlq)
			}
		})
	}
}

func TestDeadLetterQueue_Route(t *testing.T) {
	logger := zaptest.NewLogger(t)
	var producedEvents []*EventMessage

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			producedEvents = append(producedEvents, events...)
			return nil
		},
	}

	dlq, err := NewDeadLetterQueue(producer, DefaultDLQConfig(), logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		OriginalKey:   "test-key",
		OriginalValue: json.RawMessage(`{"test": "value"}`),
		Error:         "test error",
		ErrorType:     "deserialization_error",
		RetryCount:    0,
		FirstFailedAt: time.Now(),
		LastFailedAt:  time.Now(),
		Source:        "consumer",
		AgentID:       "agent-123",
		TenantID:      "tenant-456",
	}

	ctx := context.Background()
	err = dlq.Route(ctx, msg)
	require.NoError(t, err)

	assert.Len(t, producedEvents, 1)
	assert.Equal(t, "agent-123", producedEvents[0].AgentID)
	assert.Equal(t, "tenant-456", producedEvents[0].TenantID)
}

func TestDeadLetterQueue_Route_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	produceCallCount := 0

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			produceCallCount++
			return nil
		},
	}

	cfg := &DeadLetterQueueConfig{
		Enabled:      false, // Disabled
		Topic:        "edr.dlq",
		MaxRetries:   3,
		RetryBackoff: time.Second,
	}

	dlq, err := NewDeadLetterQueue(producer, cfg, logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		Error:         "test error",
	}

	ctx := context.Background()
	err = dlq.Route(ctx, msg)
	require.NoError(t, err)

	// Should not produce when disabled
	assert.Equal(t, 0, produceCallCount)
}

func TestDeadLetterQueue_Route_ProduceError(t *testing.T) {
	logger := zaptest.NewLogger(t)

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			return errors.New("produce failed")
		},
	}

	dlq, err := NewDeadLetterQueue(producer, DefaultDLQConfig(), logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		Error:         "test error",
	}

	ctx := context.Background()
	err = dlq.Route(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to produce to DLQ")
}

func TestDeadLetterQueue_RouteWithRetry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	attemptCount := 0

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			attemptCount++
			if attemptCount < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	cfg := &DeadLetterQueueConfig{
		Enabled:      true,
		Topic:        "edr.dlq",
		MaxRetries:   5,
		RetryBackoff: 10 * time.Millisecond, // Short backoff for test
	}

	dlq, err := NewDeadLetterQueue(producer, cfg, logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		Error:         "test error",
	}

	ctx := context.Background()
	err = dlq.RouteWithRetry(ctx, msg)
	require.NoError(t, err)

	assert.Equal(t, 3, attemptCount)
}

func TestDeadLetterQueue_RouteWithRetry_MaxRetriesExceeded(t *testing.T) {
	logger := zaptest.NewLogger(t)

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			return errors.New("permanent failure")
		},
	}

	cfg := &DeadLetterQueueConfig{
		Enabled:      true,
		Topic:        "edr.dlq",
		MaxRetries:   2,
		RetryBackoff: 10 * time.Millisecond,
	}

	dlq, err := NewDeadLetterQueue(producer, cfg, logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		Error:         "test error",
	}

	ctx := context.Background()
	err = dlq.RouteWithRetry(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to route to DLQ after")
}

func TestDeadLetterQueue_RouteWithRetry_ContextCanceled(t *testing.T) {
	logger := zaptest.NewLogger(t)

	producer := &mockProducer{
		produceFunc: func(ctx context.Context, events []*EventMessage) error {
			return errors.New("failure")
		},
	}

	cfg := &DeadLetterQueueConfig{
		Enabled:      true,
		Topic:        "edr.dlq",
		MaxRetries:   10,
		RetryBackoff: time.Second, // Long backoff
	}

	dlq, err := NewDeadLetterQueue(producer, cfg, logger)
	require.NoError(t, err)

	msg := &DeadLetterMessage{
		OriginalTopic: "test-topic",
		Error:         "test error",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = dlq.RouteWithRetry(ctx, msg)
	assert.Error(t, err)
}

func TestCreateDeadLetterMessage(t *testing.T) {
	event := &EventMessage{
		AgentID:  "agent-123",
		TenantID: "tenant-456",
		Headers: &MessageHeaders{
			TenantID:      "tenant-456",
			SchemaVersion: "v1",
			ContentType:   "application/json",
		},
	}

	msg := CreateDeadLetterMessage(
		"test-topic",
		"test-key",
		event,
		errors.New("processing error"),
		"processing_error",
		"consumer",
		2,
	)

	assert.Equal(t, "test-topic", msg.OriginalTopic)
	assert.Equal(t, "test-key", msg.OriginalKey)
	assert.Equal(t, "processing error", msg.Error)
	assert.Equal(t, "processing_error", msg.ErrorType)
	assert.Equal(t, 2, msg.RetryCount)
	assert.Equal(t, "consumer", msg.Source)
	assert.Equal(t, "agent-123", msg.AgentID)
	assert.Equal(t, "tenant-456", msg.TenantID)
	assert.Equal(t, "tenant-456", msg.OriginalHeaders["tenant_id"])
	assert.Equal(t, "v1", msg.OriginalHeaders["schema_version"])
}

func TestCreateDeadLetterMessage_NilEvent(t *testing.T) {
	msg := CreateDeadLetterMessage(
		"test-topic",
		"test-key",
		nil,
		errors.New("error"),
		"error_type",
		"producer",
		0,
	)

	assert.Equal(t, "test-topic", msg.OriginalTopic)
	assert.Empty(t, msg.AgentID)
	assert.Empty(t, msg.TenantID)
}

func TestDeadLetterQueue_Enabled(t *testing.T) {
	producer := &mockProducer{}

	cfg := &DeadLetterQueueConfig{
		Enabled: true,
		Topic:   "test.dlq",
	}

	dlq, err := NewDeadLetterQueue(producer, cfg, nil)
	require.NoError(t, err)

	assert.True(t, dlq.Enabled())
	assert.Equal(t, "test.dlq", dlq.Topic())
}

func TestDeadLetterQueue_Close(t *testing.T) {
	producer := &mockProducer{}

	dlq, err := NewDeadLetterQueue(producer, DefaultDLQConfig(), nil)
	require.NoError(t, err)

	err = dlq.Close()
	assert.NoError(t, err)
}
