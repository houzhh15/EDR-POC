package asset

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockStatusManager 模拟状态管理器
type mockStatusManager struct {
	heartbeats map[string]*HeartbeatInfo
}

func newMockStatusManager() *mockStatusManager {
	return &mockStatusManager{
		heartbeats: make(map[string]*HeartbeatInfo),
	}
}

func (m *mockStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *HeartbeatInfo) error {
	m.heartbeats[agentID] = info
	return nil
}

func (m *mockStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	_, ok := m.heartbeats[agentID]
	return ok, nil
}

func (m *mockStatusManager) GetStatus(ctx context.Context, agentID string) (*AgentStatus, error) {
	return nil, nil
}

func (m *mockStatusManager) ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error) {
	return nil, nil
}

func (m *mockStatusManager) CountOnlineAgents(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}

// setupServiceTestDB 创建测试数据库和服务
func setupServiceTestDB(t *testing.T) (*AssetService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Asset{}, &AssetChangeLog{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewAssetRepository(db, zap.NewNop())
	changelog := NewChangeLogger(db, zap.NewNop())
	statusMgr := newMockStatusManager()

	service := NewAssetService(repo, statusMgr, changelog, zap.NewNop())
	return service, db
}

func TestAssetService_RegisterOrUpdateAsset_NewAsset(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	req := &RegisterAssetRequest{
		AgentID:      "agent-new-001",
		TenantID:     tenantID.String(),
		Hostname:     "new-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 22.04",
		Architecture: "x64",
		IPAddresses:  []string{"192.168.1.100"},
		MACAddresses: []string{"00:11:22:33:44:55"},
		AgentVersion: "1.0.0",
	}

	// 首次注册
	asset, err := service.RegisterOrUpdateAsset(ctx, req)
	if err != nil {
		t.Fatalf("RegisterOrUpdateAsset failed: %v", err)
	}

	if asset.ID == uuid.Nil {
		t.Error("expected asset ID to be set")
	}
	if asset.AgentID != "agent-new-001" {
		t.Errorf("expected agent_id=agent-new-001, got %s", asset.AgentID)
	}
	if asset.Hostname != "new-host" {
		t.Errorf("expected hostname=new-host, got %s", asset.Hostname)
	}
	if asset.Status != AssetStatusOnline {
		t.Errorf("expected status=online, got %s", asset.Status)
	}
	if asset.LastSeenAt == nil {
		t.Error("expected last_seen_at to be set")
	}
}

func TestAssetService_RegisterOrUpdateAsset_UpdateExisting(t *testing.T) {
	service, db := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 首次注册
	req := &RegisterAssetRequest{
		AgentID:      "agent-update-001",
		TenantID:     tenantID.String(),
		Hostname:     "old-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 20.04",
		IPAddresses:  []string{"192.168.1.100"},
		AgentVersion: "1.0.0",
	}
	firstAsset, _ := service.RegisterOrUpdateAsset(ctx, req)

	// 更新请求（模拟 hostname 和 agent_version 变更）
	updateReq := &RegisterAssetRequest{
		AgentID:      "agent-update-001",
		TenantID:     tenantID.String(),
		Hostname:     "new-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 22.04",
		IPAddresses:  []string{"192.168.1.100", "10.0.0.1"},
		AgentVersion: "2.0.0",
	}
	updatedAsset, err := service.RegisterOrUpdateAsset(ctx, updateReq)
	if err != nil {
		t.Fatalf("RegisterOrUpdateAsset (update) failed: %v", err)
	}

	// 验证 ID 不变
	if updatedAsset.ID != firstAsset.ID {
		t.Error("expected asset ID to remain the same")
	}

	// 验证字段更新
	if updatedAsset.Hostname != "new-host" {
		t.Errorf("expected hostname=new-host, got %s", updatedAsset.Hostname)
	}
	if updatedAsset.OSVersion != "Ubuntu 22.04" {
		t.Errorf("expected os_version=Ubuntu 22.04, got %s", updatedAsset.OSVersion)
	}
	if updatedAsset.AgentVersion != "2.0.0" {
		t.Errorf("expected agent_version=2.0.0, got %s", updatedAsset.AgentVersion)
	}

	// 验证变更日志
	var logs []AssetChangeLog
	db.Where("asset_id = ?", firstAsset.ID).Find(&logs)

	// 应该有首次注册日志 + 4个变更字段（hostname, os_version, ip_addresses, agent_version）
	expectedChanges := 4 // 不含首次注册
	changeCount := 0
	for _, log := range logs {
		if log.FieldName != "status" { // 排除首次注册的 status 日志
			changeCount++
		}
	}
	if changeCount < expectedChanges {
		t.Errorf("expected at least %d change logs, got %d", expectedChanges, changeCount)
	}
}

func TestAssetService_RegisterOrUpdateAsset_InvalidTenantID(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()

	req := &RegisterAssetRequest{
		AgentID:  "agent-invalid",
		TenantID: "invalid-uuid",
		Hostname: "test-host",
		OSType:   "linux",
	}

	_, err := service.RegisterOrUpdateAsset(ctx, req)
	if err == nil {
		t.Error("expected error for invalid tenant_id")
	}
}

func TestAssetService_GetAsset(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建资产
	req := &RegisterAssetRequest{
		AgentID:  "agent-get-001",
		TenantID: tenantID.String(),
		Hostname: "get-test-host",
		OSType:   "windows",
	}
	created, _ := service.RegisterOrUpdateAsset(ctx, req)

	// 获取资产
	found, err := service.GetAsset(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if found.Hostname != "get-test-host" {
		t.Errorf("expected hostname=get-test-host, got %s", found.Hostname)
	}

	// 获取不存在的资产
	_, err = service.GetAsset(ctx, tenantID, uuid.New())
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound, got %v", err)
	}
}

func TestAssetService_ListAssets(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建多个资产
	for i := 0; i < 5; i++ {
		req := &RegisterAssetRequest{
			AgentID:  uuid.New().String(),
			TenantID: tenantID.String(),
			Hostname: "list-host-" + string(rune('a'+i)),
			OSType:   "linux",
		}
		service.RegisterOrUpdateAsset(ctx, req)
	}

	// 测试分页
	opts := &QueryOptions{
		Pagination: Pagination{Page: 1, PageSize: 2},
	}
	assets, total, err := service.ListAssets(ctx, tenantID, opts)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}
}

func TestAssetService_UpdateAsset(t *testing.T) {
	service, db := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建资产
	req := &RegisterAssetRequest{
		AgentID:      "agent-update-api",
		TenantID:     tenantID.String(),
		Hostname:     "old-api-host",
		OSType:       "linux",
		AgentVersion: "1.0.0",
	}
	created, _ := service.RegisterOrUpdateAsset(ctx, req)

	// 通过 API 更新
	newHostname := "new-api-host"
	newVersion := "2.0.0"
	updateReq := &UpdateAssetRequest{
		Hostname:     &newHostname,
		AgentVersion: &newVersion,
	}

	updated, err := service.UpdateAsset(ctx, tenantID, created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateAsset failed: %v", err)
	}

	if updated.Hostname != "new-api-host" {
		t.Errorf("expected hostname=new-api-host, got %s", updated.Hostname)
	}
	if updated.AgentVersion != "2.0.0" {
		t.Errorf("expected agent_version=2.0.0, got %s", updated.AgentVersion)
	}

	// 验证变更日志记录了 "api" 来源
	var logs []AssetChangeLog
	db.Where("asset_id = ? AND changed_by = ?", created.ID, "api").Find(&logs)
	if len(logs) != 2 { // hostname + agent_version
		t.Errorf("expected 2 change logs from api, got %d", len(logs))
	}
}

func TestAssetService_DeleteAsset(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建资产
	req := &RegisterAssetRequest{
		AgentID:  "agent-delete",
		TenantID: tenantID.String(),
		Hostname: "delete-host",
		OSType:   "linux",
	}
	created, _ := service.RegisterOrUpdateAsset(ctx, req)

	// 删除资产
	err := service.DeleteAsset(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("DeleteAsset failed: %v", err)
	}

	// 再次获取应该找不到
	_, err = service.GetAsset(ctx, tenantID, created.ID)
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound after delete, got %v", err)
	}
}

func TestAssetService_CountByStatus(t *testing.T) {
	service, _ := setupServiceTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建资产
	for i := 0; i < 3; i++ {
		req := &RegisterAssetRequest{
			AgentID:  uuid.New().String(),
			TenantID: tenantID.String(),
			Hostname: "count-host-" + string(rune('a'+i)),
			OSType:   "linux",
		}
		service.RegisterOrUpdateAsset(ctx, req)
	}

	counts, err := service.CountByStatus(ctx, tenantID)
	if err != nil {
		t.Fatalf("CountByStatus failed: %v", err)
	}

	if counts[AssetStatusOnline] != 3 {
		t.Errorf("expected 3 online assets, got %d", counts[AssetStatusOnline])
	}
}
