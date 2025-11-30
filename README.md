# EDR Platform

[![Build](https://github.com/edr-project/edr-platform/actions/workflows/build.yml/badge.svg)](https://github.com/edr-project/edr-platform/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**EDR (Endpoint Detection and Response) Platform** 是一个开源的终端检测与响应平台，用于实时监控和保护企业终端安全。

## 📋 功能特性

- **终端采集**：支持 Windows (ETW)、Linux (eBPF)、macOS (Endpoint Security) 多平台事件采集
- **实时检测**：基于 YARA 和 Sigma 规则的威胁检测引擎
- **云端分析**：高性能事件处理与关联分析
- **响应处置**：远程命令执行、进程隔离、文件隔离等响应能力
- **管理控制台**：直观的 Web 界面，支持告警管理、资产管理、策略配置

## 🏗️ 项目结构

```
edr-platform/
├── agent/                # 终端 Agent
│   ├── core-c/           # C 核心库 (采集、检测)
│   ├── main-go/          # Go 主程序 (业务逻辑)
│   └── agent-rust/       # Rust 备选方案 (占位)
├── cloud/                # 云端服务
│   ├── cmd/              # 服务入口
│   └── internal/         # 内部实现
├── console/              # Web 管理控制台 (React + TypeScript)
├── proto/                # Protobuf 接口定义
├── deploy/               # 部署配置
├── scripts/              # 工具脚本
└── docs/                 # 文档
```

## 🚀 快速开始

### 环境要求

| 工具 | 版本 |
|------|------|
| Go | 1.21+ |
| Node.js | 18+ LTS |
| pnpm | 8+ |
| CMake | 3.20+ |
| Docker | 24+ |

### 1. 克隆仓库

```bash
git clone https://github.com/edr-project/edr-platform.git
cd edr-platform
```

### 2. 环境检查

```bash
./scripts/setup.sh
```

### 3. 启动开发环境

```bash
# 启动依赖服务 (Kafka, PostgreSQL, Redis, OpenSearch 等)
make dev-up

# 等待服务就绪
./scripts/health-check.sh
```

### 4. 构建项目

```bash
# 构建所有模块
make build

# 或单独构建
make build-agent    # 构建 Agent
make build-cloud    # 构建 Cloud 服务
make build-console  # 构建控制台
```

### 5. 运行测试

```bash
make test
```

## 🔧 常用命令

| 命令 | 说明 |
|------|------|
| `make build` | 构建所有模块 |
| `make test` | 运行所有测试 |
| `make lint` | 代码检查 |
| `make dev-up` | 启动开发环境 |
| `make dev-down` | 停止开发环境 |
| `make dev-logs` | 查看容器日志 |
| `make help` | 显示帮助信息 |

## 📦 技术栈

| 组件 | 技术 |
|------|------|
| Agent 核心 | C11 + eBPF/ETW |
| Agent 主程序 | Go 1.21 |
| Cloud 服务 | Go 1.21 + Gin |
| 控制台 | React 18 + TypeScript + Vite |
| 消息队列 | Apache Kafka |
| 事件存储 | OpenSearch |
| 配置存储 | PostgreSQL |
| 缓存 | Redis |
| 对象存储 | MinIO |
| 链路追踪 | Jaeger |

## 📄 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可证。

## 🤝 贡献指南

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feat/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feat/amazing-feature`)
5. 创建 Pull Request

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档
- `style`: 格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 其他

## 📞 联系我们

- Issue: [GitHub Issues](https://github.com/edr-project/edr-platform/issues)
- Email: edr-team@example.com
