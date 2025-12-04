package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestDefaultConsumerConfig(t *testing.T) {
	cfg := DefaultConsumerConfig()

	assert.Equal(t, []string{"localhost:19092"}, cfg.Brokers)
	assert.Equal(t, 1024, cfg.MinBytes)
	assert.Equal(t, 10*1024*1024, cfg.MaxBytes)
	assert.Equal(t, 500*time.Millisecond, cfg.MaxWait)
	assert.Equal(t, time.Second, cfg.CommitInterval)
	assert.Equal(t, kafka.LastOffset, cfg.StartOffset)
	assert.Equal(t, 1, cfg.Concurrency)
	assert.Equal(t, 30*time.Second, cfg.SessionTimeout)
	assert.Equal(t, 3*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, 1000, cfg.MessageBufferSize)
	assert.Equal(t, 100, cfg.ErrorBufferSize)
}

func TestNewKafkaConsumer(t *testing.T) {
	tests := []struct {
		name    string
		config  *KafkaConsumerConfig
		logger  *zap.Logger
		wantErr string
	}{
		{
			name: "valid config",
			config: &KafkaConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
				GroupID: "test-group",
			},
			logger:  zap.NewNop(),
			wantErr: "",
		},
		{
			name:    "nil config uses defaults but needs topic and group",
			config:  nil,
			logger:  zap.NewNop(),
			wantErr: "topic cannot be empty",
		},
		{
			name: "empty brokers",
			config: &KafkaConsumerConfig{
				Brokers: []string{},
				Topic:   "test-topic",
				GroupID: "test-group",
			},
			logger:  zap.NewNop(),
			wantErr: "brokers list cannot be empty",
		},
		{
			name: "empty topic",
			config: &KafkaConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "",
				GroupID: "test-group",
			},
			logger:  zap.NewNop(),
			wantErr: "topic cannot be empty",
		},
		{
			name: "empty group_id",
			config: &KafkaConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
				GroupID: "",
			},
			logger:  zap.NewNop(),
			wantErr: "group_id cannot be empty",
		},
		{
			name: "nil logger uses nop",
			config: &KafkaConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
				GroupID: "test-group",
			},
			logger:  nil,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := NewKafkaConsumer(tt.config, tt.logger)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, consumer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)
				consumer.Close()
			}
		})
	}
}

func TestKafkaConsumer_DeserializeMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	// Create test message
	testEvent := &EventMessage{
		AgentID:    "agent-123",
		TenantID:   "tenant-456",
		BatchID:    "batch-789",
		Timestamp:  time.Now().UTC().Truncate(time.Second),
		ReceivedAt: time.Now().UTC().Truncate(time.Second),
		Events: []*SecurityEvent{
			{
				EventID:   "event-1",
				EventType: "process_start",
				Timestamp: time.Now().UTC().Truncate(time.Second),
				Severity:  5,
			},
		},
	}

	msgBytes, err := json.Marshal(testEvent)
	require.NoError(t, err)

	kafkaMsg := kafka.Message{
		Topic:     "test-topic",
		Partition: 2,
		Offset:    100,
		Key:       []byte("agent-123"),
		Value:     msgBytes,
		Headers: []kafka.Header{
			{Key: "tenant_id", Value: []byte("tenant-456")},
			{Key: "schema_version", Value: []byte("v1")},
			{Key: "content_type", Value: []byte("application/json")},
		},
	}

	event, err := consumer.deserializeMessage(kafkaMsg)
	require.NoError(t, err)

	assert.Equal(t, "agent-123", event.AgentID)
	assert.Equal(t, "tenant-456", event.TenantID)
	assert.Equal(t, "batch-789", event.BatchID)
	assert.Equal(t, 2, event.partition)
	assert.Equal(t, int64(100), event.offset)
	assert.NotNil(t, event.kafkaMsg)
	assert.NotNil(t, event.Headers)
	assert.Equal(t, "tenant-456", event.Headers.TenantID)
	assert.Equal(t, "v1", event.Headers.SchemaVersion)
}

func TestKafkaConsumer_DeserializeMessage_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	kafkaMsg := kafka.Message{
		Value: []byte("invalid json"),
	}

	event, err := consumer.deserializeMessage(kafkaMsg)
	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "json unmarshal")
}

func TestKafkaConsumer_StartTwice(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	ctx := context.Background()

	// First start should succeed
	// Note: This will fail to actually connect but won't return error
	// because connection happens lazily
	err = consumer.Start(ctx)
	require.NoError(t, err)

	// Second start should fail
	err = consumer.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}

func TestKafkaConsumer_CloseWithoutStart(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)

	// Close without starting should succeed
	err = consumer.Close()
	assert.NoError(t, err)

	// Close again should be idempotent
	err = consumer.Close()
	assert.NoError(t, err)
}

func TestKafkaConsumer_CommitEmptyMessages(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	ctx := context.Background()

	// Commit with no messages should succeed
	err = consumer.CommitMessages(ctx)
	assert.NoError(t, err)

	// Commit with nil kafkaMsg should succeed (no-op)
	err = consumer.CommitMessages(ctx, &EventMessage{})
	assert.NoError(t, err)
}

func TestKafkaConsumer_Channels(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &KafkaConsumerConfig{
		Brokers:           []string{"localhost:9092"},
		Topic:             "test-topic",
		GroupID:           "test-group",
		MessageBufferSize: 10,
		ErrorBufferSize:   5,
	}

	consumer, err := NewKafkaConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	// Check channels are initialized
	messages := consumer.Messages()
	assert.NotNil(t, messages)

	errors := consumer.Errors()
	assert.NotNil(t, errors)
}

// Integration tests - require Kafka/Redpanda running

func TestKafkaConsumer_Integration_ConsumeMessages(t *testing.T) {
	skipIfNoKafka(t)

	logger := zaptest.NewLogger(t)
	testTopic := "test.consumer." + time.Now().Format("20060102150405")
	testGroup := "test-group-" + time.Now().Format("20060102150405")

	// Create topic first
	tmCfg := &TopicManagerConfig{
		Brokers: []string{"localhost:19092"},
		TopicConfigs: map[string]TopicDefinition{
			testTopic: {
				Partitions:        2,
				ReplicationFactor: 1,
				RetentionMs:       3600000,
				CleanupPolicy:     "delete",
			},
		},
		AutoCreate: true,
	}

	tm, err := NewTopicManager(tmCfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = tm.EnsureTopics(ctx)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Produce some messages
	producer := NewKafkaProducer([]string{"localhost:19092"}, testTopic, logger)
	defer producer.Close()

	testEvents := []*EventMessage{
		{
			AgentID:    "agent-1",
			TenantID:   "tenant-1",
			BatchID:    "batch-1",
			Timestamp:  time.Now(),
			ReceivedAt: time.Now(),
			Events: []*SecurityEvent{
				{EventID: "e1", EventType: "test", Timestamp: time.Now(), Severity: 1},
			},
		},
		{
			AgentID:    "agent-2",
			TenantID:   "tenant-1",
			BatchID:    "batch-2",
			Timestamp:  time.Now(),
			ReceivedAt: time.Now(),
			Events: []*SecurityEvent{
				{EventID: "e2", EventType: "test", Timestamp: time.Now(), Severity: 2},
			},
		},
	}

	err = producer.ProduceBatch(ctx, testEvents)
	require.NoError(t, err)

	// Create consumer
	consumerCfg := &KafkaConsumerConfig{
		Brokers:           []string{"localhost:19092"},
		Topic:             testTopic,
		GroupID:           testGroup,
		MinBytes:          1,
		MaxBytes:          10 * 1024 * 1024,
		MaxWait:           100 * time.Millisecond,
		CommitInterval:    100 * time.Millisecond,
		StartOffset:       kafka.FirstOffset,
		Concurrency:       1,
		MessageBufferSize: 100,
		ErrorBufferSize:   10,
	}

	consumer, err := NewKafkaConsumer(consumerCfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	err = consumer.Start(ctx)
	require.NoError(t, err)

	// Collect messages
	received := make([]*EventMessage, 0)
	timeout := time.After(30 * time.Second)

loop:
	for {
		select {
		case msg := <-consumer.Messages():
			received = append(received, msg)
			err = consumer.CommitMessages(ctx, msg)
			require.NoError(t, err)
			if len(received) >= 2 {
				break loop
			}
		case err := <-consumer.Errors():
			t.Logf("consumer error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for messages")
		}
	}

	assert.Len(t, received, 2)

	// Cleanup
	err = tm.DeleteTopic(ctx, testTopic)
	require.NoError(t, err)
}

func TestKafkaConsumer_Integration_ConsumeWithHandler(t *testing.T) {
	skipIfNoKafka(t)

	logger := zaptest.NewLogger(t)
	testTopic := "test.handler." + time.Now().Format("20060102150405")
	testGroup := "test-handler-group-" + time.Now().Format("20060102150405")

	// Create topic
	tmCfg := &TopicManagerConfig{
		Brokers: []string{"localhost:19092"},
		TopicConfigs: map[string]TopicDefinition{
			testTopic: {
				Partitions:        1,
				ReplicationFactor: 1,
				RetentionMs:       3600000,
				CleanupPolicy:     "delete",
			},
		},
		AutoCreate: true,
	}

	tm, err := NewTopicManager(tmCfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = tm.EnsureTopics(ctx)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Produce message
	producer := NewKafkaProducer([]string{"localhost:19092"}, testTopic, logger)
	defer producer.Close()

	testEvent := &EventMessage{
		AgentID:    "handler-test-agent",
		TenantID:   "tenant-1",
		BatchID:    "batch-handler",
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
	}

	err = producer.ProduceBatch(ctx, []*EventMessage{testEvent})
	require.NoError(t, err)

	// Create consumer
	consumerCfg := &KafkaConsumerConfig{
		Brokers:        []string{"localhost:19092"},
		Topic:          testTopic,
		GroupID:        testGroup,
		MinBytes:       1,
		MaxWait:        100 * time.Millisecond,
		CommitInterval: 100 * time.Millisecond,
		StartOffset:    kafka.FirstOffset,
		Concurrency:    1,
	}

	consumer, err := NewKafkaConsumer(consumerCfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	handlerCtx, handlerCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer handlerCancel()

	received := make(chan *EventMessage, 1)

	go func() {
		consumer.ConsumeWithHandler(handlerCtx, func(ctx context.Context, msg *EventMessage) error {
			received <- msg
			handlerCancel() // Stop after first message
			return nil
		})
	}()

	select {
	case msg := <-received:
		assert.Equal(t, "handler-test-agent", msg.AgentID)
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for handler message")
	}

	// Cleanup
	err = tm.DeleteTopic(context.Background(), testTopic)
	require.NoError(t, err)
}
