package collector

// CollectorConfig 进程事件采集器配置
type CollectorConfig struct {
	// Enabled 是否启用进程事件采集
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// PollIntervalMs 轮询间隔(毫秒)
	PollIntervalMs int `mapstructure:"poll_interval_ms" json:"poll_interval_ms"`

	// BatchSize 批量获取事件数量
	BatchSize int `mapstructure:"batch_size" json:"batch_size"`

	// ChannelSize 事件channel容量
	ChannelSize int `mapstructure:"channel_size" json:"channel_size"`
}

// DefaultCollectorConfig 返回默认配置
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		Enabled:        true,
		PollIntervalMs: 10,
		BatchSize:      100,
		ChannelSize:    1000,
	}
}

// Validate 验证配置有效性
func (c *CollectorConfig) Validate() error {
	if c.PollIntervalMs <= 0 {
		c.PollIntervalMs = 10
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.ChannelSize <= 0 {
		c.ChannelSize = 1000
	}
	return nil
}
