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

func setupSoftwareTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Asset{}, &SoftwareInventory{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestSoftwareService_UpdateSoftwareInventory(t *testing.T) {
	db := setupSoftwareTestDB(t)
	service := NewSoftwareService(db, zap.NewNop())
	ctx := context.Background()
	assetID := uuid.New()

	// 首次更新
	items := []SoftwareItem{
		{Name: "Software A", Version: "1.0.0", Publisher: "Publisher A"},
		{Name: "Software B", Version: "2.0.0", Publisher: "Publisher B"},
	}
	err := service.UpdateSoftwareInventory(ctx, assetID, items)
	if err != nil {
		t.Fatalf("UpdateSoftwareInventory failed: %v", err)
	}

	// 验证
	software, total, _ := service.GetSoftwareByAsset(ctx, assetID, nil)
	if total != 2 {
		t.Errorf("expected 2 software items, got %d", total)
	}

	// 再次更新（全量替换）
	newItems := []SoftwareItem{
		{Name: "Software C", Version: "3.0.0"},
	}
	err = service.UpdateSoftwareInventory(ctx, assetID, newItems)
	if err != nil {
		t.Fatalf("UpdateSoftwareInventory (replace) failed: %v", err)
	}

	// 验证旧记录被删除
	software, total, _ = service.GetSoftwareByAsset(ctx, assetID, nil)
	if total != 1 {
		t.Errorf("expected 1 software item after replace, got %d", total)
	}
	if software[0].Name != "Software C" {
		t.Errorf("expected Software C, got %s", software[0].Name)
	}
}

func TestSoftwareService_GetSoftwareByAsset(t *testing.T) {
	db := setupSoftwareTestDB(t)
	service := NewSoftwareService(db, zap.NewNop())
	ctx := context.Background()
	assetID := uuid.New()

	// 创建软件记录
	items := []SoftwareItem{
		{Name: "Apache", Version: "2.4.0"},
		{Name: "MySQL", Version: "8.0.0"},
		{Name: "Nginx", Version: "1.20.0"},
	}
	service.UpdateSoftwareInventory(ctx, assetID, items)

	// 测试分页
	opts := &SoftwareQueryOptions{
		Pagination: Pagination{Page: 1, PageSize: 2},
	}
	software, total, err := service.GetSoftwareByAsset(ctx, assetID, opts)
	if err != nil {
		t.Fatalf("GetSoftwareByAsset failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(software) != 2 {
		t.Errorf("expected 2 items on page, got %d", len(software))
	}

	// 测试名称过滤
	opts = &SoftwareQueryOptions{
		Name:       "sql",
		Pagination: Pagination{Page: 1, PageSize: 10},
	}
	software, total, _ = service.GetSoftwareByAsset(ctx, assetID, opts)
	if total != 1 {
		t.Errorf("expected 1 item matching 'sql', got %d", total)
	}
}

func TestSoftwareService_UpdateEmpty(t *testing.T) {
	db := setupSoftwareTestDB(t)
	service := NewSoftwareService(db, zap.NewNop())
	ctx := context.Background()
	assetID := uuid.New()

	// 先添加一些软件
	items := []SoftwareItem{
		{Name: "Software", Version: "1.0.0"},
	}
	service.UpdateSoftwareInventory(ctx, assetID, items)

	// 用空列表更新（清空）
	err := service.UpdateSoftwareInventory(ctx, assetID, []SoftwareItem{})
	if err != nil {
		t.Fatalf("UpdateSoftwareInventory (empty) failed: %v", err)
	}

	// 验证清空
	_, total, _ := service.GetSoftwareByAsset(ctx, assetID, nil)
	if total != 0 {
		t.Errorf("expected 0 software after empty update, got %d", total)
	}
}

func TestSoftwareService_DeleteSoftwareByAsset(t *testing.T) {
	db := setupSoftwareTestDB(t)
	service := NewSoftwareService(db, zap.NewNop())
	ctx := context.Background()
	assetID := uuid.New()

	// 添加软件
	items := []SoftwareItem{
		{Name: "To Delete", Version: "1.0.0"},
	}
	service.UpdateSoftwareInventory(ctx, assetID, items)

	// 删除
	err := service.DeleteSoftwareByAsset(ctx, assetID)
	if err != nil {
		t.Fatalf("DeleteSoftwareByAsset failed: %v", err)
	}

	// 验证
	_, total, _ := service.GetSoftwareByAsset(ctx, assetID, nil)
	if total != 0 {
		t.Errorf("expected 0 software after delete, got %d", total)
	}
}

func TestSoftwareService_WithInstallDate(t *testing.T) {
	db := setupSoftwareTestDB(t)
	service := NewSoftwareService(db, zap.NewNop())
	ctx := context.Background()
	assetID := uuid.New()

	installDate := time.Now().Add(-24 * time.Hour)
	items := []SoftwareItem{
		{
			Name:        "Dated Software",
			Version:     "1.0.0",
			Publisher:   "Test Publisher",
			InstallDate: &installDate,
			InstallPath: "/usr/local/bin",
			Size:        1024000,
		},
	}
	service.UpdateSoftwareInventory(ctx, assetID, items)

	software, _, _ := service.GetSoftwareByAsset(ctx, assetID, nil)
	if software[0].InstallDate == nil {
		t.Error("expected install_date to be set")
	}
	if software[0].Size != 1024000 {
		t.Errorf("expected size=1024000, got %d", software[0].Size)
	}
}
