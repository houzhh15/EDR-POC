# EDR Agent 快速诊断脚本 (PowerShell)
# 使用方法: .\diagnose.ps1 [配置文件路径]

param(
    [string]$ConfigPath = "configs\agent.yaml"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "EDR Agent 快速诊断工具"
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 检查当前目录
Write-Host "[1] 当前目录: $PWD"
Write-Host ""

# 2. 检查可执行文件
Write-Host "[2] 检查可执行文件..."
if (Test-Path "edr-agent.exe") {
    Write-Host "    [OK] edr-agent.exe 存在" -ForegroundColor Green
    Get-Item edr-agent.exe | Format-Table Name, Length, LastWriteTime
} else {
    Write-Host "    [ERROR] edr-agent.exe 不存在" -ForegroundColor Red
    Write-Host "    请确保在正确的目录下运行此脚本"
    exit 1
}
Write-Host ""

# 3. 检查配置文件
Write-Host "[3] 检查配置文件..."
if (Test-Path $ConfigPath) {
    Write-Host "    [OK] 配置文件存在: $ConfigPath" -ForegroundColor Green
} else {
    Write-Host "    [ERROR] 配置文件不存在: $ConfigPath" -ForegroundColor Red
    Write-Host "    请创建配置文件或使用示例配置:"
    Write-Host "    Copy-Item agent.yaml.example $ConfigPath"
    exit 1
}
Write-Host ""

# 4. 显示配置内容
Write-Host "[4] 配置文件内容（前20行）:"
Write-Host "----------------------------------------"
Get-Content $ConfigPath -TotalCount 20
Write-Host "----------------------------------------"
Write-Host ""

# 5. 检查/创建日志目录
Write-Host "[5] 检查日志目录..."
if (-not (Test-Path "logs")) {
    Write-Host "    [WARN] logs 目录不存在，正在创建..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path "logs" | Out-Null
    if (Test-Path "logs") {
        Write-Host "    [OK] logs 目录创建成功" -ForegroundColor Green
    } else {
        Write-Host "    [ERROR] logs 目录创建失败" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "    [OK] logs 目录存在" -ForegroundColor Green
}
Write-Host ""

# 6. 检查管理员权限
Write-Host "[6] 检查管理员权限..."
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdmin) {
    Write-Host "    [OK] 以管理员权限运行" -ForegroundColor Green
} else {
    Write-Host "    [WARN] 未以管理员权限运行" -ForegroundColor Yellow
    Write-Host "    ETW 事件采集需要管理员权限"
    Write-Host "    建议右键点击此脚本，选择'以管理员身份运行'"
}
Write-Host ""

# 7. 检查 C 核心库
Write-Host "[7] 检查 C 核心库..."
if (Test-Path "libedr_core.dll") {
    Write-Host "    [OK] libedr_core.dll 存在" -ForegroundColor Green
    Get-Item libedr_core.dll | Format-Table Name, Length, LastWriteTime
} else {
    Write-Host "    [ERROR] libedr_core.dll 不存在" -ForegroundColor Red
    Write-Host "    请确保 DLL 与 exe 在同一目录"
    exit 1
}
Write-Host ""

# 8. 测试可执行文件
Write-Host "[8] 测试可执行文件..."
Write-Host "    运行: .\edr-agent.exe --version"
try {
    $versionOutput = & .\edr-agent.exe --version 2>&1
    Write-Host $versionOutput
    Write-Host "    [OK] 程序可执行" -ForegroundColor Green
} catch {
    Write-Host "    [ERROR] 程序执行失败: $_" -ForegroundColor Red
    Write-Host "    可能的原因:"
    Write-Host "    - 缺少依赖库 (libedr_core.dll)"
    Write-Host "    - 程序损坏"
    exit 1
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "诊断完成，准备启动 Agent"
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "配置文件: $ConfigPath"
Write-Host "日志目录: logs\"
Write-Host ""

if (-not $isAdmin) {
    Write-Host "[警告] 未使用管理员权限，可能无法采集 ETW 事件" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "按任意键启动 Agent，或 Ctrl+C 取消..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
Write-Host ""

Write-Host "[启动] .\edr-agent.exe --config $ConfigPath"
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 启动 Agent
$process = Start-Process -FilePath ".\edr-agent.exe" -ArgumentList "--config $ConfigPath" -NoNewWindow -Wait -PassThru

# 检查退出状态
$exitCode = $process.ExitCode
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Agent 已退出，退出码: $exitCode"
Write-Host "========================================" -ForegroundColor Cyan

if ($exitCode -ne 0) {
    Write-Host ""
    Write-Host "[错误分析]" -ForegroundColor Yellow
    
    switch ($exitCode) {
        1 {
            Write-Host "退出码 1: 配置或初始化错误" -ForegroundColor Red
            Write-Host "请检查:"
            Write-Host "- 配置文件格式是否正确 (YAML 语法)"
            Write-Host "- 日志文件路径是否可写"
            Write-Host "- 所有必填字段是否填写"
        }
        2 {
            Write-Host "退出码 2: 程序崩溃 (panic)" -ForegroundColor Red
            Write-Host "这是一个严重错误，请报告此问题"
        }
        default {
            Write-Host "退出码 $exitCode`: 未知错误" -ForegroundColor Red
        }
    }
    
    Write-Host ""
    Write-Host "查看日志文件获取详细信息:" -ForegroundColor Yellow
    if (Test-Path "logs\agent.log") {
        Write-Host ""
        Write-Host "[最新日志 (后20行)]:" -ForegroundColor Cyan
        Get-Content "logs\agent.log" -Tail 20
    } else {
        Write-Host "logs\agent.log 文件不存在（日志未成功写入）" -ForegroundColor Red
        Write-Host "可能原因:"
        Write-Host "- 程序在日志初始化之前就退出了"
        Write-Host "- 配置文件中 log.output 设置为 'console'"
    }
}

Write-Host ""
Write-Host "按任意键退出..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
