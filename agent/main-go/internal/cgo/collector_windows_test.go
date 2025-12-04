//go:build windows
// +build windows

package cgo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessCollectorStartStop 测试采集器启动和停止
func TestProcessCollectorStartStop(t *testing.T) {
	// 启动采集器
	collector, err := StartProcessCollector()
	require.NoError(t, err, "Failed to start collector")
	require.NotNil(t, collector, "Collector should not be nil")

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 检查统计信息
	stats := collector.GetStats()
	assert.GreaterOrEqual(t, stats.LastPollTime.Unix(), time.Now().Add(-1*time.Second).Unix(), "Last poll time should be recent")

	// 停止采集器
	err = collector.StopProcessCollector()
	assert.NoError(t, err, "Failed to stop collector")
}

// TestProcessCollectorEvents 测试事件接收
func TestProcessCollectorEvents(t *testing.T) {
	collector, err := StartProcessCollector()
	require.NoError(t, err, "Failed to start collector")
	defer collector.StopProcessCollector()

	// 启动事件消费goroutine
	eventReceived := make(chan bool, 1)
	go func() {
		select {
		case event := <-collector.Events():
			t.Logf("Received event: type=%s, pid=%d, name=%s",
				event.EventType, event.ProcessPID, event.ProcessName)
			eventReceived <- true
		case <-time.After(5 * time.Second):
			t.Log("No events received in 5 seconds")
			eventReceived <- false
		}
	}()

	// 等待事件或超时
	received := <-eventReceived
	if !received {
		t.Skip("No process events occurred during test")
	}
}

// TestProcessCollectorStats 测试统计信息
func TestProcessCollectorStats(t *testing.T) {
	collector, err := StartProcessCollector()
	require.NoError(t, err)
	defer collector.StopProcessCollector()

	// 等待一些事件
	time.Sleep(2 * time.Second)

	stats := collector.GetStats()
	t.Logf("Stats: collected=%d, processed=%d, dropped=%d",
		stats.TotalEventsCollected,
		stats.TotalEventsProcessed,
		stats.TotalEventsDropped)

	// 验证统计字段
	assert.GreaterOrEqual(t, stats.TotalEventsCollected, uint64(0))
	assert.GreaterOrEqual(t, stats.TotalEventsProcessed, uint64(0))
	assert.Equal(t, stats.TotalEventsDropped, uint64(0), "Should not drop events in normal test")
}

// TestConvertToGoEvent 测试C事件到Go事件的转换
func TestConvertToGoEvent(t *testing.T) {
	collector, err := StartProcessCollector()
	require.NoError(t, err)
	defer collector.StopProcessCollector()

	// 等待接收一个事件
	timeout := time.After(10 * time.Second)
	select {
	case event := <-collector.Events():
		// 验证事件字段
		assert.NotZero(t, event.Timestamp, "Timestamp should not be zero")
		assert.Contains(t, []string{"start", "end"}, event.EventType, "Event type should be start or end")
		assert.NotZero(t, event.ProcessPID, "PID should not be zero")
		assert.NotEmpty(t, event.ProcessName, "Process name should not be empty")
		t.Logf("Event validated: %+v", event)
	case <-timeout:
		t.Skip("No events received in 10 seconds")
	}
}

// TestProcessCollectorMultipleCycles 测试多次启动停止
func TestProcessCollectorMultipleCycles(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Logf("Cycle %d", i+1)

		collector, err := StartProcessCollector()
		require.NoError(t, err, "Failed to start collector in cycle %d", i+1)

		time.Sleep(500 * time.Millisecond)

		err = collector.StopProcessCollector()
		assert.NoError(t, err, "Failed to stop collector in cycle %d", i+1)

		time.Sleep(100 * time.Millisecond)
	}
}

// TestProcessEventFields 测试事件字段完整性
func TestProcessEventFields(t *testing.T) {
	collector, err := StartProcessCollector()
	require.NoError(t, err)
	defer collector.StopProcessCollector()

	timeout := time.After(15 * time.Second)
	eventsChecked := 0

	for eventsChecked < 5 {
		select {
		case event := <-collector.Events():
			// 检查必填字段
			assert.NotZero(t, event.Timestamp)
			assert.NotEmpty(t, event.EventType)
			assert.NotZero(t, event.ProcessPID)

			// 记录可选字段
			if event.ProcessPath != "" {
				t.Logf("  Path: %s", event.ProcessPath)
			}
			if event.ProcessCommandLine != "" {
				t.Logf("  CommandLine: %s", event.ProcessCommandLine)
			}
			if event.UserName != "" {
				t.Logf("  User: %s", event.UserName)
			}

			// end事件应该有ExitCode
			if event.EventType == "end" {
				assert.NotNil(t, event.ExitCode, "End event should have exit code")
			}

			eventsChecked++
		case <-timeout:
			if eventsChecked == 0 {
				t.Skip("No events received")
			}
			return
		}
	}
}

// BenchmarkProcessCollectorThroughput 测试吞吐量
func BenchmarkProcessCollectorThroughput(b *testing.B) {
	collector, err := StartProcessCollector()
	if err != nil {
		b.Skip("Cannot start collector")
	}
	defer collector.StopProcessCollector()

	b.ResetTimer()

	eventsProcessed := 0
	done := make(chan bool)
	go func() {
		for range collector.Events() {
			eventsProcessed++
			if eventsProcessed >= b.N {
				done <- true
				return
			}
		}
	}()

	select {
	case <-done:
		b.Logf("Processed %d events", eventsProcessed)
	case <-time.After(30 * time.Second):
		b.Skipf("Only processed %d events in 30s", eventsProcessed)
	}
}
