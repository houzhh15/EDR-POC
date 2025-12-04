package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
	"github.com/houzhh15/EDR-POC/agent/main-go/internal/log"
	pb "github.com/houzhh15/EDR-POC/agent/main-go/pkg/proto/edr/v1"
)

// EventCache 事件缓存接口（失败时存储事件）
type EventCache interface {
	Store(events []cgo.Event) error
	Load(limit int) ([]cgo.Event, error)
}

// EventClientOption 事件客户端配置选项
type EventClientOption func(*EventClient)

// WithEventCache 设置事件缓存
func WithEventCache(cache EventCache) EventClientOption {
	return func(c *EventClient) {
		c.cache = cache
	}
}

// WithLogger 设置日志器
func WithLogger(logger *log.Logger) EventClientOption {
	return func(c *EventClient) {
		c.logger = logger
	}
}

// EventClient 事件上报客户端
type EventClient struct {
	conn       *Connection
	agentID    string
	batchSize  int
	flushTimer time.Duration
	mu         sync.Mutex
	batch      []cgo.Event
	cache      EventCache  // 事件缓存（可选）
	logger     *log.Logger // 日志器

	// 序列号计数器（用于保证顺序）
	sequenceNum uint64
}

// NewEventClient 创建事件客户端
func NewEventClient(conn *Connection, agentID string, batchSize int, flushTimer time.Duration, opts ...EventClientOption) *EventClient {
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushTimer <= 0 {
		flushTimer = 5 * time.Second
	}
	c := &EventClient{
		conn:       conn,
		agentID:    agentID,
		batchSize:  batchSize,
		flushTimer: flushTimer,
		batch:      make([]cgo.Event, 0, batchSize),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
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

	// 检查连接是否可用
	grpcConn := c.conn.GetConn()
	if grpcConn == nil {
		// 连接不可用，尝试缓存事件
		if c.cache != nil {
			if err := c.cache.Store(c.batch); err != nil && c.logger != nil {
				c.logger.Warn("Failed to cache events", zap.Error(err))
			}
		}
		c.batch = c.batch[:0]
		return fmt.Errorf("connection not available")
	}

	// 创建 gRPC 客户端
	client := pb.NewAgentServiceClient(grpcConn)

	// 创建流
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.ReportEvents(ctx)
	if err != nil {
		// 发送失败，尝试缓存
		if c.cache != nil {
			if cacheErr := c.cache.Store(c.batch); cacheErr != nil && c.logger != nil {
				c.logger.Warn("Failed to cache events", zap.Error(cacheErr))
			}
		}
		c.batch = c.batch[:0]
		return fmt.Errorf("create stream: %w", err)
	}

	// 获取序列号
	seqNum := atomic.AddUint64(&c.sequenceNum, 1)

	// 转换事件为 protobuf 格式
	pbEvents := make([]*pb.SecurityEvent, 0, len(c.batch))
	for _, event := range c.batch {
		pbEvent := c.convertToProtoEvent(event)
		pbEvents = append(pbEvents, pbEvent)
	}

	// 构造 EventBatch 并发送
	eventBatch := &pb.EventBatch{
		AgentId:        c.agentID,
		BatchId:        fmt.Sprintf("batch-%d-%d", time.Now().UnixNano(), seqNum),
		SequenceNumber: int32(seqNum),
		Events:         pbEvents,
		BatchTime:      timestamppb.Now(),
	}

	if err := stream.Send(eventBatch); err != nil {
		// 发送失败，尝试缓存
		if c.cache != nil {
			if cacheErr := c.cache.Store(c.batch); cacheErr != nil && c.logger != nil {
				c.logger.Warn("Failed to cache events", zap.Error(cacheErr))
			}
		}
		c.batch = c.batch[:0]
		return fmt.Errorf("send batch: %w", err)
	}

	// 关闭流并获取响应
	resp, err := stream.CloseAndRecv()
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("Failed to receive response", zap.Error(err))
		}
		c.batch = c.batch[:0]
		return fmt.Errorf("close and recv: %w", err)
	}

	// 处理响应
	if c.logger != nil {
		c.logger.Debug("Events sent successfully",
			zap.Int32("received", resp.EventsReceived),
			zap.Int32("accepted", resp.EventsAccepted),
			zap.Int("rejected_count", len(resp.RejectedEventIds)),
		)
		if len(resp.RejectedEventIds) > 0 {
			c.logger.Warn("Some events were rejected",
				zap.Strings("rejected_ids", resp.RejectedEventIds),
			)
		}
	}

	// 清空缓冲
	c.batch = c.batch[:0]

	return nil
}

// convertToProtoEvent 将内部事件转换为 protobuf 格式
func (c *EventClient) convertToProtoEvent(event cgo.Event) *pb.SecurityEvent {
	// 解析事件 JSON 数据
	var eventData map[string]interface{}
	if len(event.Data) > 0 {
		_ = json.Unmarshal(event.Data, &eventData)
	}

	// 构建 ECS 字段
	ecsFields := make(map[string]string)
	for k, v := range eventData {
		if strVal, ok := v.(string); ok {
			ecsFields[k] = strVal
		} else if v != nil {
			// 将非字符串值序列化为 JSON
			if b, err := json.Marshal(v); err == nil {
				ecsFields[k] = string(b)
			}
		}
	}

	// 生成事件 ID
	eventID := uuid.New().String()

	// 从时间戳转换
	eventTime := time.UnixMilli(event.Timestamp)

	return &pb.SecurityEvent{
		EventId:   eventID,
		EventType: mapEventType(event.Type),
		Timestamp: timestamppb.New(eventTime),
		Severity:  1, // 默认 LOW 级别
		EcsFields: ecsFields,
		RawData:   event.Data,
	}
}

// mapEventType 映射事件类型到字符串
func mapEventType(eventType uint32) string {
	switch eventType {
	case 1:
		return "process_create"
	case 2:
		return "process_exit"
	case 3:
		return "file_create"
	case 4:
		return "file_modify"
	case 5:
		return "file_delete"
	case 6:
		return "network_connect"
	default:
		return "unknown"
	}
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
