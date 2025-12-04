// Package integration 提供事件处理管线的性能测试
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// BenchmarkBatchProcessor_Process 批处理器性能基准测试
func BenchmarkBatchProcessor_Process(b *testing.B) {
	normalizer := pipeline.NewECSNormalizer(nil)
	processor := pipeline.NewDefaultBatchProcessor(&pipeline.BatchProcessorConfig{
		Workers:        4,
		BatchSize:      1000,
		EnableParallel: true,
	}, nil, normalizer, nil)

	// 创建测试事件
	events := make([]*ecs.Event, 100)
	for i := 0; i < 100; i++ {
		events[i] = &ecs.Event{
			ID:        fmt.Sprintf("event-%d", i),
			AgentID:   "agent-001",
			EventType: "process_create",
			Timestamp: time.Now().UnixNano(),
			Process: &ecs.ProcessInfo{
				PID:         int32(1000 + i),
				PPID:        1,
				Name:        "test.exe",
				Executable:  "/usr/bin/test",
				CommandLine: "test --arg1 --arg2",
			},
		}
	}

	batch := &pipeline.Batch{
		ID:        "bench-batch",
		Events:    events,
		StartTime: time.Now(),
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := processor.Process(ctx, batch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNormalizer_Normalize 标准化器性能基准测试
func BenchmarkNormalizer_Normalize(b *testing.B) {
	normalizer := pipeline.NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "bench-event",
		AgentID:   "agent-001",
		EventType: "process_create",
		Timestamp: time.Now().UnixNano(),
		Process: &ecs.ProcessInfo{
			PID:         1234,
			PPID:        5678,
			Name:        "test.exe",
			Executable:  "/usr/bin/test",
			CommandLine: "test --arg1 --arg2",
			User:        "testuser",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := normalizer.Normalize(ctx, evt)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseRawEvent JSON解析性能基准测试
func BenchmarkParseRawEvent(b *testing.B) {
	data := []byte(`{
		"id": "bench-event",
		"agent_id": "agent-001",
		"tenant_id": "tenant-001",
		"event_type": "process_create",
		"timestamp": 1701619200000000000,
		"process": {
			"pid": 1234,
			"ppid": 5678,
			"name": "test.exe",
			"executable": "/usr/bin/test",
			"command_line": "test --arg1 --arg2",
			"user": "testuser"
		}
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := pipeline.ParseRawEvent(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestThroughput_BatchProcessor 吞吐量测试
func TestThroughput_BatchProcessor(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过吞吐量测试")
	}

	normalizer := pipeline.NewECSNormalizer(nil)
	processor := pipeline.NewDefaultBatchProcessor(&pipeline.BatchProcessorConfig{
		Workers:        8,
		BatchSize:      1000,
		EnableParallel: true,
	}, nil, normalizer, nil)

	// 创建 10000 个事件
	totalEvents := 10000
	events := make([]*ecs.Event, totalEvents)
	for i := 0; i < totalEvents; i++ {
		events[i] = &ecs.Event{
			ID:        fmt.Sprintf("event-%d", i),
			AgentID:   "agent-001",
			EventType: "process_create",
			Timestamp: time.Now().UnixNano(),
			Process: &ecs.ProcessInfo{
				PID:        int32(1000 + i%1000),
				PPID:       1,
				Name:       "test.exe",
				Executable: "/usr/bin/test",
			},
		}
	}

	batch := &pipeline.Batch{
		ID:        "throughput-batch",
		Events:    events,
		StartTime: time.Now(),
	}

	ctx := context.Background()

	start := time.Now()
	result, err := processor.Process(ctx, batch)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Events) != totalEvents {
		t.Errorf("len(Events) = %d, want %d", len(result.Events), totalEvents)
	}

	throughput := float64(totalEvents) / elapsed.Seconds()
	t.Logf("处理 %d 事件耗时 %v, 吞吐量 %.0f events/sec", totalEvents, elapsed, throughput)

	// 性能基准：应该能处理至少 10000 events/sec
	if throughput < 10000 {
		t.Logf("警告：吞吐量 %.0f events/sec 低于预期 10000 events/sec", throughput)
	}
}
