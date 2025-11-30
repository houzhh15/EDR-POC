#!/bin/bash
# ============================================================
# EDR Platform - 许可证检查脚本
# ============================================================
# 使用方式: ./scripts/license-check.sh
# 禁止许可证: GPL, AGPL, SSPL
# ============================================================

set -e

echo "============================================"
echo "EDR Platform - 许可证合规检查"
echo "============================================"
echo ""

# 禁止的许可证正则
FORBIDDEN_LICENSES="GPL|AGPL|SSPL|CC-BY-NC|CC-BY-ND"
FAILED=false

# ============================================================
# 检查 Go 依赖
# ============================================================
echo "🔍 检查 Go 依赖许可证..."
echo ""

check_go_module() {
    local dir=$1
    local name=$2

    if [ ! -f "$dir/go.mod" ]; then
        echo "  ⚠️  $name: go.mod 不存在"
        return 0
    fi

    echo "  检查 $name..."

    if command -v go-licenses &> /dev/null; then
        cd "$dir"
        if go-licenses check ./... --disallowed_types=forbidden,restricted 2>&1 | grep -qE "$FORBIDDEN_LICENSES"; then
            echo "  ❌ $name: 发现禁止的许可证!"
            cd - > /dev/null
            return 1
        else
            echo "  ✅ $name: 许可证合规"
        fi
        cd - > /dev/null
    else
        echo "  ⚠️  go-licenses 未安装，跳过检查"
        echo "     安装: go install github.com/google/go-licenses@latest"
    fi

    return 0
}

check_go_module "agent/main-go" "Agent Go" || FAILED=true
check_go_module "cloud" "Cloud" || FAILED=true

echo ""

# ============================================================
# 检查 Node.js 依赖
# ============================================================
echo "🔍 检查 Node.js 依赖许可证..."
echo ""

if [ -f "console/package.json" ]; then
    cd console

    if command -v license-checker &> /dev/null; then
        if license-checker --failOn "GPL;AGPL;SSPL" --summary > /dev/null 2>&1; then
            echo "  ✅ Console: 许可证合规"
        else
            echo "  ❌ Console: 发现禁止的许可证!"
            license-checker --onlyAllow "MIT;Apache-2.0;BSD-2-Clause;BSD-3-Clause;ISC;CC0-1.0;Unlicense" --summary 2>&1 || true
            FAILED=true
        fi
    else
        if [ -d "node_modules" ]; then
            echo "  ⚠️  license-checker 未安装，跳过检查"
            echo "     安装: npm install -g license-checker"
        else
            echo "  ⚠️  node_modules 不存在，请先运行 pnpm install"
        fi
    fi

    cd - > /dev/null
else
    echo "  ⚠️  console/package.json 不存在"
fi

echo ""
echo "============================================"

if $FAILED; then
    echo "❌ 许可证检查失败！"
    echo ""
    echo "请检查并移除使用 GPL/AGPL/SSPL 许可证的依赖。"
    echo "允许的许可证: Apache 2.0, MIT, BSD, ISC"
    echo ""
    exit 1
else
    echo "✅ 许可证检查通过！"
    echo ""
fi
