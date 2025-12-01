package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Loader 配置加载器
type Loader struct {
	v       *viper.Viper
	config  *Config
	mu      sync.RWMutex
	watches []func(*Config)
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{
		v:       viper.New(),
		config:  Default(),
		watches: make([]func(*Config), 0),
	}
}

// Load 从指定路径加载配置
// 支持多个路径，后面的配置会覆盖前面的
func (l *Loader) Load(paths ...string) (*Config, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 设置配置类型
	l.v.SetConfigType("yaml")

	// 设置环境变量前缀和自动读取
	l.v.SetEnvPrefix("EDR")
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	l.v.AutomaticEnv()

	// 设置默认值
	l.setDefaults()

	// 加载配置文件
	for _, path := range paths {
		l.v.SetConfigFile(path)
		if err := l.v.MergeInConfig(); err != nil {
			// 如果文件不存在，使用默认配置
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
			}
		}
	}

	// 解析到结构体
	cfg := &Config{}
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	l.config = cfg
	return cfg, nil
}

// setDefaults 设置默认值
func (l *Loader) setDefaults() {
	def := Default()

	l.v.SetDefault("agent.name", def.Agent.Name)
	l.v.SetDefault("cloud.endpoint", def.Cloud.Endpoint)
	l.v.SetDefault("cloud.tls.enabled", def.Cloud.TLS.Enabled)
	l.v.SetDefault("collector.enabled_types", def.Collector.EnabledTypes)
	l.v.SetDefault("collector.buffer_size", def.Collector.BufferSize)
	l.v.SetDefault("log.level", def.Log.Level)
	l.v.SetDefault("log.output", def.Log.Output)
	l.v.SetDefault("log.file_path", def.Log.FilePath)
	l.v.SetDefault("log.max_size_mb", def.Log.MaxSizeMB)
	l.v.SetDefault("log.max_backups", def.Log.MaxBackups)
}

// Get 获取当前配置（线程安全）
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// Watch 监听配置变更
// callback 会在配置变更时被调用
func (l *Loader) Watch(callback func(*Config)) error {
	l.mu.Lock()
	l.watches = append(l.watches, callback)
	l.mu.Unlock()

	l.v.OnConfigChange(func(e fsnotify.Event) {
		l.mu.Lock()
		defer l.mu.Unlock()

		// 重新加载配置
		cfg := &Config{}
		if err := l.v.Unmarshal(cfg); err != nil {
			// 配置解析失败，保持原配置
			return
		}

		// 验证配置
		if err := cfg.Validate(); err != nil {
			// 配置无效，保持原配置
			return
		}

		l.config = cfg

		// 通知所有监听者
		for _, watch := range l.watches {
			watch(cfg)
		}
	})

	l.v.WatchConfig()
	return nil
}

// LoadAndValidate 加载并验证配置
func LoadAndValidate(paths ...string) (*Config, error) {
	loader := NewLoader()
	cfg, err := loader.Load(paths...)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}
