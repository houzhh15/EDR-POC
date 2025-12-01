package config

import (
"fmt"
"os"
"time"

"gopkg.in/yaml.v3"
)

// GRPCServerConfig gRPC 服务器完整配置
type GRPCServerConfig struct {
	GRPC      GRPCConfig      `yaml:"grpc"`
	TLS       TLSConfig       `yaml:"tls"`
	Limits    LimitsConfig    `yaml:"limits"`
	Keepalive KeepaliveConfig `yaml:"keepalive"`
	Kafka     KafkaConfig     `yaml:"kafka"`
	Redis     RedisConfig     `yaml:"redis"`
	Heartbeat HeartbeatConfig `yaml:"heartbeat"`
	JWT       JWTConfig       `yaml:"jwt"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// GRPCConfig gRPC 配置
type GRPCConfig struct {
	ListenAddr  string `yaml:"listen_addr"`
	MetricsAddr string `yaml:"metrics_addr"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// LimitsConfig 限制配置
type LimitsConfig struct {
	MaxRecvMsgSize       int    `yaml:"max_recv_msg_size"`
	MaxSendMsgSize       int    `yaml:"max_send_msg_size"`
	MaxConcurrentStreams uint32 `yaml:"max_concurrent_streams"`
}

// KeepaliveConfig Keepalive 配置
type KeepaliveConfig struct {
	Time              int `yaml:"time"`
	Timeout           int `yaml:"timeout"`
	MaxConnectionIdle int `yaml:"max_connection_idle"`
	MaxConnectionAge  int `yaml:"max_connection_age"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers      []string `yaml:"brokers"`
	Topic        string   `yaml:"topic"`
	BatchSize    int      `yaml:"batch_size"`
	BatchTimeout int      `yaml:"batch_timeout"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Interval int `yaml:"interval"`
	Timeout  int `yaml:"timeout"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret    string `yaml:"secret"`
	Issuer    string `yaml:"issuer"`
	ExpiresIn int    `yaml:"expires_in"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// LoadGRPCConfig 加载配置文件
func LoadGRPCConfig(path string) (*GRPCServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &GRPCServerConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	config.setDefaults()

	return config, nil
}

// setDefaults 设置默认值
func (c *GRPCServerConfig) setDefaults() {
	if c.GRPC.ListenAddr == "" {
		c.GRPC.ListenAddr = ":9090"
	}
	if c.GRPC.MetricsAddr == "" {
		c.GRPC.MetricsAddr = ":9091"
	}
	if c.Limits.MaxRecvMsgSize == 0 {
		c.Limits.MaxRecvMsgSize = 4 * 1024 * 1024 // 4MB
	}
	if c.Limits.MaxSendMsgSize == 0 {
		c.Limits.MaxSendMsgSize = 4 * 1024 * 1024 // 4MB
	}
	if c.Limits.MaxConcurrentStreams == 0 {
		c.Limits.MaxConcurrentStreams = 1000
	}
	if c.Keepalive.Time == 0 {
		c.Keepalive.Time = 300 // 5 分钟
	}
	if c.Keepalive.Timeout == 0 {
		c.Keepalive.Timeout = 20
	}
	if c.Keepalive.MaxConnectionIdle == 0 {
		c.Keepalive.MaxConnectionIdle = 900 // 15 分钟
	}
	if c.Keepalive.MaxConnectionAge == 0 {
		c.Keepalive.MaxConnectionAge = 1800 // 30 分钟
	}
	if c.Heartbeat.Interval == 0 {
		c.Heartbeat.Interval = 30
	}
	if c.Heartbeat.Timeout == 0 {
		c.Heartbeat.Timeout = 90
	}
	if c.Kafka.BatchSize == 0 {
		c.Kafka.BatchSize = 100
	}
	if c.Kafka.BatchTimeout == 0 {
		c.Kafka.BatchTimeout = 5
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 100
	}
	if c.JWT.ExpiresIn == 0 {
		c.JWT.ExpiresIn = 86400 // 24 小时
	}
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "edr-platform"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

// GetKeepaliveTime 获取 keepalive time Duration
func (c *GRPCServerConfig) GetKeepaliveTime() time.Duration {
	return time.Duration(c.Keepalive.Time) * time.Second
}

// GetKeepaliveTimeout 获取 keepalive timeout Duration
func (c *GRPCServerConfig) GetKeepaliveTimeout() time.Duration {
	return time.Duration(c.Keepalive.Timeout) * time.Second
}

// GetMaxConnectionIdle 获取最大空闲时间 Duration
func (c *GRPCServerConfig) GetMaxConnectionIdle() time.Duration {
	return time.Duration(c.Keepalive.MaxConnectionIdle) * time.Second
}

// GetMaxConnectionAge 获取最大连接年龄 Duration
func (c *GRPCServerConfig) GetMaxConnectionAge() time.Duration {
	return time.Duration(c.Keepalive.MaxConnectionAge) * time.Second
}

// GetHeartbeatTTL 获取心跳 TTL Duration
func (c *GRPCServerConfig) GetHeartbeatTTL() time.Duration {
	return time.Duration(c.Heartbeat.Timeout) * time.Second
}

// GetKafkaBatchTimeout 获取 Kafka 批量超时 Duration
func (c *GRPCServerConfig) GetKafkaBatchTimeout() time.Duration {
	return time.Duration(c.Kafka.BatchTimeout) * time.Second
}

// GetJWTExpiresIn 获取 JWT 过期时间 Duration
func (c *GRPCServerConfig) GetJWTExpiresIn() time.Duration {
	return time.Duration(c.JWT.ExpiresIn) * time.Second
}
