// Package database 提供 PostgreSQL 数据库连接管理、迁移和事务工具
package database

import (
	"fmt"
	"time"
)

// DBConfig 数据库配置结构
type DBConfig struct {
	// 连接参数
	Host     string `yaml:"host" mapstructure:"host" json:"host"`
	Port     int    `yaml:"port" mapstructure:"port" json:"port"`
	Database string `yaml:"database" mapstructure:"database" json:"database"`
	Username string `yaml:"username" mapstructure:"username" json:"username"`
	Password string `yaml:"password" mapstructure:"password" json:"-"`
	SSLMode  string `yaml:"ssl_mode" mapstructure:"ssl_mode" json:"ssl_mode"`

	// 连接池参数
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"`

	// 健康检查参数
	HealthCheckInterval time.Duration `yaml:"health_check_interval" mapstructure:"health_check_interval" json:"health_check_interval"`
}

// DefaultDBConfig 返回带有合理默认值的配置
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		Host:                "localhost",
		Port:                15432,
		Database:            "edr",
		Username:            "edr_user",
		SSLMode:             "disable",
		MaxOpenConns:        25,
		MaxIdleConns:        5,
		ConnMaxLifetime:     5 * time.Minute,
		ConnMaxIdleTime:     5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// DSN 生成 PostgreSQL 连接字符串
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode,
	)
}

// Validate 验证配置参数
func (c *DBConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Port)
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.MaxOpenConns < 1 {
		return fmt.Errorf("max_open_conns must be at least 1")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot exceed max_open_conns")
	}
	return nil
}
