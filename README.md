# EDR Platform

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

正在开发的 EDR 平台。

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
