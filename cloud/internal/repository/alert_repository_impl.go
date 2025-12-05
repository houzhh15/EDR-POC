package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// alertRepositoryImpl AlertRepository 实现
type alertRepositoryImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAlertRepository 创建 AlertRepository 实例
func NewAlertRepository(db *gorm.DB, logger *zap.Logger) AlertRepository {
	return &alertRepositoryImpl{
		db:     db,
		logger: logger.Named("alert_repository"),
	}
}

// Create 创建告警
func (r *alertRepositoryImpl) Create(ctx context.Context, alert *models.Alert) error {
	result := r.db.WithContext(ctx).Create(alert)
	if result.Error != nil {
		r.logger.Error("Failed to create alert",
			zap.String("tenant_id", alert.TenantID.String()),
			zap.String("title", alert.Title),
			zap.Error(result.Error),
		)
		return fmt.Errorf("create alert: %w", result.Error)
	}
	return nil
}

// FindByID 根据 ID 查询告警
func (r *alertRepositoryImpl) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Alert, error) {
	var alert models.Alert
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID)).
		Preload("Policy").
		First(&alert, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find alert: %w", result.Error)
	}
	return &alert, nil
}

// FindAll 分页查询告警
func (r *alertRepositoryImpl) FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions, filter AlertFilter) ([]*models.Alert, int64, error) {
	var alerts []*models.Alert
	var total int64

	opts.Normalize()

	query := r.db.WithContext(ctx).Model(&models.Alert{}).
		Scopes(TenantScope(tenantID))

	// 应用过滤条件
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.AssetID != nil {
		query = query.Where("asset_id = ?", *filter.AssetID)
	}
	if filter.PolicyID != nil {
		query = query.Where("policy_id = ?", *filter.PolicyID)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?",
			"%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}

	// 分页查询
	result := query.Scopes(PaginationScope(opts.Limit, opts.Offset)).
		Order("created_at DESC").
		Find(&alerts)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("find alerts: %w", result.Error)
	}

	return alerts, total, nil
}

// Update 更新告警
func (r *alertRepositoryImpl) Update(ctx context.Context, alert *models.Alert) error {
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(alert.TenantID)).
		Save(alert)
	if result.Error != nil {
		return fmt.Errorf("update alert: %w", result.Error)
	}
	return nil
}

// UpdateStatus 更新告警状态
func (r *alertRepositoryImpl) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status models.AlertStatus, userID uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("update alert status: %w", result.Error)
	}
	return nil
}

// Acknowledge 确认告警
func (r *alertRepositoryImpl) Acknowledge(ctx context.Context, tenantID, id uuid.UUID, userID uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          models.AlertStatusAcknowledged,
			"acknowledged_at": now,
			"acknowledged_by": userID,
			"updated_at":      now,
		})
	if result.Error != nil {
		return fmt.Errorf("acknowledge alert: %w", result.Error)
	}
	return nil
}

// Resolve 解决告警
func (r *alertRepositoryImpl) Resolve(ctx context.Context, tenantID, id uuid.UUID, userID uuid.UUID, resolution string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.AlertStatusResolved,
			"resolved_at": now,
			"resolved_by": userID,
			"resolution":  resolution,
			"updated_at":  now,
		})
	if result.Error != nil {
		return fmt.Errorf("resolve alert: %w", result.Error)
	}
	return nil
}

// AssignTo 分配告警
func (r *alertRepositoryImpl) AssignTo(ctx context.Context, tenantID, id uuid.UUID, assigneeID uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"assigned_to": assigneeID,
			"updated_at":  now,
		})
	if result.Error != nil {
		return fmt.Errorf("assign alert: %w", result.Error)
	}
	return nil
}

// CountByStatus 按状态统计告警数量
func (r *alertRepositoryImpl) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[models.AlertStatus]int64, error) {
	type StatusCount struct {
		Status models.AlertStatus
		Count  int64
	}
	var results []StatusCount

	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("count alerts by status: %w", err)
	}

	counts := make(map[models.AlertStatus]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// CountBySeverity 按严重程度统计告警数量
func (r *alertRepositoryImpl) CountBySeverity(ctx context.Context, tenantID uuid.UUID) (map[models.AlertSeverity]int64, error) {
	type SeverityCount struct {
		Severity models.AlertSeverity
		Count    int64
	}
	var results []SeverityCount

	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Scopes(TenantScope(tenantID)).
		Select("severity, COUNT(*) as count").
		Group("severity").
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("count alerts by severity: %w", err)
	}

	counts := make(map[models.AlertSeverity]int64)
	for _, r := range results {
		counts[r.Severity] = r.Count
	}
	return counts, nil
}
