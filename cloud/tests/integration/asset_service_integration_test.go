// Package integration 提供集成测试
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/houzhh15/EDR-POC/cloud/internal/asset"
	internalgrpc "github.com/houzhh15/EDR-POC/cloud/internal/grpc"
	pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// AssetServiceIntegrationTestSuite 资产管理服务集成测试套件
// 本测试使用 gRPC Client SDK 模拟 Agent 调用，验证 Cloud 端资产管理服务的功能
// 注意：这不是真正的 E2E 测试，真正的 E2E 测试需要启动实际的 Agent 进程
type AssetServiceIntegrationTestSuite struct {
	suite.Suite
	redisClient *redis.Client
	server      *internalgrpc.Server
	grpcConn    *grpc.ClientConn
	agentClient pb.AgentServiceClient
	statusMgr   *asset.RedisAgentStatusManager
	logger      *zap.Logger
	serverAddr  string
}

// SetupSuite 初始化测试环境
func (s *AssetServiceIntegrationTestSuite) SetupSuite() {
	var err error

	// 1. 初始化 Logger
	s.logger, _ = zap.NewDevelopment()

	// 2. 连接 Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:16379"
	}
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.redisClient.Ping(ctx).Err()
	require.NoError(s.T(), err, "Redis连接失败，请确保Redis运行在 %s", redisAddr)

	// 3. 创建资产状态管理器
	s.statusMgr = asset.NewAgentStatusManager(s.redisClient, s.logger)

	// 4. 创建 AgentService 服务端（使用适配器包装 statusMgr）
	agentService := internalgrpc.NewAgentServiceServer(
		s.logger,
		&e2eMockEventProducer{},
		&agentStatusAdapter{mgr: s.statusMgr},
		nil, // assetRegistrar
		nil, // policyStore
		nil, // commandQueue
		nil, // 使用默认配置
	)

	// 5. 创建 gRPC 服务器
	serverConfig := &internalgrpc.ServerConfig{
		ListenAddr:     ":19091", // 测试端口
		MaxRecvMsgSize: 4 * 1024 * 1024,
		MaxSendMsgSize: 4 * 1024 * 1024,
	}
	s.server, err = internalgrpc.NewServer(serverConfig, s.logger, agentService, nil)
	require.NoError(s.T(), err)

	// 启动服务器
	go func() {
		if err := s.server.Start(); err != nil {
			s.logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// 等待服务启动
	time.Sleep(200 * time.Millisecond)

	s.serverAddr = "localhost:19091"

	// 6. 创建 gRPC 客户端（模拟 Agent）
	s.grpcConn, err = grpc.Dial(s.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(s.T(), err)
	s.agentClient = pb.NewAgentServiceClient(s.grpcConn)

	s.logger.Info("E2E测试环境初始化完成",
		zap.String("grpc_addr", s.serverAddr),
		zap.String("redis_addr", redisAddr),
	)
}

// TearDownSuite 清理测试环境
func (s *AssetServiceIntegrationTestSuite) TearDownSuite() {
	if s.grpcConn != nil {
		s.grpcConn.Close()
	}
	if s.server != nil {
		s.server.Stop()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
}

// SetupTest 每个测试前清理数据
func (s *AssetServiceIntegrationTestSuite) SetupTest() {
	ctx := context.Background()
	// 清理测试数据
	keys, _ := s.redisClient.Keys(ctx, "agent:status:test-*").Result()
	if len(keys) > 0 {
		s.redisClient.Del(ctx, keys...)
	}
	keys, _ = s.redisClient.Keys(ctx, "agents:online:*").Result()
	if len(keys) > 0 {
		s.redisClient.Del(ctx, keys...)
	}
}

// TestAgentHeartbeat_RegistersAsset 测试：Agent心跳注册资产
func (s *AssetServiceIntegrationTestSuite) TestAgentHeartbeat_RegistersAsset() {
	ctx := context.Background()
	agentID := "test-agent-001"

	// 模拟 Agent 发送心跳
	req := &pb.HeartbeatRequest{
		AgentId:              agentID,
		AgentVersion:         "1.0.0",
		Hostname:             "workstation-001",
		OsFamily:             "windows",
		CurrentPolicyVersion: "v1",
	}

	resp, err := s.agentClient.Heartbeat(ctx, req)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), resp.ServerTime)
	assert.Greater(s.T(), resp.HeartbeatInterval, int32(0))

	// 验证资产已注册到 Redis
	isOnline, err := s.statusMgr.IsOnline(context.Background(), agentID)
	require.NoError(s.T(), err)
	assert.True(s.T(), isOnline, "Agent应该被标记为在线")

	// 验证资产详情
	status, err := s.statusMgr.GetStatus(context.Background(), agentID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), status)
	assert.Equal(s.T(), agentID, status.AgentID)
	assert.Equal(s.T(), "workstation-001", status.Hostname)
	assert.Equal(s.T(), "1.0.0", status.AgentVersion)
	assert.Equal(s.T(), "windows", status.OSFamily)
}

// TestAgentHeartbeat_UpdatesExistingAsset 测试：心跳更新已有资产信息
func (s *AssetServiceIntegrationTestSuite) TestAgentHeartbeat_UpdatesExistingAsset() {
	ctx := context.Background()
	agentID := "test-agent-002"

	// 第一次心跳
	req1 := &pb.HeartbeatRequest{
		AgentId:      agentID,
		AgentVersion: "1.0.0",
		Hostname:     "old-hostname",
		OsFamily:     "linux",
	}
	_, err := s.agentClient.Heartbeat(ctx, req1)
	require.NoError(s.T(), err)

	// 第二次心跳（更新信息）
	req2 := &pb.HeartbeatRequest{
		AgentId:      agentID,
		AgentVersion: "1.1.0", // 版本升级
		Hostname:     "new-hostname",
		OsFamily:     "linux",
	}
	_, err = s.agentClient.Heartbeat(ctx, req2)
	require.NoError(s.T(), err)

	// 验证资产信息已更新
	status, err := s.statusMgr.GetStatus(context.Background(), agentID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "1.1.0", status.AgentVersion)
	assert.Equal(s.T(), "new-hostname", status.Hostname)
}

// TestMultipleAgents_OnlineCount 测试：多Agent在线统计
func (s *AssetServiceIntegrationTestSuite) TestMultipleAgents_OnlineCount() {
	tenantID := "tenant-multi"

	// 注册多个 Agent（直接通过 statusMgr 设置，因为 gRPC 不启用认证时 metadata 不会解析到 context）
	for i := 1; i <= 5; i++ {
		agentID := fmt.Sprintf("test-agent-multi-%d", i)
		info := &asset.HeartbeatInfo{
			AgentVersion: "1.0.0",
			Hostname:     fmt.Sprintf("host-%d", i),
			OSFamily:     "linux",
			Status:       "online",
		}
		err := s.statusMgr.UpdateHeartbeat(context.Background(), agentID, tenantID, info)
		require.NoError(s.T(), err)
	}

	// 验证在线数量
	count, err := s.statusMgr.CountOnlineAgents(context.Background(), tenantID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(5), count, "应该有5个Agent在线")
}

// TestAgentReportEvents_E2E 测试：Agent上报事件E2E流程
// 注意：此测试需要 mTLS 认证，在无证书环境下跳过
func (s *AssetServiceIntegrationTestSuite) TestAgentReportEvents_E2E() {
	// 流式 RPC 需要 mTLS 认证，当前测试环境无证书，跳过
	s.T().Skip("跳过流式RPC测试（需要mTLS认证，将在任务45中通过完整E2E环境测试）")
}

// TestAgentOffline_TTLExpiry 测试：Agent离线检测（TTL过期）
func (s *AssetServiceIntegrationTestSuite) TestAgentOffline_TTLExpiry() {
	ctx := context.Background()
	agentID := "test-agent-ttl"

	// 发送心跳
	_, err := s.agentClient.Heartbeat(ctx, &pb.HeartbeatRequest{
		AgentId:  agentID,
		Hostname: "ttl-host",
	})
	require.NoError(s.T(), err)

	// 验证TTL已设置
	ttl, err := s.redisClient.TTL(ctx, "agent:status:"+agentID).Result()
	require.NoError(s.T(), err)
	assert.Greater(s.T(), ttl.Seconds(), float64(0), "TTL应该大于0")
	assert.LessOrEqual(s.T(), ttl.Seconds(), float64(90), "TTL应该不超过90秒")
}

// TestConcurrentHeartbeats 测试：并发心跳处理
func (s *AssetServiceIntegrationTestSuite) TestConcurrentHeartbeats() {
	ctx := context.Background()
	numAgents := 20
	done := make(chan bool, numAgents)

	for i := 0; i < numAgents; i++ {
		go func(idx int) {
			req := &pb.HeartbeatRequest{
				AgentId:      fmt.Sprintf("test-concurrent-%d", idx),
				AgentVersion: "1.0.0",
				Hostname:     fmt.Sprintf("concurrent-host-%d", idx),
				OsFamily:     "linux",
			}
			_, err := s.agentClient.Heartbeat(ctx, req)
			done <- err == nil
		}(i)
	}

	// 等待所有请求完成
	successCount := 0
	for i := 0; i < numAgents; i++ {
		if <-done {
			successCount++
		}
	}
	assert.Equal(s.T(), numAgents, successCount, "所有并发心跳应该成功")
}

// === Mock 实现 ===

// e2eMockEventProducer 模拟 Kafka 生产者
type e2eMockEventProducer struct{}

func (m *e2eMockEventProducer) ProduceBatch(ctx context.Context, events []*pb.SecurityEvent) error {
	return nil
}

func (m *e2eMockEventProducer) Close() error {
	return nil
}

// agentStatusAdapter 适配器：将 asset.RedisAgentStatusManager 适配到 grpc.AgentStatusManager 接口
type agentStatusAdapter struct {
	mgr *asset.RedisAgentStatusManager
}

func (a *agentStatusAdapter) UpdateHeartbeat(ctx context.Context, agentID, tenantID, version, hostname, osType string) error {
	info := &asset.HeartbeatInfo{
		AgentVersion: version,
		Hostname:     hostname,
		OSFamily:     osType,
		Status:       "online",
	}
	return a.mgr.UpdateHeartbeat(ctx, agentID, tenantID, info)
}

func (a *agentStatusAdapter) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return a.mgr.IsOnline(ctx, agentID)
}

// === 测试入口 ===

func TestAgentAssetE2ESuite(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过E2E测试（需要Redis）")
	}
	suite.Run(t, new(AssetServiceIntegrationTestSuite))
}
