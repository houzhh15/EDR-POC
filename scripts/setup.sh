#!/bin/bash
# ============================================================
# EDR Platform - 环境初始化脚本
# ============================================================
# 使用方式: ./scripts/setup.sh
# ============================================================

set -e

echo "============================================"
echo "EDR Platform - 环境检查"
echo "============================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查命令是否存在
check_command() {
    local cmd=$1
    local required_version=$2
    local install_hint=$3

    if command -v "$cmd" &> /dev/null; then
        local version=$($cmd --version 2>&1 | head -n 1)
        echo -e "${GREEN}✅ $cmd${NC}: $version"
        return 0
    else
        echo -e "${RED}❌ $cmd${NC}: 未安装"
        if [ -n "$install_hint" ]; then
            echo -e "   ${YELLOW}安装建议: $install_hint${NC}"
        fi
        return 1
    fi
}

# 检查版本
check_version() {
    local cmd=$1
    local min_version=$2
    local current_version=$3

    # 简单版本比较 (仅比较主版本号)
    local current_major=$(echo "$current_version" | grep -oE '^[0-9]+' | head -1)
    local min_major=$(echo "$min_version" | grep -oE '^[0-9]+' | head -1)

    if [ "$current_major" -ge "$min_major" ]; then
        return 0
    else
        return 1
    fi
}

echo "📋 检查必需工具..."
echo ""

# 检查结果
ALL_OK=true

# Go
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if check_version go 1.21 "$GO_VERSION"; then
        echo -e "${GREEN}✅ Go${NC}: $GO_VERSION (>= 1.21)"
    else
        echo -e "${YELLOW}⚠️  Go${NC}: $GO_VERSION (需要 >= 1.21)"
        ALL_OK=false
    fi
else
    echo -e "${RED}❌ Go${NC}: 未安装"
    echo -e "   ${YELLOW}安装: https://golang.org/dl/${NC}"
    ALL_OK=false
fi

# Node.js
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version | sed 's/v//')
    NODE_MAJOR=$(echo "$NODE_VERSION" | cut -d. -f1)
    if [ "$NODE_MAJOR" -ge 18 ]; then
        echo -e "${GREEN}✅ Node.js${NC}: $NODE_VERSION (>= 18)"
    else
        echo -e "${YELLOW}⚠️  Node.js${NC}: $NODE_VERSION (需要 >= 18)"
        ALL_OK=false
    fi
else
    echo -e "${RED}❌ Node.js${NC}: 未安装"
    echo -e "   ${YELLOW}安装: https://nodejs.org/${NC}"
    ALL_OK=false
fi

# pnpm
check_command pnpm "" "npm install -g pnpm" || ALL_OK=false

# CMake
if command -v cmake &> /dev/null; then
    CMAKE_VERSION=$(cmake --version | head -n 1 | grep -oE '[0-9]+\.[0-9]+')
    CMAKE_MAJOR=$(echo "$CMAKE_VERSION" | cut -d. -f1)
    CMAKE_MINOR=$(echo "$CMAKE_VERSION" | cut -d. -f2)
    if [ "$CMAKE_MAJOR" -ge 3 ] && [ "$CMAKE_MINOR" -ge 20 ]; then
        echo -e "${GREEN}✅ CMake${NC}: $CMAKE_VERSION (>= 3.20)"
    else
        echo -e "${YELLOW}⚠️  CMake${NC}: $CMAKE_VERSION (需要 >= 3.20)"
        ALL_OK=false
    fi
else
    echo -e "${RED}❌ CMake${NC}: 未安装"
    echo -e "   ${YELLOW}安装: brew install cmake (macOS) / apt install cmake (Linux)${NC}"
    ALL_OK=false
fi

# Docker
check_command docker "" "https://docs.docker.com/get-docker/" || ALL_OK=false

# Docker Compose
if docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
    echo -e "${GREEN}✅ Docker Compose${NC}: $COMPOSE_VERSION"
else
    echo -e "${RED}❌ Docker Compose${NC}: 未安装"
    ALL_OK=false
fi

echo ""
echo "📋 检查可选工具..."
echo ""

# golangci-lint
check_command golangci-lint "" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# clang-format
check_command clang-format "" "brew install clang-format (macOS) / apt install clang-format (Linux)"

# yara
check_command yara "" "brew install yara (macOS) / apt install libyara-dev (Linux)"

echo ""
echo "============================================"

if $ALL_OK; then
    echo -e "${GREEN}✅ 所有必需工具已安装！${NC}"
    echo ""
    echo "下一步:"
    echo "  1. 启动开发环境: make dev-up"
    echo "  2. 构建项目: make build"
    echo "  3. 运行测试: make test"
else
    echo -e "${YELLOW}⚠️  部分工具缺失或版本不满足要求${NC}"
    echo ""
    echo "请安装缺失的工具后再次运行此脚本。"
    exit 1
fi

echo "============================================"
