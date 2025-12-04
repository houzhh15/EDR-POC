package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestNewPipeline_NilConfig(t *testing.T) {
	_, err := NewPipeline(nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Error("NewPipeline() should return error for nil config")
	}
}

func TestPipeline_IsRunning(t *testing.T) {
	config := &PipelineConfig{
		Input: InputConfig{
			Kafka: KafkaInputConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test-topic",
				ConsumerGroup: "test-group",
			},
		},
		Processing: ProcessingConfig{
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
			WorkerCount:  2,
		},
	}

	normalizer := &mockNormalizer{}
	pipeline, err := NewPipeline(config, nil, normalizer, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	if pipeline.IsRunning() {
		t.Error("IsRunning() should return false before Start()")
	}
}

func TestPipeline_Stats(t *testing.T) {
	config := &PipelineConfig{
		Input: InputConfig{
			Kafka: KafkaInputConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test-topic",
				ConsumerGroup: "test-group",
			},
		},
		Processing: ProcessingConfig{
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
			WorkerCount:  2,
		},
	}

	normalizer := &mockNormalizer{}
	pipeline, err := NewPipeline(config, nil, normalizer, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	stats := pipeline.Stats()
	if stats == nil {
		t.Fatal("Stats() should not return nil")
	}
	if stats.Running {
		t.Error("Stats().Running should be false before Start()")
	}
	if stats.BufferSize != 0 {
		t.Errorf("Stats().BufferSize = %d, want 0", stats.BufferSize)
	}
}

func TestPipeline_HealthCheck(t *testing.T) {
	config := &PipelineConfig{
		Input: InputConfig{
			Kafka: KafkaInputConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test-topic",
				ConsumerGroup: "test-group",
			},
		},
		Processing: ProcessingConfig{
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
			WorkerCount:  2,
		},
	}

	normalizer := &mockNormalizer{}
	pipeline, err := NewPipeline(config, nil, normalizer, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	err = pipeline.HealthCheck()
	if err == nil {
		t.Error("HealthCheck() should return error when not running")
	}
}

func TestDLQMessage(t *testing.T) {
	msg := DLQMessage{
		OriginalData: []byte(`{"test":"data"}`),
		Reason:       "parse_error",
		Error:        "invalid json",
		Timestamp:    time.Now(),
	}

	if msg.Reason != "parse_error" {
		t.Errorf("Reason = %s, want parse_error", msg.Reason)
	}
	if msg.Error != "invalid json" {
		t.Errorf("Error = %s, want invalid json", msg.Error)
	}
}

func TestPipelineStats(t *testing.T) {
	stats := &PipelineStats{
		Running:     true,
		BufferSize:  100,
		ConsumerLag: 500,
	}

	if !stats.Running {
		t.Error("Running should be true")
	}
	if stats.BufferSize != 100 {
		t.Errorf("BufferSize = %d, want 100", stats.BufferSize)
	}
	if stats.ConsumerLag != 500 {
		t.Errorf("ConsumerLag = %d, want 500", stats.ConsumerLag)
	}
}

// mockPipelineWriter 模拟写入器
type mockPipelineWriter struct {
	writeCount int
	closed     bool
}

func (m *mockPipelineWriter) Write(ctx context.Context, event []byte) error {
	m.writeCount++
	return nil
}

func (m *mockPipelineWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	m.writeCount += len(events)
	return nil
}

func (m *mockPipelineWriter) Close() error {
	m.closed = true
	return nil
}
