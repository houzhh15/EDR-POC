# Event Processing Pipeline Service

## 概述

事件处理管线服务（Event Processing Pipeline Service）是 EDR 云端的核心组件，负责：
- 从 Kafka 消费原始事件
- 事件富化（GeoIP、资产信息、Agent 元数据）
- ECS 8.11.0 标准化
- 输出到 Kafka 和 OpenSearch

## 架构

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Kafka     │────▶│ BatchCollector  │────▶│ BatchProcessor  │
│ (raw events)│     │   (batching)    │     │ (enrichment +   │
└─────────────┘     └─────────────────┘     │  normalization) │
                                            └────────┬────────┘
                                                     │
                    ┌────────────────────────────────┼────────────────────────────────┐
                    ▼                                ▼                                ▼
            ┌─────────────┐                  ┌─────────────┐                  ┌─────────────┐
            │   Kafka     │                  │ OpenSearch  │                  │    DLQ      │
            │ (normalized)│                  │  (storage)  │                  │  (errors)   │
            └─────────────┘                  └─────────────┘                  └─────────────┘
```

## 组件说明

### 1. BatchCollector (批次收集器)
- 收集从 Kafka 消费的事件
- 按批次大小或超时触发处理
- 线程安全的缓冲区管理

### 2. BatchProcessor (批处理器)
- 支持并行处理（多工作协程）
- 富化器链式调用
- ECS 标准化转换
- 错误隔离，单个事件失败不影响批次

### 3. Enrichers (富化器)
- **GeoIPEnricher**: IP 地理位置信息
- **AssetEnricher**: 资产元数据（主机名、OS、部门）
- **AgentEnricher**: Agent 版本和配置信息

### 4. Normalizer (标准化器)
- 支持多种事件类型：
  - process_create / process_terminate
  - file_create / file_modify / file_delete
  - network_connect / network_disconnect
  - dns_query
- 输出符合 ECS 8.11.0 标准

### 5. Writers (输出写入器)
- **KafkaWriter**: Kafka 生产者，支持批量写入和压缩
- **OpenSearchWriter**: Bulk API 写入，支持索引轮转

## 配置

配置文件位于 `configs/pipeline.yaml`：

```yaml
input:
  kafka:
    brokers:
      - localhost:19092
    topic: edr.events.raw
    consumer_group: pipeline-processor
    concurrency: 10

processing:
  batch_size: 1000
  batch_timeout: 100ms
  worker_count: 10

enrichment:
  geoip:
    enabled: true
    database_path: /data/GeoLite2-City.mmdb
  asset:
    enabled: true
    cache_ttl: 5m
  agent:
    enabled: true
    cache_ttl: 5m

output:
  kafka:
    enabled: true
    brokers:
      - localhost:19092
    topic: edr.events.normalized
  opensearch:
    enabled: true
    addresses:
      - http://localhost:9200
    index_prefix: edr-events
    bulk_size: 1000

error_handling:
  max_retries: 3
  retry_backoff: 100ms
  dlq_topic: edr.dlq
```

## 运行

### 编译
```bash
cd cloud
go build -o bin/event-processor ./cmd/event-processor
```

### 启动
```bash
./bin/event-processor -config configs/pipeline.yaml -metrics :9091
```

### 健康检查
```bash
curl http://localhost:9091/health
curl http://localhost:9091/ready
```

### 指标
```bash
curl http://localhost:9091/metrics
```

## 性能指标

- 批处理吞吐量: >100万 events/sec (并行模式)
- 单事件标准化延迟: <100μs
- 批次处理延迟: <10ms (1000 events)

### 关键 Prometheus 指标

| 指标名 | 类型 | 描述 |
|--------|------|------|
| edr_pipeline_events_consumed_total | Counter | 消费事件总数 |
| edr_pipeline_events_processed_total | Counter | 处理事件总数 |
| edr_pipeline_events_written_total | Counter | 写入事件总数 |
| edr_pipeline_processing_duration_seconds | Histogram | 处理延迟分布 |
| edr_pipeline_batch_size | Histogram | 批次大小分布 |
| edr_pipeline_errors_total | Counter | 错误计数 |
| edr_pipeline_dlq_messages_total | Counter | DLQ 消息数 |

## ECS 事件格式

输出事件符合 [Elastic Common Schema 8.11.0](https://www.elastic.co/guide/en/ecs/8.11/index.html)：

```json
{
  "@timestamp": "2024-01-15T10:30:00.000Z",
  "ecs": {
    "version": "8.11.0"
  },
  "event": {
    "id": "evt-001",
    "kind": "event",
    "category": ["process"],
    "type": ["start"],
    "created": "2024-01-15T10:30:00.000Z"
  },
  "host": {
    "hostname": "workstation-001",
    "os": {
      "family": "windows",
      "version": "10.0.19041"
    }
  },
  "process": {
    "pid": 1234,
    "parent": {
      "pid": 5678
    },
    "name": "notepad.exe",
    "executable": "C:\\Windows\\notepad.exe",
    "command_line": "notepad.exe test.txt"
  },
  "agent": {
    "id": "agent-001",
    "version": "1.0.0"
  }
}
```

## 错误处理

- 解析失败的事件发送到 DLQ
- 标准化失败的事件发送到 DLQ
- 写入失败支持指数退避重试
- 所有错误都有详细的指标记录

## 测试

```bash
# 单元测试
go test ./internal/pipeline/... -v

# 集成测试
go test ./tests/pipeline/... -v

# 性能测试
go test ./tests/pipeline/... -bench=. -benchmem
```

## 依赖

- github.com/segmentio/kafka-go - Kafka 客户端
- github.com/oschwald/geoip2-golang - GeoIP 数据库
- github.com/prometheus/client_golang - Prometheus 指标
- go.uber.org/zap - 结构化日志
- gopkg.in/yaml.v3 - YAML 配置解析
