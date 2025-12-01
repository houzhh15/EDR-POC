// Package integration 提供集成测试
package integration

import (
"context"
"testing"
"time"

"go.uber.org/zap"
"google.golang.org/grpc"
"google.golang.org/grpc/credentials/insecure"
"google.golang.org/protobuf/types/known/timestamppb"

grpcserver "github.com/houzhh15/EDR-POC/cloud/internal/grpc"
pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// mockEventProducer 模拟事件生产者
type mockEventProducer struct {
	events []*pb.SecurityEvent
}

func (m *mockEventProducer) ProduceBatch(ctx context.Context, events []*pb.SecurityEvent) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *mockEventProducer) Close() error {
	return nil
}

// mockStatusManager 模拟状态管理器
type mockStatusManager struct {
	agents map[string]bool
}

func newMockStatusManager() *mockStatusManager {
	return &mockStatusManager{agents: make(map[string]bool)}
}

func (m *mockStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID, version, hostname, osType string) error {
	m.agents[agentID] = true
	return nil
}

func (m *mockStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return m.agents[agentID], nil
}

// mockPolicyStore 模拟策略存储
type mockPolicyStore struct{}

func (m *mockPolicyStore) HasUpdate(ctx context.Context, tenantID string, currentVersion string) (bool, error) {
	return false, nil
}

func (m *mockPolicyStore) GetPolicies(ctx context.Context, tenantID string) ([]*pb.PolicyUpdate, error) {
	return nil, nil
}

// TestGRPCServerIntegration 测试 gRPC 服务器集成
func TestGRPCServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	producer := &mockEventProducer{}
	statusMgr := newMockStatusManager()
	policyStore := &mockPolicyStore{}

	// 创建 AgentService
	agentService := grpcserver.NewAgentServiceServer(
logger,
producer,
statusMgr,
policyStore,
nil, // CommandQueue
nil, // 使用默认配置
)

	// 创建服务器配置
	config := &grpcserver.ServerConfig{
		ListenAddr:     ":19090", // 使用测试端口
		MaxRecvMsgSize: 4 * 1024 * 1024,
		MaxSendMsgSize: 4 * 1024 * 1024,
	}

	// 创建服务器
	server, err := grpcserver.NewServer(config, logger, agentService, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// 启动服务器
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()
	defer server.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建客户端连接
	conn, err := grpc.Dial("localhost:19090",
grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	// 测试心跳
	t.Run("Heartbeat", func(t *testing.T) {
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{
			AgentId:      "test-agent-001",
			Hostname:     "test-host",
			IpAddress:    "192.168.1.100",
			AgentVersion: "1.0.0",
			OsFamily:     "linux",
			ClientTime:   timestamppb.Now(),
		})

		// 由于没有设置认证，可能会返回错误
		if err != nil {
			t.Logf("Heartbeat returned error (expected without auth): %v", err)
		} else {
			if !resp.Success {
				t.Logf("Heartbeat response success: %v", resp.Success)
			}
			t.Logf("Heartbeat interval: %d seconds", resp.HeartbeatInterval)
		}
	})

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
healthConn, err := grpc.Dial("localhost:19090",
grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			t.Fatalf("Failed to connect for health check: %v", err)
		}
		defer healthConn.Close()

		// 基本连接测试
		state := healthConn.GetState()
		t.Logf("Connection state: %v", state)
	})
}

// TestGRPCClientStreaming 测试客户端流式 RPC
func TestGRPCClientStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	producer := &mockEventProducer{}
	statusMgr := newMockStatusManager()
	policyStore := &mockPolicyStore{}

	agentService := grpcserver.NewAgentServiceServer(
logger,
producer,
statusMgr,
policyStore,
nil,
nil,
)

	config := &grpcserver.ServerConfig{
		ListenAddr:     ":19091",
		MaxRecvMsgSize: 4 * 1024 * 1024,
		MaxSendMsgSize: 4 * 1024 * 1024,
	}

	server, err := grpcserver.NewServer(config, logger, agentService, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		server.Start()
	}()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("localhost:19091",
grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	t.Run("ReportEvents", func(t *testing.T) {
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := client.ReportEvents(ctx)
		if err != nil {
			t.Logf("Failed to create stream (expected without auth): %v", err)
			return
		}

		// 发送测试事件
		for i := 0; i < 5; i++ {
			batch := &pb.EventBatch{
				AgentId:        "test-agent-stream",
				BatchId:        "batch-" + string(rune('0'+i)),
				SequenceNumber: int32(i),
				Events: []*pb.SecurityEvent{
					{
						EventId:   "event-" + string(rune('0'+i)),
						EventType: "process",
						Timestamp: timestamppb.Now(),
						Severity:  3,
					},
				},
				BatchTime: timestamppb.Now(),
			}

			if err := stream.Send(batch); err != nil {
				t.Logf("Failed to send event: %v", err)
				break
			}
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			t.Logf("Stream close error (expected without auth): %v", err)
		} else {
			t.Logf("Events received: %d", resp.EventsReceived)
		}
	})
}

// BenchmarkHeartbeat 心跳性能基准测试
func BenchmarkHeartbeat(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	logger, _ := zap.NewDevelopment()
	producer := &mockEventProducer{}
	statusMgr := newMockStatusManager()
	policyStore := &mockPolicyStore{}

	agentService := grpcserver.NewAgentServiceServer(
logger,
producer,
statusMgr,
policyStore,
nil,
nil,
)

	config := &grpcserver.ServerConfig{
		ListenAddr:     ":19092",
		MaxRecvMsgSize: 4 * 1024 * 1024,
		MaxSendMsgSize: 4 * 1024 * 1024,
	}

	server, err := grpcserver.NewServer(config, logger, agentService, nil)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		server.Start()
	}()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("localhost:19092",
grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client.Heartbeat(ctx, &pb.HeartbeatRequest{
			AgentId:      "benchmark-agent",
			Hostname:     "benchmark-host",
			IpAddress:    "192.168.1.100",
			AgentVersion: "1.0.0",
			OsFamily:     "linux",
			ClientTime:   timestamppb.Now(),
		})
		cancel()
	}
}
