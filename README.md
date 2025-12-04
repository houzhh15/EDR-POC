# EDR Platform

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

正在开发的 EDR 平台。

## 📋 功能特性

- **终端采集**:支持 Windows (ETW)、Linux (eBPF)、macOS (Endpoint Security) 多平台事件采集
- **实时检测**:基于 YARA 和 Sigma 规则的威胁检测引擎
- **云端分析**:高性能事件处理与关联分析
- **响应处置**:远程命令执行、进程隔离、文件隔离等响应能力
- **管理控制台**:直观的 Web 界面,支持告警管理、资产管理、策略配置

## ⚠️ 平台支持状态 (Phase 1-B1)

| 平台 | PAL 层 | 事件采集 | 说明 |
|------|--------|----------|------|
| **Windows** | 🚧 占位代码 | ✅ ETW 已实现 | 进程采集已完成,PAL层待实现 |
| **macOS** | ✅ 完整实现 | 📋 Phase 2 计划 | PAL层完整,进程采集待开发 |
| **Linux** | 🚧 占位代码 | 📋 Phase 2 计划 | 完全待实现 |

> **重要说明**: 
> - **Windows**: 进程事件采集(ETW)已完整实现(任务 `task_1764425089079736000`),包括 Ring Buffer、ETW Session、事件解析等核心模块。但由于 **PAL 层未实现**,Agent 启动时会返回 `"error":"not supported"`。
> - **macOS**: PAL 层(互斥锁、线程、文件操作等)已完整实现,但进程事件采集器(Endpoint Security)属于 Phase 2 任务 (任务 47),尚未开始。
> - **当前可测试**: 需要先实现对应平台的 PAL 层初始化,才能运行已完成的采集器代码。

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
