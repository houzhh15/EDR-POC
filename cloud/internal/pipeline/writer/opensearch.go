package writer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// OpenSearchConfig OpenSearch写入配置
type OpenSearchConfig struct {
	// Addresses OpenSearch节点地址列表
	Addresses []string `yaml:"addresses"`
	// Username 用户名
	Username string `yaml:"username"`
	// Password 密码
	Password string `yaml:"password"`
	// Index 索引名称模板
	Index string `yaml:"index"`
	// IndexRotation 索引轮转策略: daily, weekly, monthly
	IndexRotation string `yaml:"index_rotation"`
	// BatchSize 批量写入大小
	BatchSize int `yaml:"batch_size"`
	// FlushInterval 刷新间隔
	FlushInterval time.Duration `yaml:"flush_interval"`
	// TLS配置
	TLSEnabled  bool   `yaml:"tls_enabled"`
	TLSCertPath string `yaml:"tls_cert_path"`
	TLSKeyPath  string `yaml:"tls_key_path"`
	TLSCAPath   string `yaml:"tls_ca_path"`
	TLSInsecure bool   `yaml:"tls_insecure"`
	// 重试配置
	MaxRetries   int           `yaml:"max_retries"`
	RetryBackoff time.Duration `yaml:"retry_backoff"`
	// 超时配置
	Timeout time.Duration `yaml:"timeout"`
}

// OpenSearchWriter OpenSearch输出实现
type OpenSearchWriter struct {
	config    *OpenSearchConfig
	client    *http.Client
	transport *http.Transport
	buffer    [][]byte
	mu        sync.Mutex
	closed    bool
	ticker    *time.Ticker
	stopCh    chan struct{}
	indexFunc func() string
}

// NewOpenSearchWriter 创建OpenSearch写入器
func NewOpenSearchWriter(cfg *OpenSearchConfig) (*OpenSearchWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("opensearch config is nil")
	}
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("opensearch addresses is empty")
	}
	if cfg.Index == "" {
		return nil, fmt.Errorf("opensearch index is empty")
	}

	// 设置默认值
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	// 配置TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.TLSInsecure,
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	// 创建索引名称生成函数
	indexFunc := createIndexFunc(cfg.Index, cfg.IndexRotation)

	writer := &OpenSearchWriter{
		config:    cfg,
		client:    client,
		transport: transport,
		buffer:    make([][]byte, 0, cfg.BatchSize),
		stopCh:    make(chan struct{}),
		indexFunc: indexFunc,
	}

	// 启动定时刷新
	if cfg.FlushInterval > 0 {
		writer.ticker = time.NewTicker(cfg.FlushInterval)
		go writer.flushLoop()
	}

	return writer, nil
}

// createIndexFunc 创建索引名称生成函数
func createIndexFunc(baseIndex, rotation string) func() string {
	switch rotation {
	case "daily":
		return func() string {
			return fmt.Sprintf("%s-%s", baseIndex, time.Now().UTC().Format("2006.01.02"))
		}
	case "weekly":
		return func() string {
			year, week := time.Now().UTC().ISOWeek()
			return fmt.Sprintf("%s-%d.%02d", baseIndex, year, week)
		}
	case "monthly":
		return func() string {
			return fmt.Sprintf("%s-%s", baseIndex, time.Now().UTC().Format("2006.01"))
		}
	default:
		return func() string {
			return baseIndex
		}
	}
}

// flushLoop 定时刷新循环
func (w *OpenSearchWriter) flushLoop() {
	for {
		select {
		case <-w.ticker.C:
			if err := w.Flush(context.Background()); err != nil {
				// 记录错误但不退出
				fmt.Printf("opensearch flush error: %v\n", err)
			}
		case <-w.stopCh:
			return
		}
	}
}

// Write 写入单个事件
func (w *OpenSearchWriter) Write(ctx context.Context, event []byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("opensearch writer is closed")
	}

	w.buffer = append(w.buffer, event)

	if len(w.buffer) >= w.config.BatchSize {
		events := make([][]byte, len(w.buffer))
		copy(events, w.buffer)
		w.buffer = w.buffer[:0]
		w.mu.Unlock()
		return w.writeBulk(ctx, events)
	}

	w.mu.Unlock()
	return nil
}

// WriteBatch 批量写入事件
func (w *OpenSearchWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("opensearch writer is closed")
	}
	w.mu.Unlock()

	if len(events) == 0 {
		return nil
	}

	return w.writeBulk(ctx, events)
}

// writeBulk 执行批量写入
func (w *OpenSearchWriter) writeBulk(ctx context.Context, events [][]byte) error {
	if len(events) == 0 {
		return nil
	}

	index := w.indexFunc()

	// 构建 Bulk API 请求体
	var buf bytes.Buffer
	for _, event := range events {
		// 索引操作元数据
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
			},
		}
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("marshal bulk meta: %w", err)
		}
		buf.Write(metaBytes)
		buf.WriteByte('\n')
		buf.Write(event)
		buf.WriteByte('\n')
	}

	// 执行请求
	var lastErr error
	for i := 0; i <= w.config.MaxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryBackoff * time.Duration(i)):
			}
		}

		err := w.doBulkRequest(ctx, buf.Bytes())
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("opensearch bulk write failed after %d retries: %w", w.config.MaxRetries, lastErr)
}

// doBulkRequest 执行单次Bulk请求
func (w *OpenSearchWriter) doBulkRequest(ctx context.Context, body []byte) error {
	// 选择一个地址（简单轮询）
	addr := w.config.Addresses[0]
	url := fmt.Sprintf("%s/_bulk", addr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	// 设置认证
	if w.config.Username != "" && w.config.Password != "" {
		req.SetBasicAuth(w.config.Username, w.config.Password)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("opensearch error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// 解析响应检查是否有错误
	var bulkResp BulkResponse
	if err := json.Unmarshal(respBody, &bulkResp); err != nil {
		return fmt.Errorf("parse bulk response: %w", err)
	}

	if bulkResp.Errors {
		// 收集所有错误
		var errs []string
		for _, item := range bulkResp.Items {
			if item.Index.Error != nil {
				errs = append(errs, fmt.Sprintf("type=%s reason=%s", item.Index.Error.Type, item.Index.Error.Reason))
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("bulk response has errors: %v", errs)
		}
	}

	return nil
}

// BulkResponse OpenSearch Bulk API响应
type BulkResponse struct {
	Took   int        `json:"took"`
	Errors bool       `json:"errors"`
	Items  []BulkItem `json:"items"`
}

// BulkItem Bulk操作结果项
type BulkItem struct {
	Index BulkItemResult `json:"index"`
}

// BulkItemResult Bulk操作单项结果
type BulkItemResult struct {
	Index   string     `json:"_index"`
	ID      string     `json:"_id"`
	Version int        `json:"_version"`
	Result  string     `json:"result"`
	Status  int        `json:"status"`
	Error   *BulkError `json:"error,omitempty"`
}

// BulkError Bulk操作错误
type BulkError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// Flush 刷新缓冲区
func (w *OpenSearchWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}

	if len(w.buffer) == 0 {
		w.mu.Unlock()
		return nil
	}

	events := make([][]byte, len(w.buffer))
	copy(events, w.buffer)
	w.buffer = w.buffer[:0]
	w.mu.Unlock()

	return w.writeBulk(ctx, events)
}

// Close 关闭写入器
func (w *OpenSearchWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true

	if w.ticker != nil {
		w.ticker.Stop()
		close(w.stopCh)
	}
	w.mu.Unlock()

	// 刷新剩余数据
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := w.Flush(ctx); err != nil {
		return fmt.Errorf("flush on close: %w", err)
	}

	w.transport.CloseIdleConnections()
	return nil
}

// BufferSize 返回当前缓冲区大小
func (w *OpenSearchWriter) BufferSize() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.buffer)
}

// CurrentIndex 返回当前索引名称
func (w *OpenSearchWriter) CurrentIndex() string {
	return w.indexFunc()
}
