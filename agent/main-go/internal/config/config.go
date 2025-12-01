// Package config 提供 EDR Agent 的配置管理功能
package config

import "errors"

// Config 定义 Agent 的完整配置结构
type Config struct {
	Agent     AgentConfig     `mapstructure:"agent"`
	Cloud     CloudConfig     `mapstructure:"cloud"`
	Collector CollectorConfig `mapstructure:"collector"`
	Log       LogConfig       `mapstructure:"log"`
}

// AgentConfig Agent 基础配置
type AgentConfig struct {
	ID   string `mapstructure:"id"`   // Agent 唯一标识
	Name string `mapstructure:"name"` // Agent 名称
}

// CloudConfig 云端连接配置
type CloudConfig struct {
	Endpoint string    `mapstructure:"endpoint"` // 云端地址
	TLS      TLSConfig `mapstructure:"tls"`      // TLS 配置
}

// TLSConfig TLS 证书配置
type TLSConfig struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用 TLS
	CACert  string `mapstructure:"ca_cert"` // CA 证书路径
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	EnabledTypes []string `mapstructure:"enabled_types"` // 启用的采集类型
	BufferSize   int      `mapstructure:"buffer_size"`   // 事件缓冲区大小
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: debug, info, warn, error
	Output     string `mapstructure:"output"`      // 输出方式: console, file, both
	FilePath   string `mapstructure:"file_path"`   // 日志文件路径
	MaxSizeMB  int    `mapstructure:"max_size_mb"` // 单文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 最大保留文件数
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 检查云端地址
	if c.Cloud.Endpoint == "" {
		return errors.New("cloud.endpoint is required")
	}

	// 检查缓冲区大小
	if c.Collector.BufferSize <= 0 {
		return errors.New("collector.buffer_size must be greater than 0")
	}
	if c.Collector.BufferSize > 100000 {
		return errors.New("collector.buffer_size must be less than or equal to 100000")
	}

	// 检查日志级别
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if c.Log.Level != "" && !validLevels[c.Log.Level] {
		return errors.New("log.level must be one of: debug, info, warn, error")
	}

	// 检查日志输出方式
	validOutputs := map[string]bool{
		"console": true,
		"file":    true,
		"both":    true,
	}
	if c.Log.Output != "" && !validOutputs[c.Log.Output] {
		return errors.New("log.output must be one of: console, file, both")
	}

	return nil
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Agent: AgentConfig{
			ID:   "",
			Name: "edr-agent",
		},
		Cloud: CloudConfig{
			Endpoint: "localhost:8080",
			TLS: TLSConfig{
				Enabled: false,
				CACert:  "",
			},
		},
		Collector: CollectorConfig{
			EnabledTypes: []string{"process", "file", "network"},
			BufferSize:   10000,
		},
		Log: LogConfig{
			Level:      "info",
			Output:     "both",
			FilePath:   "/var/log/edr/agent.log",
			MaxSizeMB:  100,
			MaxBackups: 5,
		},
	}
}
