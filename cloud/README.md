# EDR Cloud Services

EDR 云端服务模块，采用微服务架构，负责事件处理、检测分析和告警管理。

## 📋 服务列表

| 服务 | 端口 | 说明 |
|------|------|------|
| API Gateway | 8080 | REST API / gRPC 网关 |
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
  http_port: 8080
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
