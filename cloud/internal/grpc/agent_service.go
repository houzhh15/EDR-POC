package grpc

import (
"context"
"io"
"sync"
"time"

"go.uber.org/zap"
"google.golang.org/grpc/codes"
"google.golang.org/grpc/status"
"google.golang.org/protobuf/types/known/timestamppb"

"github.com/houzhh15/EDR-POC/cloud/internal/grpc/interceptors"
pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// EventProducer Kafka 事件生产者接口
type EventProducer interface {
	ProduceBatch(ctx context.Context, events []*pb.SecurityEvent) error
	Close() error
}

// AgentStatusManager Agent 状态管理器接口
type AgentStatusManager interface {
	UpdateHeartbeat(ctx context.Context, agentID, tenantID, version, hostname, osType string) error
	IsOnline(ctx context.Context, agentID string) (bool, error)
}

// PolicyStore 策略存储接口
type PolicyStore interface {
	HasUpdate(ctx context.Context, tenantID string, currentVersion string) (bool, error)
	GetPolicies(ctx context.Context, tenantID string) ([]*pb.PolicyUpdate, error)
}

// CommandQueue 命令队列接口
type CommandQueue interface {
	Dequeue(ctx context.Context, agentID string) (*pb.Command, error)
	Ack(ctx context.Context, commandID string, result *pb.CommandResult) error
}

// AgentServiceConfig 服务配置
type AgentServiceConfig struct {
	EventBatchSize     int           // 事件批量大小，默认 100
	EventFlushInterval time.Duration // 事件刷新间隔，默认 5s
	HeartbeatTTL       time.Duration // 心跳 TTL，默认 90s
	HeartbeatInterval  int32         // 建议心跳间隔，默认 30s
}

// DefaultAgentServiceConfig 默认配置
func DefaultAgentServiceConfig() *AgentServiceConfig {
	return &AgentServiceConfig{
		EventBatchSize:     100,
		EventFlushInterval: 5 * time.Second,
		HeartbeatTTL:       90 * time.Second,
		HeartbeatInterval:  30,
	}
}

// AgentServiceServer AgentService gRPC 服务实现
type AgentServiceServer struct {
	pb.UnimplementedAgentServiceServer

	logger       *zap.Logger
	producer     EventProducer
	statusMgr    AgentStatusManager
	policyStore  PolicyStore
	commandQueue CommandQueue
	config       *AgentServiceConfig
}

// NewAgentServiceServer 创建 AgentService 实例
func NewAgentServiceServer(
logger *zap.Logger,
producer EventProducer,
statusMgr AgentStatusManager,
policyStore PolicyStore,
commandQueue CommandQueue,
config *AgentServiceConfig,
) *AgentServiceServer {
	if config == nil {
		config = DefaultAgentServiceConfig()
	}
	return &AgentServiceServer{
		logger:       logger,
		producer:     producer,
		statusMgr:    statusMgr,
		policyStore:  policyStore,
		commandQueue: commandQueue,
		config:       config,
	}
}

// Heartbeat 处理 Agent 心跳
func (s *AgentServiceServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// 从 context 获取认证信息
	agentID := interceptors.GetAgentIDFromContext(ctx)
	tenantID := interceptors.GetTenantIDFromContext(ctx)

	if agentID == "" {
		return nil, status.Error(codes.Unauthenticated, "agent_id not found in context")
	}

	s.logger.Debug("heartbeat received",
zap.String("agent_id", agentID),
zap.String("tenant_id", tenantID),
zap.String("version", req.GetAgentVersion()),
		zap.String("hostname", req.GetHostname()),
	)

	// 更新 Redis 状态
	if s.statusMgr != nil {
		if err := s.statusMgr.UpdateHeartbeat(ctx, agentID, tenantID,
req.GetAgentVersion(), req.GetHostname(), req.GetOsFamily()); err != nil {
			s.logger.Error("failed to update heartbeat", zap.Error(err))
			// 不返回错误，允许继续
		}
	}

	// 检查策略更新
	policyUpdateAvailable := false
	if s.policyStore != nil {
		hasUpdate, err := s.policyStore.HasUpdate(ctx, tenantID, req.GetCurrentPolicyVersion())
		if err != nil {
			s.logger.Warn("failed to check policy update", zap.Error(err))
		} else {
			policyUpdateAvailable = hasUpdate
		}
	}

	return &pb.HeartbeatResponse{
		ServerTime:            timestamppb.Now(),
		HeartbeatInterval:     s.config.HeartbeatInterval,
		PolicyUpdateAvailable: policyUpdateAvailable,
	}, nil
}

// ReportEvents 接收 Agent 上报的安全事件（客户端流）
func (s *AgentServiceServer) ReportEvents(stream pb.AgentService_ReportEventsServer) error {
	ctx := stream.Context()
	agentID := interceptors.GetAgentIDFromContext(ctx)
	tenantID := interceptors.GetTenantIDFromContext(ctx)

	if agentID == "" {
		return status.Error(codes.Unauthenticated, "agent_id not found in context")
	}

	var (
buffer        = make([]*pb.SecurityEvent, 0, s.config.EventBatchSize)
totalReceived int32
totalAccepted int32
mu            sync.Mutex
flushTicker   = time.NewTicker(s.config.EventFlushInterval)
)
	defer flushTicker.Stop()

	// 忽略 tenantID 未使用警告
	_ = tenantID

	// 定时刷新 goroutine
	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		for {
			select {
			case <-ctx.Done():
				return
			case <-flushTicker.C:
				mu.Lock()
				if len(buffer) > 0 {
					if err := s.flushEvents(ctx, buffer); err != nil {
						s.logger.Error("failed to flush events", zap.Error(err))
					}
					buffer = buffer[:0]
				}
				mu.Unlock()
			}
		}
	}()

	// 接收事件循环
	for {
		batch, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.logger.Error("failed to receive event batch", zap.Error(err))
			return status.Error(codes.Internal, "failed to receive events")
		}

		for _, event := range batch.GetEvents() {
			totalReceived++

			// 验证事件必填字段
			if event.GetEventId() == "" || event.GetEventType() == "" || event.GetTimestamp() == nil {
				s.logger.Warn("invalid event: missing required fields",
zap.String("event_id", event.GetEventId()),
				)
				continue
			}

			mu.Lock()
			buffer = append(buffer, event)
			totalAccepted++

			// 达到批量大小则刷新
			if len(buffer) >= s.config.EventBatchSize {
				if err := s.flushEvents(ctx, buffer); err != nil {
					s.logger.Error("failed to flush events", zap.Error(err))
				}
				buffer = buffer[:0]
			}
			mu.Unlock()
		}
	}

	// 刷新剩余事件
	mu.Lock()
	if len(buffer) > 0 {
		if err := s.flushEvents(ctx, buffer); err != nil {
			s.logger.Error("failed to flush remaining events", zap.Error(err))
		}
	}
	mu.Unlock()

	s.logger.Info("event stream completed",
zap.String("agent_id", agentID),
zap.Int32("received", totalReceived),
zap.Int32("accepted", totalAccepted),
)

	return stream.SendAndClose(&pb.ReportResponse{
		Success:        true,
		EventsReceived: totalReceived,
		EventsAccepted: totalAccepted,
	})
}

// flushEvents 将事件写入 Kafka
func (s *AgentServiceServer) flushEvents(ctx context.Context, events []*pb.SecurityEvent) error {
	if s.producer == nil || len(events) == 0 {
		return nil
	}
	return s.producer.ProduceBatch(ctx, events)
}

// SyncPolicy 同步策略更新（服务端流）
func (s *AgentServiceServer) SyncPolicy(req *pb.PolicyRequest, stream pb.AgentService_SyncPolicyServer) error {
	ctx := stream.Context()
	tenantID := interceptors.GetTenantIDFromContext(ctx)

	if tenantID == "" {
		return status.Error(codes.Unauthenticated, "tenant_id not found in context")
	}

	if s.policyStore == nil {
		return status.Error(codes.Unavailable, "policy store not configured")
	}

	// 检查是否需要更新
	hasUpdate, err := s.policyStore.HasUpdate(ctx, tenantID, req.GetCurrentVersion())
	if err != nil {
		s.logger.Error("failed to check policy update", zap.Error(err))
		return status.Error(codes.Internal, "failed to check policy update")
	}

	if !hasUpdate {
		// 发送空更新表示无需更新
		return stream.Send(&pb.PolicyUpdate{
			Version: req.GetCurrentVersion(),
		})
	}

	// 获取策略
	policies, err := s.policyStore.GetPolicies(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to get policies", zap.Error(err))
		return status.Error(codes.Internal, "failed to get policies")
	}

	// 分块发送策略（每块 < 1MB）
	for _, policy := range policies {
		if err := stream.Send(policy); err != nil {
			s.logger.Error("failed to send policy update", zap.Error(err))
			return status.Error(codes.Internal, "failed to send policy")
		}
	}

	s.logger.Info("policy sync completed",
zap.String("tenant_id", tenantID),
zap.Int("policies_sent", len(policies)),
)

	return nil
}

// ExecuteCommand 执行命令（双向流）
func (s *AgentServiceServer) ExecuteCommand(stream pb.AgentService_ExecuteCommandServer) error {
	ctx := stream.Context()
	agentID := interceptors.GetAgentIDFromContext(ctx)

	if agentID == "" {
		return status.Error(codes.Unauthenticated, "agent_id not found in context")
	}

	if s.commandQueue == nil {
		return status.Error(codes.Unavailable, "command queue not configured")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// 命令下发 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				cmd, err := s.commandQueue.Dequeue(ctx, agentID)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					// 无命令时短暂等待
					time.Sleep(100 * time.Millisecond)
					continue
				}

				if cmd != nil {
					if err := stream.Send(cmd); err != nil {
						errChan <- err
						return
					}
					s.logger.Debug("command sent",
zap.String("agent_id", agentID),
zap.String("command_id", cmd.GetCommandId()),
					)
				}
			}
		}
	}()

	// 结果接收 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			result, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				errChan <- err
				return
			}

			// 确认命令执行结果
			if err := s.commandQueue.Ack(ctx, result.GetCommandId(), result); err != nil {
				s.logger.Error("failed to ack command result",
zap.Error(err),
zap.String("command_id", result.GetCommandId()),
				)
			}

			s.logger.Debug("command result received",
zap.String("agent_id", agentID),
zap.String("command_id", result.GetCommandId()),
				zap.String("status", result.GetStatus().String()),
			)
		}
	}()

	// 等待 goroutine 完成或出错
	go func() {
		wg.Wait()
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}

	return nil
}
