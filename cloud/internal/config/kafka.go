// Package config provides configuration structures and loaders for cloud services.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// KafkaFullConfig Kafka 完整配置（顶层包装）
type KafkaFullConfig struct {
	Kafka KafkaConfigExt `yaml:"kafka"`
}

// KafkaConfigExt 扩展 Kafka 配置
type KafkaConfigExt struct {
	Brokers  []string       `yaml:"brokers"`
	Producer ProducerConfig `yaml:"producer"`
	Consumer ConsumerConfig `yaml:"consumer"`
	Topics   TopicsConfig   `yaml:"topics"`
	DLQ      DLQConfig      `yaml:"dlq"`
	Health   HealthConfig   `yaml:"health"`
}

// ProducerConfig Kafka 生产者配置
type ProducerConfig struct {
	BatchSize       int           `yaml:"batch_size"`
	BatchTimeout    time.Duration `yaml:"batch_timeout"`
	MaxRetries      int           `yaml:"max_retries"`
	RequiredAcks    string        `yaml:"required_acks"` // all, one, none
	Compression     string        `yaml:"compression"`   // none, gzip, snappy, lz4, zstd
	MaxMessageBytes int           `yaml:"max_message_bytes"`
	RetryBackoff    time.Duration `yaml:"retry_backoff"`
}

// ConsumerConfig Kafka 消费者配置
type ConsumerConfig struct {
	MinBytes          int           `yaml:"min_bytes"`
	MaxBytes          int           `yaml:"max_bytes"`
	MaxWait           time.Duration `yaml:"max_wait"`
	CommitInterval    time.Duration `yaml:"commit_interval"`
	StartOffset       string        `yaml:"start_offset"` // earliest, latest
	SessionTimeout    time.Duration `yaml:"session_timeout"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

// TopicsConfig Topic 配置集合
type TopicsConfig struct {
	EventsRaw        TopicConfig `yaml:"events_raw"`
	EventsNormalized TopicConfig `yaml:"events_normalized"`
	Alerts           TopicConfig `yaml:"alerts"`
	Commands         TopicConfig `yaml:"commands"`
	DLQ              TopicConfig `yaml:"dlq"`
}

// TopicConfig 单个 Topic 配置
type TopicConfig struct {
	Name              string `yaml:"name"`
	Partitions        int    `yaml:"partitions"`
	ReplicationFactor int    `yaml:"replication_factor"`
	RetentionMs       int64  `yaml:"retention_ms"`
	CleanupPolicy     string `yaml:"cleanup_policy"`
}

// DLQConfig 死信队列配置
type DLQConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Topic        string        `yaml:"topic"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryBackoff time.Duration `yaml:"retry_backoff"`
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	CheckInterval time.Duration `yaml:"check_interval"`
	Timeout       time.Duration `yaml:"timeout"`
}

// DefaultKafkaConfig 返回默认 Kafka 配置
func DefaultKafkaConfig() *KafkaConfigExt {
	return &KafkaConfigExt{
		Brokers: []string{"localhost:19092"},
		Producer: ProducerConfig{
			BatchSize:       100,
			BatchTimeout:    5 * time.Second,
			MaxRetries:      3,
			RequiredAcks:    "all",
			Compression:     "snappy",
			MaxMessageBytes: 1048576,
			RetryBackoff:    100 * time.Millisecond,
		},
		Consumer: ConsumerConfig{
			MinBytes:          1024,
			MaxBytes:          10 * 1024 * 1024,
			MaxWait:           500 * time.Millisecond,
			CommitInterval:    time.Second,
			StartOffset:       "latest",
			SessionTimeout:    30 * time.Second,
			HeartbeatInterval: 3 * time.Second,
		},
		Topics: TopicsConfig{
			EventsRaw: TopicConfig{
				Name:              "edr.events.raw",
				Partitions:        12,
				ReplicationFactor: 1,
				RetentionMs:       7 * 24 * 60 * 60 * 1000,
				CleanupPolicy:     "delete",
			},
			EventsNormalized: TopicConfig{
				Name:              "edr.events.normalized",
				Partitions:        12,
				ReplicationFactor: 1,
				RetentionMs:       7 * 24 * 60 * 60 * 1000,
				CleanupPolicy:     "delete",
			},
			Alerts: TopicConfig{
				Name:              "edr.alerts",
				Partitions:        6,
				ReplicationFactor: 1,
				RetentionMs:       30 * 24 * 60 * 60 * 1000,
				CleanupPolicy:     "delete",
			},
			Commands: TopicConfig{
				Name:              "edr.commands",
				Partitions:        6,
				ReplicationFactor: 1,
				RetentionMs:       24 * 60 * 60 * 1000,
				CleanupPolicy:     "delete",
			},
			DLQ: TopicConfig{
				Name:              "edr.dlq",
				Partitions:        3,
				ReplicationFactor: 1,
				RetentionMs:       30 * 24 * 60 * 60 * 1000,
				CleanupPolicy:     "delete",
			},
		},
		DLQ: DLQConfig{
			Enabled:      true,
			Topic:        "edr.dlq",
			MaxRetries:   3,
			RetryBackoff: time.Second,
		},
		Health: HealthConfig{
			CheckInterval: 30 * time.Second,
			Timeout:       5 * time.Second,
		},
	}
}

// LoadKafkaConfig 从文件加载 Kafka 配置
func LoadKafkaConfig(path string) (*KafkaConfigExt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kafka config file: %w", err)
	}

	var fullConfig KafkaFullConfig
	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return nil, fmt.Errorf("failed to parse kafka config file: %w", err)
	}

	cfg := &fullConfig.Kafka

	// 设置默认值
	if len(cfg.Brokers) == 0 {
		cfg.Brokers = []string{"localhost:19092"}
	}
	if cfg.Producer.BatchSize == 0 {
		cfg.Producer.BatchSize = 100
	}
	if cfg.Producer.BatchTimeout == 0 {
		cfg.Producer.BatchTimeout = 5 * time.Second
	}
	if cfg.Producer.MaxRetries == 0 {
		cfg.Producer.MaxRetries = 3
	}
	if cfg.Producer.RequiredAcks == "" {
		cfg.Producer.RequiredAcks = "all"
	}
	if cfg.Producer.Compression == "" {
		cfg.Producer.Compression = "snappy"
	}
	if cfg.Producer.MaxMessageBytes == 0 {
		cfg.Producer.MaxMessageBytes = 1048576
	}
	if cfg.Producer.RetryBackoff == 0 {
		cfg.Producer.RetryBackoff = 100 * time.Millisecond
	}

	if cfg.Consumer.MinBytes == 0 {
		cfg.Consumer.MinBytes = 1024
	}
	if cfg.Consumer.MaxBytes == 0 {
		cfg.Consumer.MaxBytes = 10 * 1024 * 1024
	}
	if cfg.Consumer.MaxWait == 0 {
		cfg.Consumer.MaxWait = 500 * time.Millisecond
	}
	if cfg.Consumer.CommitInterval == 0 {
		cfg.Consumer.CommitInterval = time.Second
	}
	if cfg.Consumer.StartOffset == "" {
		cfg.Consumer.StartOffset = "latest"
	}
	if cfg.Consumer.SessionTimeout == 0 {
		cfg.Consumer.SessionTimeout = 30 * time.Second
	}
	if cfg.Consumer.HeartbeatInterval == 0 {
		cfg.Consumer.HeartbeatInterval = 3 * time.Second
	}

	if cfg.DLQ.Topic == "" {
		cfg.DLQ.Topic = "edr.dlq"
	}
	if cfg.DLQ.MaxRetries == 0 {
		cfg.DLQ.MaxRetries = 3
	}
	if cfg.DLQ.RetryBackoff == 0 {
		cfg.DLQ.RetryBackoff = time.Second
	}

	if cfg.Health.CheckInterval == 0 {
		cfg.Health.CheckInterval = 30 * time.Second
	}
	if cfg.Health.Timeout == 0 {
		cfg.Health.Timeout = 5 * time.Second
	}

	return cfg, nil
}

// GetTopicByKey 根据键名获取 Topic 配置
func (t *TopicsConfig) GetTopicByKey(key string) (*TopicConfig, error) {
	switch key {
	case "events_raw", "edr.events.raw":
		return &t.EventsRaw, nil
	case "events_normalized", "edr.events.normalized":
		return &t.EventsNormalized, nil
	case "alerts", "edr.alerts":
		return &t.Alerts, nil
	case "commands", "edr.commands":
		return &t.Commands, nil
	case "dlq", "edr.dlq":
		return &t.DLQ, nil
	default:
		return nil, fmt.Errorf("unknown topic key: %s", key)
	}
}

// AllTopics 返回所有 Topic 配置
func (t *TopicsConfig) AllTopics() []TopicConfig {
	return []TopicConfig{
		t.EventsRaw,
		t.EventsNormalized,
		t.Alerts,
		t.Commands,
		t.DLQ,
	}
}

// Validate 验证 Kafka 配置
func (c *KafkaConfigExt) Validate() error {
	if len(c.Brokers) == 0 {
		return fmt.Errorf("kafka: brokers list cannot be empty")
	}

	if c.Producer.BatchSize <= 0 {
		return fmt.Errorf("kafka: producer.batch_size must be positive")
	}
	if c.Producer.MaxRetries < 0 {
		return fmt.Errorf("kafka: producer.max_retries cannot be negative")
	}

	validAcks := map[string]bool{"all": true, "one": true, "none": true}
	if !validAcks[c.Producer.RequiredAcks] {
		return fmt.Errorf("kafka: producer.required_acks must be one of: all, one, none")
	}

	validCompression := map[string]bool{"none": true, "gzip": true, "snappy": true, "lz4": true, "zstd": true}
	if !validCompression[c.Producer.Compression] {
		return fmt.Errorf("kafka: producer.compression must be one of: none, gzip, snappy, lz4, zstd")
	}

	validOffset := map[string]bool{"earliest": true, "latest": true}
	if !validOffset[c.Consumer.StartOffset] {
		return fmt.Errorf("kafka: consumer.start_offset must be one of: earliest, latest")
	}

	// 验证 Topics
	for _, topic := range c.Topics.AllTopics() {
		if topic.Name == "" {
			return fmt.Errorf("kafka: topic name cannot be empty")
		}
		if topic.Partitions <= 0 {
			return fmt.Errorf("kafka: topic %s partitions must be positive", topic.Name)
		}
		if topic.ReplicationFactor <= 0 {
			return fmt.Errorf("kafka: topic %s replication_factor must be positive", topic.Name)
		}
	}

	return nil
}
