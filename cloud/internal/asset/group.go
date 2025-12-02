package asset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// MaxGroupDepth 最大分组层级
	MaxGroupDepth = 5
)

// GroupService 分组管理服务
type GroupService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewGroupService 创建分组服务实例
func NewGroupService(db *gorm.DB, logger *zap.Logger) *GroupService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &GroupService{
		db:     db,
		logger: logger,
	}
}

// CreateGroup 创建分组
func (s *GroupService) CreateGroup(ctx context.Context, tenantID uuid.UUID, req *CreateGroupRequest) (*AssetGroup, error) {
	group := &AssetGroup{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		ParentID:    req.ParentID,
	}

	// 设置默认类型
	if group.Type == "" {
		group.Type = GroupTypeCustom
	}

	// 计算路径和层级
	if req.ParentID != nil {
		parent, err := s.GetGroup(ctx, tenantID, *req.ParentID)
		if err != nil {
			return nil, ErrGroupNotFound.WithError(err)
		}

		// 检查层级深度
		if parent.Level >= MaxGroupDepth-1 {
			return nil, ErrGroupDepthExceeded
		}

		group.Level = parent.Level + 1
		// 路径在 BeforeCreate 中设置，这里先记录父路径
		group.Path = parent.Path // 临时，创建后会更新
	} else {
		group.Level = 0
		group.Path = "/" // 根分组
	}

	// 检查同级名称唯一性
	var count int64
	query := s.db.WithContext(ctx).Model(&AssetGroup{}).
		Where("tenant_id = ? AND name = ?", tenantID, req.Name)
	if req.ParentID != nil {
		query = query.Where("parent_id = ?", *req.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	query.Count(&count)
	if count > 0 {
		return nil, ErrDuplicateGroupName
	}

	// 创建分组
	if err := s.db.WithContext(ctx).Create(group).Error; err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	// 更新路径（包含自身 ID）
	if req.ParentID != nil {
		parent, _ := s.GetGroup(ctx, tenantID, *req.ParentID)
		group.Path = fmt.Sprintf("%s%s/", parent.Path, group.ID.String())
	} else {
		group.Path = fmt.Sprintf("/%s/", group.ID.String())
	}
	s.db.WithContext(ctx).Model(group).Update("path", group.Path)

	s.logger.Info("group created",
		zap.String("id", group.ID.String()),
		zap.String("name", group.Name),
		zap.Int("level", group.Level),
	)

	return group, nil
}

// GetGroup 获取分组详情
func (s *GroupService) GetGroup(ctx context.Context, tenantID, groupID uuid.UUID) (*AssetGroup, error) {
	var group AssetGroup
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, groupID).
		First(&group).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	return &group, nil
}

// UpdateGroup 更新分组
func (s *GroupService) UpdateGroup(ctx context.Context, tenantID, groupID uuid.UUID, req *UpdateGroupRequest) (*AssetGroup, error) {
	group, err := s.GetGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		// 检查同级名称唯一性
		var count int64
		query := s.db.WithContext(ctx).Model(&AssetGroup{}).
			Where("tenant_id = ? AND name = ? AND id != ?", tenantID, *req.Name, groupID)
		if group.ParentID != nil {
			query = query.Where("parent_id = ?", *group.ParentID)
		} else {
			query = query.Where("parent_id IS NULL")
		}
		query.Count(&count)
		if count > 0 {
			return nil, ErrDuplicateGroupName
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.db.WithContext(ctx).Model(group).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("update group: %w", err)
		}
	}

	return s.GetGroup(ctx, tenantID, groupID)
}

// DeleteGroup 删除分组
func (s *GroupService) DeleteGroup(ctx context.Context, tenantID, groupID uuid.UUID) error {
	// 检查是否存在
	group, err := s.GetGroup(ctx, tenantID, groupID)
	if err != nil {
		return err
	}

	// 检查是否有子分组
	var childCount int64
	s.db.WithContext(ctx).Model(&AssetGroup{}).
		Where("tenant_id = ? AND parent_id = ?", tenantID, groupID).
		Count(&childCount)
	if childCount > 0 {
		return ErrGroupHasChildren
	}

	// 删除分组成员关联
	s.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Delete(&AssetGroupMember{})

	// 删除分组
	if err := s.db.WithContext(ctx).Delete(group).Error; err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	s.logger.Info("group deleted",
		zap.String("id", groupID.String()),
		zap.String("name", group.Name),
	)

	return nil
}

// GetGroupTree 获取分组树
func (s *GroupService) GetGroupTree(ctx context.Context, tenantID uuid.UUID) ([]*AssetGroup, error) {
	var groups []*AssetGroup
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("path ASC").
		Find(&groups).Error

	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	// 构建树结构
	return s.buildTree(groups), nil
}

// buildTree 从扁平列表构建树结构
func (s *GroupService) buildTree(groups []*AssetGroup) []*AssetGroup {
	if len(groups) == 0 {
		return nil
	}

	// 创建 ID -> Group 映射
	groupMap := make(map[uuid.UUID]*AssetGroup)
	for _, g := range groups {
		g.Children = nil // 清空子节点
		groupMap[g.ID] = g
	}

	var roots []*AssetGroup

	for _, g := range groups {
		if g.ParentID == nil {
			roots = append(roots, g)
		} else {
			if parent, ok := groupMap[*g.ParentID]; ok {
				parent.Children = append(parent.Children, g)
			}
		}
	}

	return roots
}

// AssignAsset 将资产添加到分组
func (s *GroupService) AssignAsset(ctx context.Context, tenantID, groupID, assetID uuid.UUID) error {
	// 验证分组存在
	if _, err := s.GetGroup(ctx, tenantID, groupID); err != nil {
		return err
	}

	// 检查是否已存在
	var count int64
	s.db.WithContext(ctx).Model(&AssetGroupMember{}).
		Where("group_id = ? AND asset_id = ?", groupID, assetID).
		Count(&count)
	if count > 0 {
		return ErrAssetAlreadyInGroup
	}

	// 创建关联
	member := &AssetGroupMember{
		AssetID:  assetID,
		GroupID:  groupID,
		JoinedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
		return fmt.Errorf("assign asset to group: %w", err)
	}

	s.logger.Debug("asset assigned to group",
		zap.String("asset_id", assetID.String()),
		zap.String("group_id", groupID.String()),
	)

	return nil
}

// RemoveAsset 从分组移除资产
func (s *GroupService) RemoveAsset(ctx context.Context, tenantID, groupID, assetID uuid.UUID) error {
	// 验证分组存在
	if _, err := s.GetGroup(ctx, tenantID, groupID); err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Where("group_id = ? AND asset_id = ?", groupID, assetID).
		Delete(&AssetGroupMember{})

	if result.Error != nil {
		return fmt.Errorf("remove asset from group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAssetNotInGroup
	}

	s.logger.Debug("asset removed from group",
		zap.String("asset_id", assetID.String()),
		zap.String("group_id", groupID.String()),
	)

	return nil
}

// GetAssetsByGroup 获取分组下的资产列表
func (s *GroupService) GetAssetsByGroup(ctx context.Context, tenantID, groupID uuid.UUID, opts *QueryOptions) ([]*Asset, int64, error) {
	// 验证分组存在
	if _, err := s.GetGroup(ctx, tenantID, groupID); err != nil {
		return nil, 0, err
	}

	var assets []*Asset
	var total int64

	query := s.db.WithContext(ctx).
		Model(&Asset{}).
		Joins("JOIN asset_group_members ON assets.id = asset_group_members.asset_id").
		Where("asset_group_members.group_id = ?", groupID).
		Where("assets.tenant_id = ?", tenantID).
		Where("assets.deleted_at IS NULL")

	// 应用额外过滤
	if opts != nil && opts.Status != "" {
		query = query.Where("assets.status = ?", opts.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count assets in group: %w", err)
	}

	// 分页
	if opts != nil {
		opts.Pagination.Normalize()
		query = query.Offset(opts.Pagination.Offset()).Limit(opts.Pagination.PageSize)
	}

	// 排序
	query = query.Order("assets.last_seen_at DESC NULLS LAST")

	if err := query.Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("get assets in group: %w", err)
	}

	return assets, total, nil
}

// GetGroupsByAsset 获取资产所属的所有分组
func (s *GroupService) GetGroupsByAsset(ctx context.Context, tenantID, assetID uuid.UUID) ([]*AssetGroup, error) {
	var groups []*AssetGroup

	err := s.db.WithContext(ctx).
		Model(&AssetGroup{}).
		Joins("JOIN asset_group_members ON asset_groups.id = asset_group_members.group_id").
		Where("asset_group_members.asset_id = ?", assetID).
		Where("asset_groups.tenant_id = ?", tenantID).
		Find(&groups).Error

	if err != nil {
		return nil, fmt.Errorf("get groups by asset: %w", err)
	}
	return groups, nil
}

// CountAssetsByGroup 统计分组下的资产数量
func (s *GroupService) CountAssetsByGroup(ctx context.Context, groupID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&AssetGroupMember{}).
		Where("group_id = ?", groupID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("count assets by group: %w", err)
	}
	return count, nil
}

// GetDescendantGroups 获取所有子孙分组（通过物化路径）
func (s *GroupService) GetDescendantGroups(ctx context.Context, tenantID, groupID uuid.UUID) ([]*AssetGroup, error) {
	group, err := s.GetGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}

	var groups []*AssetGroup
	// 使用 LIKE 查询物化路径
	pattern := group.Path + "%"
	err = s.db.WithContext(ctx).
		Where("tenant_id = ? AND path LIKE ? AND id != ?", tenantID, pattern, groupID).
		Order("path ASC").
		Find(&groups).Error

	if err != nil {
		return nil, fmt.Errorf("get descendant groups: %w", err)
	}
	return groups, nil
}

// GetAncestorGroups 获取所有祖先分组
func (s *GroupService) GetAncestorGroups(ctx context.Context, tenantID, groupID uuid.UUID) ([]*AssetGroup, error) {
	group, err := s.GetGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}

	if group.Path == "" || group.Path == "/" {
		return nil, nil
	}

	// 从路径中提取所有祖先 ID
	parts := strings.Split(strings.Trim(group.Path, "/"), "/")
	if len(parts) <= 1 {
		return nil, nil
	}

	var ancestorIDs []uuid.UUID
	for i := 0; i < len(parts)-1; i++ { // 排除自身
		if id, err := uuid.Parse(parts[i]); err == nil {
			ancestorIDs = append(ancestorIDs, id)
		}
	}

	if len(ancestorIDs) == 0 {
		return nil, nil
	}

	var groups []*AssetGroup
	err = s.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ancestorIDs).
		Order("level ASC").
		Find(&groups).Error

	if err != nil {
		return nil, fmt.Errorf("get ancestor groups: %w", err)
	}
	return groups, nil
}
