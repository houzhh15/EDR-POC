// Package asset 提供资产管理相关功能
package asset

import (
"context"
"fmt"
"strconv"
"time"

"github.com/redis/go-redis/v9"
"go.uber.org/zap"
)

// AgentStatusManager Agent 在线状态管理接口
type AgentStatusManager interface {
	UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *HeartbeatInfo) error
	IsOnline(ctx context.Context, agentID string) (bool, error)
	GetStatus(ctx context.Context, agentID string) (*AgentStatus, error)
	ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error)
	CountOnlineAgents(ctx context.Context, tenantID string) (int64, error)
}

// HeartbeatInfo 心跳信息
type HeartbeatInfo struct {
	Hostname        string `json:"hostname"`
	IPAddress       string `json:"ip_address"`
	AgentVersion    string `json:"agent_version"`
	OSFamily        string `json:"os_family"`
	Status          string `json:"status"`
	ConnectedServer string `json:"connected_server"`
}

// AgentStatus Agent 完整状态
type AgentStatus struct {
	AgentID         string    `json:"agent_id"`
	TenantID        string    `json:"tenant_id"`
	Status          string    `json:"status"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	Hostname        string    `json:"hostname"`
	IPAddress       string    `json:"ip_address"`
	AgentVersion    string    `json:"agent_version"`
	OSFamily        string    `json:"os_family"`
	ConnectedServer string    `json:"connected_server"`
}

// RedisAgentStatusManager Redis 实现的 Agent 状态管理
type RedisAgentStatusManager struct {
	client       *redis.Client
	logger       *zap.Logger
	heartbeatTTL time.Duration
}

// NewAgentStatusManager 创建 Agent 状态管理器
func NewAgentStatusManager(client *redis.Client, logger *zap.Logger) *RedisAgentStatusManager {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &RedisAgentStatusManager{
		client:       client,
		logger:       logger,
		heartbeatTTL: 90 * time.Second,
	}
}

// statusKey 生成 Agent 状态 Key
func statusKey(agentID string) string {
	return fmt.Sprintf("agent:status:%s", agentID)
}

// onlineKey 生成租户在线列表 Key
func onlineKey(tenantID string) string {
	return fmt.Sprintf("agents:online:%s", tenantID)
}

// UpdateHeartbeat 更新心跳状态
func (m *RedisAgentStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *HeartbeatInfo) error {
	now := time.Now()
	key := statusKey(agentID)

	statusData := map[string]interface{}{
		"agent_id":         agentID,
		"tenant_id":        tenantID,
		"status":           info.Status,
		"last_heartbeat":   now.Unix(),
		"hostname":         info.Hostname,
		"ip_address":       info.IPAddress,
		"agent_version":    info.AgentVersion,
		"os_family":        info.OSFamily,
		"connected_server": info.ConnectedServer,
	}

	pipe := m.client.Pipeline()
	pipe.HSet(ctx, key, statusData)
	pipe.Expire(ctx, key, m.heartbeatTTL)
	pipe.ZAdd(ctx, onlineKey(tenantID), redis.Z{
Score:  float64(now.Unix()),
		Member: agentID,
	})

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}
	return nil
}

// IsOnline 检查 Agent 是否在线
func (m *RedisAgentStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	exists, err := m.client.Exists(ctx, statusKey(agentID)).Result()
	if err != nil {
		return false, fmt.Errorf("check agent online: %w", err)
	}
	return exists > 0, nil
}

// GetStatus 获取 Agent 状态详情
func (m *RedisAgentStatusManager) GetStatus(ctx context.Context, agentID string) (*AgentStatus, error) {
	data, err := m.client.HGetAll(ctx, statusKey(agentID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get agent status: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var lastHeartbeat time.Time
	if ts, ok := data["last_heartbeat"]; ok {
		if unix, err := strconv.ParseInt(ts, 10, 64); err == nil {
			lastHeartbeat = time.Unix(unix, 0)
		}
	}

	return &AgentStatus{
		AgentID:         data["agent_id"],
		TenantID:        data["tenant_id"],
		Status:          data["status"],
		LastHeartbeat:   lastHeartbeat,
		Hostname:        data["hostname"],
		IPAddress:       data["ip_address"],
		AgentVersion:    data["agent_version"],
		OSFamily:        data["os_family"],
		ConnectedServer: data["connected_server"],
	}, nil
}

// ListOnlineAgents 列出租户在线 Agent
func (m *RedisAgentStatusManager) ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error) {
	minScore := float64(time.Now().Add(-m.heartbeatTTL).Unix())
	maxScore := float64(time.Now().Unix())

	agents, err := m.client.ZRangeByScore(ctx, onlineKey(tenantID), &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", minScore),
		Max: fmt.Sprintf("%f", maxScore),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("list online agents: %w", err)
	}
	return agents, nil
}

// CountOnlineAgents 统计在线 Agent 数量
func (m *RedisAgentStatusManager) CountOnlineAgents(ctx context.Context, tenantID string) (int64, error) {
	minScore := fmt.Sprintf("%d", time.Now().Add(-m.heartbeatTTL).Unix())
	maxScore := fmt.Sprintf("%d", time.Now().Unix())

	count, err := m.client.ZCount(ctx, onlineKey(tenantID), minScore, maxScore).Result()
	if err != nil {
		return 0, fmt.Errorf("count online agents: %w", err)
	}
	return count, nil
}
