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

func setupGroupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Asset{}, &AssetGroup{}, &AssetGroupMember{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestGroupService_CreateGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建根分组
	req := &CreateGroupRequest{
		Name:        "Root Group",
		Description: "Test root group",
		Type:        GroupTypeDepartment,
	}
	group, err := service.CreateGroup(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group.ID == uuid.Nil {
		t.Error("expected group ID to be set")
	}
	if group.Level != 0 {
		t.Errorf("expected level=0, got %d", group.Level)
	}
	if group.ParentID != nil {
		t.Error("expected ParentID to be nil for root group")
	}
}

func TestGroupService_CreateChildGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建父分组
	parentReq := &CreateGroupRequest{Name: "Parent"}
	parent, _ := service.CreateGroup(ctx, tenantID, parentReq)

	// 创建子分组
	childReq := &CreateGroupRequest{
		Name:     "Child",
		ParentID: &parent.ID,
	}
	child, err := service.CreateGroup(ctx, tenantID, childReq)
	if err != nil {
		t.Fatalf("CreateGroup (child) failed: %v", err)
	}

	if child.Level != 1 {
		t.Errorf("expected level=1, got %d", child.Level)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Error("expected ParentID to match parent")
	}
}

func TestGroupService_CreateGroup_DepthLimit(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建 5 层分组（达到最大深度）
	var parentID *uuid.UUID
	for i := 0; i < MaxGroupDepth; i++ {
		req := &CreateGroupRequest{
			Name:     "Level " + string(rune('0'+i)),
			ParentID: parentID,
		}
		group, err := service.CreateGroup(ctx, tenantID, req)
		if err != nil {
			t.Fatalf("CreateGroup level %d failed: %v", i, err)
		}
		parentID = &group.ID
	}

	// 尝试创建第 6 层应该失败
	req := &CreateGroupRequest{
		Name:     "Level 5 (should fail)",
		ParentID: parentID,
	}
	_, err := service.CreateGroup(ctx, tenantID, req)
	if err != ErrGroupDepthExceeded {
		t.Errorf("expected ErrGroupDepthExceeded, got %v", err)
	}
}

func TestGroupService_CreateGroup_DuplicateName(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建第一个分组
	req := &CreateGroupRequest{Name: "Duplicate"}
	_, err := service.CreateGroup(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("First CreateGroup failed: %v", err)
	}

	// 尝试创建同名分组应该失败
	_, err = service.CreateGroup(ctx, tenantID, req)
	if err != ErrDuplicateGroupName {
		t.Errorf("expected ErrDuplicateGroupName, got %v", err)
	}
}

func TestGroupService_DeleteGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建分组
	req := &CreateGroupRequest{Name: "To Delete"}
	group, _ := service.CreateGroup(ctx, tenantID, req)

	// 删除分组
	err := service.DeleteGroup(ctx, tenantID, group.ID)
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	// 验证已删除
	_, err = service.GetGroup(ctx, tenantID, group.ID)
	if err != ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound after delete, got %v", err)
	}
}

func TestGroupService_DeleteGroup_WithChildren(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建父分组和子分组
	parent, _ := service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Parent"})
	_, _ = service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Child", ParentID: &parent.ID})

	// 尝试删除父分组应该失败
	err := service.DeleteGroup(ctx, tenantID, parent.ID)
	if err != ErrGroupHasChildren {
		t.Errorf("expected ErrGroupHasChildren, got %v", err)
	}
}

func TestGroupService_GetGroupTree(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建树结构
	root1, _ := service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Root1"})
	root2, _ := service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Root2"})
	_, _ = service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Child1", ParentID: &root1.ID})
	_, _ = service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Child2", ParentID: &root1.ID})

	// 获取树
	tree, err := service.GetGroupTree(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetGroupTree failed: %v", err)
	}

	if len(tree) != 2 {
		t.Errorf("expected 2 root nodes, got %d", len(tree))
	}

	// 验证 Root1 有 2 个子节点
	for _, node := range tree {
		if node.Name == "Root1" {
			if len(node.Children) != 2 {
				t.Errorf("expected Root1 to have 2 children, got %d", len(node.Children))
			}
		}
		if node.Name == "Root2" {
			if len(node.Children) != 0 {
				t.Errorf("expected Root2 to have 0 children, got %d", len(node.Children))
			}
		}
	}

	_ = root2 // 使用变量避免警告
}

func TestGroupService_AssignAndRemoveAsset(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建分组和资产
	group, _ := service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Test Group"})

	repo := NewAssetRepository(db, zap.NewNop())
	asset := &Asset{
		AgentID:  "agent-group-test",
		TenantID: tenantID,
		Hostname: "group-test-host",
		OSType:   "linux",
		Status:   AssetStatusOnline,
	}
	repo.Create(ctx, asset)

	// 分配资产到分组
	err := service.AssignAsset(ctx, tenantID, group.ID, asset.ID)
	if err != nil {
		t.Fatalf("AssignAsset failed: %v", err)
	}

	// 重复分配应该失败
	err = service.AssignAsset(ctx, tenantID, group.ID, asset.ID)
	if err != ErrAssetAlreadyInGroup {
		t.Errorf("expected ErrAssetAlreadyInGroup, got %v", err)
	}

	// 获取分组资产
	assets, total, err := service.GetAssetsByGroup(ctx, tenantID, group.ID, nil)
	if err != nil {
		t.Fatalf("GetAssetsByGroup failed: %v", err)
	}
	if total != 1 || len(assets) != 1 {
		t.Errorf("expected 1 asset, got %d", total)
	}

	// 移除资产
	err = service.RemoveAsset(ctx, tenantID, group.ID, asset.ID)
	if err != nil {
		t.Fatalf("RemoveAsset failed: %v", err)
	}

	// 重复移除应该失败
	err = service.RemoveAsset(ctx, tenantID, group.ID, asset.ID)
	if err != ErrAssetNotInGroup {
		t.Errorf("expected ErrAssetNotInGroup, got %v", err)
	}
}

func TestGroupService_UpdateGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	service := NewGroupService(db, zap.NewNop())
	ctx := context.Background()
	tenantID := uuid.New()

	// 创建分组
	group, _ := service.CreateGroup(ctx, tenantID, &CreateGroupRequest{Name: "Original"})

	// 更新分组
	newName := "Updated"
	newDesc := "Updated description"
	updated, err := service.UpdateGroup(ctx, tenantID, group.ID, &UpdateGroupRequest{
		Name:        &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("expected name=Updated, got %s", updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected description=Updated description, got %s", updated.Description)
	}
}
