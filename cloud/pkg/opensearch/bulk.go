package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BulkIndexer 批量索引器接口
type BulkIndexer interface {
	// Add 添加文档到批量队列
	Add(ctx context.Context, item BulkItem) error

	// Flush 立即刷新缓冲区
	Flush(ctx context.Context) error

	// Stats 获取统计信息
	Stats() BulkStats

	// Close 关闭索引器，刷新剩余数据
	Close(ctx context.Context) error
}

// BulkItem 批量操作项
type BulkItem struct {
	// Action 操作类型: "index", "create", "update", "delete"
	Action string

	// Index 目标索引名称
	Index string

	// DocumentID 文档ID（可选，空则自动生成）
	DocumentID string

	// Document 文档内容
	Document interface{}

	// Routing 路由值（可选）
	Routing string

	// Pipeline Ingest pipeline（可选）
	Pipeline string
}

// bulkIndexer BulkIndexer 实现
type bulkIndexer struct {
	client Client
	config *BulkIndexerConfig

	// 缓冲区管理
	buffer    []BulkItem
	bufferMu  sync.Mutex
	bufferLen atomic.Int64

	// 字节数统计
	currentBytes atomic.Int64

	// 统计信息
	stats struct {
		numAdded     atomic.Int64
		numFlushed   atomic.Int64
		numFailed    atomic.Int64
		numRequests  atomic.Int64
		bytesFlushed atomic.Int64
	}

	// 控制信号
	ticker    *time.Ticker
	stopCh    chan struct{}
	flushCh   chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once

	// worker 控制
	wg sync.WaitGroup
}

// NewBulkIndexer 创建批量索引器
func NewBulkIndexer(client Client, opts ...BulkIndexerOption) (BulkIndexer, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: client is required", ErrInvalidConfig)
	}

	cfg := DefaultBulkIndexerConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU()
	}

	bi := &bulkIndexer{
		client:  client,
		config:  cfg,
		buffer:  make([]BulkItem, 0, cfg.BatchSize),
		stopCh:  make(chan struct{}),
		flushCh: make(chan struct{}, 1),
	}

	// 启动定时刷新
	if cfg.FlushInterval > 0 {
		bi.ticker = time.NewTicker(cfg.FlushInterval)
		bi.wg.Add(1)
		go bi.flushLoop()
	}

	return bi, nil
}

// Add 添加文档到批量队列
func (bi *bulkIndexer) Add(ctx context.Context, item BulkItem) error {
	if bi.closed.Load() {
		return ErrClientClosed
	}

	// 验证必填字段
	if item.Index == "" {
		return fmt.Errorf("%w: index is required", ErrInvalidConfig)
	}
	if item.Action == "" {
		item.Action = "index"
	}

	// 计算文档大小
	docBytes, err := bi.marshalItem(item)
	if err != nil {
		return fmt.Errorf("marshal item: %w", err)
	}
	itemSize := len(docBytes)

	bi.bufferMu.Lock()
	bi.buffer = append(bi.buffer, item)
	bufLen := len(bi.buffer)
	bi.bufferMu.Unlock()

	bi.bufferLen.Add(1)
	bi.currentBytes.Add(int64(itemSize))
	bi.stats.numAdded.Add(1)

	// 检查是否需要刷新
	shouldFlush := bufLen >= bi.config.BatchSize ||
		(bi.config.FlushBytes > 0 && bi.currentBytes.Load() >= int64(bi.config.FlushBytes))

	if shouldFlush {
		select {
		case bi.flushCh <- struct{}{}:
			// 触发异步刷新
			bi.wg.Add(1)
			go func() {
				defer bi.wg.Done()
				if err := bi.Flush(ctx); err != nil && bi.config.OnError != nil {
					bi.config.OnError(err)
				}
			}()
		default:
			// 已有刷新在进行中
		}
	}

	return nil
}

// marshalItem 序列化单个操作项为 NDJSON 格式
func (bi *bulkIndexer) marshalItem(item BulkItem) ([]byte, error) {
	var buf bytes.Buffer

	// 构建操作元数据
	meta := map[string]interface{}{
		"_index": item.Index,
	}
	if item.DocumentID != "" {
		meta["_id"] = item.DocumentID
	}
	if item.Routing != "" {
		meta["routing"] = item.Routing
	}
	if item.Pipeline != "" {
		meta["pipeline"] = item.Pipeline
	}

	action := map[string]interface{}{
		item.Action: meta,
	}

	// 写入操作行
	actionBytes, err := json.Marshal(action)
	if err != nil {
		return nil, err
	}
	buf.Write(actionBytes)
	buf.WriteByte('\n')

	// 如果不是 delete 操作，写入文档内容
	if item.Action != "delete" && item.Document != nil {
		docBytes, err := json.Marshal(item.Document)
		if err != nil {
			return nil, err
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// Flush 立即刷新缓冲区
func (bi *bulkIndexer) Flush(ctx context.Context) error {
	if bi.closed.Load() {
		return nil
	}

	// 获取并清空缓冲区
	bi.bufferMu.Lock()
	if len(bi.buffer) == 0 {
		bi.bufferMu.Unlock()
		return nil
	}

	items := make([]BulkItem, len(bi.buffer))
	copy(items, bi.buffer)
	bi.buffer = bi.buffer[:0]
	bi.bufferMu.Unlock()

	// 重置计数器
	bi.bufferLen.Store(0)
	bi.currentBytes.Store(0)

	// 执行批量写入
	return bi.executeBulk(ctx, items)
}

// executeBulk 执行批量写入
func (bi *bulkIndexer) executeBulk(ctx context.Context, items []BulkItem) error {
	if len(items) == 0 {
		return nil
	}

	// 构建请求体
	var buf bytes.Buffer
	for _, item := range items {
		itemBytes, err := bi.marshalItem(item)
		if err != nil {
			bi.stats.numFailed.Add(1)
			if bi.config.OnError != nil {
				bi.config.OnError(fmt.Errorf("marshal item: %w", err))
			}
			continue
		}
		buf.Write(itemBytes)
	}

	if buf.Len() == 0 {
		return nil
	}

	// 执行请求（带重试）
	var lastErr error
	for attempt := 0; attempt <= bi.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := 100 * time.Millisecond * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := bi.client.Bulk(ctx, bytes.NewReader(buf.Bytes()))
		bi.stats.numRequests.Add(1)

		if err != nil {
			lastErr = err
			if !IsRetryable(err) {
				break
			}
			continue
		}

		// 统计成功/失败
		bi.stats.bytesFlushed.Add(int64(buf.Len()))

		if resp.Errors {
			// 处理部分失败
			failedItems := resp.FailedItems()
			bi.stats.numFailed.Add(int64(len(failedItems)))
			bi.stats.numFlushed.Add(int64(len(items) - len(failedItems)))

			if bi.config.OnError != nil {
				bi.config.OnError(&BulkErrors{Errors: failedItems})
			}
		} else {
			bi.stats.numFlushed.Add(int64(len(items)))
		}

		// 成功回调
		if bi.config.OnSuccess != nil {
			bi.config.OnSuccess(bi.Stats())
		}

		return nil
	}

	bi.stats.numFailed.Add(int64(len(items)))
	return fmt.Errorf("bulk request failed after %d retries: %w", bi.config.MaxRetries, lastErr)
}

// flushLoop 定时刷新循环
func (bi *bulkIndexer) flushLoop() {
	defer bi.wg.Done()

	for {
		select {
		case <-bi.ticker.C:
			if bi.bufferLen.Load() > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := bi.Flush(ctx); err != nil && bi.config.OnError != nil {
					bi.config.OnError(err)
				}
				cancel()
			}
		case <-bi.flushCh:
			// 触发刷新信号（由 Add 发出）
		case <-bi.stopCh:
			return
		}
	}
}

// Stats 获取统计信息
func (bi *bulkIndexer) Stats() BulkStats {
	return BulkStats{
		NumAdded:     bi.stats.numAdded.Load(),
		NumFlushed:   bi.stats.numFlushed.Load(),
		NumFailed:    bi.stats.numFailed.Load(),
		NumRequests:  bi.stats.numRequests.Load(),
		BytesFlushed: bi.stats.bytesFlushed.Load(),
	}
}

// Close 关闭索引器
func (bi *bulkIndexer) Close(ctx context.Context) error {
	var closeErr error

	bi.closeOnce.Do(func() {
		bi.closed.Store(true)

		// 停止定时器
		if bi.ticker != nil {
			bi.ticker.Stop()
		}

		// 发送停止信号
		close(bi.stopCh)

		// 刷新剩余数据
		if bi.bufferLen.Load() > 0 {
			closeErr = bi.Flush(ctx)
		}

		// 等待所有 worker 完成
		done := make(chan struct{})
		go func() {
			bi.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			closeErr = ctx.Err()
		}
	})

	return closeErr
}
