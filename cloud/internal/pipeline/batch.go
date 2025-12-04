// Package pipeline 提供事件处理管线核心功能
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/enricher"
)

// Batch 事件批次
type Batch struct {
	// Events 原始事件列表
	Events []*ecs.Event
	// StartTime 批次开始时间
	StartTime time.Time
	// ID 批次ID
	ID string
}

// ProcessedBatch 已处理的批次
type ProcessedBatch struct {
	// Events 标准化后的ECS事件
	Events []*ecs.ECSEvent
	// FailedEvents 处理失败的事件
	FailedEvents []*FailedEvent
	// BatchID 批次ID
	BatchID string
	// ProcessingTime 处理耗时
	ProcessingTime time.Duration
}

// FailedEvent 处理失败的事件
type FailedEvent struct {
	// Event 原始事件
	Event *ecs.Event
	// Error 错误信息
	Error error
	// Stage 失败阶段
	Stage string
}

// BatchProcessor 批处理器接口
type BatchProcessor interface {
	// Process 处理事件批次
	Process(ctx context.Context, batch *Batch) (*ProcessedBatch, error)
	// ProcessAsync 异步处理事件批次
	ProcessAsync(ctx context.Context, batch *Batch) <-chan *ProcessedBatch
}

// BatchProcessorConfig 批处理器配置
type BatchProcessorConfig struct {
	// Workers 工作协程数量
	Workers int
	// BatchSize 批次大小
	BatchSize int
	// BatchTimeout 批次超时时间
	BatchTimeout time.Duration
	// EnableParallel 是否启用并行处理
	EnableParallel bool
}

// DefaultBatchProcessor 默认批处理器实现
type DefaultBatchProcessor struct {
	config     *BatchProcessorConfig
	enrichers  []enricher.Enricher
	normalizer Normalizer
	metrics    *PipelineMetrics
	mu         sync.RWMutex
}

// NewDefaultBatchProcessor 创建默认批处理器
func NewDefaultBatchProcessor(
	cfg *BatchProcessorConfig,
	enrichers []enricher.Enricher,
	normalizer Normalizer,
	metrics *PipelineMetrics,
) *DefaultBatchProcessor {
	if cfg == nil {
		cfg = &BatchProcessorConfig{
			Workers:        4,
			BatchSize:      1000,
			BatchTimeout:   100 * time.Millisecond,
			EnableParallel: true,
		}
	}

	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}

	return &DefaultBatchProcessor{
		config:     cfg,
		enrichers:  enrichers,
		normalizer: normalizer,
		metrics:    metrics,
	}
}

// Process 处理事件批次
func (p *DefaultBatchProcessor) Process(ctx context.Context, batch *Batch) (*ProcessedBatch, error) {
	if batch == nil || len(batch.Events) == 0 {
		return &ProcessedBatch{
			Events:  make([]*ecs.ECSEvent, 0),
			BatchID: batch.ID,
		}, nil
	}

	startTime := time.Now()

	var (
		processed []*ecs.ECSEvent
		failed    []*FailedEvent
	)

	if p.config.EnableParallel && len(batch.Events) > 1 {
		// 并行处理
		processed, failed = p.processParallel(ctx, batch.Events)
	} else {
		// 串行处理
		processed, failed = p.processSequential(ctx, batch.Events)
	}

	processingTime := time.Since(startTime)

	// 记录指标
	if p.metrics != nil {
		p.metrics.RecordEventProcessed("batch", true)
		for range failed {
			p.metrics.RecordEventProcessed("batch", false)
		}
		p.metrics.RecordProcessingDuration("batch_processing", processingTime.Seconds())
		p.metrics.RecordBatchSize("batch", len(batch.Events))
	}

	return &ProcessedBatch{
		Events:         processed,
		FailedEvents:   failed,
		BatchID:        batch.ID,
		ProcessingTime: processingTime,
	}, nil
}

// processSequential 串行处理事件
func (p *DefaultBatchProcessor) processSequential(ctx context.Context, events []*ecs.Event) ([]*ecs.ECSEvent, []*FailedEvent) {
	processed := make([]*ecs.ECSEvent, 0, len(events))
	failed := make([]*FailedEvent, 0)

	for _, evt := range events {
		select {
		case <-ctx.Done():
			// 将剩余事件标记为失败
			for _, remaining := range events[len(processed)+len(failed):] {
				failed = append(failed, &FailedEvent{
					Event: remaining,
					Error: ctx.Err(),
					Stage: "cancelled",
				})
			}
			return processed, failed
		default:
		}

		ecsEvent, err := p.processEvent(ctx, evt)
		if err != nil {
			failed = append(failed, &FailedEvent{
				Event: evt,
				Error: err,
				Stage: "processing",
			})
			continue
		}
		processed = append(processed, ecsEvent)
	}

	return processed, failed
}

// processParallel 并行处理事件
func (p *DefaultBatchProcessor) processParallel(ctx context.Context, events []*ecs.Event) ([]*ecs.ECSEvent, []*FailedEvent) {
	var (
		processed = make([]*ecs.ECSEvent, 0, len(events))
		failed    = make([]*FailedEvent, 0)
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	// 创建工作通道
	eventCh := make(chan *ecs.Event, len(events))

	// 启动工作协程
	for i := 0; i < p.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for evt := range eventCh {
				select {
				case <-ctx.Done():
					mu.Lock()
					failed = append(failed, &FailedEvent{
						Event: evt,
						Error: ctx.Err(),
						Stage: "cancelled",
					})
					mu.Unlock()
					continue
				default:
				}

				ecsEvent, err := p.processEvent(ctx, evt)
				mu.Lock()
				if err != nil {
					failed = append(failed, &FailedEvent{
						Event: evt,
						Error: err,
						Stage: "processing",
					})
				} else {
					processed = append(processed, ecsEvent)
				}
				mu.Unlock()
			}
		}()
	}

	// 发送事件到工作通道
	for _, evt := range events {
		eventCh <- evt
	}
	close(eventCh)

	// 等待所有工作协程完成
	wg.Wait()

	return processed, failed
}

// processEvent 处理单个事件
func (p *DefaultBatchProcessor) processEvent(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
	// 1. 富化阶段
	for _, e := range p.enrichers {
		if err := e.Enrich(ctx, evt); err != nil {
			// 富化失败不中断处理，记录错误继续
			if p.metrics != nil {
				p.metrics.RecordError("enrichment", err.Error())
			}
		}
	}

	// 2. 标准化阶段
	ecsEvent, err := p.normalizer.Normalize(ctx, evt)
	if err != nil {
		return nil, fmt.Errorf("normalize event: %w", err)
	}

	return ecsEvent, nil
}

// ProcessAsync 异步处理事件批次
func (p *DefaultBatchProcessor) ProcessAsync(ctx context.Context, batch *Batch) <-chan *ProcessedBatch {
	resultCh := make(chan *ProcessedBatch, 1)

	go func() {
		defer close(resultCh)
		result, err := p.Process(ctx, batch)
		if err != nil {
			result = &ProcessedBatch{
				Events:  make([]*ecs.ECSEvent, 0),
				BatchID: batch.ID,
				FailedEvents: []*FailedEvent{
					{
						Error: err,
						Stage: "batch_processing",
					},
				},
			}
		}
		select {
		case resultCh <- result:
		case <-ctx.Done():
		}
	}()

	return resultCh
}

// BatchCollector 批次收集器
type BatchCollector struct {
	config    *BatchProcessorConfig
	buffer    []*ecs.Event
	mu        sync.Mutex
	batchID   int64
	lastFlush time.Time
}

// NewBatchCollector 创建批次收集器
func NewBatchCollector(cfg *BatchProcessorConfig) *BatchCollector {
	return &BatchCollector{
		config:    cfg,
		buffer:    make([]*ecs.Event, 0, cfg.BatchSize),
		lastFlush: time.Now(),
	}
}

// Add 添加事件到收集器
func (c *BatchCollector) Add(evt *ecs.Event) *Batch {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buffer = append(c.buffer, evt)

	// 检查是否需要刷新
	if len(c.buffer) >= c.config.BatchSize || time.Since(c.lastFlush) >= c.config.BatchTimeout {
		return c.flush()
	}

	return nil
}

// Flush 强制刷新缓冲区
func (c *BatchCollector) Flush() *Batch {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.flush()
}

// flush 内部刷新方法（需要持有锁）
func (c *BatchCollector) flush() *Batch {
	if len(c.buffer) == 0 {
		return nil
	}

	c.batchID++
	batch := &Batch{
		Events:    c.buffer,
		StartTime: time.Now(),
		ID:        fmt.Sprintf("batch-%d", c.batchID),
	}

	c.buffer = make([]*ecs.Event, 0, c.config.BatchSize)
	c.lastFlush = time.Now()

	return batch
}

// Size 返回当前缓冲区大小
func (c *BatchCollector) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.buffer)
}

// ParseRawEvent 从原始JSON解析事件
func ParseRawEvent(data []byte) (*ecs.Event, error) {
	var evt ecs.Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}
	return &evt, nil
}
