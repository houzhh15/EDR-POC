package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// PolicyFilter 策略过滤条件
type PolicyFilter struct {
	Type    models.PolicyType `json:"type"`
	Enabled *bool             `json:"enabled"`
	Search  string            `json:"search"`
}

// PolicyRepository 策略仓储接口
type PolicyRepository interface {
	Create(ctx context.Context, policy *models.Policy) error
	Update(ctx context.Context, policy *models.Policy) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Policy, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions, filter PolicyFilter) ([]*models.Policy, int64, error)
	FindEnabled(ctx context.Context, tenantID uuid.UUID, policyType models.PolicyType) ([]*models.Policy, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	SetEnabled(ctx context.Context, tenantID, id uuid.UUID, enabled bool) error
}
