// Package writer 提供事件输出能力，支持多种目标存储
package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Writer 输出接口
type Writer interface {
	// Write 写入单个事件
	Write(ctx context.Context, event []byte) error
	// WriteBatch 批量写入事件
	WriteBatch(ctx context.Context, events [][]byte) error
	// Close 关闭写入器
	Close() error
}

// KafkaWriterConfig Kafka写入配置
type KafkaWriterConfig struct {
	// Brokers Kafka broker地址列表
	Brokers []string `yaml:"brokers"`
	// Topic 目标主题
	Topic string `yaml:"topic"`
	// BatchSize 批量写入大小
	BatchSize int `yaml:"batch_size"`
	// BatchTimeout 批量写入超时
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	// RequiredAcks 确认级别: 0=不等待, 1=Leader确认, -1=所有副本确认
	RequiredAcks int `yaml:"required_acks"`
	// Compression 压缩方式: none, gzip, snappy, lz4, zstd
	Compression string `yaml:"compression"`
	// MaxRetries 最大重试次数
	MaxRetries int `yaml:"max_retries"`
	// RetryBackoff 重试间隔
	RetryBackoff time.Duration `yaml:"retry_backoff"`
}

// KafkaWriter Kafka输出实现
type KafkaWriter struct {
	writer *kafka.Writer
	config *KafkaWriterConfig
	mu     sync.Mutex
	closed bool
}

// NewKafkaWriter 创建Kafka写入器
func NewKafkaWriter(cfg *KafkaWriterConfig) (*KafkaWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka writer config is nil")
	}
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers is empty")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is empty")
	}

	// 设置默认值
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
	}

	// 解析压缩方式
	var compression kafka.Compression
	switch cfg.Compression {
	case "gzip":
		compression = kafka.Gzip
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "zstd":
		compression = kafka.Zstd
	default:
		compression = 0 // none
	}

	// 解析确认级别
	var requiredAcks kafka.RequiredAcks
	switch cfg.RequiredAcks {
	case 0:
		requiredAcks = kafka.RequireNone
	case 1:
		requiredAcks = kafka.RequireOne
	case -1:
		requiredAcks = kafka.RequireAll
	default:
		requiredAcks = kafka.RequireOne
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: requiredAcks,
		Compression:  compression,
		MaxAttempts:  cfg.MaxRetries,
	}

	return &KafkaWriter{
		writer: writer,
		config: cfg,
	}, nil
}

// Write 写入单个事件
func (w *KafkaWriter) Write(ctx context.Context, event []byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("kafka writer is closed")
	}
	w.mu.Unlock()

	msg := kafka.Message{
		Value: event,
		Time:  time.Now(),
	}

	return w.writer.WriteMessages(ctx, msg)
}

// WriteBatch 批量写入事件
func (w *KafkaWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("kafka writer is closed")
	}
	w.mu.Unlock()

	if len(events) == 0 {
		return nil
	}

	messages := make([]kafka.Message, len(events))
	now := time.Now()
	for i, event := range events {
		messages[i] = kafka.Message{
			Value: event,
			Time:  now,
		}
	}

	return w.writer.WriteMessages(ctx, messages...)
}

// Close 关闭写入器
func (w *KafkaWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	return w.writer.Close()
}

// Stats 返回写入统计信息
func (w *KafkaWriter) Stats() kafka.WriterStats {
	return w.writer.Stats()
}

// WriteWithKey 带分区键写入
func (w *KafkaWriter) WriteWithKey(ctx context.Context, key string, event []byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("kafka writer is closed")
	}
	w.mu.Unlock()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: event,
		Time:  time.Now(),
	}

	return w.writer.WriteMessages(ctx, msg)
}

// WriteBatchWithKeys 带分区键批量写入
func (w *KafkaWriter) WriteBatchWithKeys(ctx context.Context, events map[string][]byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("kafka writer is closed")
	}
	w.mu.Unlock()

	if len(events) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(events))
	now := time.Now()
	for key, event := range events {
		messages = append(messages, kafka.Message{
			Key:   []byte(key),
			Value: event,
			Time:  now,
		})
	}

	return w.writer.WriteMessages(ctx, messages...)
}

// MultiWriter 多目标写入器
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter 创建多目标写入器
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 写入单个事件到所有目标
func (m *MultiWriter) Write(ctx context.Context, event []byte) error {
	var errs []error
	for _, w := range m.writers {
		if err := w.Write(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multi-writer errors: %v", errs)
	}
	return nil
}

// WriteBatch 批量写入事件到所有目标
func (m *MultiWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	var errs []error
	for _, w := range m.writers {
		if err := w.WriteBatch(ctx, events); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multi-writer errors: %v", errs)
	}
	return nil
}

// Close 关闭所有写入器
func (m *MultiWriter) Close() error {
	var errs []error
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multi-writer close errors: %v", errs)
	}
	return nil
}

// EventMessage 事件消息结构（用于序列化）
type EventMessage struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}

// SerializeEvent 序列化事件为JSON
func SerializeEvent(event interface{}) ([]byte, error) {
	return json.Marshal(event)
}
