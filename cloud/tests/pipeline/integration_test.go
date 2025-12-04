// Package integration 提供事件处理管线的集成测试
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/enricher"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/writer"
)

// TestPipelineE2E_ProcessCreate 端到端测试：进程创建事件
func TestPipelineE2E_ProcessCreate(t *testing.T) {
	// 跳过如果没有设置集成测试环境
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建模拟写入器
	mockWriter := &mockCollectWriter{events: make([][]byte, 0)}

	// 创建配置
	cfg := &pipeline.PipelineConfig{
		Input: pipeline.InputConfig{
			Kafka: pipeline.KafkaInputConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test-events",
				ConsumerGroup: "test-group",
			},
		},
		Processing: pipeline.ProcessingConfig{
			BatchSize:    10,
			BatchTimeout: 100 * time.Millisecond,
			WorkerCount:  2,
		},
	}

	// 创建组件
	normalizer := pipeline.NewECSNormalizer(nil)
	metrics := pipeline.NewPipelineMetrics("test")

	// 创建批处理器
	processorCfg := &pipeline.BatchProcessorConfig{
		Workers:        2,
		BatchSize:      10,
		BatchTimeout:   100 * time.Millisecond,
		EnableParallel: true,
	}
	processor := pipeline.NewDefaultBatchProcessor(processorCfg, nil, normalizer, metrics)

	// 创建测试事件
	evt := &ecs.Event{
		ID:        "test-001",
		AgentID:   "agent-001",
		TenantID:  "tenant-001",
		EventType: "process_create",
		Timestamp: time.Now().UnixNano(),
		Process: &ecs.ProcessInfo{
			PID:         1234,
			PPID:        5678,
			Name:        "test.exe",
			Executable:  "C:\\test\\test.exe",
			CommandLine: "test.exe --arg1",
			User:        "testuser",
		},
	}

	// 创建批次
	batch := &pipeline.Batch{
		ID:        "batch-001",
		Events:    []*ecs.Event{evt},
		StartTime: time.Now(),
	}

	// 处理批次
	ctx := context.Background()
	result, err := processor.Process(ctx, batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// 验证结果
	if len(result.Events) != 1 {
		t.Errorf("len(Events) = %d, want 1", len(result.Events))
	}

	if len(result.FailedEvents) != 0 {
		t.Errorf("len(FailedEvents) = %d, want 0", len(result.FailedEvents))
	}

	// 验证 ECS 格式
	ecsEvent := result.Events[0]
	if ecsEvent.ECS.Version != "8.11.0" {
		t.Errorf("ECS.Version = %s, want 8.11.0", ecsEvent.ECS.Version)
	}
	if ecsEvent.Event.Kind != "event" {
		t.Errorf("Event.Kind = %s, want event", ecsEvent.Event.Kind)
	}
	if ecsEvent.Event.Category[0] != "process" {
		t.Errorf("Event.Category[0] = %s, want process", ecsEvent.Event.Category[0])
	}
	if ecsEvent.Process == nil {
		t.Fatal("Process is nil")
	}
	if ecsEvent.Process.PID != 1234 {
		t.Errorf("Process.PID = %d, want 1234", ecsEvent.Process.PID)
	}

	// 序列化并写入
	for _, e := range result.Events {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		mockWriter.Write(ctx, data)
	}

	// 验证写入
	if len(mockWriter.events) != 1 {
		t.Errorf("mockWriter.events = %d, want 1", len(mockWriter.events))
	}

	_ = cfg // 使用配置
}

// TestPipelineE2E_BatchProcessing 批量处理测试
func TestPipelineE2E_BatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建组件
	normalizer := pipeline.NewECSNormalizer(nil)
	metrics := pipeline.NewPipelineMetrics("test")

	processorCfg := &pipeline.BatchProcessorConfig{
		Workers:        4,
		BatchSize:      100,
		BatchTimeout:   100 * time.Millisecond,
		EnableParallel: true,
	}
	processor := pipeline.NewDefaultBatchProcessor(processorCfg, nil, normalizer, metrics)

	// 创建多个测试事件
	events := make([]*ecs.Event, 50)
	for i := 0; i < 50; i++ {
		events[i] = &ecs.Event{
			ID:        fmt.Sprintf("event-%03d", i),
			AgentID:   "agent-001",
			EventType: "process_create",
			Timestamp: time.Now().UnixNano(),
			Process: &ecs.ProcessInfo{
				PID:        int32(1000 + i),
				PPID:       1,
				Name:       "test.exe",
				Executable: "/usr/bin/test",
			},
		}
	}

	batch := &pipeline.Batch{
		ID:        "batch-002",
		Events:    events,
		StartTime: time.Now(),
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// 验证所有事件都被处理
	if len(result.Events) != 50 {
		t.Errorf("len(Events) = %d, want 50", len(result.Events))
	}

	// 验证处理时间合理
	if result.ProcessingTime > 5*time.Second {
		t.Errorf("ProcessingTime = %v, too slow", result.ProcessingTime)
	}
}

// TestPipelineE2E_WithEnrichers 带富化器的测试
func TestPipelineE2E_WithEnrichers(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建模拟富化器
	mockEnricher := &mockEnricher{
		enrichFn: func(ctx context.Context, evt *ecs.Event) error {
			if evt.Enrichment == nil {
				evt.Enrichment = &ecs.EnrichmentData{}
			}
			evt.Enrichment.Asset = &ecs.AssetInfo{
				Hostname:   "test-host",
				OSFamily:   "linux",
				Department: "IT",
			}
			return nil
		},
	}

	normalizer := pipeline.NewECSNormalizer(nil)
	processor := pipeline.NewDefaultBatchProcessor(nil, []enricher.Enricher{mockEnricher}, normalizer, nil)

	evt := &ecs.Event{
		ID:        "test-003",
		AgentID:   "agent-001",
		EventType: "process_create",
		Timestamp: time.Now().UnixNano(),
		Process: &ecs.ProcessInfo{
			PID:  1234,
			Name: "test.exe",
		},
	}

	batch := &pipeline.Batch{
		ID:        "batch-003",
		Events:    []*ecs.Event{evt},
		StartTime: time.Now(),
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(result.Events))
	}

	// 验证富化数据被应用
	ecsEvent := result.Events[0]
	if ecsEvent.Host.Hostname != "test-host" {
		t.Errorf("Host.Hostname = %s, want test-host", ecsEvent.Host.Hostname)
	}
}

// TestPipelineE2E_ErrorHandling 错误处理测试
func TestPipelineE2E_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建会失败的标准化器
	mockNorm := &mockNormalizer{
		normalizeFn: func(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
			if evt.ID == "fail-event" {
				return nil, fmt.Errorf("intentional error")
			}
			return &ecs.ECSEvent{
				ECS: ecs.ECSMeta{Version: "8.11.0"},
				Event: ecs.ECSEventMeta{
					ID:   evt.ID,
					Kind: "event",
				},
			}, nil
		},
	}

	processor := pipeline.NewDefaultBatchProcessor(nil, nil, mockNorm, nil)

	events := []*ecs.Event{
		{ID: "good-event-1", EventType: "process_create"},
		{ID: "fail-event", EventType: "process_create"},
		{ID: "good-event-2", EventType: "process_create"},
	}

	batch := &pipeline.Batch{
		ID:        "batch-004",
		Events:    events,
		StartTime: time.Now(),
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, batch)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// 应该有2个成功，1个失败
	if len(result.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(result.Events))
	}
	if len(result.FailedEvents) != 1 {
		t.Errorf("len(FailedEvents) = %d, want 1", len(result.FailedEvents))
	}
}

// 辅助类型
type mockCollectWriter struct {
	events [][]byte
}

func (m *mockCollectWriter) Write(ctx context.Context, event []byte) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockCollectWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *mockCollectWriter) Close() error {
	return nil
}

type mockEnricher struct {
	enrichFn func(ctx context.Context, evt *ecs.Event) error
}

func (m *mockEnricher) Name() string  { return "mock" }
func (m *mockEnricher) Enabled() bool { return true }
func (m *mockEnricher) Close() error  { return nil }
func (m *mockEnricher) Enrich(ctx context.Context, evt *ecs.Event) error {
	if m.enrichFn != nil {
		return m.enrichFn(ctx, evt)
	}
	return nil
}

type mockNormalizer struct {
	normalizeFn func(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error)
}

func (m *mockNormalizer) Normalize(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
	if m.normalizeFn != nil {
		return m.normalizeFn(ctx, evt)
	}
	return &ecs.ECSEvent{}, nil
}

func (m *mockNormalizer) SupportedTypes() []string {
	return []string{"process_create"}
}

var _ writer.Writer = (*mockCollectWriter)(nil)
var _ enricher.Enricher = (*mockEnricher)(nil)
var _ pipeline.Normalizer = (*mockNormalizer)(nil)
