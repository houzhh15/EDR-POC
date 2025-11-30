# EDR Agent Go Module

EDR Agent 的 Go 主程序模块，负责业务逻辑、策略管理和与云端通信。

## 📋 模块结构

```
main-go/
├── cmd/agent/         # 入口程序
│   └── main.go
├── internal/          # 内部实现
│   ├── cgo/           # CGO 封装 (调用 C 核心库)
│   ├── comm/          # gRPC 通信
│   ├── policy/        # 策略管理
│   ├── config/        # 配置管理
│   └── log/           # 日志封装
├── pkg/               # 可导出包
├── go.mod
└── go.sum
```

## 🔧 依赖说明

| 依赖 | 版本 | 用途 |
|------|------|------|
| google.golang.org/grpc | v1.60.0 | gRPC 通信 |
| google.golang.org/protobuf | v1.32.0 | Protobuf 序列化 |
| go.uber.org/zap | v1.26.0 | 结构化日志 |
| github.com/spf13/viper | v1.18.0 | 配置管理 |
| github.com/fsnotify/fsnotify | v1.7.0 | 文件监控 |

## 🏗️ 编译方式

### 前置条件

1. 先编译 C 核心库：
   ```bash
   cd ../core-c
   mkdir build && cd build
   cmake .. -DCMAKE_BUILD_TYPE=Release
   make
   ```

2. 设置 CGO 环境：
   ```bash
   export CGO_ENABLED=1
   export CGO_LDFLAGS="-L../core-c/build -ledr_core"
   ```

### 编译

```bash
# 在 main-go 目录
go build -o edr-agent ./cmd/agent

# 带版本信息编译
go build -ldflags "-X main.Version=0.1.0 -X main.GitCommit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o edr-agent ./cmd/agent
```

### 使用根目录 Makefile

```bash
# 在项目根目录
make build-agent-go
```

## 🚀 运行

```bash
# 设置库路径
export LD_LIBRARY_PATH=../core-c/build:$LD_LIBRARY_PATH  # Linux
export DYLD_LIBRARY_PATH=../core-c/build:$DYLD_LIBRARY_PATH  # macOS

# 运行
./edr-agent

# 带配置文件运行
./edr-agent --config /etc/edr/agent.yaml
```

## 📦 配置示例

```yaml
# agent.yaml
server:
  endpoint: "cloud.edr.example.com:443"
  tls:
    enabled: true
    cert_file: "/etc/edr/certs/agent.crt"
    key_file: "/etc/edr/certs/agent.key"

collector:
  enabled: true
  batch_size: 100
  flush_interval: 5s

detector:
  yara_rules_path: "/etc/edr/rules/yara"
  sigma_rules_path: "/etc/edr/rules/sigma"

log:
  level: "info"
  format: "json"
  output: "/var/log/edr/agent.log"
```

## 📝 开发指南

### 添加新的内部模块

1. 在 `internal/` 下创建目录
2. 包名使用小写
3. 遵循 Go 代码规范

### 测试

```bash
go test ./...
```

### 代码检查

```bash
golangci-lint run
```
