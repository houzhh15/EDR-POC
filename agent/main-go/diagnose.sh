#!/bin/bash
# EDR Agent 快速诊断脚本（Linux/macOS）

echo "========================================"
echo "EDR Agent 诊断工具"
echo "========================================"
echo ""

# 检查当前目录
echo "[1] 当前目录: $(pwd)"
echo ""

# 检查可执行文件
echo "[2] 检查可执行文件..."
if [ -f "edr-agent" ] || [ -f "edr-agent.exe" ]; then
    echo "    [OK] Agent 可执行文件存在"
    ls -lh edr-agent* 2>/dev/null
else
    echo "    [ERROR] Agent 可执行文件不存在"
    echo "    请确保在正确的目录下运行此脚本"
    exit 1
fi
echo ""

# 检查配置文件
echo "[3] 检查配置文件..."
CONFIG_PATH="${1:-configs/agent.yaml}"
if [ -f "$CONFIG_PATH" ]; then
    echo "    [OK] 配置文件存在: $CONFIG_PATH"
else
    echo "    [ERROR] 配置文件不存在: $CONFIG_PATH"
    echo "    请创建配置文件或使用示例配置:"
    echo "    cp agent.yaml.example $CONFIG_PATH"
    exit 1
fi
echo ""

# 显示配置内容
echo "[4] 配置文件内容（前20行）:"
echo "----------------------------------------"
head -20 "$CONFIG_PATH"
echo "----------------------------------------"
echo ""

# 检查/创建日志目录
echo "[5] 检查日志目录..."
if [ ! -d "logs" ]; then
    echo "    [WARN] logs 目录不存在，正在创建..."
    mkdir -p logs
    if [ -d "logs" ]; then
        echo "    [OK] logs 目录创建成功"
    else
        echo "    [ERROR] logs 目录创建失败"
        exit 1
    fi
else
    echo "    [OK] logs 目录存在"
fi
echo ""

# 检查 C 核心库
echo "[6] 检查 C 核心库..."
if [ -f "libedr_core.so" ] || [ -f "libedr_core.dylib" ] || [ -f "libedr_core.dll" ]; then
    echo "    [OK] C 核心库存在"
    ls -lh libedr_core.* 2>/dev/null
else
    echo "    [ERROR] C 核心库不存在"
    echo "    请确保 libedr_core.so/dylib/dll 与可执行文件在同一目录"
    exit 1
fi
echo ""

# 测试版本信息
echo "[7] 测试可执行文件..."
echo "    运行: ./edr-agent --version"
if [ -f "edr-agent" ]; then
    ./edr-agent --version
elif [ -f "edr-agent.exe" ]; then
    ./edr-agent.exe --version
fi

if [ $? -ne 0 ]; then
    echo "    [ERROR] 程序执行失败"
    echo "    可能的原因:"
    echo "    - 缺少依赖库"
    echo "    - 程序损坏"
    exit 1
fi
echo ""

echo "========================================"
echo "诊断完成，准备启动 Agent"
echo "========================================"
echo ""
echo "配置文件: $CONFIG_PATH"
echo "日志目录: logs/"
echo ""
echo "按 Enter 启动，或 Ctrl+C 取消..."
read

echo ""
echo "[启动] ./edr-agent --config $CONFIG_PATH"
echo "========================================"

# 启动 Agent
if [ -f "edr-agent" ]; then
    ./edr-agent --config "$CONFIG_PATH"
elif [ -f "edr-agent.exe" ]; then
    ./edr-agent.exe --config "$CONFIG_PATH"
fi

# 检查退出状态
EXIT_CODE=$?
echo ""
echo "========================================"
echo "Agent 已退出，退出码: $EXIT_CODE"
echo "========================================"

if [ $EXIT_CODE -ne 0 ]; then
    echo ""
    echo "[错误分析]"
    if [ $EXIT_CODE -eq 1 ]; then
        echo "退出码 1: 配置或初始化错误"
        echo "请检查:"
        echo "- 配置文件格式是否正确"
        echo "- 日志文件路径是否可写"
        echo "- 网络配置是否正确"
    fi
    echo ""
    echo "查看日志文件获取详细信息:"
    if [ -f "logs/agent.log" ]; then
        echo ""
        echo "[最新日志]:"
        tail -20 logs/agent.log
    else
        echo "logs/agent.log 文件不存在（日志未成功写入）"
    fi
fi

echo ""
