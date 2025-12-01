// Package event 提供事件处理相关功能
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"go.uber.org/zap"
)

// Producer Kafka 生产者接口
type Producer interface {
	// ProduceBatch 批量写入事件
	ProduceBatch(ctx context.Context, events []*EventMessage) error
	// Close 关闭连接
	Close() error
}

// SecurityEvent 安全事件结构
type SecurityEvent struct {
	EventID   string            `json:"event_id"`
	EventType string            `json:"event_type"`
	Timestamp time.Time         `json:"timestamp"`
	Severity  int               `json:"severity"`
	ECSFields map[string]string `json:"ecs_fields,omitempty"`
	RawData   []byte            `json:"raw_data,omitempty"`
}

// EventMessage 事件消息
type EventMessage struct {
	AgentID    string           `json:"agent_id"`
	TenantID   string           `json:"tenant_id"`
	BatchID    string           `json:"batch_id"`
	Events     []*SecurityEvent `json:"events"`
	Timestamp  time.Time        `json:"timestamp"`
	ReceivedAt time.Time        `json:"received_at"`
}

// KafkaProducer Kafka 生产者实现
type KafkaProducer struct {
	writer *kafka.Writer
	topic  string
	logger *zap.Logger
}

// KafkaProducerConfig Kafka 生产者配置
type KafkaProducerConfig struct {
	Brokers      []string      `json:"brokers" yaml:"brokers"`
	Topic        string        `json:"topic" yaml:"topic"`
	BatchSize    int           `json:"batch_size" yaml:"batch_size"`
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout"`
}

// DefaultKafkaConfig 返回默认配置
func DefaultKafkaConfig() *KafkaProducerConfig {
	return &KafkaProducerConfig{
		Brokers:      []string{"localhost:9092"},
		Topic:        "edr.events.raw",
		BatchSize:    100,
		BatchTimeout: 5 * time.Second,
	}
}

// NewKafkaProducer 创建 Kafka 生产者
func NewKafkaProducer(brokers []string, topic string, logger *zap.Logger) *KafkaProducer {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // 按 key 哈希分区，确保同一 Agent 事件有序
		RequiredAcks: kafka.RequireAll,
		BatchSize:    100,
		BatchTimeout: 5 * time.Second,
		Compression:  compress.Snappy,
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Debug(fmt.Sprintf(msg, args...), zap.String("component", "kafka"))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error(fmt.Sprintf(msg, args...), zap.String("component", "kafka"))
		}),
	}

	return &KafkaProducer{
		writer: writer,
		topic:  topic,
		logger: logger,
	}
}

// NewKafkaProducerWithConfig 使用配置创建 Kafka 生产者
func NewKafkaProducerWithConfig(cfg *KafkaProducerConfig, logger *zap.Logger) *KafkaProducer {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Compression:  compress.Snappy,
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Debug(fmt.Sprintf(msg, args...), zap.String("component", "kafka"))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error(fmt.Sprintf(msg, args...), zap.String("component", "kafka"))
		}),
	}

	return &KafkaProducer{
		writer: writer,
		topic:  cfg.Topic,
		logger: logger,
	}
}

// ProduceBatch 批量写入事件
func (p *KafkaProducer) ProduceBatch(ctx context.Context, events []*EventMessage) error {
	if len(events) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(events))

	for _, evt := range events {
		evt.ReceivedAt = time.Now()
		value, err := json.Marshal(evt)
		if err != nil {
			p.logger.Error("failed to marshal event",
				zap.String("agent_id", evt.AgentID),
				zap.String("batch_id", evt.BatchID),
				zap.Error(err),
			)
			continue
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(evt.AgentID), // 同一 Agent 事件有序
			Value: value,
			Headers: []kafka.Header{
				{Key: "tenant_id", Value: []byte(evt.TenantID)},
				{Key: "schema_version", Value: []byte("v1")},
				{Key: "content_type", Value: []byte("application/json")},
			},
		})
	}

	if len(messages) == 0 {
		return nil
	}

	// 带重试的写入
	return p.writeWithRetry(ctx, messages, 3)
}

// writeWithRetry 带重试的写入
func (p *KafkaProducer) writeWithRetry(ctx context.Context, messages []kafka.Message, maxRetries int) error {
	var lastErr error
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.writer.WriteMessages(ctx, messages...)
		if err == nil {
			p.logger.Debug("kafka write success",
				zap.Int("message_count", len(messages)),
				zap.Int("attempt", attempt+1),
			)
			return nil
		}

		lastErr = err
		p.logger.Warn("kafka write failed, retrying",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", maxRetries),
			zap.Duration("backoff", backoff),
		)

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(backoff):
		}

		// 指数退避，最大 2 秒
		backoff *= 2
		if backoff > 2*time.Second {
			backoff = 2 * time.Second
		}
	}

	return fmt.Errorf("kafka write failed after %d retries: %w", maxRetries, lastErr)
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			p.logger.Error("failed to close kafka writer", zap.Error(err))
			return fmt.Errorf("close kafka writer: %w", err)
		}
		p.logger.Info("kafka producer closed")
	}
	return nil
}
