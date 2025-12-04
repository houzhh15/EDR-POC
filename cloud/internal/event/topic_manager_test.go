package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// skipIfNoKafka skips the test if Kafka is not available.
func skipIfNoKafka(t *testing.T) {
	if os.Getenv("KAFKA_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set KAFKA_INTEGRATION_TEST=true to run.")
	}
}

func TestDefaultTopicConfigs(t *testing.T) {
	configs := DefaultTopicConfigs()

	assert.Len(t, configs, 5)

	// edr.events.raw
	raw, ok := configs["edr.events.raw"]
	assert.True(t, ok)
	assert.Equal(t, 12, raw.Partitions)
	assert.Equal(t, 1, raw.ReplicationFactor)
	assert.Equal(t, int64(7*24*60*60*1000), raw.RetentionMs)
	assert.Equal(t, "delete", raw.CleanupPolicy)

	// edr.events.normalized
	normalized, ok := configs["edr.events.normalized"]
	assert.True(t, ok)
	assert.Equal(t, 12, normalized.Partitions)

	// edr.alerts
	alerts, ok := configs["edr.alerts"]
	assert.True(t, ok)
	assert.Equal(t, 6, alerts.Partitions)
	assert.Equal(t, int64(30*24*60*60*1000), alerts.RetentionMs)

	// edr.commands
	commands, ok := configs["edr.commands"]
	assert.True(t, ok)
	assert.Equal(t, 6, commands.Partitions)
	assert.Equal(t, int64(24*60*60*1000), commands.RetentionMs)

	// edr.dlq
	dlq, ok := configs["edr.dlq"]
	assert.True(t, ok)
	assert.Equal(t, 3, dlq.Partitions)
}

func TestDefaultTopicManagerConfig(t *testing.T) {
	cfg := DefaultTopicManagerConfig()

	assert.Equal(t, []string{"localhost:19092"}, cfg.Brokers)
	assert.True(t, cfg.AutoCreate)
	assert.Equal(t, 10*time.Second, cfg.DialTimeout)
	assert.Len(t, cfg.TopicConfigs, 5)
}

func TestNewTopicManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *TopicManagerConfig
		logger  *zap.Logger
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			logger:  zap.NewNop(),
			wantErr: false,
		},
		{
			name: "valid config",
			config: &TopicManagerConfig{
				Brokers:      []string{"localhost:9092"},
				TopicConfigs: DefaultTopicConfigs(),
				AutoCreate:   true,
			},
			logger:  zap.NewNop(),
			wantErr: false,
		},
		{
			name: "empty brokers",
			config: &TopicManagerConfig{
				Brokers: []string{},
			},
			logger:  zap.NewNop(),
			wantErr: true,
		},
		{
			name: "nil logger uses nop",
			config: &TopicManagerConfig{
				Brokers: []string{"localhost:9092"},
			},
			logger:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := NewTopicManager(tt.config, tt.logger)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tm)
				tm.Close()
			}
		})
	}
}

// Integration tests - require Kafka/Redpanda running

func TestTopicManager_Integration_CreateAndList(t *testing.T) {
	skipIfNoKafka(t)

	logger := zaptest.NewLogger(t)
	testTopicName := "test.integration." + time.Now().Format("20060102150405")

	cfg := &TopicManagerConfig{
		Brokers: []string{"localhost:19092"},
		TopicConfigs: map[string]TopicDefinition{
			testTopicName: {
				Partitions:        3,
				ReplicationFactor: 1,
				RetentionMs:       3600000, // 1 hour
				CleanupPolicy:     "delete",
			},
		},
		AutoCreate:  true,
		DialTimeout: 10 * time.Second,
	}

	tm, err := NewTopicManager(cfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create topic
	err = tm.CreateTopic(ctx, testTopicName, cfg.TopicConfigs[testTopicName])
	require.NoError(t, err)

	// Wait for topic to be available
	time.Sleep(2 * time.Second)

	// List topics
	topics, err := tm.ListTopics(ctx)
	require.NoError(t, err)

	found := false
	for _, topic := range topics {
		if topic == testTopicName {
			found = true
			break
		}
	}
	assert.True(t, found, "created topic should be in list")

	// Check topic exists
	exists, err := tm.TopicExists(ctx, testTopicName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Get topic metadata
	metadata, err := tm.GetTopicMetadata(ctx, testTopicName)
	require.NoError(t, err)
	assert.Equal(t, testTopicName, metadata.Name)
	assert.Equal(t, 3, metadata.Partitions)

	// Delete topic (cleanup)
	err = tm.DeleteTopic(ctx, testTopicName)
	require.NoError(t, err)

	// Verify deletion
	time.Sleep(2 * time.Second)
	exists, err = tm.TopicExists(ctx, testTopicName)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestTopicManager_Integration_EnsureTopics(t *testing.T) {
	skipIfNoKafka(t)

	logger := zaptest.NewLogger(t)
	testPrefix := "test.ensure." + time.Now().Format("20060102150405")

	cfg := &TopicManagerConfig{
		Brokers: []string{"localhost:19092"},
		TopicConfigs: map[string]TopicDefinition{
			testPrefix + ".topic1": {
				Partitions:        2,
				ReplicationFactor: 1,
				RetentionMs:       3600000,
				CleanupPolicy:     "delete",
			},
			testPrefix + ".topic2": {
				Partitions:        2,
				ReplicationFactor: 1,
				RetentionMs:       3600000,
				CleanupPolicy:     "delete",
			},
		},
		AutoCreate:  true,
		DialTimeout: 10 * time.Second,
	}

	tm, err := NewTopicManager(cfg, logger)
	require.NoError(t, err)
	defer func() {
		ctx := context.Background()
		tm.DeleteTopic(ctx, testPrefix+".topic1")
		tm.DeleteTopic(ctx, testPrefix+".topic2")
		tm.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure topics
	err = tm.EnsureTopics(ctx)
	require.NoError(t, err)

	// Wait for topics to be available
	time.Sleep(2 * time.Second)

	// Verify topics exist
	exists1, err := tm.TopicExists(ctx, testPrefix+".topic1")
	require.NoError(t, err)
	assert.True(t, exists1)

	exists2, err := tm.TopicExists(ctx, testPrefix+".topic2")
	require.NoError(t, err)
	assert.True(t, exists2)

	// Run EnsureTopics again - should be idempotent
	err = tm.EnsureTopics(ctx)
	require.NoError(t, err)
}

func TestTopicManager_Integration_AutoCreateDisabled(t *testing.T) {
	skipIfNoKafka(t)

	logger := zaptest.NewLogger(t)

	cfg := &TopicManagerConfig{
		Brokers: []string{"localhost:19092"},
		TopicConfigs: map[string]TopicDefinition{
			"test.no.create": {
				Partitions:        1,
				ReplicationFactor: 1,
				RetentionMs:       3600000,
				CleanupPolicy:     "delete",
			},
		},
		AutoCreate:  false, // Disabled
		DialTimeout: 10 * time.Second,
	}

	tm, err := NewTopicManager(cfg, logger)
	require.NoError(t, err)
	defer tm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// EnsureTopics should not create anything
	err = tm.EnsureTopics(ctx)
	require.NoError(t, err)

	// Topic should not exist
	exists, err := tm.TopicExists(ctx, "test.no.create")
	require.NoError(t, err)
	assert.False(t, exists)
}
