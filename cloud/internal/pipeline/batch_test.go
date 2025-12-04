package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// mockNormalizer 模拟标准化器
type mockNormalizer struct {
	normalizeFn func(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error)
}

func (m *mockNormalizer) Normalize(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
	if m.normalizeFn != nil {
		return m.normalizeFn(ctx, evt)
	}
	return &ecs.ECSEvent{
		Event: ecs.ECSEventMeta{
			ID: evt.ID,
		},
	}, nil
}

func (m *mockNormalizer) SupportedTypes() []string {
	return []string{"process_create", "file_create"}
}

func TestDefaultBatchProcessor_Process(t *testing.T) {
	normalizer := &mockNormalizer{}

	processor := NewDefaultBatchProcessor(nil, nil, normalizer, nil)

	batch := &Batch{
		ID: "test-batch-001",
		Events: []*ecs.Event{
			{ID: "event-001", EventType: "process_create"},
			{ID: "event-002", EventType: "file_create"},
		},
		StartTime: time.Now(),
	}

	result, err := processor.Process(context.Background(), batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.BatchID != "test-batch-001" {
		t.Errorf("BatchID = %s, want test-batch-001", result.BatchID)
	}

	if len(result.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(result.Events))
	}

	if len(result.FailedEvents) != 0 {
		t.Errorf("len(FailedEvents) = %d, want 0", len(result.FailedEvents))
	}
}

func TestDefaultBatchProcessor_ProcessEmpty(t *testing.T) {
	normalizer := &mockNormalizer{}
	processor := NewDefaultBatchProcessor(nil, nil, normalizer, nil)

	result, err := processor.Process(context.Background(), &Batch{ID: "empty"})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Events) != 0 {
		t.Errorf("len(Events) = %d, want 0", len(result.Events))
	}
}

func TestDefaultBatchProcessor_ProcessWithErrors(t *testing.T) {
	normalizer := &mockNormalizer{
		normalizeFn: func(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
			if evt.ID == "event-002" {
				return nil, errors.New("normalize error")
			}
			return &ecs.ECSEvent{Event: ecs.ECSEventMeta{ID: evt.ID}}, nil
		},
	}

	processor := NewDefaultBatchProcessor(&BatchProcessorConfig{
		EnableParallel: false,
	}, nil, normalizer, nil)

	batch := &Batch{
		ID: "test-batch-002",
		Events: []*ecs.Event{
			{ID: "event-001", EventType: "process_create"},
			{ID: "event-002", EventType: "file_create"},
			{ID: "event-003", EventType: "process_create"},
		},
	}

	result, err := processor.Process(context.Background(), batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(result.Events))
	}

	if len(result.FailedEvents) != 1 {
		t.Errorf("len(FailedEvents) = %d, want 1", len(result.FailedEvents))
	}
}

func TestDefaultBatchProcessor_ProcessParallel(t *testing.T) {
	normalizer := &mockNormalizer{}

	processor := NewDefaultBatchProcessor(&BatchProcessorConfig{
		Workers:        4,
		EnableParallel: true,
	}, nil, normalizer, nil)

	// 创建大批量事件
	events := make([]*ecs.Event, 100)
	for i := 0; i < 100; i++ {
		events[i] = &ecs.Event{
			ID:        "event-" + string(rune('0'+i%10)),
			EventType: "process_create",
		}
	}

	batch := &Batch{
		ID:     "parallel-batch",
		Events: events,
	}

	result, err := processor.Process(context.Background(), batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Events) != 100 {
		t.Errorf("len(Events) = %d, want 100", len(result.Events))
	}
}

func TestDefaultBatchProcessor_ProcessWithCancel(t *testing.T) {
	normalizer := &mockNormalizer{
		normalizeFn: func(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
			// 模拟慢处理
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
			return &ecs.ECSEvent{Event: ecs.ECSEventMeta{ID: evt.ID}}, nil
		},
	}

	processor := NewDefaultBatchProcessor(&BatchProcessorConfig{
		EnableParallel: false,
	}, nil, normalizer, nil)

	events := make([]*ecs.Event, 10)
	for i := 0; i < 10; i++ {
		events[i] = &ecs.Event{ID: "event-" + string(rune('0'+i))}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	batch := &Batch{
		ID:     "cancel-batch",
		Events: events,
	}

	result, err := processor.Process(ctx, batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// 应该有部分事件被取消
	total := len(result.Events) + len(result.FailedEvents)
	if total != 10 {
		t.Errorf("total events = %d, want 10", total)
	}
}

func TestDefaultBatchProcessor_ProcessAsync(t *testing.T) {
	normalizer := &mockNormalizer{}
	processor := NewDefaultBatchProcessor(nil, nil, normalizer, nil)

	batch := &Batch{
		ID: "async-batch",
		Events: []*ecs.Event{
			{ID: "event-001", EventType: "process_create"},
		},
	}

	resultCh := processor.ProcessAsync(context.Background(), batch)

	select {
	case result := <-resultCh:
		if result.BatchID != "async-batch" {
			t.Errorf("BatchID = %s, want async-batch", result.BatchID)
		}
		if len(result.Events) != 1 {
			t.Errorf("len(Events) = %d, want 1", len(result.Events))
		}
	case <-time.After(time.Second):
		t.Fatal("ProcessAsync() timeout")
	}
}

func TestBatchCollector_Add(t *testing.T) {
	cfg := &BatchProcessorConfig{
		BatchSize:    3,
		BatchTimeout: time.Hour, // 设置长超时，只测试大小触发
	}
	collector := NewBatchCollector(cfg)

	// 添加不足BatchSize的事件
	batch := collector.Add(&ecs.Event{ID: "event-001"})
	if batch != nil {
		t.Error("Add() should return nil when buffer not full")
	}

	batch = collector.Add(&ecs.Event{ID: "event-002"})
	if batch != nil {
		t.Error("Add() should return nil when buffer not full")
	}

	// 添加达到BatchSize的事件
	batch = collector.Add(&ecs.Event{ID: "event-003"})
	if batch == nil {
		t.Fatal("Add() should return batch when buffer is full")
	}

	if len(batch.Events) != 3 {
		t.Errorf("len(Events) = %d, want 3", len(batch.Events))
	}

	// 缓冲区应该已清空
	if collector.Size() != 0 {
		t.Errorf("Size() = %d, want 0", collector.Size())
	}
}

func TestBatchCollector_Flush(t *testing.T) {
	cfg := &BatchProcessorConfig{
		BatchSize:    10,
		BatchTimeout: time.Hour,
	}
	collector := NewBatchCollector(cfg)

	collector.Add(&ecs.Event{ID: "event-001"})
	collector.Add(&ecs.Event{ID: "event-002"})

	batch := collector.Flush()
	if batch == nil {
		t.Fatal("Flush() should return batch")
	}

	if len(batch.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(batch.Events))
	}

	// 再次Flush应该返回nil
	batch = collector.Flush()
	if batch != nil {
		t.Error("Flush() should return nil for empty buffer")
	}
}

func TestBatchCollector_Size(t *testing.T) {
	cfg := &BatchProcessorConfig{
		BatchSize:    10,
		BatchTimeout: time.Hour,
	}
	collector := NewBatchCollector(cfg)

	if collector.Size() != 0 {
		t.Errorf("Size() = %d, want 0", collector.Size())
	}

	collector.Add(&ecs.Event{ID: "event-001"})
	if collector.Size() != 1 {
		t.Errorf("Size() = %d, want 1", collector.Size())
	}

	collector.Add(&ecs.Event{ID: "event-002"})
	if collector.Size() != 2 {
		t.Errorf("Size() = %d, want 2", collector.Size())
	}
}

func TestParseRawEvent(t *testing.T) {
	data := []byte(`{"id":"test-001","agent_id":"agent-001","event_type":"process_create"}`)

	evt, err := ParseRawEvent(data)
	if err != nil {
		t.Fatalf("ParseRawEvent() error = %v", err)
	}

	if evt.ID != "test-001" {
		t.Errorf("ID = %s, want test-001", evt.ID)
	}
	if evt.AgentID != "agent-001" {
		t.Errorf("AgentID = %s, want agent-001", evt.AgentID)
	}
	if evt.EventType != "process_create" {
		t.Errorf("EventType = %s, want process_create", evt.EventType)
	}
}

func TestParseRawEvent_Invalid(t *testing.T) {
	data := []byte(`invalid json`)

	_, err := ParseRawEvent(data)
	if err == nil {
		t.Error("ParseRawEvent() should return error for invalid JSON")
	}
}

func TestBatchProcessorConfig_Defaults(t *testing.T) {
	processor := NewDefaultBatchProcessor(nil, nil, &mockNormalizer{}, nil)

	if processor.config.Workers != 4 {
		t.Errorf("Workers = %d, want 4", processor.config.Workers)
	}
	if processor.config.BatchSize != 1000 {
		t.Errorf("BatchSize = %d, want 1000", processor.config.BatchSize)
	}
	if processor.config.BatchTimeout != 100*time.Millisecond {
		t.Errorf("BatchTimeout = %v, want 100ms", processor.config.BatchTimeout)
	}
}
