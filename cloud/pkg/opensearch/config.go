package opensearch

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// Config OpenSearch 客户端配置
type Config struct {
	// Addresses 集群地址列表，如 ["https://localhost:9200"]
	Addresses []string `json:"addresses" yaml:"addresses"`

	// Username 用户名
	Username string `json:"username" yaml:"username"`

	// Password 密码
	Password string `json:"password" yaml:"password"`

	// APIKey API 密钥认证 (优先于 Username/Password)
	APIKey string `json:"api_key" yaml:"api_key"`

	// TLS TLS 配置
	TLS *TLSConfig `json:"tls" yaml:"tls"`

	// MaxIdleConns 最大空闲连接数，默认 100
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// MaxConnsPerHost 每个主机最大连接数，默认 10
	MaxConnsPerHost int `json:"max_conns_per_host" yaml:"max_conns_per_host"`

	// IdleConnTimeout 空闲连接超时，默认 90s
	IdleConnTimeout time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`

	// RequestTimeout 请求超时，默认 30s
	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// RetryOnStatus 需要重试的 HTTP 状态码，默认 [502, 503, 504]
	RetryOnStatus []int `json:"retry_on_status" yaml:"retry_on_status"`

	// MaxRetries 最大重试次数，默认 3
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryBackoff 重试退避基数，默认 100ms
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff"`

	// EnableMetrics 是否启用 Prometheus 指标，默认 true
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`

	// CompressRequestBody 是否压缩请求体，默认 false
	CompressRequestBody bool `json:"compress_request_body" yaml:"compress_request_body"`

	// DiscoverNodes 是否自动发现集群节点，默认 false
	DiscoverNodes bool `json:"discover_nodes" yaml:"discover_nodes"`

	// DisableRetry 禁用重试，默认 false
	DisableRetry bool `json:"disable_retry" yaml:"disable_retry"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	// Enabled 是否启用 TLS
	Enabled bool `json:"enabled" yaml:"enabled"`

	// InsecureSkipVerify 跳过证书验证（仅用于开发环境）
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`

	// CertPath 客户端证书路径
	CertPath string `json:"cert_path" yaml:"cert_path"`

	// KeyPath 客户端密钥路径
	KeyPath string `json:"key_path" yaml:"key_path"`

	// CAPath CA 证书路径
	CAPath string `json:"ca_path" yaml:"ca_path"`

	// ServerName 服务器名称（用于 SNI）
	ServerName string `json:"server_name" yaml:"server_name"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:    100,
		MaxConnsPerHost: 10,
		IdleConnTimeout: 90 * time.Second,
		RequestTimeout:  30 * time.Second,
		RetryOnStatus:   []int{502, 503, 504},
		MaxRetries:      3,
		RetryBackoff:    100 * time.Millisecond,
		EnableMetrics:   true,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.Addresses) == 0 {
		return fmt.Errorf("%w: addresses is required", ErrInvalidConfig)
	}

	for i, addr := range c.Addresses {
		if addr == "" {
			return fmt.Errorf("%w: address[%d] is empty", ErrInvalidConfig, i)
		}
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: max_idle_conns must be non-negative", ErrInvalidConfig)
	}

	if c.MaxConnsPerHost < 0 {
		return fmt.Errorf("%w: max_conns_per_host must be non-negative", ErrInvalidConfig)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("%w: max_retries must be non-negative", ErrInvalidConfig)
	}

	if c.TLS != nil && c.TLS.Enabled {
		if err := c.TLS.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ApplyDefaults 应用默认值
func (c *Config) ApplyDefaults() {
	defaults := DefaultConfig()

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = defaults.MaxIdleConns
	}
	if c.MaxConnsPerHost == 0 {
		c.MaxConnsPerHost = defaults.MaxConnsPerHost
	}
	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = defaults.IdleConnTimeout
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaults.RequestTimeout
	}
	if len(c.RetryOnStatus) == 0 {
		c.RetryOnStatus = defaults.RetryOnStatus
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = defaults.MaxRetries
	}
	if c.RetryBackoff == 0 {
		c.RetryBackoff = defaults.RetryBackoff
	}
}

// Validate 验证 TLS 配置
func (t *TLSConfig) Validate() error {
	if !t.Enabled {
		return nil
	}

	// 如果提供了客户端证书，必须同时提供密钥
	if (t.CertPath != "" && t.KeyPath == "") || (t.CertPath == "" && t.KeyPath != "") {
		return fmt.Errorf("%w: both cert_path and key_path must be provided for mTLS", ErrInvalidConfig)
	}

	// 检查文件是否存在
	if t.CertPath != "" {
		if _, err := os.Stat(t.CertPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: cert file not found: %s", ErrInvalidConfig, t.CertPath)
		}
	}
	if t.KeyPath != "" {
		if _, err := os.Stat(t.KeyPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: key file not found: %s", ErrInvalidConfig, t.KeyPath)
		}
	}
	if t.CAPath != "" {
		if _, err := os.Stat(t.CAPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: CA file not found: %s", ErrInvalidConfig, t.CAPath)
		}
	}

	return nil
}

// BuildTLSConfig 构建 tls.Config
func (t *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if !t.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: t.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	if t.ServerName != "" {
		tlsConfig.ServerName = t.ServerName
	}

	// 加载客户端证书 (mTLS)
	if t.CertPath != "" && t.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(t.CertPath, t.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载 CA 证书
	if t.CAPath != "" {
		caCert, err := os.ReadFile(t.CAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// BulkIndexerConfig 批量索引器配置
type BulkIndexerConfig struct {
	// NumWorkers 并发 worker 数量，默认 runtime.NumCPU()
	NumWorkers int `json:"num_workers" yaml:"num_workers"`

	// BatchSize 每批文档数量阈值，默认 5000
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// FlushInterval 定时刷新间隔，默认 5s
	FlushInterval time.Duration `json:"flush_interval" yaml:"flush_interval"`

	// FlushBytes 字节数阈值，默认 5MB
	FlushBytes int `json:"flush_bytes" yaml:"flush_bytes"`

	// MaxRetries 最大重试次数，默认 3
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// OnError 错误回调
	OnError func(error) `json:"-" yaml:"-"`

	// OnSuccess 成功回调
	OnSuccess func(BulkStats) `json:"-" yaml:"-"`

	// Pipeline Ingest pipeline 名称
	Pipeline string `json:"pipeline" yaml:"pipeline"`

	// Refresh 写入后是否刷新，可选值: "true", "false", "wait_for"
	Refresh string `json:"refresh" yaml:"refresh"`
}

// DefaultBulkIndexerConfig 返回默认批量索引器配置
func DefaultBulkIndexerConfig() *BulkIndexerConfig {
	return &BulkIndexerConfig{
		NumWorkers:    0, // 0 表示使用 runtime.NumCPU()
		BatchSize:     5000,
		FlushInterval: 5 * time.Second,
		FlushBytes:    5 * 1024 * 1024, // 5MB
		MaxRetries:    3,
	}
}

// BulkStats 批量操作统计
type BulkStats struct {
	// NumAdded 添加的文档数
	NumAdded int64 `json:"num_added"`

	// NumFlushed 已刷新的文档数
	NumFlushed int64 `json:"num_flushed"`

	// NumFailed 失败的文档数
	NumFailed int64 `json:"num_failed"`

	// NumRequests 发送的请求数
	NumRequests int64 `json:"num_requests"`

	// BytesFlushed 已刷新的字节数
	BytesFlushed int64 `json:"bytes_flushed"`
}
