package grpc

import (
"context"
"testing"
"time"

"go.uber.org/zap"
"google.golang.org/protobuf/types/known/timestamppb"

"github.com/houzhh15/EDR-POC/cloud/internal/grpc/interceptors"
pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// MockEventProducer 模拟 Kafka 生产者
type MockEventProducer struct {
	events []*pb.SecurityEvent
}

func (m *MockEventProducer) ProduceBatch(ctx context.Context, events []*pb.SecurityEvent) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *MockEventProducer) Close() error {
	return nil
}

// MockAgentStatusManager 模拟 Agent 状态管理器
type MockAgentStatusManager struct {
	agents map[string]bool
}

func NewMockAgentStatusManager() *MockAgentStatusManager {
	return &MockAgentStatusManager{
		agents: make(map[string]bool),
	}
}

func (m *MockAgentStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID, version, hostname, osType string) error {
	m.agents[agentID] = true
	return nil
}

func (m *MockAgentStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return m.agents[agentID], nil
}

// MockAssetRegistrar 模拟资产注册器
type MockAssetRegistrar struct{}

func (m *MockAssetRegistrar) RegisterOrUpdateFromHeartbeat(ctx context.Context, agentID, tenantID, hostname, osType, agentVersion string, ipAddresses []string) error {
	return nil
}

// MockPolicyStore 模拟策略存储
type MockPolicyStore struct {
	hasUpdate bool
	policies  []*pb.PolicyUpdate
}

func (m *MockPolicyStore) HasUpdate(ctx context.Context, tenantID string, currentVersion string) (bool, error) {
	return m.hasUpdate, nil
}

func (m *MockPolicyStore) GetPolicies(ctx context.Context, tenantID string) ([]*pb.PolicyUpdate, error) {
	return m.policies, nil
}

// MockCommandQueue 模拟命令队列
type MockCommandQueue struct {
	commands []*pb.Command
	results  map[string]*pb.CommandResult
}

func NewMockCommandQueue() *MockCommandQueue {
	return &MockCommandQueue{
		commands: make([]*pb.Command, 0),
		results:  make(map[string]*pb.CommandResult),
	}
}

func (m *MockCommandQueue) Dequeue(ctx context.Context, agentID string) (*pb.Command, error) {
	if len(m.commands) == 0 {
		return nil, nil
	}
	cmd := m.commands[0]
	m.commands = m.commands[1:]
	return cmd, nil
}

func (m *MockCommandQueue) Ack(ctx context.Context, commandID string, result *pb.CommandResult) error {
	m.results[commandID] = result
	return nil
}

func TestDefaultAgentServiceConfig(t *testing.T) {
	config := DefaultAgentServiceConfig()

	if config.EventBatchSize != 100 {
		t.Errorf("EventBatchSize = %d, want 100", config.EventBatchSize)
	}
	if config.EventFlushInterval != 5*time.Second {
		t.Errorf("EventFlushInterval = %v, want 5s", config.EventFlushInterval)
	}
	if config.HeartbeatTTL != 90*time.Second {
		t.Errorf("HeartbeatTTL = %v, want 90s", config.HeartbeatTTL)
	}
	if config.HeartbeatInterval != 30 {
		t.Errorf("HeartbeatInterval = %d, want 30", config.HeartbeatInterval)
	}
}

func TestNewAgentServiceServer(t *testing.T) {
	logger := zap.NewNop()
	producer := &MockEventProducer{}
	statusMgr := NewMockAgentStatusManager()
	assetRegistrar := &MockAssetRegistrar{}
	policyStore := &MockPolicyStore{}
	commandQueue := NewMockCommandQueue()

	server := NewAgentServiceServer(logger, producer, statusMgr, assetRegistrar, policyStore, commandQueue, nil)

	if server == nil {
		t.Fatal("NewAgentServiceServer returned nil")
	}
	if server.config == nil {
		t.Error("config should not be nil")
	}
	if server.config.EventBatchSize != 100 {
		t.Errorf("default EventBatchSize = %d, want 100", server.config.EventBatchSize)
	}
}

func TestNewAgentServiceServerWithCustomConfig(t *testing.T) {
	logger := zap.NewNop()
	config := &AgentServiceConfig{
		EventBatchSize:     50,
		EventFlushInterval: 2 * time.Second,
		HeartbeatTTL:       60 * time.Second,
		HeartbeatInterval:  20,
	}

	server := NewAgentServiceServer(logger, nil, nil, nil, nil, nil, config)

	if server.config.EventBatchSize != 50 {
		t.Errorf("EventBatchSize = %d, want 50", server.config.EventBatchSize)
	}
}

func TestHeartbeatWithoutAgentID(t *testing.T) {
	logger := zap.NewNop()
	server := NewAgentServiceServer(logger, nil, nil, nil, nil, nil, nil)

	// context 中没有 agent_id
	ctx := context.Background()
	_, err := server.Heartbeat(ctx, &pb.HeartbeatRequest{})

	if err == nil {
		t.Error("Heartbeat should fail without agent_id")
	}
}

func TestHeartbeatSuccess(t *testing.T) {
	logger := zap.NewNop()
	statusMgr := NewMockAgentStatusManager()
	policyStore := &MockPolicyStore{hasUpdate: true}

	server := NewAgentServiceServer(logger, nil, statusMgr, nil, policyStore, nil, nil)

	// 模拟认证后的 context
	ctx := context.WithValue(context.Background(), interceptors.AgentIDKey, "agent-123")
	ctx = context.WithValue(ctx, interceptors.TenantIDKey, "tenant-456")

	req := &pb.HeartbeatRequest{
		AgentVersion:         "1.0.0",
		Hostname:             "test-host",
		OsFamily:             "linux",
		CurrentPolicyVersion: "v1",
	}

	resp, err := server.Heartbeat(ctx, req)

	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
	if resp.HeartbeatInterval != 30 {
		t.Errorf("HeartbeatInterval = %d, want 30", resp.HeartbeatInterval)
	}
	if !resp.PolicyUpdateAvailable {
		t.Error("PolicyUpdateAvailable should be true")
	}
	if resp.ServerTime == nil {
		t.Error("ServerTime should not be nil")
	}
	if !statusMgr.agents["agent-123"] {
		t.Error("agent-123 should be in status manager")
	}
}

func TestFlushEvents(t *testing.T) {
	logger := zap.NewNop()
	producer := &MockEventProducer{}

	server := NewAgentServiceServer(logger, producer, nil, nil, nil, nil)

	events := []*pb.SecurityEvent{
		{EventId: "event-1", EventType: "process_create", Timestamp: timestamppb.Now()},
		{EventId: "event-2", EventType: "file_write", Timestamp: timestamppb.Now()},
	}

	err := server.flushEvents(context.Background(), events)

	if err != nil {
		t.Fatalf("flushEvents failed: %v", err)
	}
	if len(producer.events) != 2 {
		t.Errorf("producer.events length = %d, want 2", len(producer.events))
	}
}

func TestFlushEventsWithNilProducer(t *testing.T) {
	logger := zap.NewNop()
	server := NewAgentServiceServer(logger, nil, nil, nil, nil, nil, nil)

	events := []*pb.SecurityEvent{
		{EventId: "event-1", EventType: "process_create", Timestamp: timestamppb.Now()},
	}

	err := server.flushEvents(context.Background(), events)

	if err != nil {
		t.Errorf("flushEvents should not fail with nil producer: %v", err)
	}
}

func TestFlushEventsWithEmptyEvents(t *testing.T) {
	logger := zap.NewNop()
	producer := &MockEventProducer{}

	server := NewAgentServiceServer(logger, producer, nil, nil, nil, nil)

	err := server.flushEvents(context.Background(), []*pb.SecurityEvent{})

	if err != nil {
		t.Errorf("flushEvents should not fail with empty events: %v", err)
	}
	if len(producer.events) != 0 {
		t.Errorf("producer.events should be empty")
	}
}

func TestMockEventProducer(t *testing.T) {
	producer := &MockEventProducer{}

	events := []*pb.SecurityEvent{
		{EventId: "event-1"},
		{EventId: "event-2"},
	}

	if err := producer.ProduceBatch(context.Background(), events); err != nil {
		t.Fatalf("ProduceBatch failed: %v", err)
	}

	if len(producer.events) != 2 {
		t.Errorf("events length = %d, want 2", len(producer.events))
	}

	if err := producer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMockAgentStatusManager(t *testing.T) {
	mgr := NewMockAgentStatusManager()

	// 初始状态
	online, _ := mgr.IsOnline(context.Background(), "agent-123")
	if online {
		t.Error("agent-123 should not be online initially")
	}

	// 更新心跳
	if err := mgr.UpdateHeartbeat(context.Background(), "agent-123", "tenant-456", "1.0.0", "host", "linux"); err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	// 检查状态
	online, _ = mgr.IsOnline(context.Background(), "agent-123")
	if !online {
		t.Error("agent-123 should be online after UpdateHeartbeat")
	}
}

func TestMockPolicyStore(t *testing.T) {
	store := &MockPolicyStore{
		hasUpdate: true,
		policies: []*pb.PolicyUpdate{
			{PolicyId: "policy-1", Version: "v1"},
		},
	}

	hasUpdate, _ := store.HasUpdate(context.Background(), "tenant-123", "v0")
	if !hasUpdate {
		t.Error("HasUpdate should return true")
	}

	policies, _ := store.GetPolicies(context.Background(), "tenant-123")
	if len(policies) != 1 {
		t.Errorf("policies length = %d, want 1", len(policies))
	}
}

func TestMockCommandQueue(t *testing.T) {
	queue := NewMockCommandQueue()

	// 添加命令
	queue.commands = append(queue.commands, &pb.Command{
		CommandId:   "cmd-1",
		CommandType: "collect_logs",
	})

	// Dequeue
	cmd, _ := queue.Dequeue(context.Background(), "agent-123")
	if cmd == nil || cmd.CommandId != "cmd-1" {
		t.Error("Dequeue should return cmd-1")
	}

	// Dequeue 空队列
	cmd, _ = queue.Dequeue(context.Background(), "agent-123")
	if cmd != nil {
		t.Error("Dequeue should return nil for empty queue")
	}

	// Ack
	result := &pb.CommandResult{
		CommandId: "cmd-1",
		Status:    pb.CommandStatus_COMMAND_STATUS_SUCCESS,
	}
	if err := queue.Ack(context.Background(), "cmd-1", result); err != nil {
		t.Fatalf("Ack failed: %v", err)
	}

	if queue.results["cmd-1"] != result {
		t.Error("result should be stored in queue.results")
	}
}
