package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// HealthChecker 数据库健康检查器
type HealthChecker struct {
	db       *gorm.DB
	interval time.Duration
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *gorm.DB, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		db:       db,
		interval: interval,
	}
}

// Check 执行单次健康检查
func (h *HealthChecker) Check(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Stats 获取连接池统计信息
func (h *HealthChecker) Stats() (*PoolStats, error) {
	sqlDB, err := h.db.DB()
	if err != nil {
		return nil, err
	}
	stats := sqlDB.Stats()
	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// Interval 返回健康检查间隔
func (h *HealthChecker) Interval() time.Duration {
	return h.interval
}

// PoolStats 连接池统计
type PoolStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}
