#!/bin/bash
# ============================================================
# EDR Platform - Protobuf 代码生成脚本
# ============================================================
# 使用方式: ./scripts/proto-gen.sh
# 依赖: protoc, protoc-gen-go, protoc-gen-go-grpc
# ============================================================

set -e

PROTO_DIR="proto"
AGENT_GO_OUT="agent/main-go/pkg/proto"
CLOUD_GO_OUT="cloud/pkg/proto"
CONSOLE_TS_OUT="console/src/api/proto"

echo "============================================"
echo "EDR Platform - Protobuf 代码生成"
echo "============================================"
echo ""

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc 未安装"
    echo ""
    echo "安装方式:"
    echo "  macOS: brew install protobuf"
    echo "  Linux: apt install protobuf-compiler"
    echo ""
    exit 1
fi

# 检查 Go 插件
if ! command -v protoc-gen-go &> /dev/null; then
    echo "⚠️  protoc-gen-go 未安装，正在安装..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "⚠️  protoc-gen-go-grpc 未安装，正在安装..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录
mkdir -p "$AGENT_GO_OUT"
mkdir -p "$CLOUD_GO_OUT"
mkdir -p "$CONSOLE_TS_OUT"

# 查找所有 .proto 文件
PROTO_FILES=$(find "$PROTO_DIR" -name "*.proto" 2>/dev/null)

if [ -z "$PROTO_FILES" ]; then
    echo "⚠️  未找到 .proto 文件"
    echo "请在 $PROTO_DIR 目录下创建 .proto 文件"
    exit 0
fi

echo "📝 找到以下 .proto 文件:"
echo "$PROTO_FILES"
echo ""

# 生成 Go 代码
echo "🔧 生成 Go 代码..."
for proto in $PROTO_FILES; do
    echo "  处理: $proto"

    # Agent Go
    protoc \
        --proto_path="$PROTO_DIR" \
        --go_out="$AGENT_GO_OUT" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$AGENT_GO_OUT" \
        --go-grpc_opt=paths=source_relative \
        "$proto" || true

    # Cloud Go
    protoc \
        --proto_path="$PROTO_DIR" \
        --go_out="$CLOUD_GO_OUT" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$CLOUD_GO_OUT" \
        --go-grpc_opt=paths=source_relative \
        "$proto" || true
done

echo ""
echo "✅ Protobuf 代码生成完成！"
echo ""
echo "生成目录:"
echo "  - Agent Go: $AGENT_GO_OUT"
echo "  - Cloud Go: $CLOUD_GO_OUT"
echo ""
