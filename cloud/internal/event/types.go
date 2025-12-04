// Package event 提供事件处理相关功能
package event

import (
	"time"

	"github.com/segmentio/kafka-go"
)

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
	Headers    *MessageHeaders  `json:"headers,omitempty"`

	// 内部字段（不序列化）- 用于消费者提交偏移量
	kafkaMsg  *kafka.Message `json:"-"`
	partition int            `json:"-"`
	offset    int64          `json:"-"`
}

// SetKafkaMessage 设置原始 Kafka 消息（供消费者使用）
func (e *EventMessage) SetKafkaMessage(msg *kafka.Message) {
	e.kafkaMsg = msg
	if msg != nil {
		e.partition = msg.Partition
		e.offset = msg.Offset
	}
}

// GetKafkaMessage 获取原始 Kafka 消息
func (e *EventMessage) GetKafkaMessage() *kafka.Message {
	return e.kafkaMsg
}

// GetPartition 获取分区号
func (e *EventMessage) GetPartition() int {
	return e.partition
}

// GetOffset 获取偏移量
func (e *EventMessage) GetOffset() int64 {
	return e.offset
}

// MessageHeaders Kafka 消息头
type MessageHeaders struct {
	TenantID      string `json:"tenant_id"`
	SchemaVersion string `json:"schema_version"`
	ContentType   string `json:"content_type"`
	TraceID       string `json:"trace_id,omitempty"`
	SourceService string `json:"source_service,omitempty"`
}

// ParseHeaders 从 kafka.Header 切片解析 MessageHeaders
func ParseHeaders(headers []kafka.Header) *MessageHeaders {
	h := &MessageHeaders{}
	for _, header := range headers {
		switch header.Key {
		case "tenant_id":
			h.TenantID = string(header.Value)
		case "schema_version":
			h.SchemaVersion = string(header.Value)
		case "content_type":
			h.ContentType = string(header.Value)
		case "trace_id":
			h.TraceID = string(header.Value)
		case "source_service":
			h.SourceService = string(header.Value)
		}
	}
	return h
}

// ToKafkaHeaders 将 MessageHeaders 转换为 kafka.Header 切片
func (h *MessageHeaders) ToKafkaHeaders() []kafka.Header {
	headers := []kafka.Header{
		{Key: "tenant_id", Value: []byte(h.TenantID)},
		{Key: "schema_version", Value: []byte(h.SchemaVersion)},
		{Key: "content_type", Value: []byte(h.ContentType)},
	}
	if h.TraceID != "" {
		headers = append(headers, kafka.Header{Key: "trace_id", Value: []byte(h.TraceID)})
	}
	if h.SourceService != "" {
		headers = append(headers, kafka.Header{Key: "source_service", Value: []byte(h.SourceService)})
	}
	return headers
}

// DefaultMessageHeaders 返回默认消息头
func DefaultMessageHeaders(tenantID string) *MessageHeaders {
	return &MessageHeaders{
		TenantID:      tenantID,
		SchemaVersion: "v1",
		ContentType:   "application/json",
	}
}
