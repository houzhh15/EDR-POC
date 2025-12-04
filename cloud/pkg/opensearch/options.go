package opensearch

import (
	"context"
	"time"
)

// ClientOption 客户端选项函数类型
type ClientOption func(*clientOptions)

// clientOptions 客户端内部选项
type clientOptions struct {
	config *Config
}

// WithConfig 设置配置
func WithConfig(cfg *Config) ClientOption {
	return func(o *clientOptions) {
		o.config = cfg
	}
}

// WithAddresses 设置集群地址
func WithAddresses(addresses ...string) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.Addresses = addresses
	}
}

// WithBasicAuth 设置基本认证
func WithBasicAuth(username, password string) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.Username = username
		o.config.Password = password
	}
}

// WithAPIKey 设置 API Key 认证
func WithAPIKey(apiKey string) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.APIKey = apiKey
	}
}

// WithTLS 设置 TLS 配置
func WithTLS(tlsConfig *TLSConfig) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.TLS = tlsConfig
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.MaxRetries = n
	}
}

// WithRequestTimeout 设置请求超时
func WithRequestTimeout(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.RequestTimeout = d
	}
}

// WithRetryBackoff 设置重试退避时间
func WithRetryBackoff(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.RetryBackoff = d
	}
}

// WithConnectionPool 设置连接池参数
func WithConnectionPool(maxIdle, maxPerHost int, idleTimeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.MaxIdleConns = maxIdle
		o.config.MaxConnsPerHost = maxPerHost
		o.config.IdleConnTimeout = idleTimeout
	}
}

// WithMetrics 设置是否启用指标
func WithMetrics(enabled bool) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.EnableMetrics = enabled
	}
}

// WithCompression 设置是否压缩请求体
func WithCompression(enabled bool) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.CompressRequestBody = enabled
	}
}

// WithNodeDiscovery 设置是否自动发现节点
func WithNodeDiscovery(enabled bool) ClientOption {
	return func(o *clientOptions) {
		if o.config == nil {
			o.config = DefaultConfig()
		}
		o.config.DiscoverNodes = enabled
	}
}

// RequestOption 请求选项函数类型
type RequestOption func(*requestOptions)

// requestOptions 请求内部选项
type requestOptions struct {
	ctx            context.Context
	timeout        time.Duration
	retries        int
	headers        map[string]string
	refresh        string
	routing        string
	pipeline       string
	waitForRefresh bool
}

// defaultRequestOptions 返回默认请求选项
func defaultRequestOptions() *requestOptions {
	return &requestOptions{
		ctx:     context.Background(),
		headers: make(map[string]string),
	}
}

// applyRequestOptions 应用请求选项
func applyRequestOptions(opts ...RequestOption) *requestOptions {
	o := defaultRequestOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithContext 设置请求上下文
func WithContext(ctx context.Context) RequestOption {
	return func(o *requestOptions) {
		o.ctx = ctx
	}
}

// WithTimeout 设置请求超时
func WithTimeout(d time.Duration) RequestOption {
	return func(o *requestOptions) {
		o.timeout = d
	}
}

// WithRetries 设置请求重试次数
func WithRetries(n int) RequestOption {
	return func(o *requestOptions) {
		o.retries = n
	}
}

// WithHeader 添加请求头
func WithHeader(key, value string) RequestOption {
	return func(o *requestOptions) {
		o.headers[key] = value
	}
}

// WithHeaders 设置多个请求头
func WithHeaders(headers map[string]string) RequestOption {
	return func(o *requestOptions) {
		for k, v := range headers {
			o.headers[k] = v
		}
	}
}

// WithRefresh 设置刷新策略
func WithRefresh(refresh string) RequestOption {
	return func(o *requestOptions) {
		o.refresh = refresh
	}
}

// WithWaitForRefresh 设置等待刷新
func WithWaitForRefresh() RequestOption {
	return func(o *requestOptions) {
		o.waitForRefresh = true
		o.refresh = "wait_for"
	}
}

// WithRouting 设置路由
func WithRouting(routing string) RequestOption {
	return func(o *requestOptions) {
		o.routing = routing
	}
}

// WithPipeline 设置 ingest pipeline
func WithPipeline(pipeline string) RequestOption {
	return func(o *requestOptions) {
		o.pipeline = pipeline
	}
}

// BulkIndexerOption 批量索引器选项函数类型
type BulkIndexerOption func(*BulkIndexerConfig)

// WithNumWorkers 设置 worker 数量
func WithNumWorkers(n int) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.NumWorkers = n
	}
}

// WithBatchSize 设置批量大小
func WithBatchSize(n int) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.BatchSize = n
	}
}

// WithFlushInterval 设置刷新间隔
func WithFlushInterval(d time.Duration) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.FlushInterval = d
	}
}

// WithFlushBytes 设置字节阈值
func WithFlushBytes(n int) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.FlushBytes = n
	}
}

// WithBulkMaxRetries 设置批量操作最大重试次数
func WithBulkMaxRetries(n int) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.MaxRetries = n
	}
}

// WithOnError 设置错误回调
func WithOnError(fn func(error)) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.OnError = fn
	}
}

// WithOnSuccess 设置成功回调
func WithOnSuccess(fn func(BulkStats)) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.OnSuccess = fn
	}
}

// WithBulkPipeline 设置 ingest pipeline
func WithBulkPipeline(pipeline string) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.Pipeline = pipeline
	}
}

// WithBulkRefresh 设置刷新策略
func WithBulkRefresh(refresh string) BulkIndexerOption {
	return func(c *BulkIndexerConfig) {
		c.Refresh = refresh
	}
}
