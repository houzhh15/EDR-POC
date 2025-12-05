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

// userRepositoryImpl UserRepository 实现
type userRepositoryImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewUserRepository 创建 UserRepository 实例
func NewUserRepository(db *gorm.DB, logger *zap.Logger) UserRepository {
	return &userRepositoryImpl{
		db:     db,
		logger: logger.Named("user_repository"),
	}
}

// Create 创建用户
func (r *userRepositoryImpl) Create(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		r.logger.Error("Failed to create user",
			zap.String("tenant_id", user.TenantID.String()),
			zap.String("username", user.Username),
			zap.Error(result.Error),
		)
		return fmt.Errorf("create user: %w", result.Error)
	}
	r.logger.Info("User created",
		zap.String("id", user.ID.String()),
		zap.String("username", user.Username),
	)
	return nil
}

// FindByID 根据 ID 查询用户
func (r *userRepositoryImpl) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", result.Error)
	}
	return &user, nil
}

// FindByUsername 根据用户名查询
func (r *userRepositoryImpl) FindByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		First(&user, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by username: %w", result.Error)
	}
	return &user, nil
}

// FindByEmail 根据邮箱查询
func (r *userRepositoryImpl) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		First(&user, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email: %w", result.Error)
	}
	return &user, nil
}

// FindAll 分页查询所有用户
func (r *userRepositoryImpl) FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	opts.Normalize()

	query := r.db.WithContext(ctx).Model(&models.User{}).
		Scopes(TenantScope(tenantID), NotDeletedScope())

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// 分页查询
	result := query.Scopes(
		PaginationScope(opts.Limit, opts.Offset),
		OrderScope(opts.OrderBy, opts.Order),
	).Find(&users)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("find all users: %w", result.Error)
	}

	return users, total, nil
}

// Update 更新用户
func (r *userRepositoryImpl) Update(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).
		Scopes(TenantScope(user.TenantID), NotDeletedScope()).
		Save(user)
	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}
	return nil
}

// Delete 软删除用户
func (r *userRepositoryImpl) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Scopes(TenantScope(tenantID)).
		Where("id = ?", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}
	return nil
}

// UpdatePassword 更新密码
func (r *userRepositoryImpl) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", id).
		Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("update password: %w", result.Error)
	}
	return nil
}

// UpdateLastLogin 更新最后登录信息
func (r *userRepositoryImpl) UpdateLastLogin(ctx context.Context, tenantID, id uuid.UUID, ip string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Scopes(TenantScope(tenantID), NotDeletedScope()).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"last_login_ip": ip,
			"login_count":   gorm.Expr("login_count + 1"),
			"updated_at":    now,
		})
	if result.Error != nil {
		return fmt.Errorf("update last login: %w", result.Error)
	}
	return nil
}
