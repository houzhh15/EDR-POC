#!/bin/bash
# ============================================================
# EDR Platform - 服务健康检查脚本
# ============================================================
# 使用方式: ./scripts/health-check.sh
# ============================================================

set -e

TIMEOUT=${TIMEOUT:-120}
INTERVAL=5
ELAPSED=0

echo "============================================"
echo "EDR Platform - 服务健康检查"
echo "============================================"
echo ""

# 服务列表: 名称:主机:端口
services=(
    "Kafka:localhost:9092"
    "OpenSearch:localhost:9200"
    "PostgreSQL:localhost:5432"
    "Redis:localhost:6379"
    "ClickHouse:localhost:8123"
    "MinIO:localhost:9001"
    "Jaeger:localhost:16686"
)

# 检查端口是否可用
check_port() {
    local name=$1
    local host=$2
    local port=$3

    if nc -z "$host" "$port" 2>/dev/null; then
        echo -e "  ✅ $name ($host:$port)"
        return 0
    else
        echo -e "  ⏳ $name ($host:$port) - 等待中..."
        return 1
    fi
}

# 主循环
while [ $ELAPSED -lt $TIMEOUT ]; do
    all_ready=true

    echo "🔍 检查服务状态... (已等待 ${ELAPSED}s / ${TIMEOUT}s)"
    echo ""

    for service in "${services[@]}"; do
        IFS=':' read -r name host port <<< "$service"
        if ! check_port "$name" "$host" "$port"; then
            all_ready=false
        fi
    done

    if $all_ready; then
        echo ""
        echo "============================================"
        echo "✅ 所有服务已就绪！"
        echo "============================================"
        echo ""
        echo "服务地址:"
        echo "  - Kafka:      localhost:9092"
        echo "  - OpenSearch: localhost:9200"
        echo "  - PostgreSQL: localhost:5432"
        echo "  - Redis:      localhost:6379"
        echo "  - ClickHouse: localhost:8123"
        echo "  - MinIO API:  localhost:9001"
        echo "  - MinIO UI:   localhost:9002"
        echo "  - Jaeger UI:  localhost:16686"
        echo ""
        exit 0
    fi

    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
    echo ""
done

echo ""
echo "============================================"
echo "❌ 超时：部分服务未就绪"
echo "============================================"
echo ""
echo "请检查 Docker 容器状态: docker compose ps"
echo "查看容器日志: docker compose logs"
echo ""
exit 1
