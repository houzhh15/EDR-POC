# EDR Core C Library

EDR Agent 的 C 核心库,负责平台相关的事件采集和检测引擎实现。

## ⚠️ 平台支持状态

| 平台 | PAL 层 | 采集器实现 | 说明 |
|------|--------|-----------|------|
| **Windows** | 🚧 占位代码 | ✅ ETW 已完成 | 进程采集器已实现,PAL层阻塞运行 |
| **macOS** | ✅ 完整实现 | 📋 Phase 2 | PAL层可用,采集器待开发 |
| **Linux** | 🚧 占位代码 | 📋 Phase 2 | 完全待实现 |

**详细说明**:

### Windows 平台
- ✅ **已实现**: ETW 进程事件采集器 (`src/collector/windows/`)
  - `etw_session.c/h`: ETW Session 管理
  - `etw_process.c/h`: 进程事件解析
  - `event_buffer.c/h`: Ring Buffer 无锁队列
  - `edr_log.c/h`: 日志框架
- 🚧 **阻塞项**: `src/pal/pal_windows.c` 中 `pal_init()` 直接返回 `EDR_ERR_NOT_SUPPORTED`
- 📋 **待办**: 实现 Windows PAL 层(互斥锁、线程、时间、文件操作)

### macOS 平台  
- ✅ **已实现**: `src/pal/pal_macos.c` - 完整的 PAL 层实现
- 📋 **待开发**: Endpoint Security 进程采集器 (任务 47, Phase 2)

### Linux 平台
- 🚧 **状态**: 完全待实现(Phase 2)

> **重要**: 当前 Agent 启动失败的根本原因是 **PAL 层占位实现**,不是采集器问题。Windows 的 ETW 采集器代码已完整,一旦 PAL 层实现即可运行。

## 📋 功能模块

| 模块 | 目录 | 说明 |
|------|------|------|
| Collector | `src/collector/` | 事件采集器 (Windows ETW / Linux eBPF / macOS ES) |
| Detector | `src/detector/` | 检测引擎 (YARA / Sigma) |
| Response | `src/response/` | 响应执行器 |
| Common | `src/common/` | 公共工具函数 |

## 🔧 依赖要求

### 编译工具

- CMake 3.20+
- GCC 9+ / Clang 11+ / MSVC 2019+

### 平台依赖

| 平台 | 依赖库 | 安装命令 |
|------|--------|----------|
| Linux | libbpf, libyara | `apt install libbpf-dev libyara-dev` |
| macOS | yara | `brew install yara` |
| Windows | yara (vcpkg) | `vcpkg install yara` |

## 🏗️ 编译方式

### 标准编译

```bash
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make
```

### Debug 编译

```bash
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Debug
make
```

### 使用根目录 Makefile

```bash
# 在项目根目录
make build-agent-c
```

## 📦 构建产物

| 平台 | 产物 |
|------|------|
| Linux | `libedr_core.so` |
| macOS | `libedr_core.dylib` |
| Windows | `edr_core.dll` |

## 🔗 Go 集成

此库通过 CGO 被 `agent/main-go` 调用：

```go
// #cgo LDFLAGS: -L${SRCDIR}/../../core-c/build -ledr_core
// #include "edr_core.h"
import "C"
```

## 📝 代码规范

- 遵循 `.clang-format` 配置
- 函数命名：`edr_<module>_<action>` (如 `edr_collector_start`)
- 错误处理：返回 `int` 错误码，0 表示成功

## ⚠️ 许可证注意

- **libbpf (LGPL)**: 必须动态链接，不能静态链接
- **yara (BSD-3)**: 可自由使用
