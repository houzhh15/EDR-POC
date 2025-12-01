package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Agent.Name != "edr-agent" {
		t.Errorf("expected agent.name to be 'edr-agent', got %s", cfg.Agent.Name)
	}

	if cfg.Collector.BufferSize != 10000 {
		t.Errorf("expected collector.buffer_size to be 10000, got %d", cfg.Collector.BufferSize)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("expected log.level to be 'info', got %s", cfg.Log.Level)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "empty endpoint",
			modify: func(c *Config) {
				c.Cloud.Endpoint = ""
			},
			wantErr: true,
		},
		{
			name: "zero buffer size",
			modify: func(c *Config) {
				c.Collector.BufferSize = 0
			},
			wantErr: true,
		},
		{
			name: "negative buffer size",
			modify: func(c *Config) {
				c.Collector.BufferSize = -1
			},
			wantErr: true,
		},
		{
			name: "buffer size too large",
			modify: func(c *Config) {
				c.Collector.BufferSize = 100001
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Log.Level = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid log output",
			modify: func(c *Config) {
				c.Log.Output = "invalid"
			},
			wantErr: true,
		},
		{
			name: "valid custom config",
			modify: func(c *Config) {
				c.Cloud.Endpoint = "example.com:443"
				c.Collector.BufferSize = 50000
				c.Log.Level = "debug"
				c.Log.Output = "file"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
agent:
  id: "test-agent-001"
  name: "test-agent"
cloud:
  endpoint: "test.example.com:8080"
  tls:
    enabled: true
    ca_cert: "/etc/ssl/ca.pem"
collector:
  enabled_types:
    - process
  buffer_size: 5000
log:
  level: "debug"
  output: "console"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证加载的值
	if cfg.Agent.ID != "test-agent-001" {
		t.Errorf("expected agent.id = 'test-agent-001', got %s", cfg.Agent.ID)
	}

	if cfg.Cloud.Endpoint != "test.example.com:8080" {
		t.Errorf("expected cloud.endpoint = 'test.example.com:8080', got %s", cfg.Cloud.Endpoint)
	}

	if !cfg.Cloud.TLS.Enabled {
		t.Error("expected cloud.tls.enabled = true")
	}

	if cfg.Collector.BufferSize != 5000 {
		t.Errorf("expected collector.buffer_size = 5000, got %d", cfg.Collector.BufferSize)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("expected log.level = 'debug', got %s", cfg.Log.Level)
	}
}

func TestLoader_LoadWithEnvOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("EDR_CLOUD_ENDPOINT", "env.example.com:9090")
	defer os.Unsetenv("EDR_CLOUD_ENDPOINT")

	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
cloud:
  endpoint: "file.example.com:8080"
collector:
  buffer_size: 5000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 环境变量应该覆盖文件配置
	if cfg.Cloud.Endpoint != "env.example.com:9090" {
		t.Errorf("expected cloud.endpoint = 'env.example.com:9090' (from env), got %s", cfg.Cloud.Endpoint)
	}
}

func TestLoadAndValidate(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// 有效配置
	validConfig := `
cloud:
  endpoint: "example.com:8080"
collector:
  buffer_size: 5000
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadAndValidate(configPath)
	if err != nil {
		t.Errorf("LoadAndValidate() error = %v", err)
	}

	if cfg == nil {
		t.Error("expected config to be non-nil")
	}
}

func TestLoadAndValidate_Invalid(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// 无效配置 - 空 endpoint
	invalidConfig := `
cloud:
  endpoint: ""
collector:
  buffer_size: 5000
`

	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadAndValidate(configPath)
	if err == nil {
		t.Error("expected LoadAndValidate() to return error for invalid config")
	}
}
