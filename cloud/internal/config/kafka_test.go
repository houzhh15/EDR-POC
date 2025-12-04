package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultKafkaConfig(t *testing.T) {
	cfg := DefaultKafkaConfig()

	assert.Equal(t, []string{"localhost:19092"}, cfg.Brokers)

	// Producer defaults
	assert.Equal(t, 100, cfg.Producer.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.Producer.BatchTimeout)
	assert.Equal(t, 3, cfg.Producer.MaxRetries)
	assert.Equal(t, "all", cfg.Producer.RequiredAcks)
	assert.Equal(t, "snappy", cfg.Producer.Compression)
	assert.Equal(t, 1048576, cfg.Producer.MaxMessageBytes)
	assert.Equal(t, 100*time.Millisecond, cfg.Producer.RetryBackoff)

	// Consumer defaults
	assert.Equal(t, 1024, cfg.Consumer.MinBytes)
	assert.Equal(t, 10*1024*1024, cfg.Consumer.MaxBytes)
	assert.Equal(t, 500*time.Millisecond, cfg.Consumer.MaxWait)
	assert.Equal(t, time.Second, cfg.Consumer.CommitInterval)
	assert.Equal(t, "latest", cfg.Consumer.StartOffset)
	assert.Equal(t, 30*time.Second, cfg.Consumer.SessionTimeout)
	assert.Equal(t, 3*time.Second, cfg.Consumer.HeartbeatInterval)

	// Topics
	assert.Equal(t, "edr.events.raw", cfg.Topics.EventsRaw.Name)
	assert.Equal(t, 12, cfg.Topics.EventsRaw.Partitions)
	assert.Equal(t, 1, cfg.Topics.EventsRaw.ReplicationFactor)

	// DLQ
	assert.True(t, cfg.DLQ.Enabled)
	assert.Equal(t, "edr.dlq", cfg.DLQ.Topic)
	assert.Equal(t, 3, cfg.DLQ.MaxRetries)

	// Health
	assert.Equal(t, 30*time.Second, cfg.Health.CheckInterval)
	assert.Equal(t, 5*time.Second, cfg.Health.Timeout)
}

func TestLoadKafkaConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "kafka.yaml")

	configContent := `
kafka:
  brokers:
    - "broker1:9092"
    - "broker2:9092"
  producer:
    batch_size: 200
    batch_timeout: 10s
    max_retries: 5
    required_acks: one
    compression: gzip
    max_message_bytes: 2097152
    retry_backoff: 200ms
  consumer:
    min_bytes: 2048
    max_bytes: 20971520
    max_wait: 1s
    commit_interval: 2s
    start_offset: earliest
    session_timeout: 60s
    heartbeat_interval: 5s
  topics:
    events_raw:
      name: "custom.events.raw"
      partitions: 24
      replication_factor: 3
      retention_ms: 1209600000
      cleanup_policy: delete
    events_normalized:
      name: "custom.events.normalized"
      partitions: 24
      replication_factor: 3
      retention_ms: 1209600000
      cleanup_policy: delete
    alerts:
      name: "custom.alerts"
      partitions: 12
      replication_factor: 3
      retention_ms: 5184000000
      cleanup_policy: delete
    commands:
      name: "custom.commands"
      partitions: 12
      replication_factor: 3
      retention_ms: 172800000
      cleanup_policy: delete
    dlq:
      name: "custom.dlq"
      partitions: 6
      replication_factor: 3
      retention_ms: 5184000000
      cleanup_policy: delete
  dlq:
    enabled: true
    topic: "custom.dlq"
    max_retries: 5
    retry_backoff: 2s
  health:
    check_interval: 60s
    timeout: 10s
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadKafkaConfig(configPath)
	require.NoError(t, err)

	// 验证加载的配置
	assert.Equal(t, []string{"broker1:9092", "broker2:9092"}, cfg.Brokers)
	assert.Equal(t, 200, cfg.Producer.BatchSize)
	assert.Equal(t, 10*time.Second, cfg.Producer.BatchTimeout)
	assert.Equal(t, 5, cfg.Producer.MaxRetries)
	assert.Equal(t, "one", cfg.Producer.RequiredAcks)
	assert.Equal(t, "gzip", cfg.Producer.Compression)
	assert.Equal(t, 2097152, cfg.Producer.MaxMessageBytes)
	assert.Equal(t, 200*time.Millisecond, cfg.Producer.RetryBackoff)

	assert.Equal(t, 2048, cfg.Consumer.MinBytes)
	assert.Equal(t, 20971520, cfg.Consumer.MaxBytes)
	assert.Equal(t, time.Second, cfg.Consumer.MaxWait)
	assert.Equal(t, 2*time.Second, cfg.Consumer.CommitInterval)
	assert.Equal(t, "earliest", cfg.Consumer.StartOffset)
	assert.Equal(t, 60*time.Second, cfg.Consumer.SessionTimeout)
	assert.Equal(t, 5*time.Second, cfg.Consumer.HeartbeatInterval)

	assert.Equal(t, "custom.events.raw", cfg.Topics.EventsRaw.Name)
	assert.Equal(t, 24, cfg.Topics.EventsRaw.Partitions)
	assert.Equal(t, 3, cfg.Topics.EventsRaw.ReplicationFactor)
	assert.Equal(t, int64(1209600000), cfg.Topics.EventsRaw.RetentionMs)

	assert.True(t, cfg.DLQ.Enabled)
	assert.Equal(t, "custom.dlq", cfg.DLQ.Topic)
	assert.Equal(t, 5, cfg.DLQ.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.DLQ.RetryBackoff)

	assert.Equal(t, 60*time.Second, cfg.Health.CheckInterval)
	assert.Equal(t, 10*time.Second, cfg.Health.Timeout)
}

func TestLoadKafkaConfig_FileNotFound(t *testing.T) {
	_, err := LoadKafkaConfig("/nonexistent/path/kafka.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read kafka config file")
}

func TestLoadKafkaConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "kafka.yaml")

	invalidYAML := `
kafka:
  brokers: [
    - broken yaml
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	_, err = LoadKafkaConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse kafka config file")
}

func TestLoadKafkaConfig_Defaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "kafka.yaml")

	minimalConfig := `
kafka:
  brokers:
    - "localhost:9092"
  topics:
    events_raw:
      name: "test.events.raw"
      partitions: 1
      replication_factor: 1
      retention_ms: 86400000
      cleanup_policy: delete
    events_normalized:
      name: "test.events.normalized"
      partitions: 1
      replication_factor: 1
      retention_ms: 86400000
      cleanup_policy: delete
    alerts:
      name: "test.alerts"
      partitions: 1
      replication_factor: 1
      retention_ms: 86400000
      cleanup_policy: delete
    commands:
      name: "test.commands"
      partitions: 1
      replication_factor: 1
      retention_ms: 86400000
      cleanup_policy: delete
    dlq:
      name: "test.dlq"
      partitions: 1
      replication_factor: 1
      retention_ms: 86400000
      cleanup_policy: delete
`
	err := os.WriteFile(configPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	cfg, err := LoadKafkaConfig(configPath)
	require.NoError(t, err)

	// 验证默认值被应用
	assert.Equal(t, 100, cfg.Producer.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.Producer.BatchTimeout)
	assert.Equal(t, "all", cfg.Producer.RequiredAcks)
	assert.Equal(t, "snappy", cfg.Producer.Compression)

	assert.Equal(t, 1024, cfg.Consumer.MinBytes)
	assert.Equal(t, "latest", cfg.Consumer.StartOffset)

	assert.Equal(t, "edr.dlq", cfg.DLQ.Topic)
	assert.Equal(t, 3, cfg.DLQ.MaxRetries)
}

func TestTopicsConfig_GetTopicByKey(t *testing.T) {
	cfg := DefaultKafkaConfig()

	tests := []struct {
		key      string
		expected string
		hasError bool
	}{
		{"events_raw", "edr.events.raw", false},
		{"edr.events.raw", "edr.events.raw", false},
		{"events_normalized", "edr.events.normalized", false},
		{"edr.events.normalized", "edr.events.normalized", false},
		{"alerts", "edr.alerts", false},
		{"edr.alerts", "edr.alerts", false},
		{"commands", "edr.commands", false},
		{"edr.commands", "edr.commands", false},
		{"dlq", "edr.dlq", false},
		{"edr.dlq", "edr.dlq", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			topic, err := cfg.Topics.GetTopicByKey(tt.key)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, topic.Name)
			}
		})
	}
}

func TestTopicsConfig_AllTopics(t *testing.T) {
	cfg := DefaultKafkaConfig()
	topics := cfg.Topics.AllTopics()

	assert.Len(t, topics, 5)

	topicNames := make(map[string]bool)
	for _, topic := range topics {
		topicNames[topic.Name] = true
	}

	assert.True(t, topicNames["edr.events.raw"])
	assert.True(t, topicNames["edr.events.normalized"])
	assert.True(t, topicNames["edr.alerts"])
	assert.True(t, topicNames["edr.commands"])
	assert.True(t, topicNames["edr.dlq"])
}

func TestKafkaConfigExt_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*KafkaConfigExt)
		wantErr string
	}{
		{
			name:    "valid config",
			modify:  func(c *KafkaConfigExt) {},
			wantErr: "",
		},
		{
			name: "empty brokers",
			modify: func(c *KafkaConfigExt) {
				c.Brokers = []string{}
			},
			wantErr: "brokers list cannot be empty",
		},
		{
			name: "invalid batch size",
			modify: func(c *KafkaConfigExt) {
				c.Producer.BatchSize = 0
			},
			wantErr: "producer.batch_size must be positive",
		},
		{
			name: "negative max retries",
			modify: func(c *KafkaConfigExt) {
				c.Producer.MaxRetries = -1
			},
			wantErr: "producer.max_retries cannot be negative",
		},
		{
			name: "invalid required_acks",
			modify: func(c *KafkaConfigExt) {
				c.Producer.RequiredAcks = "invalid"
			},
			wantErr: "producer.required_acks must be one of",
		},
		{
			name: "invalid compression",
			modify: func(c *KafkaConfigExt) {
				c.Producer.Compression = "invalid"
			},
			wantErr: "producer.compression must be one of",
		},
		{
			name: "invalid start_offset",
			modify: func(c *KafkaConfigExt) {
				c.Consumer.StartOffset = "invalid"
			},
			wantErr: "consumer.start_offset must be one of",
		},
		{
			name: "empty topic name",
			modify: func(c *KafkaConfigExt) {
				c.Topics.EventsRaw.Name = ""
			},
			wantErr: "topic name cannot be empty",
		},
		{
			name: "invalid partitions",
			modify: func(c *KafkaConfigExt) {
				c.Topics.EventsRaw.Partitions = 0
			},
			wantErr: "partitions must be positive",
		},
		{
			name: "invalid replication factor",
			modify: func(c *KafkaConfigExt) {
				c.Topics.Alerts.ReplicationFactor = 0
			},
			wantErr: "replication_factor must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultKafkaConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
