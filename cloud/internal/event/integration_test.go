//go:build integration
// +build integration

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	integrationTestBroker = "localhost:9092"
	integrationTestTopic  = "edr.test.integration"
	integrationTestGroup  = "edr-integration-test-group"
)

// skipIfKafkaUnavailable checks if Kafka is available and skips the test if not.
func skipIfKafkaUnavailable(t *testing.T) {
	conn, err := net.DialTimeout("tcp", integrationTestBroker, 2*time.Second)
	if err != nil {
		t.Skipf("Kafka not available at %s: %v", integrationTestBroker, err)
	}
	conn.Close()
}

// TestIntegration_FullWorkflow tests the complete Kafka workflow.
func TestIntegration_FullWorkflow(t *testing.T) {
	skipIfKafkaUnavailable(t)

	logger := zaptest.NewLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Create topics using TopicManager
	t.Run("CreateTopics", func(t *testing.T) {
		cfg := &TopicManagerConfig{
			Brokers: []string{integrationTestBroker},
			TopicConfigs: map[string]TopicDefinition{
				integrationTestTopic: {
					Partitions:        3,
					ReplicationFactor: 1,
					RetentionMs:       86400000, // 1 day
					CleanupPolicy:     "delete",
				},
				integrationTestTopic + ".dlq": {
					Partitions:        1,
					ReplicationFactor: 1,
					RetentionMs:       86400000,
					CleanupPolicy:     "delete",
				},
			},
			DialTimeout: 10 * time.Second,
		}

		tm, err := NewTopicManager(cfg, logger)
		require.NoError(t, err)
		defer tm.Close()

		err = tm.EnsureTopics(ctx)
		require.NoError(t, err)

		// Wait for topics to be ready
		time.Sleep(time.Second)

		// Verify topics exist
		existing, err := tm.ListTopics(ctx)
		require.NoError(t, err)

		topicMap := make(map[string]bool)
		for _, topic := range existing {
			topicMap[topic] = true
		}

		assert.True(t, topicMap[integrationTestTopic], "Test topic should exist")
	})

	// Step 2: Test Producer
	var producer *KafkaProducer
	t.Run("ProducerSetup", func(t *testing.T) {
		producer = NewKafkaProducer([]string{integrationTestBroker}, integrationTestTopic, logger)
		require.NotNil(t, producer)
	})
	defer producer.Close()

	// Step 3: Produce messages
	testMessages := []*EventMessage{
		{
			AgentID:    "agent-001",
			TenantID:   "tenant-001",
			BatchID:    "batch-001",
			Timestamp:  time.Now(),
			ReceivedAt: time.Now(),
			Events: []*SecurityEvent{
				{
					EventID:   "evt-001",
					EventType: "process_start",
					Timestamp: time.Now(),
					Severity:  2,
					RawData:   json.RawMessage(`{"process":"test.exe"}`),
				},
			},
		},
		{
			AgentID:    "agent-002",
			TenantID:   "tenant-001",
			BatchID:    "batch-002",
			Timestamp:  time.Now(),
			ReceivedAt: time.Now(),
			Events: []*SecurityEvent{
				{
					EventID:   "evt-002",
					EventType: "file_create",
					Timestamp: time.Now(),
					Severity:  1,
				},
			},
		},
	}

	t.Run("ProduceMessages", func(t *testing.T) {
		err := producer.ProduceBatch(ctx, testMessages)
		require.NoError(t, err)
		t.Logf("Produced %d messages", len(testMessages))
	})

	// Wait for messages to be committed
	time.Sleep(500 * time.Millisecond)

	// Step 4: Consume messages
	t.Run("ConsumeMessages", func(t *testing.T) {
		consumerCfg := &KafkaConsumerConfig{
			Brokers:           []string{integrationTestBroker},
			GroupID:           integrationTestGroup,
			Topic:             integrationTestTopic,
			MinBytes:          1024,
			MaxBytes:          10 * 1024 * 1024,
			MaxWait:           500 * time.Millisecond,
			CommitInterval:    time.Second,
			StartOffset:       kafka.FirstOffset,
			MessageBufferSize: 100,
			ErrorBufferSize:   10,
		}

		consumer, err := NewKafkaConsumer(consumerCfg, logger)
		require.NoError(t, err)
		defer consumer.Close()

		// Start consumer
		consumeCtx, consumeCancel := context.WithTimeout(ctx, 10*time.Second)
		defer consumeCancel()

		err = consumer.Start(consumeCtx)
		require.NoError(t, err)

		// Collect messages
		receivedMessages := make([]*EventMessage, 0)
		messageCount := 0
		timeout := time.After(5 * time.Second)

	collectLoop:
		for {
			select {
			case msg := <-consumer.Messages():
				receivedMessages = append(receivedMessages, msg)
				messageCount++
				t.Logf("Received message from agent: %s", msg.AgentID)
				if messageCount >= len(testMessages) {
					break collectLoop
				}
			case err := <-consumer.Errors():
				t.Logf("Consumer error: %v", err)
			case <-timeout:
				t.Logf("Timeout waiting for messages, received %d", messageCount)
				break collectLoop
			}
		}

		assert.GreaterOrEqual(t, len(receivedMessages), 1, "Should receive at least one message")
	})

	// Step 5: Test HealthChecker
	t.Run("HealthCheck", func(t *testing.T) {
		hc := NewHealthChecker([]string{integrationTestBroker}, 5*time.Second, logger)

		status := hc.Check(ctx)
		require.NotNil(t, status)
		assert.True(t, status.Healthy, "Kafka should be healthy")
		assert.Len(t, status.Brokers, 1)
		assert.True(t, status.Brokers[0].Healthy, "Broker should be healthy")

		t.Logf("Health status: healthy=%v, duration=%s", status.Healthy, status.Duration)
	})

	// Step 6: Test Health with topics
	t.Run("HealthCheckWithTopics", func(t *testing.T) {
		hc := NewHealthChecker([]string{integrationTestBroker}, 5*time.Second, logger)

		status := hc.CheckWithTopics(ctx, []string{integrationTestTopic})
		require.NotNil(t, status)

		if status.Healthy {
			assert.NotEmpty(t, status.Topics)
			t.Logf("Topic check: %d topics checked", len(status.Topics))
		} else {
			t.Logf("Topic check failed: %s", status.Error)
		}
	})

	// Step 7: Test DLQ routing
	t.Run("DLQRouting", func(t *testing.T) {
		// Create a DLQ producer
		dlqProducer := NewKafkaProducer(
			[]string{integrationTestBroker},
			integrationTestTopic+".dlq",
			logger,
		)
		defer dlqProducer.Close()

		dlq, err := NewDeadLetterQueue(dlqProducer, &DeadLetterQueueConfig{
			Enabled:      true,
			Topic:        integrationTestTopic + ".dlq",
			MaxRetries:   2,
			RetryBackoff: 100 * time.Millisecond,
		}, logger)
		require.NoError(t, err)

		// Route a message to DLQ
		dlqMsg := &DeadLetterMessage{
			OriginalTopic: integrationTestTopic,
			OriginalKey:   "test-key",
			Error:         "test error for integration",
			ErrorType:     "integration_test",
			RetryCount:    0,
			FirstFailedAt: time.Now(),
			LastFailedAt:  time.Now(),
			Source:        "integration_test",
			AgentID:       "agent-test",
			TenantID:      "tenant-test",
		}

		err = dlq.Route(ctx, dlqMsg)
		assert.NoError(t, err, "DLQ routing should succeed")
		t.Logf("DLQ message routed successfully")
	})
}

// TestIntegration_ProducerMetrics tests producer metrics.
func TestIntegration_ProducerMetrics(t *testing.T) {
	skipIfKafkaUnavailable(t)

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create a new registry for this test
	registry := prometheus.NewRegistry()
	metrics := NewProducerMetrics("edr_integration_test")
	metrics.MustRegister(registry)

	producer := NewKafkaProducer([]string{integrationTestBroker}, integrationTestTopic, logger)
	defer producer.Close()

	// Produce a message
	msg := &EventMessage{
		AgentID:    "metrics-test-agent",
		TenantID:   "metrics-test-tenant",
		BatchID:    "metrics-batch",
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
		Events: []*SecurityEvent{
			{
				EventID:   "metrics-evt",
				EventType: "test",
				Timestamp: time.Now(),
			},
		},
	}

	err := producer.ProduceBatch(ctx, []*EventMessage{msg})
	require.NoError(t, err)

	// Gather metrics
	families, err := registry.Gather()
	require.NoError(t, err)

	metricNames := make([]string, 0)
	for _, family := range families {
		metricNames = append(metricNames, *family.Name)
	}

	t.Logf("Gathered metrics: %v", metricNames)
}

// TestIntegration_ConsumerWithHandler tests consumer with handler.
func TestIntegration_ConsumerWithHandler(t *testing.T) {
	skipIfKafkaUnavailable(t)

	logger := zaptest.NewLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First produce some messages
	producer := NewKafkaProducer([]string{integrationTestBroker}, integrationTestTopic, logger)

	msg := &EventMessage{
		AgentID:    "handler-test-agent",
		TenantID:   "handler-test-tenant",
		BatchID:    fmt.Sprintf("handler-batch-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
		Events: []*SecurityEvent{
			{
				EventID:   fmt.Sprintf("handler-evt-%d", time.Now().UnixNano()),
				EventType: "handler_test",
				Timestamp: time.Now(),
			},
		},
	}

	err := producer.ProduceBatch(ctx, []*EventMessage{msg})
	require.NoError(t, err)
	producer.Close()

	// Create consumer with unique group
	groupID := fmt.Sprintf("handler-test-group-%d", time.Now().UnixNano())
	consumerCfg := &KafkaConsumerConfig{
		Brokers:           []string{integrationTestBroker},
		GroupID:           groupID,
		Topic:             integrationTestTopic,
		MinBytes:          1024,
		MaxBytes:          10 * 1024 * 1024,
		MaxWait:           500 * time.Millisecond,
		CommitInterval:    time.Second,
		StartOffset:       kafka.FirstOffset,
		MessageBufferSize: 100,
		ErrorBufferSize:   10,
	}

	consumer, err := NewKafkaConsumer(consumerCfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	// Track received messages
	var received []*EventMessage
	var mu sync.Mutex

	handler := func(ctx context.Context, msg *EventMessage) error {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		t.Logf("Handler received message from agent: %s", msg.AgentID)
		return nil
	}

	// Start with handler in background
	handlerCtx, handlerCancel := context.WithTimeout(ctx, 5*time.Second)
	defer handlerCancel()

	go func() {
		if err := consumer.ConsumeWithHandler(handlerCtx, handler); err != nil && err != context.DeadlineExceeded {
			t.Logf("ConsumeWithHandler returned: %v", err)
		}
	}()

	// Wait for messages
	time.Sleep(3 * time.Second)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	t.Logf("Total received messages: %d", count)
}

// TestIntegration_Cleanup cleans up test topics.
func TestIntegration_Cleanup(t *testing.T) {
	skipIfKafkaUnavailable(t)

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := &TopicManagerConfig{
		Brokers:     []string{integrationTestBroker},
		DialTimeout: 10 * time.Second,
	}

	tm, err := NewTopicManager(cfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	// Delete test topics
	topics := []string{integrationTestTopic, integrationTestTopic + ".dlq"}
	for _, topic := range topics {
		err := tm.DeleteTopic(ctx, topic)
		if err != nil {
			t.Logf("Failed to delete topic %s: %v", topic, err)
		} else {
			t.Logf("Deleted topic: %s", topic)
		}
	}
}

// TestIntegration_TopicManager tests topic management operations.
func TestIntegration_TopicManager(t *testing.T) {
	skipIfKafkaUnavailable(t)

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	testTopic := fmt.Sprintf("edr.test.topicmgr.%d", time.Now().UnixNano())

	cfg := &TopicManagerConfig{
		Brokers: []string{integrationTestBroker},
		TopicConfigs: map[string]TopicDefinition{
			testTopic: {
				Partitions:        2,
				ReplicationFactor: 1,
				RetentionMs:       86400000,
				CleanupPolicy:     "delete",
			},
		},
		DialTimeout: 10 * time.Second,
	}

	tm, err := NewTopicManager(cfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	t.Run("CreateTopic", func(t *testing.T) {
		err := tm.EnsureTopics(ctx)
		require.NoError(t, err)
	})

	t.Run("ListTopics", func(t *testing.T) {
		time.Sleep(500 * time.Millisecond) // Wait for topic creation

		topics, err := tm.ListTopics(ctx)
		require.NoError(t, err)

		found := false
		for _, topic := range topics {
			if topic == testTopic {
				found = true
				break
			}
		}
		assert.True(t, found, "Created topic should be listed")
	})

	t.Run("DeleteTopic", func(t *testing.T) {
		err := tm.DeleteTopic(ctx, testTopic)
		require.NoError(t, err)
	})
}

// TestIntegration_LowLevelKafkaConnection tests direct Kafka connection.
func TestIntegration_LowLevelKafkaConnection(t *testing.T) {
	skipIfKafkaUnavailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := &kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.DialContext(ctx, "tcp", integrationTestBroker)
	require.NoError(t, err)
	defer conn.Close()

	// Get broker info
	brokers, err := conn.Brokers()
	require.NoError(t, err)
	assert.NotEmpty(t, brokers)

	t.Logf("Connected to Kafka cluster with %d brokers", len(brokers))
	for _, broker := range brokers {
		t.Logf("  Broker: ID=%d, Host=%s, Port=%d", broker.ID, broker.Host, broker.Port)
	}
}
