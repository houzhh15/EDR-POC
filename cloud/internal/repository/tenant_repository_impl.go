package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// tenantRepositoryImpl TenantRepository 实现
type tenantRepositoryImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTenantRepository 创建 TenantRepository 实例
func NewTenantRepository(db *gorm.DB, logger *zap.Logger) TenantRepository {
	return &tenantRepositoryImpl{
		db:     db,
		logger: logger.Named("tenant_repository"),
	}
}

// Create 创建租户
func (r *tenantRepositoryImpl) Create(ctx context.Context, tenant *models.Tenant) error {
	result := r.db.WithContext(ctx).Create(tenant)
	if result.Error != nil {
		r.logger.Error("Failed to create tenant",
			zap.String("name", tenant.Name),
			zap.Error(result.Error),
		)
		return fmt.Errorf("create tenant: %w", result.Error)
	}
	r.logger.Info("Tenant created",
		zap.String("id", tenant.ID.String()),
		zap.String("name", tenant.Name),
	)
	return nil
}

// FindByID 根据 ID 查询租户
func (r *tenantRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	result := r.db.WithContext(ctx).First(&tenant, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 未找到返回 nil，不返回错误
		}
		return nil, fmt.Errorf("find tenant by id: %w", result.Error)
	}
	return &tenant, nil
}

// FindByName 根据名称查询租户
func (r *tenantRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Tenant, error) {
	var tenant models.Tenant
	result := r.db.WithContext(ctx).First(&tenant, "name = ?", name)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find tenant by name: %w", result.Error)
	}
	return &tenant, nil
}

// FindAll 分页查询所有租户
func (r *tenantRepositoryImpl) FindAll(ctx context.Context, opts models.ListOptions) ([]*models.Tenant, int64, error) {
	var tenants []*models.Tenant
	var total int64

	opts.Normalize()

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	// 分页查询
	result := r.db.WithContext(ctx).
		Scopes(
			PaginationScope(opts.Limit, opts.Offset),
			OrderScope(opts.OrderBy, opts.Order),
		).
		Find(&tenants)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("find all tenants: %w", result.Error)
	}

	return tenants, total, nil
}

// Update 更新租户
func (r *tenantRepositoryImpl) Update(ctx context.Context, tenant *models.Tenant) error {
	result := r.db.WithContext(ctx).Save(tenant)
	if result.Error != nil {
		return fmt.Errorf("update tenant: %w", result.Error)
	}
	return nil
}

// Delete 删除租户（物理删除）
func (r *tenantRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Tenant{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete tenant: %w", result.Error)
	}
	return nil
}

// UpdateStatus 更新租户状态
func (r *tenantRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TenantStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.Tenant{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("update tenant status: %w", result.Error)
	}
	return nil
}
