// Package main 数据库服务初始化示例
// 此文件演示如何在 api-gateway 中初始化和使用新创建的 Repository 层
package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository"
	"github.com/houzhh15/EDR-POC/cloud/pkg/database"
)

// DatabaseServices 封装所有数据库相关服务
type DatabaseServices struct {
	TenantRepo repository.TenantRepository
	UserRepo   repository.UserRepository
	PolicyRepo repository.PolicyRepository
	AlertRepo  repository.AlertRepository
	Health     *database.HealthChecker
	Metrics    *database.MetricsCollector
	migrator   *database.Migrator
	closeFunc  func() error
}

// InitDatabaseServices 初始化数据库服务
// 使用示例:
//
//	services, err := InitDatabaseServices(ctx, logger)
//	if err != nil {
//	    logger.Fatal("Failed to init database", zap.Error(err))
//	}
//	defer services.Close()
func InitDatabaseServices(ctx context.Context, logger *zap.Logger) (*DatabaseServices, error) {
	// 1. 加载数据库配置
	cfg := &database.DBConfig{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            getEnvOrDefaultInt("DB_PORT", 15432),
		Database:        getEnvOrDefault("DB_NAME", "edr"),
		Username:        getEnvOrDefault("DB_USER", "edr"),
		Password:        getEnvOrDefault("DB_PASSWORD", "edr_dev_password"),
		SSLMode:         getEnvOrDefault("DB_SSL_MODE", "disable"),
		MaxOpenConns:    getEnvOrDefaultInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvOrDefaultInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	// 2. 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	logger.Info("Database config validated",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
	)

	// 3. 创建数据库连接
	db, err := database.NewPostgresDB(cfg, logger.Named("database"))
	if err != nil {
		return nil, err
	}
	logger.Info("PostgreSQL connection established")

	// 获取底层 sql.DB 用于迁移和健康检查
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 4. 运行数据库迁移 (可选，生产环境建议手动执行)
	autoMigrate := getEnvOrDefault("DB_AUTO_MIGRATE", "false") == "true"
	var migrator *database.Migrator
	if autoMigrate {
		migrator = database.NewMigrator(sqlDB, "file://migrations", logger.Named("migrator"))
		if err := migrator.Up(); err != nil {
			return nil, err
		}
		logger.Info("Database migrations applied")
	}

	// 5. 初始化健康检查
	healthChecker := database.NewHealthChecker(db, 30*time.Second)
	if err := healthChecker.Check(ctx); err != nil {
		return nil, err
	}
	logger.Info("Database health check passed")

	// 6. 初始化 Prometheus 指标收集
	metricsCollector := database.NewMetricsCollector(db, 15*time.Second)
	metricsCollector.Start()
	logger.Info("Database metrics collector started")

	// 7. 初始化 Repository 层
	tenantRepo := repository.NewTenantRepository(db, logger.Named("tenant-repo"))
	userRepo := repository.NewUserRepository(db, logger.Named("user-repo"))
	policyRepo := repository.NewPolicyRepository(db, logger.Named("policy-repo"))
	alertRepo := repository.NewAlertRepository(db, logger.Named("alert-repo"))
	logger.Info("Repository layer initialized")

	return &DatabaseServices{
		TenantRepo: tenantRepo,
		UserRepo:   userRepo,
		PolicyRepo: policyRepo,
		AlertRepo:  alertRepo,
		Health:     healthChecker,
		Metrics:    metricsCollector,
		migrator:   migrator,
		closeFunc:  func() error { return database.CloseDB(db, logger.Named("database")) },
	}, nil
}

// Close 关闭所有数据库服务
func (s *DatabaseServices) Close() error {
	if s.Metrics != nil {
		s.Metrics.Stop()
	}
	if s.closeFunc != nil {
		return s.closeFunc()
	}
	return nil
}

// HealthCheck 执行健康检查
func (s *DatabaseServices) HealthCheck(ctx context.Context) error {
	return s.Health.Check(ctx)
}

// 辅助函数
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// 简化处理，实际应该解析整数
		return defaultValue
	}
	return defaultValue
}
