// Package enricher provides asset enrichment functionality.
package enricher

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// AssetEnricher 资产丰富化器
type AssetEnricher struct {
	cache   *AssetCache
	enabled bool
	logger  *zap.Logger
}

// AssetEnricherConfig 资产丰富化器配置
type AssetEnricherConfig struct {
	Enabled  bool          `yaml:"enabled"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
}

// AssetCache 资产缓存
type AssetCache struct {
	data map[string]*assetCacheEntry
	mu   sync.RWMutex
	ttl  time.Duration
}

type assetCacheEntry struct {
	asset     *ecs.AssetInfo
	expiresAt time.Time
}

// NewAssetCache 创建资产缓存
func NewAssetCache(ttl time.Duration) *AssetCache {
	return &AssetCache{data: make(map[string]*assetCacheEntry), ttl: ttl}
}

// Get 从缓存获取资产信息
func (c *AssetCache) Get(agentID string) (*ecs.AssetInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[agentID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.asset, true
}

// Set 设置缓存
func (c *AssetCache) Set(agentID string, asset *ecs.AssetInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[agentID] = &assetCacheEntry{asset: asset, expiresAt: time.Now().Add(c.ttl)}
}

// Clear 清空缓存
func (c *AssetCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*assetCacheEntry)
}

// NewAssetEnricher 创建资产丰富化器
func NewAssetEnricher(cfg *AssetEnricherConfig, logger *zap.Logger) (*AssetEnricher, error) {
	if !cfg.Enabled {
		return &AssetEnricher{enabled: false, logger: logger}, nil
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ttl := cfg.CacheTTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	logger.Info("Asset enricher initialized", zap.Duration("cache_ttl", ttl))
	return &AssetEnricher{cache: NewAssetCache(ttl), enabled: true, logger: logger}, nil
}

// Name 返回丰富化器名称
func (e *AssetEnricher) Name() string { return "asset" }

// Enabled 返回是否启用
func (e *AssetEnricher) Enabled() bool { return e.enabled }

// Enrich 丰富化事件
func (e *AssetEnricher) Enrich(ctx context.Context, evt *ecs.Event) error {
	if !e.enabled || evt.AgentID == "" {
		return nil
	}

	asset, ok := e.cache.Get(evt.AgentID)
	if !ok {
		return nil
	}

	if evt.Enrichment == nil {
		evt.Enrichment = &ecs.EnrichmentData{}
	}
	evt.Enrichment.Asset = asset
	return nil
}

// SetAsset 设置资产信息（用于测试或预加载）
func (e *AssetEnricher) SetAsset(agentID string, asset *ecs.AssetInfo) {
	if e.cache != nil {
		e.cache.Set(agentID, asset)
	}
}

// Close 关闭丰富化器
func (e *AssetEnricher) Close() error {
	if e.cache != nil {
		e.cache.Clear()
	}
	return nil
}
