package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEventMessageSerialization(t *testing.T) {
	msg := &EventMessage{
		AgentID:   "agent-12345",
		TenantID:  "tenant-001",
		BatchID:   "batch-uuid-001",
		Timestamp: time.Now(),
		Events: []*SecurityEvent{
			{
				EventID:   "evt-001",
				EventType: "process",
				Timestamp: time.Now(),
				Severity:  3,
				ECSFields: map[string]string{
					"process.name": "cmd.exe",
					"process.pid":  "1234",
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal EventMessage: %v", err)
	}

	var decoded EventMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EventMessage: %v", err)
	}

	if decoded.AgentID != msg.AgentID {
		t.Errorf("AgentID mismatch: got %s, want %s", decoded.AgentID, msg.AgentID)
	}
	if decoded.TenantID != msg.TenantID {
		t.Errorf("TenantID mismatch: got %s, want %s", decoded.TenantID, msg.TenantID)
	}
	if len(decoded.Events) != len(msg.Events) {
		t.Errorf("Events count mismatch: got %d, want %d", len(decoded.Events), len(msg.Events))
	}
}

func TestSecurityEventSerialization(t *testing.T) {
	evt := &SecurityEvent{
		EventID:   "evt-001",
		EventType: "file",
		Timestamp: time.Now(),
		Severity:  5,
		ECSFields: map[string]string{
			"file.path": "/etc/passwd",
			"file.name": "passwd",
		},
		RawData: []byte("raw event data"),
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("failed to marshal SecurityEvent: %v", err)
	}

	var decoded SecurityEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SecurityEvent: %v", err)
	}

	if decoded.EventID != evt.EventID {
		t.Errorf("EventID mismatch: got %s, want %s", decoded.EventID, evt.EventID)
	}
	if decoded.EventType != evt.EventType {
		t.Errorf("EventType mismatch: got %s, want %s", decoded.EventType, evt.EventType)
	}
	if decoded.Severity != evt.Severity {
		t.Errorf("Severity mismatch: got %d, want %d", decoded.Severity, evt.Severity)
	}
}

func TestDefaultKafkaConfig(t *testing.T) {
	cfg := DefaultKafkaConfig()

	if len(cfg.Brokers) == 0 {
		t.Error("Brokers should not be empty")
	}
	if cfg.Topic != "edr.events.raw" {
		t.Errorf("Topic mismatch: got %s, want %s", cfg.Topic, "edr.events.raw")
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize mismatch: got %d, want %d", cfg.BatchSize, 100)
	}
	if cfg.BatchTimeout != 5*time.Second {
		t.Errorf("BatchTimeout mismatch: got %v, want %v", cfg.BatchTimeout, 5*time.Second)
	}
}

// MockProducer 模拟生产者用于测试
type MockProducer struct {
	Messages     []*EventMessage
	Closed       bool
	ErrOnProduce error
}

// NewMockProducer 创建模拟生产者
func NewMockProducer() *MockProducer {
	return &MockProducer{
		Messages: make([]*EventMessage, 0),
	}
}

// ProduceBatch 模拟批量写入
func (m *MockProducer) ProduceBatch(ctx context.Context, events []*EventMessage) error {
	if m.ErrOnProduce != nil {
		return m.ErrOnProduce
	}
	m.Messages = append(m.Messages, events...)
	return nil
}

// Close 模拟关闭
func (m *MockProducer) Close() error {
	m.Closed = true
	return nil
}

func TestMockProducer(t *testing.T) {
	mock := NewMockProducer()

	events := []*EventMessage{
		{AgentID: "agent-1", TenantID: "tenant-1"},
		{AgentID: "agent-2", TenantID: "tenant-1"},
	}

	err := mock.ProduceBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Messages) != 2 {
		t.Errorf("Messages count mismatch: got %d, want %d", len(mock.Messages), 2)
	}

	if err := mock.Close(); err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}

	if !mock.Closed {
		t.Error("MockProducer should be marked as closed")
	}
}

func TestProduceBatchEmpty(t *testing.T) {
	mock := NewMockProducer()
	err := mock.ProduceBatch(context.Background(), []*EventMessage{})
	if err != nil {
		t.Fatalf("empty batch should not return error: %v", err)
	}
	if len(mock.Messages) != 0 {
		t.Errorf("Messages should be empty for empty batch")
	}
}
