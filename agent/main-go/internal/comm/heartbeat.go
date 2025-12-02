// Package comm 提供 Agent 与云端的通信功能
package comm

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/edr-project/edr-platform/agent/internal/log"
	pb "github.com/edr-project/edr-platform/agent/pkg/proto/edr/v1"
)

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	AgentID       string        // Agent ID
	AgentVersion  string        // Agent 版本
	Interval      time.Duration // 心跳间隔
	RetryInterval time.Duration // 重试间隔
}

// HeartbeatClient 心跳客户端
type HeartbeatClient struct {
	config HeartbeatConfig
	conn   *Connection
	logger *log.Logger

	mu            sync.RWMutex
	lastHeartbeat time.Time
	isRunning     bool
}

// NewHeartbeatClient 创建心跳客户端
func NewHeartbeatClient(conn *Connection, config HeartbeatConfig, logger *log.Logger) *HeartbeatClient {
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 5 * time.Second
	}

	return &HeartbeatClient{
		config: config,
		conn:   conn,
		logger: logger,
	}
}

// Start 启动心跳发送
func (h *HeartbeatClient) Start(ctx context.Context) {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return
	}
	h.isRunning = true
	h.mu.Unlock()

	h.logger.Info("Heartbeat client starting",
		zap.String("agent_id", h.config.AgentID),
		zap.Duration("interval", h.config.Interval),
	)

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	// 立即发送第一次心跳
	h.sendHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Heartbeat client stopped")
			h.mu.Lock()
			h.isRunning = false
			h.mu.Unlock()
			return
		case <-ticker.C:
			h.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat 发送心跳
func (h *HeartbeatClient) sendHeartbeat(ctx context.Context) {
	conn := h.conn.GetConn()
	if conn == nil {
		h.logger.Warn("Cannot send heartbeat: not connected")
		return
	}

	client := pb.NewAgentServiceClient(conn)

	// 获取主机信息
	hostname, _ := os.Hostname()

	req := &pb.HeartbeatRequest{
		AgentId:      h.config.AgentID,
		Hostname:     hostname,
		AgentVersion: h.config.AgentVersion,
		OsFamily:     runtime.GOOS,
		ClientTime:   timestamppb.Now(),
		Status:       pb.AgentStatus_AGENT_STATUS_RUNNING,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.Heartbeat(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			h.logger.Warn("Heartbeat failed",
				zap.String("code", st.Code().String()),
				zap.String("message", st.Message()),
			)
			// 如果是认证错误，可以尝试重新注册
			if st.Code() == codes.Unauthenticated {
				h.logger.Error("Agent not authenticated, may need to re-register")
			}
		} else {
			h.logger.Warn("Heartbeat failed", zap.Error(err))
		}
		return
	}

	h.mu.Lock()
	h.lastHeartbeat = time.Now()
	h.mu.Unlock()

	h.logger.Debug("Heartbeat sent successfully",
		zap.String("agent_id", h.config.AgentID),
		zap.Bool("success", resp.Success),
		zap.Int32("heartbeat_interval", resp.HeartbeatInterval),
	)
}

// LastHeartbeat 返回最后一次成功心跳的时间
func (h *HeartbeatClient) LastHeartbeat() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastHeartbeat
}

// IsRunning 返回心跳客户端是否正在运行
func (h *HeartbeatClient) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isRunning
}
