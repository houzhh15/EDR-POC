package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AssetRepository 资产数据访问层
type AssetRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAssetRepository 创建资产仓储实例
func NewAssetRepository(db *gorm.DB, logger *zap.Logger) *AssetRepository {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &AssetRepository{
		db:     db,
		logger: logger,
	}
}

// ==================== GORM Scopes ====================

// TenantScope 多租户过滤条件
func TenantScope(tenantID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// NotDeletedScope 软删除过滤条件
func NotDeletedScope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL")
	}
}

// StatusScope 状态过滤条件
func StatusScope(status AssetStatus) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status == "" {
			return db
		}
		return db.Where("status = ?", status)
	}
}

// ==================== Asset CRUD ====================

// Create 创建新资产
func (r *AssetRepository) Create(ctx context.Context, asset *Asset) (*Asset, error) {
	// BeforeCreate hook 会处理 ID、时间戳和默认状态
	if err := r.db.WithContext(ctx).Create(asset).Error; err != nil {
		r.logger.Error("failed to create asset",
			zap.String("agent_id", asset.AgentID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("create asset: %w", err)
	}

	r.logger.Info("asset created",
		zap.String("id", asset.ID.String()),
		zap.String("agent_id", asset.AgentID),
		zap.String("hostname", asset.Hostname),
	)
	return asset, nil
}

// Update 更新资产信息
func (r *AssetRepository) Update(ctx context.Context, asset *Asset) (*Asset, error) {
	asset.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).
		Model(asset).
		Scopes(TenantScope(asset.TenantID), NotDeletedScope()).
		Updates(map[string]interface{}{
			"hostname":      asset.Hostname,
			"os_type":       asset.OSType,
			"os_version":    asset.OSVersion,
			"architecture":  asset.Architecture,
			"ip_addresses":  asset.IPAddresses,
			"mac_addresses": asset.MACAddresses,
			"agent_version": asset.AgentVersion,
			"status":        asset.Status,
			"last_seen_at":  asset.LastSeenAt,
			"updated_at":    asset.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.Error("failed to update asset",
			zap.String("id", asset.ID.String()),
			zap.Error(result.Error),
		)
		return nil, fmt.Errorf("update asset: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, ErrAssetNotFound
	}

	return asset, nil
}

// FindByID 按 ID 查询资产
func (r *AssetRepository) FindByID(ctx context.Context, tenantID, assetID uuid.UUID) (*Asset, error) {
	var asset Asset
	err := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", assetID).
		First(&asset).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, fmt.Errorf("find asset by id: %w", err)
	}
	return &asset, nil
}

// FindByAgentID 按 Agent ID 查询资产
func (r *AssetRepository) FindByAgentID(ctx context.Context, tenantID uuid.UUID, agentID string) (*Asset, error) {
	var asset Asset
	err := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("agent_id = ?", agentID).
		First(&asset).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, fmt.Errorf("find asset by agent_id: %w", err)
	}
	return &asset, nil
}

// FindAll 分页查询资产列表
func (r *AssetRepository) FindAll(ctx context.Context, tenantID uuid.UUID, opts *QueryOptions) ([]*Asset, int64, error) {
	var assets []*Asset
	var total int64

	query := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(TenantScope(tenantID), NotDeletedScope())

	// 应用过滤条件
	query = r.applyFilters(query, opts)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count assets: %w", err)
	}

	// 排序
	query = r.applySorting(query, opts)

	// 分页
	opts.Pagination.Normalize()
	query = query.Offset(opts.Pagination.Offset()).Limit(opts.Pagination.PageSize)

	// 执行查询
	if err := query.Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("find assets: %w", err)
	}

	return assets, total, nil
}

// applyFilters 应用过滤条件
func (r *AssetRepository) applyFilters(query *gorm.DB, opts *QueryOptions) *gorm.DB {
	if opts == nil {
		return query
	}

	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}

	if opts.OSType != "" {
		query = query.Where("os_type = ?", opts.OSType)
	}

	if opts.Hostname != "" {
		// 支持模糊搜索，将 * 转换为 %
		pattern := strings.ReplaceAll(opts.Hostname, "*", "%")
		if !strings.Contains(pattern, "%") {
			pattern = "%" + pattern + "%"
		}
		query = query.Where("hostname ILIKE ?", pattern)
	}

	if opts.IP != "" {
		// IP 地址在 TEXT[] 数组中搜索
		pattern := strings.ReplaceAll(opts.IP, "*", "%")
		if !strings.Contains(pattern, "%") {
			pattern = "%" + pattern + "%"
		}
		query = query.Where("EXISTS (SELECT 1 FROM unnest(ip_addresses) ip WHERE ip LIKE ?)", pattern)
	}

	if opts.GroupID != nil {
		// 通过关联表过滤
		query = query.Where("id IN (SELECT asset_id FROM asset_group_members WHERE group_id = ?)", *opts.GroupID)
	}

	return query
}

// applySorting 应用排序
func (r *AssetRepository) applySorting(query *gorm.DB, opts *QueryOptions) *gorm.DB {
	if opts == nil {
		return query.Order("last_seen_at DESC NULLS LAST")
	}

	sortField := opts.SortBy
	if sortField == "" {
		sortField = "last_seen_at"
	}

	// 验证排序字段，防止 SQL 注入
	validFields := map[string]string{
		"last_seen_at":  "last_seen_at",
		"hostname":      "hostname",
		"created_at":    "created_at",
		"first_seen_at": "first_seen_at",
		"os_type":       "os_type",
	}

	dbField, ok := validFields[sortField]
	if !ok {
		dbField = "last_seen_at"
	}

	order := "DESC"
	if opts.SortOrder == "asc" {
		order = "ASC"
	}

	// 处理 NULL 值排序
	nullsOrder := "NULLS LAST"
	if order == "ASC" {
		nullsOrder = "NULLS FIRST"
	}

	return query.Order(fmt.Sprintf("%s %s %s", dbField, order, nullsOrder))
}

// SoftDelete 软删除资产
func (r *AssetRepository) SoftDelete(ctx context.Context, tenantID, assetID uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", assetID).
		Update("deleted_at", now)

	if result.Error != nil {
		return fmt.Errorf("soft delete asset: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrAssetNotFound
	}

	r.logger.Info("asset soft deleted",
		zap.String("id", assetID.String()),
		zap.String("tenant_id", tenantID.String()),
	)
	return nil
}

// UpdateStatus 更新资产状态
func (r *AssetRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, assetID uuid.UUID, status AssetStatus) error {
	result := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", assetID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("update asset status: %w", result.Error)
	}
	return nil
}

// BatchUpdateStatus 批量更新资产状态
func (r *AssetRepository) BatchUpdateStatus(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, status AssetStatus) error {
	if len(assetIDs) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id IN ?", assetIDs).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("batch update asset status: %w", result.Error)
	}

	r.logger.Info("batch updated asset status",
		zap.Int("count", int(result.RowsAffected)),
		zap.String("status", string(status)),
	)
	return nil
}

// UpdateStatusByAgentID 按 AgentID 更新状态
// 如果 tenantID 为 uuid.Nil，则跨租户更新（用于状态监控）
func (r *AssetRepository) UpdateStatusByAgentID(ctx context.Context, tenantID uuid.UUID, agentID string, status AssetStatus) error {
	query := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(NotDeletedScope()).
		Where("agent_id = ?", agentID)

	// 只有当 tenantID 非空时才加租户过滤
	if tenantID != uuid.Nil {
		query = query.Scopes(TenantScope(tenantID))
	}

	result := query.Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return fmt.Errorf("update asset status by agent_id: %w", result.Error)
	}
	return nil
}

// GetOnlineAssets 获取所有在线资产的 AgentID 列表
func (r *AssetRepository) GetOnlineAssets(ctx context.Context, tenantID uuid.UUID) ([]OnlineAssetInfo, error) {
	var results []OnlineAssetInfo

	err := r.db.WithContext(ctx).
		Model(&Asset{}).
		Select("id, agent_id").
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("status = ?", AssetStatusOnline).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("get online assets: %w", err)
	}
	return results, nil
}

// OnlineAssetInfo 在线资产简要信息
type OnlineAssetInfo struct {
	ID      uuid.UUID `gorm:"column:id"`
	AgentID string    `gorm:"column:agent_id"`
}

// GetAllOnlineAssets 获取所有租户的在线资产（用于状态监控）
func (r *AssetRepository) GetAllOnlineAssets(ctx context.Context) ([]OnlineAssetInfo, error) {
	var results []OnlineAssetInfo

	err := r.db.WithContext(ctx).
		Model(&Asset{}).
		Select("id, agent_id").
		Scopes(NotDeletedScope()).
		Where("status = ?", AssetStatusOnline).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("get all online assets: %w", err)
	}
	return results, nil
}

// UpdateLastSeen 更新最后在线时间
func (r *AssetRepository) UpdateLastSeen(ctx context.Context, tenantID uuid.UUID, agentID string, lastSeenAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&Asset{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"last_seen_at": lastSeenAt,
			"status":       AssetStatusOnline,
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("update last seen: %w", result.Error)
	}
	return nil
}

// CountByStatus 按状态统计资产数量
func (r *AssetRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[AssetStatus]int64, error) {
	type StatusCount struct {
		Status AssetStatus `gorm:"column:status"`
		Count  int64       `gorm:"column:count"`
	}

	var results []StatusCount
	err := r.db.WithContext(ctx).
		Model(&Asset{}).
		Select("status, COUNT(*) as count").
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}

	counts := make(map[AssetStatus]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}
