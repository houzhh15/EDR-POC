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

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/log"
	pb "github.com/houzhh15/EDR-POC/agent/main-go/pkg/proto/edr/v1"
)

// PolicyUpdateCallback 策略更新回调函数类型
type PolicyUpdateCallback func(ctx context.Context) error

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	AgentID         string        // Agent ID
	AgentVersion    string        // Agent 版本
	Interval        time.Duration // 心跳间隔（初始值）
	MinInterval     time.Duration // 最小心跳间隔
	MaxInterval     time.Duration // 最大心跳间隔
	RetryInterval   time.Duration // 重试间隔
	MaxFailureCount int           // 最大连续失败次数（触发告警）
}

// HeartbeatClient 心跳客户端
type HeartbeatClient struct {
	config HeartbeatConfig
	conn   *Connection
	logger *log.Logger

	mu              sync.RWMutex
	lastHeartbeat   time.Time
	isRunning       bool
	currentInterval time.Duration // 当前动态心跳间隔
	failureCount    int           // 连续失败计数
	policyCallback  PolicyUpdateCallback
	stopCh          chan struct{} // 用于重置 ticker
}

// NewHeartbeatClient 创建心跳客户端
func NewHeartbeatClient(conn *Connection, config HeartbeatConfig, logger *log.Logger) *HeartbeatClient {
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}
	if config.MinInterval <= 0 {
		config.MinInterval = 10 * time.Second
	}
	if config.MaxInterval <= 0 {
		config.MaxInterval = 120 * time.Second
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 5 * time.Second
	}
	if config.MaxFailureCount <= 0 {
		config.MaxFailureCount = 3
	}

	return &HeartbeatClient{
		config:          config,
		conn:            conn,
		logger:          logger,
		currentInterval: config.Interval,
		stopCh:          make(chan struct{}),
	}
}

// SetPolicyCallback 设置策略更新回调
func (h *HeartbeatClient) SetPolicyCallback(callback PolicyUpdateCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.policyCallback = callback
}

// Start 启动心跳发送
func (h *HeartbeatClient) Start(ctx context.Context) {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return
	}
	h.isRunning = true
	h.stopCh = make(chan struct{})
	h.mu.Unlock()

	h.logger.Info("Heartbeat client starting",
		zap.String("agent_id", h.config.AgentID),
		zap.Duration("interval", h.currentInterval),
	)

	ticker := time.NewTicker(h.currentInterval)
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
		case <-h.stopCh:
			// 收到重置信号，更新 ticker 间隔
			h.mu.RLock()
			newInterval := h.currentInterval
			h.mu.RUnlock()
			ticker.Reset(newInterval)
			h.logger.Info("Heartbeat interval updated",
				zap.Duration("new_interval", newInterval),
			)
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
		h.incrementFailure()
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
		h.incrementFailure()
		st, ok := status.FromError(err)
		if ok {
			h.logger.Warn("Heartbeat failed",
				zap.String("code", st.Code().String()),
				zap.String("message", st.Message()),
				zap.Int("failure_count", h.getFailureCount()),
			)
			// 如果是认证错误，可以尝试重新注册
			if st.Code() == codes.Unauthenticated {
				h.logger.Error("Agent not authenticated, may need to re-register")
			}
		} else {
			h.logger.Warn("Heartbeat failed",
				zap.Error(err),
				zap.Int("failure_count", h.getFailureCount()),
			)
		}
		return
	}

	// 心跳成功，重置失败计数
	h.mu.Lock()
	h.lastHeartbeat = time.Now()
	h.failureCount = 0
	h.mu.Unlock()

	h.logger.Debug("Heartbeat sent successfully",
		zap.String("agent_id", h.config.AgentID),
		zap.Bool("success", resp.Success),
		zap.Int32("heartbeat_interval", resp.HeartbeatInterval),
	)

	// 处理服务端返回的动态心跳间隔
	if resp.HeartbeatInterval > 0 {
		newInterval := time.Duration(resp.HeartbeatInterval) * time.Second
		h.updateInterval(newInterval)
	}

	// 检查是否需要触发策略更新
	if resp.PolicyUpdateAvailable {
		h.triggerPolicyUpdate(ctx)
	}
}

// updateInterval 更新心跳间隔（带边界限制）
func (h *HeartbeatClient) updateInterval(newInterval time.Duration) {
	// 边界检查
	if newInterval < h.config.MinInterval {
		newInterval = h.config.MinInterval
	}
	if newInterval > h.config.MaxInterval {
		newInterval = h.config.MaxInterval
	}

	h.mu.Lock()
	if h.currentInterval != newInterval {
		h.currentInterval = newInterval
		h.mu.Unlock()
		// 通知 ticker 重置
		select {
		case h.stopCh <- struct{}{}:
		default:
		}
	} else {
		h.mu.Unlock()
	}
}

// triggerPolicyUpdate 触发策略更新
func (h *HeartbeatClient) triggerPolicyUpdate(ctx context.Context) {
	h.mu.RLock()
	callback := h.policyCallback
	h.mu.RUnlock()

	if callback != nil {
		h.logger.Info("Triggering policy update from heartbeat response")
		go func() {
			if err := callback(ctx); err != nil {
				h.logger.Error("Policy update failed", zap.Error(err))
			}
		}()
	}
}

// incrementFailure 增加失败计数
func (h *HeartbeatClient) incrementFailure() {
	h.mu.Lock()
	h.failureCount++
	count := h.failureCount
	maxCount := h.config.MaxFailureCount
	h.mu.Unlock()

	if count >= maxCount {
		h.logger.Error("Heartbeat failure threshold reached",
			zap.Int("failure_count", count),
			zap.Int("max_failures", maxCount),
		)
	}
}

// getFailureCount 获取失败计数
func (h *HeartbeatClient) getFailureCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.failureCount
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

// CurrentInterval 返回当前心跳间隔
func (h *HeartbeatClient) CurrentInterval() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentInterval
}

// FailureCount 返回连续失败次数
func (h *HeartbeatClient) FailureCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.failureCount
}

// IsHealthy 返回心跳是否健康（失败次数未超阈值）
func (h *HeartbeatClient) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.failureCount < h.config.MaxFailureCount
}
