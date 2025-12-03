@echo off
REM EDR Agent 启动脚本 (Windows)
REM 用于诊断启动问题

echo ========================================
echo EDR Agent 启动诊断
echo ========================================
echo.

REM 检查当前目录
echo [1] 当前目录: %CD%
echo.

REM 检查可执行文件
echo [2] 检查可执行文件...
if exist "edr-agent.exe" (
    echo    [OK] edr-agent.exe 存在
) else (
    echo    [ERROR] edr-agent.exe 不存在
    echo    请确保在正确的目录下运行此脚本
    pause
    exit /b 1
)
echo.

REM 检查配置文件
echo [3] 检查配置文件...
set CONFIG_PATH=configs\agent.yaml
if "%1" NEQ "" set CONFIG_PATH=%1

if exist "%CONFIG_PATH%" (
    echo    [OK] 配置文件存在: %CONFIG_PATH%
) else (
    echo    [ERROR] 配置文件不存在: %CONFIG_PATH%
    echo    请检查配置文件路径
    pause
    exit /b 1
)
echo.

REM 显示配置内容（前20行）
echo [4] 配置文件内容（前20行）:
echo ----------------------------------------
powershell -Command "Get-Content '%CONFIG_PATH%' -TotalCount 20"
echo ----------------------------------------
echo.

REM 检查/创建日志目录
echo [5] 检查日志目录...
if not exist "logs" (
    echo    [WARN] logs 目录不存在，正在创建...
    mkdir logs
    if exist "logs" (
        echo    [OK] logs 目录创建成功
    ) else (
        echo    [ERROR] logs 目录创建失败
        pause
        exit /b 1
    )
) else (
    echo    [OK] logs 目录存在
)
echo.

REM 检查管理员权限
echo [6] 检查管理员权限...
net session >nul 2>&1
if %errorLevel% == 0 (
    echo    [OK] 以管理员权限运行
) else (
    echo    [WARN] 未以管理员权限运行
    echo    ETW 事件采集需要管理员权限
    echo    请右键点击此脚本，选择"以管理员身份运行"
    echo.
    echo    是否继续启动（可能无法采集事件）？[Y/N]
    choice /C YN /N
    if errorlevel 2 exit /b 1
)
echo.

REM 检查 DLL 依赖
echo [7] 检查 C 核心库...
if exist "libedr_core.dll" (
    echo    [OK] libedr_core.dll 存在
) else (
    echo    [ERROR] libedr_core.dll 不存在
    echo    请确保 DLL 与 exe 在同一目录
    pause
    exit /b 1
)
echo.

REM 测试版本信息
echo [8] 测试可执行文件...
echo    运行: edr-agent.exe --version
edr-agent.exe --version
if %errorLevel% NEQ 0 (
    echo    [ERROR] 程序执行失败，退出码: %errorLevel%
    echo    可能的原因:
    echo    - 缺少依赖库 (libedr_core.dll)
    echo    - 程序损坏
    pause
    exit /b 1
)
echo.

echo ========================================
echo 诊断完成，准备启动 Agent
echo ========================================
echo.
echo 配置文件: %CONFIG_PATH%
echo 日志目录: logs\
echo.
echo 按任意键启动，或 Ctrl+C 取消...
pause >nul
echo.

REM 启动 Agent
echo [启动] edr-agent.exe --config %CONFIG_PATH%
echo ========================================
edr-agent.exe --config "%CONFIG_PATH%"

REM 检查退出状态
set EXIT_CODE=%errorLevel%
echo.
echo ========================================
echo Agent 已退出，退出码: %EXIT_CODE%
echo ========================================

if %EXIT_CODE% NEQ 0 (
    echo.
    echo [错误分析]
    if %EXIT_CODE% == 1 (
        echo 退出码 1: 配置或初始化错误
        echo 请检查:
        echo - 配置文件格式是否正确
        echo - 日志文件路径是否可写
        echo - 网络配置是否正确
    )
    echo.
    echo 查看日志文件获取详细信息:
    if exist "logs\agent.log" (
        echo.
        echo [最新日志]:
        powershell -Command "Get-Content 'logs\agent.log' -Tail 20"
    ) else (
        echo logs\agent.log 文件不存在（日志未成功写入）
    )
)

echo.
pause
