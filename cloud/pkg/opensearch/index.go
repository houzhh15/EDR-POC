package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// IndexManager 索引管理器接口
type IndexManager interface {
	// EnsureIndexTemplate 确保索引模板存在
	EnsureIndexTemplate(ctx context.Context, name string, template *IndexTemplate) error

	// EnsureISMPolicy 确保 ISM 策略存在
	EnsureISMPolicy(ctx context.Context, name string, policy *ISMPolicy) error

	// CreateTimeBasedIndex 创建基于时间的索引
	CreateTimeBasedIndex(ctx context.Context, prefix string, timestamp time.Time) (string, error)

	// CreateIndexWithAlias 创建索引并设置别名
	CreateIndexWithAlias(ctx context.Context, name string, alias string, settings map[string]interface{}) error

	// GetIndexStats 获取索引统计信息
	GetIndexStats(ctx context.Context, pattern string) (*IndexStats, error)

	// Rollover 执行索引滚动
	Rollover(ctx context.Context, alias string, conditions RolloverConditions) (*RolloverResult, error)

	// DeleteOldIndices 删除过期索引
	DeleteOldIndices(ctx context.Context, pattern string, olderThan time.Duration) ([]string, error)

	// RefreshIndex 刷新索引
	RefreshIndex(ctx context.Context, indices ...string) error
}

// indexManager IndexManager 实现
type indexManager struct {
	client Client
}

// NewIndexManager 创建索引管理器
func NewIndexManager(client Client) IndexManager {
	return &indexManager{client: client}
}

// EnsureIndexTemplate 确保索引模板存在
func (im *indexManager) EnsureIndexTemplate(ctx context.Context, name string, template *IndexTemplate) error {
	return im.client.PutIndexTemplate(ctx, name, template)
}

// EnsureISMPolicy 确保 ISM 策略存在
func (im *indexManager) EnsureISMPolicy(ctx context.Context, name string, policy *ISMPolicy) error {
	return im.client.PutISMPolicy(ctx, name, policy)
}

// CreateTimeBasedIndex 创建基于时间的索引
func (im *indexManager) CreateTimeBasedIndex(ctx context.Context, prefix string, timestamp time.Time) (string, error) {
	indexName := fmt.Sprintf("%s-%s", prefix, timestamp.UTC().Format("2006.01.02"))

	err := im.client.CreateIndex(ctx, indexName, nil)
	if err != nil {
		// 忽略索引已存在的错误
		if !IsConflict(err) {
			return "", fmt.Errorf("create index %s: %w", indexName, err)
		}
	}

	return indexName, nil
}

// CreateIndexWithAlias 创建索引并设置别名
func (im *indexManager) CreateIndexWithAlias(ctx context.Context, name string, alias string, settings map[string]interface{}) error {
	if settings == nil {
		settings = make(map[string]interface{})
	}

	if alias != "" {
		if settings["aliases"] == nil {
			settings["aliases"] = make(map[string]interface{})
		}
		aliases := settings["aliases"].(map[string]interface{})
		aliases[alias] = map[string]interface{}{
			"is_write_index": true,
		}
	}

	return im.client.CreateIndex(ctx, name, settings)
}

// GetIndexStats 获取索引统计信息
func (im *indexManager) GetIndexStats(ctx context.Context, pattern string) (*IndexStats, error) {
	c, ok := im.client.(*opensearchClient)
	if !ok {
		return nil, fmt.Errorf("invalid client type")
	}

	path := fmt.Sprintf("/%s/_stats", pattern)
	respBody, err := c.doRequestWithRetry(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var statsResp indexStatsResponse
	if err := json.Unmarshal(respBody, &statsResp); err != nil {
		return nil, fmt.Errorf("parse stats response: %w", err)
	}

	// 汇总统计
	stats := &IndexStats{
		Indices: make(map[string]SingleIndexStats),
	}

	stats.DocsCount = statsResp.All.Primaries.Docs.Count
	stats.DocsDeleted = statsResp.All.Primaries.Docs.Deleted
	stats.StoreSizeBytes = statsResp.All.Primaries.Store.SizeInBytes
	stats.IndexCount = len(statsResp.Indices)

	for name, idx := range statsResp.Indices {
		stats.Indices[name] = SingleIndexStats{
			DocsCount:      idx.Primaries.Docs.Count,
			DocsDeleted:    idx.Primaries.Docs.Deleted,
			StoreSizeBytes: idx.Primaries.Store.SizeInBytes,
		}
	}

	return stats, nil
}

// Rollover 执行索引滚动
func (im *indexManager) Rollover(ctx context.Context, alias string, conditions RolloverConditions) (*RolloverResult, error) {
	c, ok := im.client.(*opensearchClient)
	if !ok {
		return nil, fmt.Errorf("invalid client type")
	}

	body := map[string]interface{}{
		"conditions": conditions.toMap(),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal rollover body: %w", err)
	}

	path := fmt.Sprintf("/%s/_rollover", alias)
	respBody, err := c.doRequestWithRetry(ctx, "POST", path, bodyBytes, nil)
	if err != nil {
		return nil, err
	}

	var rolloverResp RolloverResult
	if err := json.Unmarshal(respBody, &rolloverResp); err != nil {
		return nil, fmt.Errorf("parse rollover response: %w", err)
	}

	return &rolloverResp, nil
}

// DeleteOldIndices 删除过期索引
func (im *indexManager) DeleteOldIndices(ctx context.Context, pattern string, olderThan time.Duration) ([]string, error) {
	c, ok := im.client.(*opensearchClient)
	if !ok {
		return nil, fmt.Errorf("invalid client type")
	}

	// 获取匹配的索引列表
	path := fmt.Sprintf("/_cat/indices/%s?format=json", pattern)
	respBody, err := c.doRequestWithRetry(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var indices []catIndex
	if err := json.Unmarshal(respBody, &indices); err != nil {
		return nil, fmt.Errorf("parse cat indices response: %w", err)
	}

	var deleted []string
	cutoff := time.Now().Add(-olderThan)

	for _, idx := range indices {
		// 解析索引名称中的日期
		indexTime, err := parseIndexDate(idx.Index)
		if err != nil {
			continue // 无法解析日期，跳过
		}

		if indexTime.Before(cutoff) {
			if err := im.client.DeleteIndex(ctx, idx.Index); err != nil {
				return deleted, fmt.Errorf("delete index %s: %w", idx.Index, err)
			}
			deleted = append(deleted, idx.Index)
		}
	}

	return deleted, nil
}

// RefreshIndex 刷新索引
func (im *indexManager) RefreshIndex(ctx context.Context, indices ...string) error {
	c, ok := im.client.(*opensearchClient)
	if !ok {
		return fmt.Errorf("invalid client type")
	}

	path := "/_refresh"
	if len(indices) > 0 {
		path = fmt.Sprintf("/%s/_refresh", joinIndices(indices))
	}

	_, err := c.doRequestWithRetry(ctx, "POST", path, nil, nil)
	return err
}

// parseIndexDate 从索引名称解析日期
func parseIndexDate(indexName string) (time.Time, error) {
	// 尝试多种日期格式
	formats := []struct {
		layout string
		suffix int
	}{
		{"2006.01.02", 10}, // daily
		{"2006.01", 7},     // monthly
		{"2006-01-02", 10}, // daily with dash
	}

	for _, f := range formats {
		if len(indexName) >= f.suffix {
			dateStr := indexName[len(indexName)-f.suffix:]
			if t, err := time.Parse(f.layout, dateStr); err == nil {
				return t, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse date from index name: %s", indexName)
}

// joinIndices 拼接索引名称
func joinIndices(indices []string) string {
	result := ""
	for i, idx := range indices {
		if i > 0 {
			result += ","
		}
		result += idx
	}
	return result
}

// IndexStats 索引统计信息
type IndexStats struct {
	DocsCount      int64                       `json:"docs_count"`
	DocsDeleted    int64                       `json:"docs_deleted"`
	StoreSizeBytes int64                       `json:"store_size_bytes"`
	IndexCount     int                         `json:"index_count"`
	Indices        map[string]SingleIndexStats `json:"indices"`
}

// SingleIndexStats 单个索引统计
type SingleIndexStats struct {
	DocsCount      int64 `json:"docs_count"`
	DocsDeleted    int64 `json:"docs_deleted"`
	StoreSizeBytes int64 `json:"store_size_bytes"`
}

// RolloverConditions 滚动条件
type RolloverConditions struct {
	MaxAge     string `json:"max_age,omitempty"`
	MaxDocs    int64  `json:"max_docs,omitempty"`
	MaxSize    string `json:"max_size,omitempty"`
	MaxPrimary string `json:"max_primary_shard_size,omitempty"`
}

// toMap 转换为 map
func (rc RolloverConditions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if rc.MaxAge != "" {
		m["max_age"] = rc.MaxAge
	}
	if rc.MaxDocs > 0 {
		m["max_docs"] = rc.MaxDocs
	}
	if rc.MaxSize != "" {
		m["max_size"] = rc.MaxSize
	}
	if rc.MaxPrimary != "" {
		m["max_primary_shard_size"] = rc.MaxPrimary
	}
	return m
}

// RolloverResult 滚动结果
type RolloverResult struct {
	Acknowledged       bool                   `json:"acknowledged"`
	ShardsAcknowledged bool                   `json:"shards_acknowledged"`
	OldIndex           string                 `json:"old_index"`
	NewIndex           string                 `json:"new_index"`
	RolledOver         bool                   `json:"rolled_over"`
	DryRun             bool                   `json:"dry_run"`
	Conditions         map[string]interface{} `json:"conditions"`
}

// indexStatsResponse 索引统计 API 响应
type indexStatsResponse struct {
	All struct {
		Primaries struct {
			Docs struct {
				Count   int64 `json:"count"`
				Deleted int64 `json:"deleted"`
			} `json:"docs"`
			Store struct {
				SizeInBytes int64 `json:"size_in_bytes"`
			} `json:"store"`
		} `json:"primaries"`
	} `json:"_all"`
	Indices map[string]struct {
		Primaries struct {
			Docs struct {
				Count   int64 `json:"count"`
				Deleted int64 `json:"deleted"`
			} `json:"docs"`
			Store struct {
				SizeInBytes int64 `json:"size_in_bytes"`
			} `json:"store"`
		} `json:"primaries"`
	} `json:"indices"`
}

// catIndex cat indices API 响应项
type catIndex struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Index        string `json:"index"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

// ============ 预定义索引模板 ============

// NewEventsIndexTemplate 创建事件索引模板
func NewEventsIndexTemplate() *IndexTemplate {
	return &IndexTemplate{
		IndexPatterns: []string{"edr-events-*"},
		Priority:      100,
		Template: &IndexTemplateBody{
			Settings: map[string]interface{}{
				"number_of_shards":                 3,
				"number_of_replicas":               1,
				"refresh_interval":                 "5s",
				"index.mapping.total_fields.limit": 2000,
			},
			Mappings: map[string]interface{}{
				"dynamic": "strict",
				"properties": map[string]interface{}{
					"@timestamp": map[string]interface{}{"type": "date"},
					"event": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":       map[string]interface{}{"type": "keyword"},
							"category": map[string]interface{}{"type": "keyword"},
							"type":     map[string]interface{}{"type": "keyword"},
							"action":   map[string]interface{}{"type": "keyword"},
							"outcome":  map[string]interface{}{"type": "keyword"},
							"severity": map[string]interface{}{"type": "integer"},
						},
					},
					"host": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":       map[string]interface{}{"type": "keyword"},
							"name":     map[string]interface{}{"type": "keyword"},
							"hostname": map[string]interface{}{"type": "keyword"},
							"ip":       map[string]interface{}{"type": "ip"},
							"os": map[string]interface{}{
								"properties": map[string]interface{}{
									"family":  map[string]interface{}{"type": "keyword"},
									"name":    map[string]interface{}{"type": "keyword"},
									"version": map[string]interface{}{"type": "keyword"},
								},
							},
						},
					},
					"process": map[string]interface{}{
						"properties": map[string]interface{}{
							"pid":          map[string]interface{}{"type": "long"},
							"name":         map[string]interface{}{"type": "keyword"},
							"executable":   map[string]interface{}{"type": "keyword"},
							"command_line": map[string]interface{}{"type": "text"},
							"hash": map[string]interface{}{
								"properties": map[string]interface{}{
									"md5":    map[string]interface{}{"type": "keyword"},
									"sha256": map[string]interface{}{"type": "keyword"},
								},
							},
							"parent": map[string]interface{}{
								"properties": map[string]interface{}{
									"pid":  map[string]interface{}{"type": "long"},
									"name": map[string]interface{}{"type": "keyword"},
								},
							},
						},
					},
					"file": map[string]interface{}{
						"properties": map[string]interface{}{
							"path":      map[string]interface{}{"type": "keyword"},
							"name":      map[string]interface{}{"type": "keyword"},
							"extension": map[string]interface{}{"type": "keyword"},
							"size":      map[string]interface{}{"type": "long"},
							"hash": map[string]interface{}{
								"properties": map[string]interface{}{
									"md5":    map[string]interface{}{"type": "keyword"},
									"sha256": map[string]interface{}{"type": "keyword"},
								},
							},
						},
					},
					"network": map[string]interface{}{
						"properties": map[string]interface{}{
							"direction": map[string]interface{}{"type": "keyword"},
							"protocol":  map[string]interface{}{"type": "keyword"},
							"transport": map[string]interface{}{"type": "keyword"},
						},
					},
					"source": map[string]interface{}{
						"properties": map[string]interface{}{
							"ip":   map[string]interface{}{"type": "ip"},
							"port": map[string]interface{}{"type": "integer"},
						},
					},
					"destination": map[string]interface{}{
						"properties": map[string]interface{}{
							"ip":     map[string]interface{}{"type": "ip"},
							"port":   map[string]interface{}{"type": "integer"},
							"domain": map[string]interface{}{"type": "keyword"},
						},
					},
					"agent": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":      map[string]interface{}{"type": "keyword"},
							"version": map[string]interface{}{"type": "keyword"},
						},
					},
				},
			},
		},
	}
}

// NewAlertsIndexTemplate 创建告警索引模板
func NewAlertsIndexTemplate() *IndexTemplate {
	return &IndexTemplate{
		IndexPatterns: []string{"edr-alerts-*"},
		Priority:      100,
		Template: &IndexTemplateBody{
			Settings: map[string]interface{}{
				"number_of_shards":   1,
				"number_of_replicas": 1,
			},
			Mappings: map[string]interface{}{
				"properties": map[string]interface{}{
					"@timestamp": map[string]interface{}{"type": "date"},
					"alert": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":        map[string]interface{}{"type": "keyword"},
							"name":      map[string]interface{}{"type": "keyword"},
							"severity":  map[string]interface{}{"type": "keyword"},
							"status":    map[string]interface{}{"type": "keyword"},
							"rule_id":   map[string]interface{}{"type": "keyword"},
							"rule_name": map[string]interface{}{"type": "keyword"},
						},
					},
					"event": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":  map[string]interface{}{"type": "keyword"},
							"ids": map[string]interface{}{"type": "keyword"},
						},
					},
					"host": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":   map[string]interface{}{"type": "keyword"},
							"name": map[string]interface{}{"type": "keyword"},
						},
					},
					"assignee": map[string]interface{}{"type": "keyword"},
					"tags":     map[string]interface{}{"type": "keyword"},
				},
			},
		},
	}
}

// NewAssetsIndexTemplate 创建资产索引模板
func NewAssetsIndexTemplate() *IndexTemplate {
	return &IndexTemplate{
		IndexPatterns: []string{"edr-assets"},
		Priority:      100,
		Template: &IndexTemplateBody{
			Settings: map[string]interface{}{
				"number_of_shards":   1,
				"number_of_replicas": 1,
			},
			Mappings: map[string]interface{}{
				"properties": map[string]interface{}{
					"asset_id":     map[string]interface{}{"type": "keyword"},
					"hostname":     map[string]interface{}{"type": "keyword"},
					"ip_addresses": map[string]interface{}{"type": "ip"},
					"os": map[string]interface{}{
						"properties": map[string]interface{}{
							"family":  map[string]interface{}{"type": "keyword"},
							"name":    map[string]interface{}{"type": "keyword"},
							"version": map[string]interface{}{"type": "keyword"},
						},
					},
					"agent": map[string]interface{}{
						"properties": map[string]interface{}{
							"id":      map[string]interface{}{"type": "keyword"},
							"version": map[string]interface{}{"type": "keyword"},
							"status":  map[string]interface{}{"type": "keyword"},
						},
					},
					"tags":       map[string]interface{}{"type": "keyword"},
					"last_seen":  map[string]interface{}{"type": "date"},
					"first_seen": map[string]interface{}{"type": "date"},
					"updated_at": map[string]interface{}{"type": "date"},
				},
			},
		},
	}
}

// NewEDRLifecyclePolicy 创建 EDR 生命周期策略
func NewEDRLifecyclePolicy() *ISMPolicy {
	return &ISMPolicy{
		Description:  "EDR events lifecycle policy",
		DefaultState: "hot",
		States: []ISMState{
			{
				Name: "hot",
				Actions: []ISMAction{
					{
						Rollover: &RolloverAction{
							MinDocCount: 50000000,
							MinSize:     "50gb",
							MinIndexAge: "1d",
						},
					},
				},
				Transitions: []ISMTransition{
					{
						StateName:  "warm",
						Conditions: &ISMConditions{MinIndexAge: "7d"},
					},
				},
			},
			{
				Name: "warm",
				Actions: []ISMAction{
					{ReadOnly: &struct{}{}},
					{ForceMerge: &ForceMergeAction{MaxNumSegments: 1}},
					{IndexPriority: &IndexPriorityAction{Priority: 50}},
				},
				Transitions: []ISMTransition{
					{
						StateName:  "cold",
						Conditions: &ISMConditions{MinIndexAge: "30d"},
					},
				},
			},
			{
				Name: "cold",
				Actions: []ISMAction{
					{ReadOnly: &struct{}{}},
					{ReplicaCount: &ReplicaCountAction{NumberOfReplicas: 0}},
				},
				Transitions: []ISMTransition{
					{
						StateName:  "delete",
						Conditions: &ISMConditions{MinIndexAge: "90d"},
					},
				},
			},
			{
				Name: "delete",
				Actions: []ISMAction{
					{Delete: &struct{}{}},
				},
			},
		},
		ISMTemplate: []ISMTemplate{
			{
				IndexPatterns: []string{"edr-events-*"},
				Priority:      100,
			},
		},
	}
}
