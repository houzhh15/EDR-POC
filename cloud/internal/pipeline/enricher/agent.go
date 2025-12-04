// Package enricher provides agent metadata enrichment functionality.
package enricher

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// AgentEnricher Agent元数据丰富化器
type AgentEnricher struct {
	cache   *AgentCache
	enabled bool
	logger  *zap.Logger
}

// AgentEnricherConfig Agent丰富化器配置
type AgentEnricherConfig struct {
	Enabled  bool          `yaml:"enabled"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
}

// AgentCache Agent缓存
type AgentCache struct {
	data map[string]*agentCacheEntry
	mu   sync.RWMutex
	ttl  time.Duration
}

type agentCacheEntry struct {
	agent     *ecs.AgentInfo
	expiresAt time.Time
}

// NewAgentCache 创建Agent缓存
func NewAgentCache(ttl time.Duration) *AgentCache {
	return &AgentCache{data: make(map[string]*agentCacheEntry), ttl: ttl}
}

// Get 从缓存获取Agent信息
func (c *AgentCache) Get(agentID string) (*ecs.AgentInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[agentID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.agent, true
}

// Set 设置缓存
func (c *AgentCache) Set(agentID string, agent *ecs.AgentInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[agentID] = &agentCacheEntry{agent: agent, expiresAt: time.Now().Add(c.ttl)}
}

// Clear 清空缓存
func (c *AgentCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*agentCacheEntry)
}

// NewAgentEnricher 创建Agent丰富化器
func NewAgentEnricher(cfg *AgentEnricherConfig, logger *zap.Logger) (*AgentEnricher, error) {
	if !cfg.Enabled {
		return &AgentEnricher{enabled: false, logger: logger}, nil
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ttl := cfg.CacheTTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	logger.Info("Agent enricher initialized", zap.Duration("cache_ttl", ttl))
	return &AgentEnricher{cache: NewAgentCache(ttl), enabled: true, logger: logger}, nil
}

// Name 返回丰富化器名称
func (e *AgentEnricher) Name() string { return "agent" }

// Enabled 返回是否启用
func (e *AgentEnricher) Enabled() bool { return e.enabled }

// Enrich 丰富化事件
func (e *AgentEnricher) Enrich(ctx context.Context, evt *ecs.Event) error {
	if !e.enabled || evt.AgentID == "" {
		return nil
	}

	agent, ok := e.cache.Get(evt.AgentID)
	if !ok {
		return nil
	}

	if evt.Enrichment == nil {
		evt.Enrichment = &ecs.EnrichmentData{}
	}
	evt.Enrichment.Agent = agent
	return nil
}

// SetAgent 设置Agent信息（用于测试或预加载）
func (e *AgentEnricher) SetAgent(agentID string, agent *ecs.AgentInfo) {
	if e.cache != nil {
		e.cache.Set(agentID, agent)
	}
}

// Close 关闭丰富化器
func (e *AgentEnricher) Close() error {
	if e.cache != nil {
		e.cache.Clear()
	}
	return nil
}
