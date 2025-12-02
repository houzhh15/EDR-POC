package asset

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChangeLogger 资产变更日志记录器
type ChangeLogger struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewChangeLogger 创建变更日志记录器
func NewChangeLogger(db *gorm.DB, logger *zap.Logger) *ChangeLogger {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &ChangeLogger{
		db:     db,
		logger: logger,
	}
}

// LogChange 记录字段变更
func (c *ChangeLogger) LogChange(ctx context.Context, assetID uuid.UUID, fieldName, oldValue, newValue, changedBy string) error {
	log := &AssetChangeLog{
		AssetID:   assetID,
		FieldName: fieldName,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}

	if err := c.db.WithContext(ctx).Create(log).Error; err != nil {
		c.logger.Error("failed to log change",
			zap.String("asset_id", assetID.String()),
			zap.String("field", fieldName),
			zap.Error(err),
		)
		return fmt.Errorf("log change: %w", err)
	}

	c.logger.Debug("change logged",
		zap.String("asset_id", assetID.String()),
		zap.String("field", fieldName),
		zap.String("old", oldValue),
		zap.String("new", newValue),
	)
	return nil
}

// LogMultipleChanges 批量记录多个字段变更
func (c *ChangeLogger) LogMultipleChanges(ctx context.Context, assetID uuid.UUID, changes []FieldChange, changedBy string) error {
	if len(changes) == 0 {
		return nil
	}

	logs := make([]*AssetChangeLog, len(changes))
	now := time.Now()
	for i, change := range changes {
		logs[i] = &AssetChangeLog{
			AssetID:   assetID,
			FieldName: change.FieldName,
			OldValue:  change.OldValue,
			NewValue:  change.NewValue,
			ChangedBy: changedBy,
			ChangedAt: now,
		}
	}

	if err := c.db.WithContext(ctx).Create(&logs).Error; err != nil {
		c.logger.Error("failed to log multiple changes",
			zap.String("asset_id", assetID.String()),
			zap.Int("count", len(changes)),
			zap.Error(err),
		)
		return fmt.Errorf("log multiple changes: %w", err)
	}

	c.logger.Debug("multiple changes logged",
		zap.String("asset_id", assetID.String()),
		zap.Int("count", len(changes)),
	)
	return nil
}

// FieldChange 字段变更记录
type FieldChange struct {
	FieldName string
	OldValue  string
	NewValue  string
}

// GetChangeHistory 获取变更历史
func (c *ChangeLogger) GetChangeHistory(ctx context.Context, assetID uuid.UUID, opts *ChangeLogQueryOptions) ([]*AssetChangeLog, int64, error) {
	var logs []*AssetChangeLog
	var total int64

	query := c.db.WithContext(ctx).
		Model(&AssetChangeLog{}).
		Where("asset_id = ?", assetID)

	// 应用过滤条件
	if opts != nil {
		if opts.FieldName != "" {
			query = query.Where("field_name = ?", opts.FieldName)
		}
		if opts.StartTime != nil {
			query = query.Where("changed_at >= ?", *opts.StartTime)
		}
		if opts.EndTime != nil {
			query = query.Where("changed_at <= ?", *opts.EndTime)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count change logs: %w", err)
	}

	// 分页
	if opts != nil {
		opts.Pagination.Normalize()
		query = query.Offset(opts.Pagination.Offset()).Limit(opts.Pagination.PageSize)
	}

	// 按时间倒序
	query = query.Order("changed_at DESC")

	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("find change logs: %w", err)
	}

	return logs, total, nil
}

// GetLatestChanges 获取最近的变更记录
func (c *ChangeLogger) GetLatestChanges(ctx context.Context, assetID uuid.UUID, limit int) ([]*AssetChangeLog, error) {
	var logs []*AssetChangeLog

	if limit <= 0 {
		limit = 10
	}

	err := c.db.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Order("changed_at DESC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("get latest changes: %w", err)
	}
	return logs, nil
}
