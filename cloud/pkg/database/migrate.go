package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// Migrator 数据库迁移器
type Migrator struct {
	db     *sql.DB
	path   string
	logger *zap.Logger
}

// NewMigrator 创建迁移器
func NewMigrator(db *sql.DB, migrationsPath string, logger *zap.Logger) *Migrator {
	return &Migrator{
		db:     db,
		path:   migrationsPath,
		logger: logger.Named("migrator"),
	}
}

// Up 执行所有待处理的迁移
func (m *Migrator) Up() error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	version, dirty, err := migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	m.logger.Info("Current migration version",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	newVersion, _, _ := migrator.Version()
	m.logger.Info("Migration completed",
		zap.Uint("from_version", version),
		zap.Uint("to_version", newVersion),
	)
	return nil
}

// Down 回滚一个版本
func (m *Migrator) Down() error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Steps(-1); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

// Version 获取当前版本
func (m *Migrator) Version() (uint, bool, error) {
	migrator, err := m.newMigrate()
	if err != nil {
		return 0, false, err
	}
	defer migrator.Close()
	return migrator.Version()
}

// Goto 迁移到指定版本
func (m *Migrator) Goto(version uint) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to version %d failed: %w", version, err)
	}
	return nil
}

// Force 强制设置版本（用于修复 dirty 状态）
func (m *Migrator) Force(version int) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Force(version); err != nil {
		return fmt.Errorf("force version %d failed: %w", version, err)
	}
	m.logger.Warn("Forced migration version", zap.Int("version", version))
	return nil
}

func (m *Migrator) newMigrate() (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}
	return migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.path),
		"postgres",
		driver,
	)
}
