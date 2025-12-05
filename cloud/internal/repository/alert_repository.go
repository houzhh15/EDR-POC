package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

// AlertFilter 告警过滤条件
type AlertFilter struct {
	Status    models.AlertStatus   `json:"status"`
	Severity  models.AlertSeverity `json:"severity"`
	AssetID   *uuid.UUID           `json:"asset_id"`
	PolicyID  *uuid.UUID           `json:"policy_id"`
	StartTime *time.Time           `json:"start_time"`
	EndTime   *time.Time           `json:"end_time"`
	Search    string               `json:"search"`
}

// AlertRepository 告警仓储接口
type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	Update(ctx context.Context, alert *models.Alert) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Alert, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, opts models.ListOptions, filter AlertFilter) ([]*models.Alert, int64, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status models.AlertStatus, userID uuid.UUID) error
	Acknowledge(ctx context.Context, tenantID, id uuid.UUID, userID uuid.UUID) error
	Resolve(ctx context.Context, tenantID, id uuid.UUID, userID uuid.UUID, resolution string) error
	AssignTo(ctx context.Context, tenantID, id uuid.UUID, assigneeID uuid.UUID) error
	CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[models.AlertStatus]int64, error)
	CountBySeverity(ctx context.Context, tenantID uuid.UUID) (map[models.AlertSeverity]int64, error)
}
