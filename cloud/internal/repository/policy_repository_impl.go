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

// policyRepositoryImpl PolicyRepository 实现
type policyRepositoryImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPolicyRepository 创建 PolicyRepository 实例
func NewPolicyRepository(db *gorm.DB, logger *zap.Logger) PolicyRepository {
	return &policyRepositoryImpl{
		db:     db,
		logger: logger.Named("policy_repository"),
	}
}

// Create 创建策略
func (r *policyRepositoryImpl) Create(ctx context.Context, policy *models.Policy) error {
	result := r.db.WithContext(ctx).Create(policy)
	if result.Error != nil {
		r.logger.Error("Failed to create policy",
			zap.String("tenant_id", policy.TenantID.String()),
			zap.String("name", policy.Name),
			zap.Error(result.Error),
		)
		return fmt.Errorf("create policy: %w", result.Error)
	}
	return nil
}

// FindByID 根据 ID 查询策略
func (r *policyRepositoryImpl) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Policy, error) {
	var policy models.Policy
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		First(&policy, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find policy: %w", result.Error)
	}
	return &policy, nil
}

// FindAll 分页查询策略
func (r *policyRepositoryImpl) FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions, filter PolicyFilter) ([]*models.Policy, int64, error) {
	var policies []*models.Policy
	var total int64

	opts.Normalize()

	query := r.db.WithContext(ctx).Model(&models.Policy{}).
		Scopes(TenantScope(tenantID), NotDeletedScope())

	// 应用过滤条件
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}
	if filter.Search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?",
			"%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count policies: %w", err)
	}

	// 分页查询
	result := query.Scopes(PaginationScope(opts.Limit, opts.Offset)).
		Order("priority DESC, created_at DESC").
		Find(&policies)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("find policies: %w", result.Error)
	}

	return policies, total, nil
}

// FindEnabled 查询已启用的策略
func (r *policyRepositoryImpl) FindEnabled(ctx context.Context, tenantID uuid.UUID, policyType models.PolicyType) ([]*models.Policy, error) {
	var policies []*models.Policy
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope(), EnabledScope(true)).
		Where("type = ?", policyType).
		Order("priority DESC").
		Find(&policies)
	if result.Error != nil {
		return nil, fmt.Errorf("find enabled policies: %w", result.Error)
	}
	return policies, nil
}

// Update 更新策略
func (r *policyRepositoryImpl) Update(ctx context.Context, policy *models.Policy) error {
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(policy.TenantID), NotDeletedScope()).
		Save(policy)
	if result.Error != nil {
		return fmt.Errorf("update policy: %w", result.Error)
	}
	return nil
}

// Delete 软删除策略
func (r *policyRepositoryImpl) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Policy{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return fmt.Errorf("delete policy: %w", result.Error)
	}
	return nil
}

// SetEnabled 设置启用状态
func (r *policyRepositoryImpl) SetEnabled(ctx context.Context, tenantID, id uuid.UUID, enabled bool) error {
	result := r.db.WithContext(ctx).
		Model(&models.Policy{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", id).
		Update("enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("set policy enabled: %w", result.Error)
	}
	return nil
}
