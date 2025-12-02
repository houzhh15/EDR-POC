# EDR-POC 项目关键上下文

> 本文档为AI大模型执行任务的贯穿上下文，包含项目核心约束、技术选型、架构边界。
> 任何任务执行前应先阅读本文档，确保与项目整体设计一致。

---

## 一、项目定位

**EDR（终端检测与响应）商业友好开源技术栈方案**

- **目标**: 构建可私有化部署的EDR平台，支持SASE/ZTNA集成
- **许可要求**: 核心代码可闭源商用，无GPL污染
- **规模目标**: Phase1(100终端) → Phase2(1000终端) → Phase3(10000+终端)

---

## 二、技术栈约束 (⚠️强制)

### 2.1 许可证规则

| 状态 | 许可证 | 说明 |
|------|--------|------|
| ✅ 允许 | Apache 2.0, MIT, BSD, Public Domain | 优先选择 |
| ⚠️ 受限 | LGPL 2.1/3.0 | **仅允许动态链接** (如libbpf) |
| ❌ 禁止 | GPL, AGPL, SSPL | 任何形式均禁止 |

### 2.2 语言与架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Agent 架构                           │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │   C 核心库      │    │         Go 主程序               │ │
│  │ (采集/检测)     │◄──►│  (通信/管理/业务逻辑)           │ │
│  │ ETW/eBPF/ES    │CGO │  gRPC Client, 配置管理          │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

| 层级 | 语言 | 说明 |
|------|------|------|
| Agent核心 | **C** | 采集器(ETW/eBPF/ES)、检测引擎、平台API |
| Agent主程序 | **Go** | 业务逻辑、通信、配置管理，通过CGO调用C |
| Cloud | **Go** | 全部后端服务 |
| Console | **TypeScript + React 18** | Vite构建, Ant Design 5, Zustand状态管理 |

### 2.3 平台事件采集技术

| 平台 | 采集技术 | 关键API |
|------|----------|---------|
| **Windows** | ETW (Event Tracing for Windows) | StartTrace, ProcessTrace |
| **Linux** | eBPF + libbpf (**动态链接**) | bpf_prog_load, ring_buffer |
| **macOS** | Endpoint Security Framework | es_new_client, NEFilterDataProvider |

### 2.4 存储层

| 用途 | 技术 | 许可证 |
|------|------|--------|
| 配置/元数据/用户 | PostgreSQL | PostgreSQL License |
| 事件日志/全文检索 | OpenSearch | Apache 2.0 |
| OLAP统计分析 | ClickHouse | Apache 2.0 |
| 缓存/IOC匹配 | Redis | BSD |
| Agent本地缓存 | SQLite | Public Domain |
| 对象存储(取证) | MinIO | Apache 2.0 |

### 2.5 消息与通信

| 组件 | 技术 | 说明 |
|------|------|------|
| 消息队列 | Apache Kafka | 分区按agent_id哈希 |
| Agent-Cloud通信 | gRPC + Protobuf | TLS 1.3 + mTLS双向认证 |
| 事件格式 | **ECS (Elastic Common Schema)** | 统一三平台事件结构 |

---

## 三、检测引擎

| 引擎 | 用途 | 许可证 |
|------|------|--------|
| **YARA** | 文件/进程特征匹配 | BSD 3-Clause |
| **Sigma** | 行为规则检测 | LGPL 2.1 (规则文本) |
| **IOC** | 威胁指标匹配 | - |
| 行为规则 | 自研攻击模式检测 | - |

---

## 四、核心模块边界

### 4.1 Agent模块 (任务02-11, 33, 39, 47-53)

- **core-c/**: C核心库 - 采集器、检测引擎、平台抽象
- **main-go/**: Go主程序 - 通信、策略、生命周期管理
- **平台隔离**: 采集代码按平台分离 (windows/, linux/, macos/)

### 4.2 Cloud模块 (任务12-23, 31-32, 34-38, 40-46)

- **事件处理链**: Kafka接收 → 标准化 → 检测引擎 → 告警生成
- **API层**: gRPC(Agent) + REST(Console)
- **多租户**: PostgreSQL RLS行级安全隔离

### 4.3 Console模块 (任务24-30)

- **技术栈**: React 18 + TypeScript + Vite + Ant Design 5
- **状态管理**: Zustand (轻量级)
- **路由**: React Router 6

---

## 五、关键设计决策

### 5.1 为什么选择C+Go混合架构（而非Rust）

| 因素 | C+Go | Rust |
|------|------|------|
| 内核API绑定 | 原生支持 | 需要unsafe FFI |
| ETW/eBPF集成 | 成熟生态 | 生态较新 |
| 开发效率 | Go部分高效 | 学习曲线陡峭 |
| 招聘难度 | 较易 | 较难 |

### 5.2 eBPF许可证处理

```
libbpf (LGPL 2.1) 必须动态链接
├── 编译时: -l bpf (动态库)
├── 运行时: 依赖系统libbpf.so
└── 不能静态链接到闭源Agent
```

### 5.3 ECS事件格式

所有平台事件统一转换为ECS格式，确保：
- 云端处理逻辑统一
- 检测规则跨平台复用
- 与OpenSearch原生兼容

---

## 六、Phase规划

| Phase | 周期 | 核心目标 | 关键任务 |
|-------|------|----------|----------|
| **Phase 1** | 3月 | MVP验证，100终端 | 01-30 (基础框架+Agent+Cloud+Console) |
| **Phase 2** | 6月 | 能力增强，1000终端 | 31-53 (高级检测+多租户+macOS) |
| **Phase 3** | 12月 | 企业级，10000+终端 | 扩展性+AI检测+SOAR |

---

## 七、文件结构约定

```
EDR-POC/
├── agent/
│   ├── core-c/           # C核心库
│   │   ├── src/
│   │   │   ├── collector/  # 采集器 (windows/, linux/, macos/)
│   │   │   ├── detector/   # 检测引擎 (yara/, sigma/)
│   │   │   └── common/     # 公共代码
│   │   └── CMakeLists.txt
│   └── main-go/          # Go主程序
│       ├── cmd/agent/
│       ├── internal/
│       │   ├── cgo/        # CGO封装
│       │   ├── comm/       # gRPC通信
│       │   └── policy/     # 策略管理
│       └── go.mod
├── cloud/
│   ├── cmd/              # 服务入口
│   ├── internal/         # 业务逻辑
│   ├── pkg/              # 公共包
│   └── api/proto/        # Protobuf定义
├── console/
│   ├── src/
│   │   ├── pages/        # 页面组件
│   │   ├── components/   # 公共组件
│   │   ├── stores/       # Zustand状态
│   │   └── api/          # API调用
│   └── vite.config.ts
└── deploy/               # 部署配置
```

**端口分配表**:

| 端口 | 服务 | 说明 |
|-----|------|------|
| 15432 | PostgreSQL | 主数据库 |
| 16379 | Redis | 缓存/状态存储 |
| 9080 | API Gateway | Cloud REST API |
| 9000 | ClickHouse | Native 接口 |
| 8123 | ClickHouse | HTTP 接口 |
| 9200 | OpenSearch | 事件检索 |

---

## 八、任务执行检查清单

执行任何任务前，确认：

- [ ] 使用的依赖许可证是否在白名单？
- [ ] C代码是否放在agent/core-c/？
- [ ] Go代码是否放在agent/main-go/或cloud/？
- [ ] 事件格式是否遵循ECS规范？
- [ ] 平台特定代码是否正确隔离？
- [ ] libbpf是否动态链接？
- [ ] 是否使用了Casbin做权限控制？
- [ ] 前端是否使用Ant Design 5组件？

---

## 九、变更记录

| 日期 | 变更项 | 影响范围 |
|------|--------|----------|
| 2025-11-30 | 初始版本 | - |

---

> **使用说明**: 
> 1. 新会话开始时，将本文档作为系统上下文提供
> 2. 当设计变更时，更新本文档对应章节
> 3. 任务执行结果与本文档冲突时，以本文档为准并反馈调整建议
