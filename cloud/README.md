# EDR Cloud Services

EDR 云端服务模块，采用微服务架构，负责事件处理、检测分析和告警管理。

## 📋 服务列表

| 服务 | 端口 | 说明 |
|------|------|------|
| API Gateway | 9080 | REST API / gRPC 网关 |
| Event Processor | - | 事件消费和存储 |
| Detection Engine | - | 规则检测引擎 |
| Alert Manager | - | 告警管理 |

## 🏗️ 模块结构

```
cloud/
├── cmd/                       # 服务入口
│   ├── api-gateway/
│   ├── event-processor/
│   ├── detection-engine/
│   └── alert-manager/
├── internal/                  # 内部实现
│   ├── event/                 # 事件处理
│   ├── detection/             # 检测逻辑
│   ├── alert/                 # 告警管理
│   ├── asset/                 # 资产管理
│   ├── policy/                # 策略管理
│   └── storage/               # 存储抽象
├── pkg/                       # 公共库
│   ├── middleware/            # HTTP 中间件
│   ├── auth/                  # 认证鉴权
│   └── utils/                 # 工具函数
├── go.mod
└── README.md
```

## 🔧 依赖说明

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/gin-gonic/gin | v1.9.1 | HTTP 框架 |
| github.com/segmentio/kafka-go | v0.4.45 | Kafka 客户端 |
| go.uber.org/zap | v1.26.0 | 结构化日志 |
| gorm.io/gorm | v1.25.5 | ORM |
| gorm.io/driver/postgres | v1.5.4 | PostgreSQL 驱动 |

## 🚀 启动方式

### 1. 启动依赖服务

```bash
# 在项目根目录
make dev-up
```

### 2. 编译服务

```bash
# 编译所有服务
make build-cloud

# 或单独编译
cd cloud
go build -o ../build/bin/api-gateway ./cmd/api-gateway
go build -o ../build/bin/event-processor ./cmd/event-processor
go build -o ../build/bin/detection-engine ./cmd/detection-engine
go build -o ../build/bin/alert-manager ./cmd/alert-manager
```

### 3. 运行服务

```bash
# 运行 API Gateway
./build/bin/api-gateway

# 运行 Event Processor
./build/bin/event-processor

# 运行 Detection Engine
./build/bin/detection-engine

# 运行 Alert Manager
./build/bin/alert-manager
```

## 📦 配置示例

```yaml
# config.yaml
server:
  http_port: 9080
  grpc_port: 9090

database:
  host: localhost
  port: 5432
  user: edr
  password: ${POSTGRES_PASSWORD}
  database: edr

kafka:
  brokers:
    - localhost:9092
  topics:
    events: edr-events
    alerts: edr-alerts

opensearch:
  addresses:
    - http://localhost:9200

redis:
  address: localhost:6379

log:
  level: info
  format: json
```

## 🔌 Kafka 组件

### 概述

云端服务使用 Kafka 进行事件流处理，主要组件包括：

- **Producer**: 生产消息到 Kafka Topic
- **Consumer**: 消费 Kafka 消息并处理
- **TopicManager**: 管理 Topic 生命周期
- **DeadLetterQueue (DLQ)**: 处理失败消息
- **HealthChecker**: 检查 Kafka 集群健康状态

### Topic 设计

| Topic | 用途 | 分区数 | 保留期 |
|-------|------|--------|--------|
| `edr.events.raw` | 原始事件 | 12 | 7天 |
| `edr.events.normalized` | 标准化事件 | 12 | 7天 |
| `edr.alerts` | 告警事件 | 6 | 30天 |
| `edr.commands` | 响应命令 | 6 | 3天 |
| `edr.dlq` | 死信队列 | 3 | 14天 |

### 代码示例

#### 生产者使用

```go
import "github.com/houzhh15/EDR-POC/cloud/internal/event"

// 创建生产者
producer, err := event.NewKafkaProducer(
    "localhost:19092",
    "edr.events.raw",
    logger,
)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// 设置 Prometheus 指标
metrics := event.NewProducerMetrics("edr")
producer.SetMetrics(metrics)

// 发送消息
msg := &event.EventMessage{
    AgentID:    "agent-001",
    TenantID:   "tenant-001",
    BatchID:    "batch-001",
    Timestamp:  time.Now(),
    ReceivedAt: time.Now(),
    Events: []*event.SecurityEvent{
        {
            EventID:   "evt-001",
            EventType: "process_start",
            Timestamp: time.Now(),
            Severity:  2,
        },
    },
}

err = producer.ProduceBatch(ctx, []*event.EventMessage{msg})
```

#### 消费者使用

```go
import "github.com/houzhh15/EDR-POC/cloud/internal/event"

// 创建消费者
consumer, err := event.NewKafkaConsumer(
    []string{"localhost:19092"},
    "event-processor-group",
    "edr.events.raw",
    logger,
)
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// 使用 Handler 模式消费
handler := func(ctx context.Context, msgs []*event.EventMessage) error {
    for _, msg := range msgs {
        // 处理消息
        log.Printf("Received event from agent: %s", msg.AgentID)
    }
    return nil
}

err = consumer.ConsumeWithHandler(ctx, handler)
```

#### Topic 管理

```go
import "github.com/houzhh15/EDR-POC/cloud/internal/event"

// 创建 Topic 管理器
tm := event.NewTopicManager([]string{"localhost:19092"}, logger)

// 确保 Topic 存在
topics := []event.TopicDefinition{
    {Name: "edr.events.raw", Partitions: 12, ReplicationFactor: 1},
    {Name: "edr.alerts", Partitions: 6, ReplicationFactor: 1},
}
err := tm.EnsureTopics(ctx, topics)

// 列出所有 Topic
existing, err := tm.ListTopics(ctx)
```

#### DLQ 使用

```go
import "github.com/houzhh15/EDR-POC/cloud/internal/event"

// 创建 DLQ
dlqProducer, _ := event.NewKafkaProducer("localhost:19092", "edr.dlq", logger)
dlq, err := event.NewDeadLetterQueue(dlqProducer, &event.DeadLetterQueueConfig{
    Enabled:      true,
    Topic:        "edr.dlq",
    MaxRetries:   3,
    RetryBackoff: time.Second,
}, logger)

// 路由失败消息到 DLQ
dlqMsg := event.CreateDeadLetterMessage(
    "edr.events.raw",       // 原始 Topic
    "agent-001",            // Key
    originalEvent,          // 原始事件
    err,                    // 错误信息
    "deserialization_error",// 错误类型
    "consumer",             // 来源
    0,                      // 重试次数
)
err = dlq.Route(ctx, dlqMsg)
```

#### 健康检查

```go
import "github.com/houzhh15/EDR-POC/cloud/internal/event"

// 创建健康检查器
hc := event.NewHealthChecker([]string{"localhost:19092"}, 5*time.Second, logger)

// 检查 Broker 健康
status := hc.Check(ctx)
if status.Healthy {
    log.Printf("Kafka healthy, latency: %s", status.Duration)
} else {
    log.Printf("Kafka unhealthy: %s", status.Error)
}

// 检查 Topic 健康
status = hc.CheckWithTopics(ctx, []string{"edr.events.raw", "edr.alerts"})
```

### Prometheus 指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `edr_kafka_producer_messages_total` | Counter | 生产消息总数 |
| `edr_kafka_producer_bytes_total` | Counter | 生产字节总数 |
| `edr_kafka_producer_errors_total` | Counter | 生产错误总数 |
| `edr_kafka_producer_latency_seconds` | Histogram | 生产延迟 |
| `edr_kafka_consumer_messages_total` | Counter | 消费消息总数 |
| `edr_kafka_consumer_bytes_total` | Counter | 消费字节总数 |
| `edr_kafka_consumer_lag` | Gauge | 消费延迟 |
| `edr_kafka_consumer_errors_total` | Counter | 消费错误总数 |
| `edr_kafka_dlq_messages_total` | Counter | DLQ 消息总数 |
| `edr_kafka_health_check_status` | Gauge | 健康检查状态 |
| `edr_kafka_health_brokers_up` | Gauge | 健康 Broker 数 |

### 配置文件

完整 Kafka 配置示例 (`configs/kafka.yaml`)：

```yaml
kafka:
  brokers:
    - localhost:19092
  
  producer:
    batch_size: 100
    batch_timeout: 100ms
    max_attempts: 3
    compression: snappy
    required_acks: -1  # all replicas
  
  consumer:
    min_bytes: 10KB
    max_bytes: 10MB
    max_wait: 500ms
    commit_interval: 1s
    start_offset: earliest
  
  topics:
    events_raw:
      name: edr.events.raw
      partitions: 12
      replication_factor: 1
      retention: 168h  # 7 days
    
    events_normalized:
      name: edr.events.normalized
      partitions: 12
      replication_factor: 1
      retention: 168h
    
    alerts:
      name: edr.alerts
      partitions: 6
      replication_factor: 1
      retention: 720h  # 30 days
    
    commands:
      name: edr.commands
      partitions: 6
      replication_factor: 1
      retention: 72h  # 3 days
    
    dlq:
      name: edr.dlq
      partitions: 3
      replication_factor: 1
      retention: 336h  # 14 days
  
  dlq:
    enabled: true
    topic: edr.dlq
    max_retries: 3
    retry_backoff: 1s
  
  health:
    check_interval: 30s
    timeout: 5s
```

## 📊 服务架构

```
                    ┌─────────────┐
                    │   Console   │
                    └─────┬───────┘
                          │ REST API
                    ┌─────▼───────┐
                    │ API Gateway │◄──────────┐
                    └─────┬───────┘           │ gRPC
                          │                   │
         ┌────────────────┼────────────────┐ │
         │                │                │ │
    ┌────▼────┐     ┌────▼─────┐    ┌────▼─┴──┐
    │  Event  │     │Detection │    │  Alert  │
    │Processor│     │ Engine   │    │ Manager │
    └────┬────┘     └────┬─────┘    └────┬────┘
         │               │               │
    ┌────▼───────────────▼───────────────▼────┐
    │                 Kafka                    │
    └──────────────────────────────────────────┘
         │               │               │
    ┌────▼────┐    ┌────▼────┐     ┌────▼────┐
    │OpenSearch│   │PostgreSQL│    │  Redis  │
    └─────────┘    └─────────┘     └─────────┘
```

## 📝 开发指南

### 添加新服务

1. 在 `cmd/` 下创建服务目录
2. 实现 `main.go` 入口
3. 在 `internal/` 下添加业务逻辑
4. 更新 Makefile 构建目标
5. 更新 Docker Compose 配置

### 测试

```bash
go test ./...
```

### 代码检查

```bash
golangci-lint run
```
