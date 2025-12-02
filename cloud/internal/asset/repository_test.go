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

// setupTestDB 创建内存 SQLite 测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&Asset{}, &AssetGroup{}, &AssetGroupMember{}, &SoftwareInventory{}, &AssetChangeLog{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestAssetRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:      "agent-001",
		TenantID:     tenantID,
		Hostname:     "test-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 22.04",
		Architecture: "x64",
		IPAddresses:  StringSlice{"192.168.1.100"},
		MACAddresses: StringSlice{"00:11:22:33:44:55"},
		AgentVersion: "1.0.0",
		Status:       AssetStatusOnline,
	}

	created, err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if created.AgentID != "agent-001" {
		t.Errorf("expected agent_id=agent-001, got %s", created.AgentID)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestAssetRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-002",
		TenantID: tenantID,
		Hostname: "find-by-id-host",
		OSType:   "windows",
		Status:   AssetStatusOnline,
	}

	created, _ := repo.Create(ctx, asset)

	// 正常查找
	found, err := repo.FindByID(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Hostname != "find-by-id-host" {
		t.Errorf("expected hostname=find-by-id-host, got %s", found.Hostname)
	}

	// 查找不存在的
	_, err = repo.FindByID(ctx, tenantID, uuid.New())
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound, got %v", err)
	}

	// 跨租户隔离
	otherTenant := uuid.New()
	_, err = repo.FindByID(ctx, otherTenant, created.ID)
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound for other tenant, got %v", err)
	}
}

func TestAssetRepository_FindByAgentID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-find-003",
		TenantID: tenantID,
		Hostname: "find-by-agent-host",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}

	repo.Create(ctx, asset)

	found, err := repo.FindByAgentID(ctx, tenantID, "agent-find-003")
	if err != nil {
		t.Fatalf("FindByAgentID failed: %v", err)
	}
	if found.Hostname != "find-by-agent-host" {
		t.Errorf("expected hostname=find-by-agent-host, got %s", found.Hostname)
	}

	// 查找不存在的
	_, err = repo.FindByAgentID(ctx, tenantID, "non-existent")
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound, got %v", err)
	}
}

func TestAssetRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()

	// 创建多个资产
	for i := 0; i < 5; i++ {
		status := AssetStatusOnline
		if i%2 == 0 {
			status = AssetStatusOffline
		}
		asset := &Asset{
			AgentID:  uuid.New().String(),
			TenantID: tenantID,
			Hostname: "host-" + string(rune('a'+i)),
			OSType:   "linux",
			Status:   status,
		}
		repo.Create(ctx, asset)
	}

	// 测试分页
	opts := &QueryOptions{
		Pagination: Pagination{Page: 1, PageSize: 2},
	}
	assets, total, err := repo.FindAll(ctx, tenantID, opts)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}

	// 测试状态过滤
	opts = &QueryOptions{
		Status:     "online",
		Pagination: Pagination{Page: 1, PageSize: 10},
	}
	assets, total, err = repo.FindAll(ctx, tenantID, opts)
	if err != nil {
		t.Fatalf("FindAll with status filter failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2 online assets, got %d", total)
	}
}

func TestAssetRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-update",
		TenantID: tenantID,
		Hostname: "old-hostname",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}

	created, _ := repo.Create(ctx, asset)

	// 更新
	created.Hostname = "new-hostname"
	created.OSVersion = "Ubuntu 24.04"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Hostname != "new-hostname" {
		t.Errorf("expected hostname=new-hostname, got %s", updated.Hostname)
	}

	// 验证持久化
	found, _ := repo.FindByID(ctx, tenantID, created.ID)
	if found.Hostname != "new-hostname" {
		t.Errorf("expected persisted hostname=new-hostname, got %s", found.Hostname)
	}
}

func TestAssetRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-delete",
		TenantID: tenantID,
		Hostname: "to-delete",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}

	created, _ := repo.Create(ctx, asset)

	// 软删除
	err := repo.SoftDelete(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// 查询应该找不到
	_, err = repo.FindByID(ctx, tenantID, created.ID)
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound after soft delete, got %v", err)
	}

	// 再次删除应该报错
	err = repo.SoftDelete(ctx, tenantID, created.ID)
	if err != ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound for already deleted, got %v", err)
	}
}

func TestAssetRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-status",
		TenantID: tenantID,
		Hostname: "status-test",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}

	created, _ := repo.Create(ctx, asset)

	// 更新状态
	err := repo.UpdateStatus(ctx, tenantID, created.ID, AssetStatusOffline)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// 验证
	found, _ := repo.FindByID(ctx, tenantID, created.ID)
	if found.Status != AssetStatusOffline {
		t.Errorf("expected status=offline, got %s", found.Status)
	}
}

func TestAssetRepository_GetOnlineAssets(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()

	// 创建在线和离线资产
	onlineAsset := &Asset{
		AgentID:  "agent-online",
		TenantID: tenantID,
		Hostname: "online-host",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}
	offlineAsset := &Asset{
		AgentID:  "agent-offline",
		TenantID: tenantID,
		Hostname: "offline-host",
		OSType:   "linux",
		Status:   AssetStatusOffline,
	}

	repo.Create(ctx, onlineAsset)
	repo.Create(ctx, offlineAsset)

	// 获取在线资产
	onlineAssets, err := repo.GetOnlineAssets(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOnlineAssets failed: %v", err)
	}

	if len(onlineAssets) != 1 {
		t.Errorf("expected 1 online asset, got %d", len(onlineAssets))
	}
	if onlineAssets[0].AgentID != "agent-online" {
		t.Errorf("expected agent_id=agent-online, got %s", onlineAssets[0].AgentID)
	}
}

func TestAssetRepository_UpdateLastSeen(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()
	asset := &Asset{
		AgentID:  "agent-lastseen",
		TenantID: tenantID,
		Hostname: "lastseen-test",
		OSType:   "linux",
		Status:   AssetStatusOffline,
	}

	repo.Create(ctx, asset)

	// 更新最后在线时间
	now := time.Now()
	err := repo.UpdateLastSeen(ctx, tenantID, "agent-lastseen", now)
	if err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	// 验证状态也变为在线
	found, _ := repo.FindByAgentID(ctx, tenantID, "agent-lastseen")
	if found.Status != AssetStatusOnline {
		t.Errorf("expected status=online after UpdateLastSeen, got %s", found.Status)
	}
	if found.LastSeenAt == nil {
		t.Error("expected last_seen_at to be set")
	}
}

func TestAssetRepository_BatchUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		asset := &Asset{
			AgentID:  uuid.New().String(),
			TenantID: tenantID,
			Hostname: "batch-" + string(rune('a'+i)),
			OSType:   "linux",
			Status:   AssetStatusOnline,
		}
		created, _ := repo.Create(ctx, asset)
		ids = append(ids, created.ID)
	}

	// 批量更新状态
	err := repo.BatchUpdateStatus(ctx, tenantID, ids, AssetStatusOffline)
	if err != nil {
		t.Fatalf("BatchUpdateStatus failed: %v", err)
	}

	// 验证所有资产状态都变为离线
	for _, id := range ids {
		found, _ := repo.FindByID(ctx, tenantID, id)
		if found.Status != AssetStatusOffline {
			t.Errorf("expected status=offline for %s, got %s", id, found.Status)
		}
	}
}

func TestAssetRepository_CountByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAssetRepository(db, zap.NewNop())
	ctx := context.Background()

	tenantID := uuid.New()

	// 创建不同状态的资产
	for i := 0; i < 5; i++ {
		status := AssetStatusOnline
		if i < 2 {
			status = AssetStatusOffline
		} else if i < 3 {
			status = AssetStatusUnknown
		}
		asset := &Asset{
			AgentID:  uuid.New().String(),
			TenantID: tenantID,
			Hostname: "count-" + string(rune('a'+i)),
			OSType:   "linux",
			Status:   status,
		}
		repo.Create(ctx, asset)
	}

	counts, err := repo.CountByStatus(ctx, tenantID)
	if err != nil {
		t.Fatalf("CountByStatus failed: %v", err)
	}

	if counts[AssetStatusOnline] != 2 {
		t.Errorf("expected 2 online, got %d", counts[AssetStatusOnline])
	}
	if counts[AssetStatusOffline] != 2 {
		t.Errorf("expected 2 offline, got %d", counts[AssetStatusOffline])
	}
	if counts[AssetStatusUnknown] != 1 {
		t.Errorf("expected 1 unknown, got %d", counts[AssetStatusUnknown])
	}
}
