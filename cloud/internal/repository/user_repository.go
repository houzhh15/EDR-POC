package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.User, error)
	FindByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*models.User, error)
	FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*models.User, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions) ([]*models.User, int64, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, tenantID, id uuid.UUID, ip string) error
}
