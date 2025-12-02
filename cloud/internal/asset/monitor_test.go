package asset

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testStatusManager 测试用状态管理器
type testStatusManager struct {
	onlineAgents map[string]bool
}

func newTestStatusManager() *testStatusManager {
	return &testStatusManager{
		onlineAgents: make(map[string]bool),
	}
}

func (m *testStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *HeartbeatInfo) error {
	m.onlineAgents[agentID] = true
	return nil
}

func (m *testStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return m.onlineAgents[agentID], nil
}

func (m *testStatusManager) GetStatus(ctx context.Context, agentID string) (*AgentStatus, error) {
	return nil, nil
}

func (m *testStatusManager) ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error) {
	return nil, nil
}

func (m *testStatusManager) CountOnlineAgents(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}

func (m *testStatusManager) SetOffline(agentID string) {
	m.onlineAgents[agentID] = false
}

func setupMonitorTestDB(t *testing.T) (*gorm.DB, *AssetRepository) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Asset{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewAssetRepository(db, zap.NewNop())
	return db, repo
}

func TestStatusMonitor_ScanOnce(t *testing.T) {
	_, repo := setupMonitorTestDB(t)
	statusMgr := newTestStatusManager()
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建两个在线资产
	asset1 := &Asset{
		AgentID:  "agent-monitor-001",
		TenantID: tenantID,
		Hostname: "host-1",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}
	asset2 := &Asset{
		AgentID:  "agent-monitor-002",
		TenantID: tenantID,
		Hostname: "host-2",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}
	repo.Create(ctx, asset1)
	repo.Create(ctx, asset2)

	// 设置 agent1 在 Redis 中在线，agent2 离线
	statusMgr.onlineAgents["agent-monitor-001"] = true
	statusMgr.onlineAgents["agent-monitor-002"] = false

	// 创建监控器并执行扫描
	config := &MonitorConfig{
		ScanInterval: 1 * time.Second,
		HeartbeatTTL: 90 * time.Second,
		BatchSize:    100,
	}
	monitor := NewStatusMonitor(repo, statusMgr, config, zap.NewNop())

	// 执行一次扫描
	monitor.ScanOnce(ctx)

	// 验证 agent2 被标记为离线
	found1, _ := repo.FindByAgentID(ctx, tenantID, "agent-monitor-001")
	found2, _ := repo.FindByAgentID(ctx, tenantID, "agent-monitor-002")

	if found1.Status != AssetStatusOnline {
		t.Errorf("expected agent-monitor-001 to remain online, got %s", found1.Status)
	}
	if found2.Status != AssetStatusOffline {
		t.Errorf("expected agent-monitor-002 to be offline, got %s", found2.Status)
	}
}

func TestStatusMonitor_HandleHeartbeat(t *testing.T) {
	_, repo := setupMonitorTestDB(t)
	statusMgr := newTestStatusManager()
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建一个离线资产
	asset := &Asset{
		AgentID:  "agent-heartbeat-001",
		TenantID: tenantID,
		Hostname: "heartbeat-host",
		OSType:   "linux",
		Status:   AssetStatusOffline,
	}
	repo.Create(ctx, asset)

	config := DefaultMonitorConfig()
	monitor := NewStatusMonitor(repo, statusMgr, config, zap.NewNop())

	// 处理心跳
	info := &HeartbeatInfo{
		Hostname:     "heartbeat-host",
		IPAddress:    "192.168.1.100",
		AgentVersion: "1.0.0",
		OSFamily:     "linux",
		Status:       "online",
	}
	err := monitor.HandleHeartbeat(ctx, tenantID, "agent-heartbeat-001", info)
	if err != nil {
		t.Fatalf("HandleHeartbeat failed: %v", err)
	}

	// 验证资产变为在线
	found, _ := repo.FindByAgentID(ctx, tenantID, "agent-heartbeat-001")
	if found.Status != AssetStatusOnline {
		t.Errorf("expected asset to be online after heartbeat, got %s", found.Status)
	}
	if found.LastSeenAt == nil {
		t.Error("expected last_seen_at to be set")
	}

	// 验证 Redis 也被更新
	if !statusMgr.onlineAgents["agent-heartbeat-001"] {
		t.Error("expected agent to be online in Redis")
	}
}

func TestStatusMonitor_StartStop(t *testing.T) {
	_, repo := setupMonitorTestDB(t)
	statusMgr := newTestStatusManager()

	config := &MonitorConfig{
		ScanInterval: 100 * time.Millisecond, // 快速扫描用于测试
		HeartbeatTTL: 90 * time.Second,
		BatchSize:    100,
	}
	monitor := NewStatusMonitor(repo, statusMgr, config, zap.NewNop())

	// 启动
	monitor.Start()

	// 等待几个扫描周期
	time.Sleep(350 * time.Millisecond)

	// 停止
	done := make(chan struct{})
	go func() {
		monitor.Stop()
		close(done)
	}()

	select {
	case <-done:
		// 正常停止
	case <-time.After(2 * time.Second):
		t.Error("monitor.Stop() did not complete in time")
	}
}

func TestStatusMonitor_DefaultConfig(t *testing.T) {
	config := DefaultMonitorConfig()

	if config.ScanInterval != 30*time.Second {
		t.Errorf("expected ScanInterval=30s, got %v", config.ScanInterval)
	}
	if config.HeartbeatTTL != 90*time.Second {
		t.Errorf("expected HeartbeatTTL=90s, got %v", config.HeartbeatTTL)
	}
	if config.BatchSize != 100 {
		t.Errorf("expected BatchSize=100, got %d", config.BatchSize)
	}
}

func TestStatusMonitor_GetConfig(t *testing.T) {
	_, repo := setupMonitorTestDB(t)
	statusMgr := newTestStatusManager()

	customConfig := &MonitorConfig{
		ScanInterval: 60 * time.Second,
		HeartbeatTTL: 120 * time.Second,
		BatchSize:    50,
	}
	monitor := NewStatusMonitor(repo, statusMgr, customConfig, zap.NewNop())

	gotConfig := monitor.GetConfig()
	if gotConfig.ScanInterval != 60*time.Second {
		t.Errorf("expected ScanInterval=60s, got %v", gotConfig.ScanInterval)
	}
}
