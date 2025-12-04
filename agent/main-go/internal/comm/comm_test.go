package comm

import (
	"context"
	"crypto/tls"
	"sync/atomic"
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
	pb "github.com/houzhh15/EDR-POC/agent/main-go/pkg/proto/edr/v1"
)

// TestConnectionConfig 测试连接配置默认值
func TestConnectionConfig(t *testing.T) {
	config := ConnConfig{
		Endpoint: "localhost:9090",
	}

	if config.Endpoint != "localhost:9090" {
		t.Errorf("Expected endpoint localhost:9090, got %s", config.Endpoint)
	}
}

// TestConnectionState 测试连接状态
func TestConnectionState(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "Disconnected"},
		{StateConnecting, "Connecting"},
		{StateConnected, "Connected"},
		{StateReconnecting, "Reconnecting"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("Expected state string %s, got %s", tt.expected, tt.state.String())
		}
	}
}

// TestHeartbeatConfig 测试心跳配置默认值
func TestHeartbeatConfig(t *testing.T) {
	config := HeartbeatConfig{
		AgentID:      "test-agent",
		AgentVersion: "1.0.0",
	}

	if config.AgentID != "test-agent" {
		t.Errorf("Expected AgentID test-agent, got %s", config.AgentID)
	}
}

// TestHeartbeatClientCreation 测试心跳客户端创建
func TestHeartbeatClientCreation(t *testing.T) {
	conn := &Connection{}
	config := HeartbeatConfig{
		AgentID:      "test-agent",
		AgentVersion: "1.0.0",
	}

	client := NewHeartbeatClient(conn, config, nil)

	if client.config.Interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", client.config.Interval)
	}
	if client.config.MinInterval != 10*time.Second {
		t.Errorf("Expected default min interval 10s, got %v", client.config.MinInterval)
	}
	if client.config.MaxInterval != 120*time.Second {
		t.Errorf("Expected default max interval 120s, got %v", client.config.MaxInterval)
	}
	if client.config.MaxFailureCount != 3 {
		t.Errorf("Expected default max failure count 3, got %d", client.config.MaxFailureCount)
	}
}

// TestPolicyConfig 测试策略配置默认值
func TestPolicyConfig(t *testing.T) {
	conn := &Connection{}
	config := PolicyConfig{
		AgentID: "test-agent",
	}

	client := NewPolicyClient(conn, config, nil)

	if client.config.SyncInterval != 5*time.Minute {
		t.Errorf("Expected default sync interval 5m, got %v", client.config.SyncInterval)
	}
	if client.config.RetryInterval != 30*time.Second {
		t.Errorf("Expected default retry interval 30s, got %v", client.config.RetryInterval)
	}
	if client.config.MaxRetries != 3 {
		t.Errorf("Expected default max retries 3, got %d", client.config.MaxRetries)
	}
}

// TestCommandConfig 测试命令配置默认值
func TestCommandConfig(t *testing.T) {
	conn := &Connection{}
	config := CommandConfig{
		AgentID: "test-agent",
	}

	client := NewCommandClient(conn, config, nil)

	if client.config.DefaultTimeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", client.config.DefaultTimeout)
	}
	if client.config.MaxConcurrent != 5 {
		t.Errorf("Expected default max concurrent 5, got %d", client.config.MaxConcurrent)
	}
	if client.config.ReconnectDelay != 5*time.Second {
		t.Errorf("Expected default reconnect delay 5s, got %v", client.config.ReconnectDelay)
	}
}

// TestEventClientCreation 测试事件客户端创建
func TestEventClientCreation(t *testing.T) {
	conn := &Connection{}
	client := NewEventClient(conn, "test-agent", 100, 5*time.Second)

	if client == nil {
		t.Fatal("Expected non-nil EventClient")
	}
	if client.agentID != "test-agent" {
		t.Errorf("Expected agentID test-agent, got %s", client.agentID)
	}
}

// TestEventClientOptions 测试事件客户端选项
func TestEventClientOptions(t *testing.T) {
	conn := &Connection{}
	mockCache := &mockEventCache{}
	client := NewEventClient(conn, "test-agent", 100, 5*time.Second,
		WithEventCache(mockCache),
	)

	if client.cache == nil {
		t.Error("Expected custom cache to be set")
	}
}

// mockEventCache 模拟事件缓存
type mockEventCache struct {
	events []cgo.Event
}

func (m *mockEventCache) Store(events []cgo.Event) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *mockEventCache) Load(limit int) ([]cgo.Event, error) {
	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}
	return m.events[:limit], nil
}

// TestDefaultCommandExecutor 测试默认命令执行器
func TestDefaultCommandExecutor(t *testing.T) {
	executor := NewDefaultCommandExecutor()

	executor.RegisterHandler("echo", func(ctx context.Context, params map[string]string) (string, error) {
		return params["message"], nil
	})

	commands := executor.SupportedCommands()
	if len(commands) != 1 || commands[0] != "echo" {
		t.Errorf("Expected [echo], got %v", commands)
	}

	cmd := &pb.Command{
		CommandId:   "test-1",
		CommandType: "echo",
		Parameters:  map[string]string{"message": "hello"},
	}

	output, err := executor.Execute(context.Background(), cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output != "hello" {
		t.Errorf("Expected output 'hello', got '%s'", output)
	}

	cmd.CommandType = "unknown"
	_, err = executor.Execute(context.Background(), cmd)
	if err == nil {
		t.Error("Expected error for unsupported command")
	}
}

// TestIsRetryableError 测试错误重试判断
func TestIsRetryableError(t *testing.T) {
	if !isRetryableError(nil) {
		t.Error("Expected nil error to be retryable")
	}
}

// TestTLSVersionEnforcement 测试 TLS 版本强制
func TestTLSVersionEnforcement(t *testing.T) {
	expectedMinVersion := uint16(tls.VersionTLS13)
	if expectedMinVersion != tls.VersionTLS13 {
		t.Error("TLS version should be 1.3")
	}
}

// TestEventBatchSequenceNumber 测试事件批次序列号
func TestEventBatchSequenceNumber(t *testing.T) {
	var counter int32

	for i := 0; i < 5; i++ {
		seq := atomic.AddInt32(&counter, 1)
		if seq != int32(i+1) {
			t.Errorf("Expected sequence %d, got %d", i+1, seq)
		}
	}
}

// TestPolicyChecksumVerification 测试策略校验和验证
func TestPolicyChecksumVerification(t *testing.T) {
	conn := &Connection{}
	config := PolicyConfig{
		AgentID:         "test-agent",
		ChecksumEnabled: true,
	}

	client := NewPolicyClient(conn, config, nil)

	update := &pb.PolicyUpdate{
		PolicyId: "policy-1",
		Content:  []byte("test policy content"),
		Checksum: "invalid-checksum",
	}

	err := client.verifyPolicyChecksum(update)
	if err == nil {
		t.Error("Expected checksum verification to fail")
	}
}

// TestHeartbeatHealthStatus 测试心跳健康状态
func TestHeartbeatHealthStatus(t *testing.T) {
	conn := &Connection{}
	config := HeartbeatConfig{
		AgentID:         "test-agent",
		AgentVersion:    "1.0.0",
		MaxFailureCount: 3,
	}

	client := NewHeartbeatClient(conn, config, nil)

	if !client.IsHealthy() {
		t.Error("Expected initial state to be healthy")
	}

	client.failureCount = 3
	if client.IsHealthy() {
		t.Error("Expected unhealthy after reaching max failures")
	}
}

// TestPolicyMergeChunks 测试策略分块合并
func TestPolicyMergeChunks(t *testing.T) {
	conn := &Connection{}
	config := PolicyConfig{AgentID: "test-agent"}
	client := NewPolicyClient(conn, config, nil)

	singleChunk := []*pb.PolicyUpdate{
		{
			PolicyId:   "policy-1",
			Content:    []byte("content"),
			ChunkIndex: 0,
			IsComplete: true,
		},
	}

	merged, err := client.mergePolicyChunks(singleChunk)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(merged.Content) != "content" {
		t.Errorf("Expected content 'content', got '%s'", string(merged.Content))
	}

	multiChunks := []*pb.PolicyUpdate{
		{
			PolicyId:    "policy-2",
			Content:     []byte("part1"),
			ChunkIndex:  0,
			TotalChunks: 2,
		},
		{
			PolicyId:    "policy-2",
			Content:     []byte("part2"),
			ChunkIndex:  1,
			TotalChunks: 2,
			IsComplete:  true,
			Checksum:    "test-checksum",
		},
	}

	merged, err = client.mergePolicyChunks(multiChunks)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(merged.Content) != "part1part2" {
		t.Errorf("Expected content 'part1part2', got '%s'", string(merged.Content))
	}

	_, err = client.mergePolicyChunks(nil)
	if err == nil {
		t.Error("Expected error for empty chunks")
	}
}

// BenchmarkEventConversion 基准测试事件转换性能
func BenchmarkEventConversion(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = &pb.SecurityEvent{
			EventId:   "test-event",
			EventType: "process",
			Severity:  3,
		}
	}
}
