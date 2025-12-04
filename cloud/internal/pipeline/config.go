// Package pipeline provides the event processing pipeline service for EDR cloud.
package pipeline

import (
	"time"
)

// PipelineConfig 管线服务配置
type PipelineConfig struct {
	Input         InputConfig         `yaml:"input"`
	Processing    ProcessingConfig    `yaml:"processing"`
	Enrichment    EnrichmentConfig    `yaml:"enrichment"`
	Output        OutputConfig        `yaml:"output"`
	ErrorHandling ErrorConfig         `yaml:"error_handling"`
	Observability ObservabilityConfig `yaml:"observability"`
}

// InputConfig 输入配置
type InputConfig struct {
	Kafka KafkaInputConfig `yaml:"kafka"`
}

// KafkaInputConfig Kafka输入配置
type KafkaInputConfig struct {
	Brokers        []string      `yaml:"brokers"`
	Topic          string        `yaml:"topic"`
	ConsumerGroup  string        `yaml:"consumer_group"`
	Concurrency    int           `yaml:"concurrency"`
	MinBytes       int           `yaml:"min_bytes"`
	MaxBytes       int           `yaml:"max_bytes"`
	MaxWait        time.Duration `yaml:"max_wait"`
	CommitInterval time.Duration `yaml:"commit_interval"`
}

// ProcessingConfig 处理配置
type ProcessingConfig struct {
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	WorkerCount  int           `yaml:"worker_count"`
}

// EnrichmentConfig 丰富化配置
type EnrichmentConfig struct {
	GeoIP GeoIPConfig `yaml:"geoip"`
	Asset AssetConfig `yaml:"asset"`
	Agent AgentConfig `yaml:"agent"`
}

// GeoIPConfig GeoIP配置
type GeoIPConfig struct {
	Enabled      bool   `yaml:"enabled"`
	DatabasePath string `yaml:"database_path"`
}

// AssetConfig 资产配置
type AssetConfig struct {
	Enabled  bool          `yaml:"enabled"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
}

// AgentConfig Agent元数据配置
type AgentConfig struct {
	Enabled  bool          `yaml:"enabled"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	Kafka      KafkaOutputConfig      `yaml:"kafka"`
	OpenSearch OpenSearchOutputConfig `yaml:"opensearch"`
}

// KafkaOutputConfig Kafka输出配置
type KafkaOutputConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Brokers      []string      `yaml:"brokers"`
	Topic        string        `yaml:"topic"`
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
}

// OpenSearchOutputConfig OpenSearch输出配置
type OpenSearchOutputConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Addresses     []string      `yaml:"addresses"`
	IndexPrefix   string        `yaml:"index_prefix"`
	BulkSize      int           `yaml:"bulk_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	Username      string        `yaml:"username"`
	Password      string        `yaml:"password"`
	TLSEnabled    bool          `yaml:"tls_enabled"`
	TLSSkipVerify bool          `yaml:"tls_skip_verify"`
}

// ErrorConfig 错误处理配置
type ErrorConfig struct {
	MaxRetries    int           `yaml:"max_retries"`
	RetryBackoff  time.Duration `yaml:"retry_backoff"`
	MaxBackoff    time.Duration `yaml:"max_backoff"`
	BackoffFactor float64       `yaml:"backoff_factor"`
	DLQTopic      string        `yaml:"dlq_topic"`
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	MetricsPort    int    `yaml:"metrics_port"`
	MetricsPath    string `yaml:"metrics_path"`
	TracingEnabled bool   `yaml:"tracing_enabled"`
	OTLPEndpoint   string `yaml:"otlp_endpoint"`
	ServiceName    string `yaml:"service_name"`
}

// DefaultPipelineConfig 返回默认配置
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		Input: InputConfig{
			Kafka: KafkaInputConfig{
				Brokers:        []string{"localhost:19092"},
				Topic:          "edr.events.raw",
				ConsumerGroup:  "pipeline-processor",
				Concurrency:    10,
				MinBytes:       1024,             // 1KB
				MaxBytes:       10 * 1024 * 1024, // 10MB
				MaxWait:        500 * time.Millisecond,
				CommitInterval: time.Second,
			},
		},
		Processing: ProcessingConfig{
			BatchSize:    1000,
			BatchTimeout: 100 * time.Millisecond,
			WorkerCount:  10,
		},
		Enrichment: EnrichmentConfig{
			GeoIP: GeoIPConfig{
				Enabled:      true,
				DatabasePath: "/data/GeoLite2-City.mmdb",
			},
			Asset: AssetConfig{
				Enabled:  true,
				CacheTTL: 5 * time.Minute,
			},
			Agent: AgentConfig{
				Enabled:  true,
				CacheTTL: 5 * time.Minute,
			},
		},
		Output: OutputConfig{
			Kafka: KafkaOutputConfig{
				Enabled:      true,
				Brokers:      []string{"localhost:19092"},
				Topic:        "edr.events.normalized",
				BatchSize:    100,
				BatchTimeout: 5 * time.Second,
			},
			OpenSearch: OpenSearchOutputConfig{
				Enabled:       true,
				Addresses:     []string{"http://localhost:9200"},
				IndexPrefix:   "edr-events",
				BulkSize:      1000,
				FlushInterval: 100 * time.Millisecond,
				TLSEnabled:    false,
				TLSSkipVerify: true,
			},
		},
		ErrorHandling: ErrorConfig{
			MaxRetries:    3,
			RetryBackoff:  100 * time.Millisecond,
			MaxBackoff:    2 * time.Second,
			BackoffFactor: 2.0,
			DLQTopic:      "edr.dlq",
		},
		Observability: ObservabilityConfig{
			MetricsPort:    9091,
			MetricsPath:    "/metrics",
			TracingEnabled: true,
			OTLPEndpoint:   "localhost:4317",
			ServiceName:    "edr-pipeline",
		},
	}
}

// Validate 验证配置
func (c *PipelineConfig) Validate() error {
	if len(c.Input.Kafka.Brokers) == 0 {
		return &ConfigError{Field: "input.kafka.brokers", Message: "brokers list cannot be empty"}
	}
	if c.Input.Kafka.Topic == "" {
		return &ConfigError{Field: "input.kafka.topic", Message: "topic cannot be empty"}
	}
	if c.Input.Kafka.ConsumerGroup == "" {
		return &ConfigError{Field: "input.kafka.consumer_group", Message: "consumer_group cannot be empty"}
	}
	if c.Processing.BatchSize <= 0 {
		return &ConfigError{Field: "processing.batch_size", Message: "batch_size must be positive"}
	}
	if c.Processing.WorkerCount <= 0 {
		return &ConfigError{Field: "processing.worker_count", Message: "worker_count must be positive"}
	}
	return nil
}

// ConfigError 配置错误
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + ": " + e.Message
}
