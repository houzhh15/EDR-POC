package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client OpenSearch 客户端接口
type Client interface {
	// Bulk 执行批量操作
	Bulk(ctx context.Context, body io.Reader, opts ...RequestOption) (*BulkResponse, error)

	// Search 执行搜索查询
	Search(ctx context.Context, indices []string, query map[string]interface{}, opts ...RequestOption) (*SearchResponse, error)

	// Index 索引单个文档
	Index(ctx context.Context, index string, docID string, body interface{}, opts ...RequestOption) error

	// CreateIndex 创建索引
	CreateIndex(ctx context.Context, name string, settings map[string]interface{}) error

	// DeleteIndex 删除索引
	DeleteIndex(ctx context.Context, name string) error

	// PutIndexTemplate 创建/更新索引模板
	PutIndexTemplate(ctx context.Context, name string, template *IndexTemplate) error

	// PutISMPolicy 创建/更新 ISM 策略
	PutISMPolicy(ctx context.Context, name string, policy *ISMPolicy) error

	// Health 检查集群健康状态
	Health(ctx context.Context) (*ClusterHealth, error)

	// Close 关闭客户端
	Close() error
}

// opensearchClient Client 接口实现
type opensearchClient struct {
	config    *Config
	transport *http.Transport
	client    *http.Client

	// 连接池状态
	nodeIndex atomic.Int64
	nodes     []string

	// 关闭控制
	closed    atomic.Bool
	closeOnce sync.Once
}

// NewClient 创建 OpenSearch 客户端
func NewClient(opts ...ClientOption) (Client, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cfg := options.config
	if cfg == nil {
		cfg = DefaultConfig()
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 创建 HTTP Transport
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
	}

	// 构建 TLS 配置
	if cfg.TLS != nil && cfg.TLS.Enabled {
		tlsConfig, err := cfg.TLS.BuildTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("build TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	}

	// 创建 HTTP Client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
	}

	client := &opensearchClient{
		config:    cfg,
		transport: transport,
		client:    httpClient,
		nodes:     cfg.Addresses,
	}

	return client, nil
}

// getNode 获取下一个节点地址（轮询）
func (c *opensearchClient) getNode() string {
	if len(c.nodes) == 1 {
		return c.nodes[0]
	}
	idx := c.nodeIndex.Add(1) % int64(len(c.nodes))
	return c.nodes[idx]
}

// doRequest 执行 HTTP 请求
func (c *opensearchClient) doRequest(ctx context.Context, method, path string, body io.Reader, opts *requestOptions) (*http.Response, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	node := c.getNode()
	url := strings.TrimSuffix(node, "/") + "/" + strings.TrimPrefix(path, "/")

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置默认 Content-Type
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置认证
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.config.APIKey)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// 应用请求选项中的头
	if opts != nil {
		for k, v := range opts.headers {
			req.Header.Set(k, v)
		}
	}

	return c.client.Do(req)
}

// doRequestWithRetry 带重试的请求
func (c *opensearchClient) doRequestWithRetry(ctx context.Context, method, path string, body []byte, opts *requestOptions) ([]byte, error) {
	maxRetries := c.config.MaxRetries
	if opts != nil && opts.retries > 0 {
		maxRetries = opts.retries
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := c.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		resp, err := c.doRequest(ctx, method, path, bodyReader, opts)
		if err != nil {
			lastErr = err
			if !IsRetryable(err) {
				return nil, err
			}
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		// 检查是否需要重试的状态码
		if c.shouldRetry(resp.StatusCode) && attempt < maxRetries {
			lastErr = &ResponseError{
				StatusCode: resp.StatusCode,
				Type:       "http_error",
				Reason:     string(respBody),
			}
			continue
		}

		// 处理错误响应
		if resp.StatusCode >= 400 {
			return nil, c.parseErrorResponse(resp.StatusCode, respBody)
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// shouldRetry 检查是否应该重试
func (c *opensearchClient) shouldRetry(statusCode int) bool {
	for _, code := range c.config.RetryOnStatus {
		if code == statusCode {
			return true
		}
	}
	return false
}

// parseErrorResponse 解析错误响应
func (c *opensearchClient) parseErrorResponse(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrIndexNotFound
	case 429:
		return ErrRateLimited
	}

	var errResp struct {
		Error struct {
			Type      string `json:"type"`
			Reason    string `json:"reason"`
			RootCause []struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"root_cause"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Type != "" {
		return &ResponseError{
			StatusCode: statusCode,
			Type:       errResp.Error.Type,
			Reason:     errResp.Error.Reason,
		}
	}

	return &ResponseError{
		StatusCode: statusCode,
		Type:       "unknown",
		Reason:     string(body),
	}
}

// Bulk 执行批量操作
func (c *opensearchClient) Bulk(ctx context.Context, body io.Reader, opts ...RequestOption) (*BulkResponse, error) {
	reqOpts := applyRequestOptions(opts...)

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// 设置 Content-Type 为 ndjson
	reqOpts.headers["Content-Type"] = "application/x-ndjson"

	respBody, err := c.doRequestWithRetry(ctx, http.MethodPost, "/_bulk", bodyBytes, reqOpts)
	if err != nil {
		return nil, err
	}

	var bulkResp BulkResponse
	if err := json.Unmarshal(respBody, &bulkResp); err != nil {
		return nil, fmt.Errorf("parse bulk response: %w", err)
	}

	return &bulkResp, nil
}

// Search 执行搜索查询
func (c *opensearchClient) Search(ctx context.Context, indices []string, query map[string]interface{}, opts ...RequestOption) (*SearchResponse, error) {
	reqOpts := applyRequestOptions(opts...)

	path := "/_search"
	if len(indices) > 0 {
		path = "/" + strings.Join(indices, ",") + "/_search"
	}

	bodyBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	respBody, err := c.doRequestWithRetry(ctx, http.MethodPost, path, bodyBytes, reqOpts)
	if err != nil {
		return nil, err
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	return &searchResp, nil
}

// Index 索引单个文档
func (c *opensearchClient) Index(ctx context.Context, index string, docID string, body interface{}, opts ...RequestOption) error {
	reqOpts := applyRequestOptions(opts...)

	path := "/" + index + "/_doc"
	if docID != "" {
		path = "/" + index + "/_doc/" + docID
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	method := http.MethodPost
	if docID != "" {
		method = http.MethodPut
	}

	_, err = c.doRequestWithRetry(ctx, method, path, bodyBytes, reqOpts)
	return err
}

// CreateIndex 创建索引
func (c *opensearchClient) CreateIndex(ctx context.Context, name string, settings map[string]interface{}) error {
	reqOpts := applyRequestOptions()

	var bodyBytes []byte
	var err error
	if settings != nil {
		bodyBytes, err = json.Marshal(settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
	}

	_, err = c.doRequestWithRetry(ctx, http.MethodPut, "/"+name, bodyBytes, reqOpts)
	return err
}

// DeleteIndex 删除索引
func (c *opensearchClient) DeleteIndex(ctx context.Context, name string) error {
	reqOpts := applyRequestOptions()
	_, err := c.doRequestWithRetry(ctx, http.MethodDelete, "/"+name, nil, reqOpts)
	return err
}

// PutIndexTemplate 创建/更新索引模板
func (c *opensearchClient) PutIndexTemplate(ctx context.Context, name string, template *IndexTemplate) error {
	reqOpts := applyRequestOptions()

	bodyBytes, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}

	_, err = c.doRequestWithRetry(ctx, http.MethodPut, "/_index_template/"+name, bodyBytes, reqOpts)
	return err
}

// PutISMPolicy 创建/更新 ISM 策略
func (c *opensearchClient) PutISMPolicy(ctx context.Context, name string, policy *ISMPolicy) error {
	reqOpts := applyRequestOptions()

	wrapper := map[string]interface{}{
		"policy": policy,
	}

	bodyBytes, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}

	_, err = c.doRequestWithRetry(ctx, http.MethodPut, "/_plugins/_ism/policies/"+name, bodyBytes, reqOpts)
	return err
}

// Health 检查集群健康状态
func (c *opensearchClient) Health(ctx context.Context) (*ClusterHealth, error) {
	reqOpts := applyRequestOptions()

	respBody, err := c.doRequestWithRetry(ctx, http.MethodGet, "/_cluster/health", nil, reqOpts)
	if err != nil {
		return nil, err
	}

	var health ClusterHealth
	if err := json.Unmarshal(respBody, &health); err != nil {
		return nil, fmt.Errorf("parse health response: %w", err)
	}

	return &health, nil
}

// Close 关闭客户端
func (c *opensearchClient) Close() error {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.transport.CloseIdleConnections()
	})
	return nil
}

// ClusterHealth 集群健康状态
type ClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// IsGreen 检查集群是否为绿色状态
func (h *ClusterHealth) IsGreen() bool {
	return h.Status == "green"
}

// IsYellow 检查集群是否为黄色状态
func (h *ClusterHealth) IsYellow() bool {
	return h.Status == "yellow"
}

// IsRed 检查集群是否为红色状态
func (h *ClusterHealth) IsRed() bool {
	return h.Status == "red"
}

// BulkResponse Bulk API 响应
type BulkResponse struct {
	Took   int                           `json:"took"`
	Errors bool                          `json:"errors"`
	Items  []map[string]BulkItemResponse `json:"items"`
}

// BulkItemResponse Bulk 操作单项响应
type BulkItemResponse struct {
	Index   string         `json:"_index"`
	ID      string         `json:"_id"`
	Version int            `json:"_version"`
	Result  string         `json:"result"`
	Status  int            `json:"status"`
	Error   *BulkItemError `json:"error,omitempty"`
}

// BulkItemError Bulk 操作错误
type BulkItemError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// FailedItems 返回失败的项
func (r *BulkResponse) FailedItems() []*BulkError {
	if !r.Errors {
		return nil
	}

	var failed []*BulkError
	for _, item := range r.Items {
		for action, result := range item {
			if result.Error != nil {
				failed = append(failed, &BulkError{
					Index:      result.Index,
					DocumentID: result.ID,
					Type:       result.Error.Type,
					Reason:     result.Error.Reason,
					Status:     result.Status,
				})
			}
			_ = action // 避免 unused variable
		}
	}
	return failed
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits         SearchHits                 `json:"hits"`
	Aggregations map[string]json.RawMessage `json:"aggregations,omitempty"`
}

// SearchHits 搜索命中结果
type SearchHits struct {
	Total    SearchTotal `json:"total"`
	MaxScore *float64    `json:"max_score"`
	Hits     []SearchHit `json:"hits"`
}

// SearchTotal 搜索总数
type SearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// SearchHit 单个搜索命中
type SearchHit struct {
	Index  string          `json:"_index"`
	ID     string          `json:"_id"`
	Score  *float64        `json:"_score"`
	Source json.RawMessage `json:"_source"`
	Sort   []interface{}   `json:"sort,omitempty"`
}

// IndexTemplate 索引模板
type IndexTemplate struct {
	IndexPatterns []string           `json:"index_patterns"`
	Priority      int                `json:"priority,omitempty"`
	Template      *IndexTemplateBody `json:"template,omitempty"`
	ComposedOf    []string           `json:"composed_of,omitempty"`
	Version       int                `json:"version,omitempty"`
	Meta          map[string]string  `json:"_meta,omitempty"`
}

// IndexTemplateBody 索引模板内容
type IndexTemplateBody struct {
	Settings map[string]interface{} `json:"settings,omitempty"`
	Mappings map[string]interface{} `json:"mappings,omitempty"`
	Aliases  map[string]interface{} `json:"aliases,omitempty"`
}

// ISMPolicy ISM 策略定义
type ISMPolicy struct {
	Description  string        `json:"description,omitempty"`
	DefaultState string        `json:"default_state"`
	States       []ISMState    `json:"states"`
	ISMTemplate  []ISMTemplate `json:"ism_template,omitempty"`
}

// ISMState ISM 状态
type ISMState struct {
	Name        string          `json:"name"`
	Actions     []ISMAction     `json:"actions,omitempty"`
	Transitions []ISMTransition `json:"transitions,omitempty"`
}

// ISMAction ISM 动作
type ISMAction struct {
	Rollover      *RolloverAction      `json:"rollover,omitempty"`
	ReadOnly      *struct{}            `json:"read_only,omitempty"`
	ForceMerge    *ForceMergeAction    `json:"force_merge,omitempty"`
	ReplicaCount  *ReplicaCountAction  `json:"replica_count,omitempty"`
	Delete        *struct{}            `json:"delete,omitempty"`
	IndexPriority *IndexPriorityAction `json:"index_priority,omitempty"`
}

// RolloverAction Rollover 动作
type RolloverAction struct {
	MinDocCount int64  `json:"min_doc_count,omitempty"`
	MinSize     string `json:"min_size,omitempty"`
	MinIndexAge string `json:"min_index_age,omitempty"`
}

// ForceMergeAction Force Merge 动作
type ForceMergeAction struct {
	MaxNumSegments int `json:"max_num_segments"`
}

// ReplicaCountAction 副本数动作
type ReplicaCountAction struct {
	NumberOfReplicas int `json:"number_of_replicas"`
}

// IndexPriorityAction 索引优先级动作
type IndexPriorityAction struct {
	Priority int `json:"priority"`
}

// ISMTransition ISM 状态转换
type ISMTransition struct {
	StateName  string         `json:"state_name"`
	Conditions *ISMConditions `json:"conditions,omitempty"`
}

// ISMConditions ISM 转换条件
type ISMConditions struct {
	MinIndexAge string `json:"min_index_age,omitempty"`
	MinDocCount int64  `json:"min_doc_count,omitempty"`
	MinSize     string `json:"min_size,omitempty"`
}

// ISMTemplate ISM 模板
type ISMTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
	Priority      int      `json:"priority,omitempty"`
}
