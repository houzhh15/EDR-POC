package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// TenantRepository 租户仓储接口
type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	Update(ctx context.Context, tenant *models.Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	FindByName(ctx context.Context, name string) (*models.Tenant, error)
	FindAll(ctx context.Context, opts models.ListOptions) ([]*models.Tenant, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.TenantStatus) error
}
