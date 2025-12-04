// Package enricher provides event enrichment functionality.
package enricher

import (
	"context"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// Enricher 事件丰富化器接口
type Enricher interface {
	Name() string
	Enrich(ctx context.Context, event *ecs.Event) error
	Enabled() bool
	Close() error
}

// EnricherChain 丰富化器链
type EnricherChain struct {
	enrichers []Enricher
}

// NewEnricherChain 创建丰富化器链
func NewEnricherChain(enrichers ...Enricher) *EnricherChain {
	return &EnricherChain{enrichers: enrichers}
}

// Enrich 依次调用所有丰富化器
func (c *EnricherChain) Enrich(ctx context.Context, event *ecs.Event) error {
	for _, enricher := range c.enrichers {
		if !enricher.Enabled() {
			continue
		}
		_ = enricher.Enrich(ctx, event)
	}
	return nil
}

// Close 关闭所有丰富化器
func (c *EnricherChain) Close() error {
	var lastErr error
	for _, enricher := range c.enrichers {
		if err := enricher.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
