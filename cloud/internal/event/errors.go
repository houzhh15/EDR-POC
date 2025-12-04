// Package event 提供事件处理相关功能
package event

import (
	"errors"
	"fmt"
)

// 生产者错误
var (
	// ErrProducerClosed 生产者已关闭
	ErrProducerClosed = errors.New("producer is closed")
	// ErrBatchEmpty 批次为空
	ErrBatchEmpty = errors.New("batch is empty")
	// ErrMessageTooLarge 消息过大
	ErrMessageTooLarge = errors.New("message too large")
	// ErrTopicNotFound Topic 不存在
	ErrTopicNotFound = errors.New("topic not found")
)

// 消费者错误
var (
	// ErrConsumerClosed 消费者已关闭
	ErrConsumerClosed = errors.New("consumer is closed")
	// ErrCommitFailed 提交偏移量失败
	ErrCommitFailed = errors.New("commit failed")
	// ErrDeserializeFailed 反序列化失败
	ErrDeserializeFailed = errors.New("deserialization failed")
)

// 连接错误
var (
	// ErrBrokerUnavailable Broker 不可用
	ErrBrokerUnavailable = errors.New("broker unavailable")
	// ErrTimeout 操作超时
	ErrTimeout = errors.New("operation timeout")
)

// KafkaError 包装 Kafka 错误，提供更多上下文信息
type KafkaError struct {
	Op      string // 操作类型: produce, consume, commit, create_topic
	Topic   string // 相关 Topic
	Err     error  // 原始错误
	Retries int    // 已重试次数
}

// Error 实现 error 接口
func (e *KafkaError) Error() string {
	if e.Topic != "" {
		return fmt.Sprintf("kafka %s on topic %s: %v (retries: %d)",
			e.Op, e.Topic, e.Err, e.Retries)
	}
	return fmt.Sprintf("kafka %s: %v (retries: %d)", e.Op, e.Err, e.Retries)
}

// Unwrap 返回原始错误，支持 errors.Is 和 errors.As
func (e *KafkaError) Unwrap() error {
	return e.Err
}

// NewKafkaError 创建 KafkaError
func NewKafkaError(op, topic string, err error, retries int) *KafkaError {
	return &KafkaError{
		Op:      op,
		Topic:   topic,
		Err:     err,
		Retries: retries,
	}
}

// IsRetryable 判断错误是否可重试
// 网络错误、临时错误可重试；消息格式错误、认证错误不可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		// 已达到最大重试次数
		if kafkaErr.Retries >= 3 {
			return false
		}
	}

	// 不可重试的错误类型
	if errors.Is(err, ErrMessageTooLarge) ||
		errors.Is(err, ErrDeserializeFailed) ||
		errors.Is(err, ErrProducerClosed) ||
		errors.Is(err, ErrConsumerClosed) {
		return false
	}

	// 可重试的错误类型
	if errors.Is(err, ErrBrokerUnavailable) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrCommitFailed) {
		return true
	}

	// 默认认为可重试（网络错误等）
	return true
}

// WrapProduceError 包装生产者错误
func WrapProduceError(topic string, err error, retries int) error {
	return NewKafkaError("produce", topic, err, retries)
}

// WrapConsumeError 包装消费者错误
func WrapConsumeError(topic string, err error, retries int) error {
	return NewKafkaError("consume", topic, err, retries)
}

// WrapCommitError 包装提交错误
func WrapCommitError(topic string, err error, retries int) error {
	return NewKafkaError("commit", topic, err, retries)
}
