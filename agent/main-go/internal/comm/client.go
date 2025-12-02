package comm

import (
	"context"
	"sync"
	"time"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
)

// EventClient 事件上报客户端
type EventClient struct {
	conn       *Connection
	batchSize  int
	flushTimer time.Duration
	mu         sync.Mutex
	batch      []cgo.Event
}

// NewEventClient 创建事件客户端
func NewEventClient(conn *Connection, batchSize int, flushTimer time.Duration) *EventClient {
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushTimer <= 0 {
		flushTimer = 5 * time.Second
	}
	return &EventClient{
		conn:       conn,
		batchSize:  batchSize,
		flushTimer: flushTimer,
		batch:      make([]cgo.Event, 0, batchSize),
	}
}

// SendEvent 发送单个事件（缓冲后批量发送）
func (c *EventClient) SendEvent(event cgo.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, event)

	// 达到批量大小时立即发送
	if len(c.batch) >= c.batchSize {
		return c.flush()
	}

	return nil
}

// flush 刷新发送缓冲的事件
func (c *EventClient) flush() error {
	if len(c.batch) == 0 {
		return nil
	}

	// TODO: 调用 gRPC 服务发送事件
	// 当前为占位实现，后续实现 proto 定义后替换
	// pb.NewEventServiceClient(c.conn.GetConn()).ReportEvents(ctx, &pb.ReportEventsRequest{...})

	// 清空缓冲
	c.batch = c.batch[:0]

	return nil
}

// Flush 手动刷新缓冲区
func (c *EventClient) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.flush()
}

// StartBatchSender 启动批量发送 goroutine
// 从 eventChan 读取事件并批量发送
func (c *EventClient) StartBatchSender(ctx context.Context, eventChan <-chan cgo.Event) {
	ticker := time.NewTicker(c.flushTimer)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 退出前刷新剩余事件
			c.Flush()
			return

		case event, ok := <-eventChan:
			if !ok {
				c.Flush()
				return
			}
			c.SendEvent(event)

		case <-ticker.C:
			// 定时刷新
			c.Flush()
		}
	}
}
