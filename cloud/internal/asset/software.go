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

// SoftwareService 软件清单服务
type SoftwareService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewSoftwareService 创建软件清单服务实例
func NewSoftwareService(db *gorm.DB, logger *zap.Logger) *SoftwareService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &SoftwareService{
		db:     db,
		logger: logger,
	}
}

// UpdateSoftwareInventory 更新软件清单（全量替换策略）
func (s *SoftwareService) UpdateSoftwareInventory(ctx context.Context, assetID uuid.UUID, items []SoftwareItem) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧记录
		if err := tx.Where("asset_id = ?", assetID).Delete(&SoftwareInventory{}).Error; err != nil {
			return fmt.Errorf("delete old software: %w", err)
		}

		// 批量插入新记录
		if len(items) == 0 {
			return nil
		}

		now := time.Now()
		records := make([]*SoftwareInventory, len(items))
		for i, item := range items {
			records[i] = &SoftwareInventory{
				AssetID:     assetID,
				Name:        item.Name,
				Version:     item.Version,
				Publisher:   item.Publisher,
				InstallDate: item.InstallDate,
				InstallPath: item.InstallPath,
				Size:        item.Size,
				UpdatedAt:   now,
			}
		}

		if err := tx.Create(&records).Error; err != nil {
			return fmt.Errorf("insert software: %w", err)
		}

		s.logger.Info("software inventory updated",
			zap.String("asset_id", assetID.String()),
			zap.Int("count", len(items)),
		)

		return nil
	})
}

// GetSoftwareByAsset 获取资产的软件清单
func (s *SoftwareService) GetSoftwareByAsset(ctx context.Context, assetID uuid.UUID, opts *SoftwareQueryOptions) ([]*SoftwareInventory, int64, error) {
	var software []*SoftwareInventory
	var total int64

	query := s.db.WithContext(ctx).
		Model(&SoftwareInventory{}).
		Where("asset_id = ?", assetID)

	// 应用过滤（使用 LOWER + LIKE 保持跨数据库兼容性）
	if opts != nil {
		if opts.Name != "" {
			pattern := "%" + strings.ToLower(opts.Name) + "%"
			query = query.Where("LOWER(name) LIKE ?", pattern)
		}
		if opts.Publisher != "" {
			pattern := "%" + strings.ToLower(opts.Publisher) + "%"
			query = query.Where("LOWER(publisher) LIKE ?", pattern)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count software: %w", err)
	}

	// 分页
	if opts != nil {
		opts.Pagination.Normalize()
		query = query.Offset(opts.Pagination.Offset()).Limit(opts.Pagination.PageSize)
	}

	// 排序
	query = query.Order("name ASC")

	if err := query.Find(&software).Error; err != nil {
		return nil, 0, fmt.Errorf("get software: %w", err)
	}

	return software, total, nil
}

// SearchSoftware 跨资产搜索软件
func (s *SoftwareService) SearchSoftware(ctx context.Context, tenantID uuid.UUID, name string, opts *Pagination) ([]*SoftwareSearchResult, int64, error) {
	var results []*SoftwareSearchResult
	var total int64

	// 搜索软件名称
	pattern := "%" + strings.ToLower(name) + "%"

	query := s.db.WithContext(ctx).
		Table("software_inventory si").
		Select("si.*, a.hostname, a.id as asset_uuid").
		Joins("JOIN assets a ON si.asset_id = a.id").
		Where("a.tenant_id = ?", tenantID).
		Where("a.deleted_at IS NULL").
		Where("LOWER(si.name) LIKE ?", pattern)

	// 统计总数（使用子查询）
	countQuery := s.db.WithContext(ctx).
		Table("software_inventory si").
		Joins("JOIN assets a ON si.asset_id = a.id").
		Where("a.tenant_id = ?", tenantID).
		Where("a.deleted_at IS NULL").
		Where("LOWER(si.name) LIKE ?", pattern)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	// 分页
	if opts != nil {
		opts.Normalize()
		query = query.Offset(opts.Offset()).Limit(opts.PageSize)
	}

	// 排序
	query = query.Order("si.name ASC, a.hostname ASC")

	if err := query.Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("search software: %w", err)
	}

	return results, total, nil
}

// SoftwareSearchResult 软件搜索结果
type SoftwareSearchResult struct {
	SoftwareInventory
	Hostname  string    `gorm:"column:hostname" json:"hostname"`
	AssetUUID uuid.UUID `gorm:"column:asset_uuid" json:"asset_uuid"`
}

// GetSoftwareStats 获取软件统计信息
func (s *SoftwareService) GetSoftwareStats(ctx context.Context, tenantID uuid.UUID) (*SoftwareStats, error) {
	var stats SoftwareStats

	// 统计软件总数
	err := s.db.WithContext(ctx).
		Table("software_inventory si").
		Joins("JOIN assets a ON si.asset_id = a.id").
		Where("a.tenant_id = ?", tenantID).
		Where("a.deleted_at IS NULL").
		Count(&stats.TotalCount).Error
	if err != nil {
		return nil, fmt.Errorf("count total software: %w", err)
	}

	// 统计唯一软件数
	err = s.db.WithContext(ctx).
		Table("software_inventory si").
		Select("COUNT(DISTINCT si.name)").
		Joins("JOIN assets a ON si.asset_id = a.id").
		Where("a.tenant_id = ?", tenantID).
		Where("a.deleted_at IS NULL").
		Scan(&stats.UniqueCount).Error
	if err != nil {
		return nil, fmt.Errorf("count unique software: %w", err)
	}

	return &stats, nil
}

// SoftwareStats 软件统计
type SoftwareStats struct {
	TotalCount  int64 `json:"total_count"`
	UniqueCount int64 `json:"unique_count"`
}

// GetTopSoftware 获取安装最多的软件
func (s *SoftwareService) GetTopSoftware(ctx context.Context, tenantID uuid.UUID, limit int) ([]*SoftwarePopularity, error) {
	if limit <= 0 {
		limit = 10
	}

	var results []*SoftwarePopularity

	err := s.db.WithContext(ctx).
		Table("software_inventory si").
		Select("si.name, si.version, COUNT(*) as install_count").
		Joins("JOIN assets a ON si.asset_id = a.id").
		Where("a.tenant_id = ?", tenantID).
		Where("a.deleted_at IS NULL").
		Group("si.name, si.version").
		Order("install_count DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("get top software: %w", err)
	}

	return results, nil
}

// SoftwarePopularity 软件流行度
type SoftwarePopularity struct {
	Name         string `gorm:"column:name" json:"name"`
	Version      string `gorm:"column:version" json:"version"`
	InstallCount int64  `gorm:"column:install_count" json:"install_count"`
}

// DeleteSoftwareByAsset 删除资产的所有软件记录
func (s *SoftwareService) DeleteSoftwareByAsset(ctx context.Context, assetID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Delete(&SoftwareInventory{})

	if result.Error != nil {
		return fmt.Errorf("delete software by asset: %w", result.Error)
	}

	s.logger.Debug("software deleted for asset",
		zap.String("asset_id", assetID.String()),
		zap.Int64("count", result.RowsAffected),
	)

	return nil
}
