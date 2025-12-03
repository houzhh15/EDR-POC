# EDR Agent 启动故障排查指南

## 问题：Agent 启动后立即退出，无任何输出

### 症状
```bash
.\edr-agent.exe --config .\configs\agent.yaml
# 立即退出，无任何输出
# logs 目录下没有日志文件
```

### 可能原因及解决方案

#### 1. 配置文件路径错误

**检查**：
```powershell
# 确认配置文件存在
Test-Path .\configs\agent.yaml
```

**解决**：
```powershell
# 确保配置文件路径正确
.\edr-agent.exe --config .\configs\agent.yaml  # Windows
./edr-agent --config ./configs/agent.yaml      # Linux/macOS
```

#### 2. 配置文件格式错误

**检查**：
```powershell
# 查看配置文件内容
Get-Content .\configs\agent.yaml
```

**常见错误**：
- YAML 缩进错误（必须使用空格，不能用 Tab）
- 缺少必填字段
- 字段名拼写错误

**解决**：使用正确的配置模板
```yaml
agent:
  id: "test-agent-001"

collector:
  enabled_types: ["process"]
  buffer_size: 4096

log:
  level: "info"
  output: "file"          # 或 "console" 用于调试
  file_path: "logs/agent.log"
  max_size_mb: 100
  max_backups: 3

cloud:
  endpoint: ""            # 留空表示独立模式
  tls:
    enabled: false
```

#### 3. 日志目录不存在

**检查**：
```powershell
# 检查 logs 目录
Test-Path logs
```

**解决**：
```powershell
# 创建日志目录
New-Item -ItemType Directory -Force -Path logs
```

**说明**：最新版本已自动创建日志目录，但建议手动检查。

#### 4. 缺少 C 核心库 (libedr_core.dll)

**检查**：
```powershell
# Windows
Test-Path libedr_core.dll

# Linux
ls libedr_core.so

# macOS
ls libedr_core.dylib
```

**解决**：
```powershell
# 确保 DLL 与 exe 在同一目录
# 从 CI 构建产物中复制
Copy-Item build\bin\libedr_core.dll .\
```

#### 5. 配置了 cloud.endpoint 但无法连接

**现象**：
- 配置了 `endpoint: "grpc://10.146.25.35:9090"`
- Agent 启动失败或立即退出

**检查**：
```yaml
# 查看配置
cloud:
  endpoint: "grpc://10.146.25.35:9090"
```

**解决方案 A - 使用独立模式**：
```yaml
# 改为独立模式（推荐用于初期测试）
cloud:
  endpoint: ""  # 留空
```

**解决方案 B - 启用控制台日志查看错误**：
```yaml
log:
  output: "both"  # 同时输出到文件和控制台
```

重新启动后可在控制台看到详细错误信息。

#### 6. 无管理员权限

**现象**：
- ETW 采集需要管理员权限
- 普通用户启动可能失败

**解决**：
```powershell
# 以管理员身份运行 PowerShell
Start-Process powershell -Verb RunAs

# 在管理员窗口中
cd C:\edr-agent
.\edr-agent.exe --config .\configs\agent.yaml
```

## 诊断步骤

### 步骤 1：使用诊断脚本

```powershell
# 使用提供的诊断脚本
.\start-agent.bat
```

脚本会自动检查：
- 可执行文件是否存在
- 配置文件是否存在
- 日志目录是否存在
- C 核心库是否存在
- 是否有管理员权限
- 程序是否能正常执行

### 步骤 2：启用控制台输出

修改配置文件：
```yaml
log:
  level: "debug"        # 启用详细日志
  output: "console"     # 输出到控制台（而非文件）
```

重新启动：
```powershell
.\edr-agent.exe --config .\configs\agent.yaml
```

现在可以在控制台看到所有日志输出。

### 步骤 3：检查版本信息

```powershell
# 测试程序是否能正常执行
.\edr-agent.exe --version
```

**预期输出**：
```
EDR Agent 0.1.0 (commit: xxx, built: xxx)
Core Library: 0.1.0
```

如果此命令失败，说明：
- 程序本身有问题
- 缺少 libedr_core.dll
- 缺少其他依赖

### 步骤 4：查看详细配置

```powershell
# 显示当前配置
Get-Content .\configs\agent.yaml | Select-String -NotMatch "^#|^$"
```

### 步骤 5：手动测试日志写入

```powershell
# 测试日志目录是否可写
New-Item -ItemType File -Force -Path "logs\test.log"
"test" | Out-File -Append "logs\test.log"
Get-Content "logs\test.log"
```

## 推荐的测试配置

**最简配置（独立模式）**：
```yaml
agent:
  id: "test-agent-001"

collector:
  enabled_types: ["process"]
  buffer_size: 4096

log:
  level: "debug"
  output: "console"  # 先用控制台调试

cloud:
  endpoint: ""
```

**调试配置（带文件日志）**：
```yaml
agent:
  id: "test-agent-001"

collector:
  enabled_types: ["process"]
  buffer_size: 4096

log:
  level: "debug"
  output: "both"     # 同时输出到控制台和文件
  file_path: "logs/agent.log"
  max_size_mb: 100
  max_backups: 3

cloud:
  endpoint: ""
```

## 常见错误信息

### "Failed to load config: ..."

**原因**：配置文件解析失败

**解决**：
1. 检查 YAML 格式
2. 检查字段名是否正确
3. 检查缩进（必须用空格）

### "Failed to initialize logger: ..."

**原因**：日志系统初始化失败

**解决**：
1. 创建 logs 目录：`mkdir logs`
2. 检查目录权限
3. 检查磁盘空间

### "Failed to initialize core library: ..."

**原因**：C 核心库加载失败

**解决**：
1. 确认 libedr_core.dll 存在
2. 将 DLL 放在与 exe 同目录
3. 检查 DLL 是否损坏：`.\edr-agent.exe --version`

### 无任何输出直接退出

**可能原因**：
1. 配置文件不存在或路径错误
2. YAML 格式严重错误导致 panic
3. 缺少 DLL

**诊断**：
```powershell
# 1. 使用 --version 测试基本功能
.\edr-agent.exe --version

# 2. 使用控制台输出
# 修改配置 log.output: "console"

# 3. 使用诊断脚本
.\start-agent.bat
```

## 获取帮助

如果以上方法都无法解决问题，请收集以下信息：

1. **系统信息**：
   ```powershell
   systeminfo | findstr /B /C:"OS Name" /C:"OS Version"
   ```

2. **文件检查**：
   ```powershell
   Get-ChildItem | Select-Object Name, Length
   ```

3. **配置文件**：
   ```powershell
   Get-Content .\configs\agent.yaml
   ```

4. **版本信息**：
   ```powershell
   .\edr-agent.exe --version 2>&1
   ```

5. **日志文件**（如果存在）：
   ```powershell
   Get-Content logs\agent.log -Tail 50
   ```

提交 Issue 到：https://github.com/houzhh15/EDR-POC/issues
