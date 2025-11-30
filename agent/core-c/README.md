# EDR Core C Library

EDR Agent 的 C 核心库，负责平台相关的事件采集和检测引擎实现。

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
