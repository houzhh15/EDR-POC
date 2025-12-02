package asset

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MonitorConfig 状态监控配置
type MonitorConfig struct {
	ScanInterval time.Duration // 扫描间隔，默认 30s
	HeartbeatTTL time.Duration // 心跳超时，默认 90s
	BatchSize    int           // 批量处理数，默认 100
}

// DefaultMonitorConfig 默认配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		ScanInterval: 30 * time.Second,
		HeartbeatTTL: 90 * time.Second,
		BatchSize:    100,
	}
}

// StatusMonitor 资产状态监控器
type StatusMonitor struct {
	repo      *AssetRepository
	statusMgr AgentStatusManager
	config    *MonitorConfig
	logger    *zap.Logger

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewStatusMonitor 创建状态监控器
func NewStatusMonitor(repo *AssetRepository, statusMgr AgentStatusManager, config *MonitorConfig, logger *zap.Logger) *StatusMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &StatusMonitor{
		repo:      repo,
		statusMgr: statusMgr,
		config:    config,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动状态监控协程
func (m *StatusMonitor) Start() {
	m.wg.Add(1)
	go m.runScanLoop()
	m.logger.Info("status monitor started",
		zap.Duration("scan_interval", m.config.ScanInterval),
		zap.Duration("heartbeat_ttl", m.config.HeartbeatTTL),
	)
}

// Stop 停止状态监控（优雅关闭）
func (m *StatusMonitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("status monitor stopped")
}

// runScanLoop 扫描循环
func (m *StatusMonitor) runScanLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	// 启动时立即执行一次扫描
	m.scanAndUpdateStatus()

	for {
		select {
		case <-ticker.C:
			m.scanAndUpdateStatus()
		case <-m.stopCh:
			return
		}
	}
}

// scanAndUpdateStatus 扫描并更新离线资产状态
func (m *StatusMonitor) scanAndUpdateStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取所有在线资产
	onlineAssets, err := m.repo.GetAllOnlineAssets(ctx)
	if err != nil {
		m.logger.Error("failed to get online assets", zap.Error(err))
		return
	}

	if len(onlineAssets) == 0 {
		return
	}

	m.logger.Debug("scanning online assets", zap.Int("count", len(onlineAssets)))

	var offlineAssets []OnlineAssetInfo

	// 检查每个资产的 Redis 状态
	for _, asset := range onlineAssets {
		isOnline, err := m.statusMgr.IsOnline(ctx, asset.AgentID)
		if err != nil {
			m.logger.Warn("failed to check agent status",
				zap.String("agent_id", asset.AgentID),
				zap.Error(err),
			)
			continue
		}

		if !isOnline {
			offlineAssets = append(offlineAssets, asset)
		}
	}

	if len(offlineAssets) == 0 {
		return
	}

	// 批量更新离线状态
	m.markAssetsOffline(ctx, offlineAssets)
}

// markAssetsOffline 标记资产为离线
func (m *StatusMonitor) markAssetsOffline(ctx context.Context, assets []OnlineAssetInfo) {
	// 按批次处理
	batchSize := m.config.BatchSize
	for i := 0; i < len(assets); i += batchSize {
		end := i + batchSize
		if end > len(assets) {
			end = len(assets)
		}
		batch := assets[i:end]

		ids := make([]uuid.UUID, len(batch))
		agentIDs := make([]string, len(batch))
		for j, asset := range batch {
			ids[j] = asset.ID
			agentIDs[j] = asset.AgentID
		}

		// 这里我们逐个更新（因为可能属于不同租户）
		// 实际生产环境可以优化为按租户分组批量更新
		for _, asset := range batch {
			if err := m.repo.UpdateStatusByAgentID(ctx, uuid.Nil, asset.AgentID, AssetStatusOffline); err != nil {
				m.logger.Warn("failed to mark asset offline",
					zap.String("agent_id", asset.AgentID),
					zap.Error(err),
				)
				continue
			}

			m.logger.Info("asset marked offline",
				zap.String("id", asset.ID.String()),
				zap.String("agent_id", asset.AgentID),
			)
		}
	}
}

// HandleHeartbeat 处理心跳（同时更新 Redis 和 PostgreSQL）
func (m *StatusMonitor) HandleHeartbeat(ctx context.Context, tenantID uuid.UUID, agentID string, info *HeartbeatInfo) error {
	now := time.Now()

	// 更新 Redis 状态
	if err := m.statusMgr.UpdateHeartbeat(ctx, agentID, tenantID.String(), info); err != nil {
		m.logger.Warn("failed to update Redis heartbeat",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		// Redis 失败不阻断 PostgreSQL 更新
	}

	// 更新 PostgreSQL last_seen_at
	if err := m.repo.UpdateLastSeen(ctx, tenantID, agentID, now); err != nil {
		m.logger.Error("failed to update last_seen_at",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// ScanOnce 手动触发一次扫描（用于测试或管理接口）
func (m *StatusMonitor) ScanOnce(ctx context.Context) error {
	m.scanAndUpdateStatus()
	return nil
}

// GetConfig 获取配置
func (m *StatusMonitor) GetConfig() *MonitorConfig {
	return m.config
}
