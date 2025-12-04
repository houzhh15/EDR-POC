// Package pipeline 提供事件处理管线主协调器
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/enricher"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/writer"
	"github.com/segmentio/kafka-go"
)

// Pipeline 事件处理管线
type Pipeline struct {
	config    *PipelineConfig
	consumer  *kafka.Reader
	processor BatchProcessor
	writers   []writer.Writer
	dlqWriter writer.Writer
	collector *BatchCollector
	metrics   *PipelineMetrics

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool
}

// NewPipeline 创建新的事件处理管线
func NewPipeline(
	config *PipelineConfig,
	enrichers []enricher.Enricher,
	normalizer Normalizer,
	writers []writer.Writer,
	dlqWriter writer.Writer,
	metrics *PipelineMetrics,
) (*Pipeline, error) {
	if config == nil {
		return nil, fmt.Errorf("pipeline config is nil")
	}

	// 创建 Kafka 消费者
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Input.Kafka.Brokers,
		Topic:          config.Input.Kafka.Topic,
		GroupID:        config.Input.Kafka.ConsumerGroup,
		MinBytes:       1e3,  // 1KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        config.Processing.BatchTimeout,
		CommitInterval: time.Second,
	})

	// 创建批处理器
	processorConfig := &BatchProcessorConfig{
		Workers:        config.Processing.WorkerCount,
		BatchSize:      config.Processing.BatchSize,
		BatchTimeout:   config.Processing.BatchTimeout,
		EnableParallel: config.Processing.WorkerCount > 1,
	}
	processor := NewDefaultBatchProcessor(processorConfig, enrichers, normalizer, metrics)

	// 创建批次收集器
	collector := NewBatchCollector(processorConfig)

	return &Pipeline{
		config:    config,
		consumer:  consumer,
		processor: processor,
		writers:   writers,
		dlqWriter: dlqWriter,
		collector: collector,
		metrics:   metrics,
	}, nil
}

// Start 启动管线
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline is already running")
	}
	p.running = true
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.mu.Unlock()

	// 启动消费循环
	p.wg.Add(1)
	go p.consumeLoop()

	// 启动定时刷新
	p.wg.Add(1)
	go p.flushLoop()

	return nil
}

// Stop 停止管线
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.cancel()
	p.mu.Unlock()

	// 等待所有协程退出
	p.wg.Wait()

	// 刷新剩余数据
	if batch := p.collector.Flush(); batch != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.processBatch(ctx, batch)
	}

	// 关闭消费者
	if err := p.consumer.Close(); err != nil {
		return fmt.Errorf("close consumer: %w", err)
	}

	// 关闭写入器
	for _, w := range p.writers {
		if err := w.Close(); err != nil {
			return fmt.Errorf("close writer: %w", err)
		}
	}

	if p.dlqWriter != nil {
		if err := p.dlqWriter.Close(); err != nil {
			return fmt.Errorf("close dlq writer: %w", err)
		}
	}

	return nil
}

// consumeLoop 消费循环
func (p *Pipeline) consumeLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// 从 Kafka 读取消息
		msg, err := p.consumer.FetchMessage(p.ctx)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			if p.metrics != nil {
				p.metrics.RecordError("consume", err.Error())
			}
			continue
		}

		// 记录消费指标
		if p.metrics != nil {
			p.metrics.RecordEventConsumed(msg.Topic, 1)
		}

		// 解析事件
		evt, err := ParseRawEvent(msg.Value)
		if err != nil {
			// 解析失败，发送到 DLQ
			p.sendToDLQ(msg.Value, "parse_error", err)
			p.commitMessage(msg)
			continue
		}

		// 添加到批次收集器
		batch := p.collector.Add(evt)
		if batch != nil {
			p.processBatch(p.ctx, batch)
		}

		// 提交消息
		p.commitMessage(msg)
	}
}

// flushLoop 定时刷新循环
func (p *Pipeline) flushLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.Processing.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if batch := p.collector.Flush(); batch != nil {
				p.processBatch(p.ctx, batch)
			}
		}
	}
}

// processBatch 处理批次
func (p *Pipeline) processBatch(ctx context.Context, batch *Batch) {
	startTime := time.Now()

	// 处理批次
	result, err := p.processor.Process(ctx, batch)
	if err != nil {
		if p.metrics != nil {
			p.metrics.RecordError("processing", err.Error())
		}
		return
	}

	// 写入成功处理的事件
	if len(result.Events) > 0 {
		p.writeEvents(ctx, result.Events)
	}

	// 处理失败的事件
	for _, failed := range result.FailedEvents {
		var eventData []byte
		if failed.Event != nil {
			eventData, _ = json.Marshal(failed.Event)
		}
		p.sendToDLQ(eventData, failed.Stage, failed.Error)
	}

	// 记录处理耗时
	if p.metrics != nil {
		p.metrics.RecordProcessingDuration("batch_total", time.Since(startTime).Seconds())
	}
}

// writeEvents 写入事件到所有输出
func (p *Pipeline) writeEvents(ctx context.Context, events []*ecs.ECSEvent) {
	// 序列化事件
	serialized := make([][]byte, 0, len(events))
	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			if p.metrics != nil {
				p.metrics.RecordError("serialize", err.Error())
			}
			continue
		}
		serialized = append(serialized, data)
	}

	// 写入所有目标
	for i, w := range p.writers {
		startTime := time.Now()
		err := w.WriteBatch(ctx, serialized)
		duration := time.Since(startTime)

		if p.metrics != nil {
			writerName := fmt.Sprintf("writer_%d", i)
			p.metrics.RecordWriterLatency(writerName, duration.Seconds())
			if err != nil {
				p.metrics.RecordEventWritten(writerName, len(serialized), false)
				p.metrics.RecordError("write", err.Error())
			} else {
				p.metrics.RecordEventWritten(writerName, len(serialized), true)
			}
		}
	}
}

// sendToDLQ 发送消息到死信队列
func (p *Pipeline) sendToDLQ(data []byte, reason string, err error) {
	if p.dlqWriter == nil {
		return
	}

	dlqMsg := DLQMessage{
		OriginalData: data,
		Reason:       reason,
		Error:        err.Error(),
		Timestamp:    time.Now(),
	}

	msgData, marshalErr := json.Marshal(dlqMsg)
	if marshalErr != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if writeErr := p.dlqWriter.Write(ctx, msgData); writeErr != nil {
		if p.metrics != nil {
			p.metrics.RecordError("dlq_write", writeErr.Error())
		}
	} else {
		if p.metrics != nil {
			p.metrics.RecordDLQMessage(reason)
		}
	}
}

// commitMessage 提交消息偏移量
func (p *Pipeline) commitMessage(msg kafka.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.consumer.CommitMessages(ctx, msg); err != nil {
		if p.metrics != nil {
			p.metrics.RecordError("commit", err.Error())
		}
	}
}

// DLQMessage 死信队列消息
type DLQMessage struct {
	OriginalData []byte    `json:"original_data"`
	Reason       string    `json:"reason"`
	Error        string    `json:"error"`
	Timestamp    time.Time `json:"timestamp"`
}

// IsRunning 返回管线是否正在运行
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Stats 返回管线统计信息
type PipelineStats struct {
	Running     bool  `json:"running"`
	BufferSize  int   `json:"buffer_size"`
	ConsumerLag int64 `json:"consumer_lag"`
}

// Stats 返回管线统计信息
func (p *Pipeline) Stats() *PipelineStats {
	return &PipelineStats{
		Running:    p.IsRunning(),
		BufferSize: p.collector.Size(),
	}
}

// HealthCheck 健康检查
func (p *Pipeline) HealthCheck() error {
	if !p.IsRunning() {
		return fmt.Errorf("pipeline is not running")
	}
	return nil
}
