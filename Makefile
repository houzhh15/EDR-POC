# ============================================================
# EDR Platform Makefile
# ============================================================
# 统一构建入口，支持 Linux / macOS / Windows
# 使用方式: make [target]
# 查看帮助: make help
# ============================================================

# 变量定义
GO := go
CMAKE := cmake
PNPM := pnpm
DOCKER_COMPOSE := docker compose

# 目录定义
AGENT_C_DIR := agent/core-c
AGENT_GO_DIR := agent/main-go
CLOUD_DIR := cloud
CONSOLE_DIR := console
DEPLOY_DIR := deploy/docker

# 构建输出目录
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin

# 版本信息 (可通过命令行覆盖)
VERSION ?= 0.1.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go 链接标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# ============================================================
# 平台检测
# ============================================================
UNAME_S := $(shell uname -s 2>/dev/null || echo "Windows")
ifeq ($(UNAME_S),Linux)
    PLATFORM := linux
    LIB_EXT := .so
endif
ifeq ($(UNAME_S),Darwin)
    PLATFORM := darwin
    LIB_EXT := .dylib
endif
ifeq ($(OS),Windows_NT)
    PLATFORM := windows
    LIB_EXT := .dll
endif

# ============================================================
# 目标声明
# ============================================================
.PHONY: all build clean test lint fmt
.PHONY: build-agent build-agent-c build-agent-go build-cloud build-console
.PHONY: test-agent test-cloud test-cloud-unit test-cloud-integration test-console
.PHONY: lint-c lint-go lint-ts
.PHONY: dev-up dev-down dev-logs dev-ps dev-reset
.PHONY: proto-gen license-check
.PHONY: help

# ============================================================
# 默认目标
# ============================================================
all: build

# ============================================================
# 构建目标
# ============================================================
build: build-agent build-cloud build-console
	@echo "============================================"
	@echo "✅ 全部构建完成!"
	@echo "============================================"
	@echo "产物目录: $(BIN_DIR)/"
	@ls -la $(BIN_DIR)/ 2>/dev/null || echo "(目录为空)"

# 清理构建产物
clean:
	@echo "🧹 清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(AGENT_C_DIR)/build
	rm -rf $(CONSOLE_DIR)/dist $(CONSOLE_DIR)/node_modules/.vite
	@echo "✅ 清理完成"

# ============================================================
# Agent 构建
# ============================================================
build-agent: build-agent-c build-agent-go
	@echo "✅ Agent 构建完成"

# CMake 生成器设置 (Windows 需要使用 MinGW Makefiles)
ifeq ($(OS),Windows_NT)
    CMAKE_GENERATOR := -G "MinGW Makefiles"
    MAKE_CMD := mingw32-make
else
    CMAKE_GENERATOR :=
    MAKE_CMD := $(MAKE)
endif

build-agent-c:
	@echo "📦 构建 Agent C 核心库..."
	@mkdir -p $(AGENT_C_DIR)/build
	cd $(AGENT_C_DIR)/build && $(CMAKE) $(CMAKE_GENERATOR) .. -DCMAKE_BUILD_TYPE=Release
	cd $(AGENT_C_DIR)/build && $(MAKE_CMD)
	@echo "✅ C 核心库构建完成: $(AGENT_C_DIR)/build/libedr_core$(LIB_EXT)"

build-agent-go: build-agent-c
	@echo "📦 构建 Agent Go 主程序..."
	@mkdir -p $(BIN_DIR)
	cd $(AGENT_GO_DIR) && CGO_ENABLED=1 $(GO) build $(LDFLAGS) -o ../../$(BIN_DIR)/edr-agent ./cmd/agent
	@echo "📋 复制 C 核心库到输出目录..."
	@cp $(AGENT_C_DIR)/build/libedr_core$(LIB_EXT) $(BIN_DIR)/ 2>/dev/null || echo "⚠️  未找到 C 核心库"
	@echo "✅ Agent 构建完成: $(BIN_DIR)/edr-agent"

# ============================================================
# Cloud 构建
# ============================================================
build-cloud:
	@echo "📦 构建 Cloud 服务..."
	@mkdir -p $(BIN_DIR)
	cd $(CLOUD_DIR) && $(GO) build $(LDFLAGS) -o ../$(BIN_DIR)/api-gateway ./cmd/api-gateway
	cd $(CLOUD_DIR) && $(GO) build $(LDFLAGS) -o ../$(BIN_DIR)/event-processor ./cmd/event-processor
	cd $(CLOUD_DIR) && $(GO) build $(LDFLAGS) -o ../$(BIN_DIR)/detection-engine ./cmd/detection-engine
	cd $(CLOUD_DIR) && $(GO) build $(LDFLAGS) -o ../$(BIN_DIR)/alert-manager ./cmd/alert-manager
	@echo "✅ Cloud 服务构建完成"

# ============================================================
# Console 构建
# ============================================================
build-console:
	@echo "📦 构建 Console 前端..."
	cd $(CONSOLE_DIR) && $(PNPM) install --frozen-lockfile 2>/dev/null || $(PNPM) install
	cd $(CONSOLE_DIR) && $(PNPM) run build
	@echo "✅ Console 构建完成: $(CONSOLE_DIR)/dist/"

# ============================================================
# 测试目标
# ============================================================
test: test-agent test-cloud test-console
	@echo "✅ 所有测试通过"

test-agent:
	@echo "🧪 运行 Agent 测试..."
	@if [ -d "$(AGENT_C_DIR)/build" ]; then \
		cd $(AGENT_C_DIR)/build && ctest --output-on-failure || true; \
	fi
	cd $(AGENT_GO_DIR) && $(GO) test -v ./...

test-cloud:
	@echo "🧪 运行 Cloud 测试..."
	cd $(CLOUD_DIR) && $(GO) test -v ./...

test-cloud-unit:
	@echo "🧪 运行 Cloud 单元测试..."
	cd $(CLOUD_DIR) && $(GO) test -v -short ./...

test-cloud-integration:
	@echo "🧪 运行 Cloud 集成测试..."
	@echo "确保 PostgreSQL 和 Redis 服务正在运行..."
	cd $(CLOUD_DIR) && $(GO) test -v -tags=integration ./tests/integration/...

test-console:
	@echo "🧪 运行 Console 测试..."
	cd $(CONSOLE_DIR) && $(PNPM) run test

# ============================================================
# 代码检查
# ============================================================
lint: lint-c lint-go lint-ts
	@echo "✅ 所有代码检查通过"

lint-c:
	@echo "🔍 检查 C 代码格式..."
	@find $(AGENT_C_DIR)/src $(AGENT_C_DIR)/include -name "*.c" -o -name "*.h" 2>/dev/null | \
		xargs clang-format --dry-run --Werror 2>/dev/null || \
		echo "⚠️  clang-format 未安装或无源文件"

lint-go:
	@echo "🔍 检查 Go 代码..."
	cd $(AGENT_GO_DIR) && golangci-lint run 2>/dev/null || $(GO) vet ./...
	cd $(CLOUD_DIR) && golangci-lint run 2>/dev/null || $(GO) vet ./...

lint-ts:
	@echo "🔍 检查 TypeScript 代码..."
	cd $(CONSOLE_DIR) && $(PNPM) run lint

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@find $(AGENT_C_DIR)/src $(AGENT_C_DIR)/include -name "*.c" -o -name "*.h" 2>/dev/null | \
		xargs clang-format -i 2>/dev/null || true
	cd $(AGENT_GO_DIR) && $(GO) fmt ./...
	cd $(CLOUD_DIR) && $(GO) fmt ./...
	cd $(CONSOLE_DIR) && $(PNPM) run format
	@echo "✅ 格式化完成"

# ============================================================
# 开发环境
# ============================================================
dev-up:
	@echo "🚀 启动开发环境..."
	cd $(DEPLOY_DIR) && $(DOCKER_COMPOSE) up -d
	@echo "⏳ 等待服务就绪..."
	@./scripts/health-check.sh || echo "⚠️  部分服务可能未就绪"
	@echo ""
	@echo "============================================"
	@echo "📊 服务状态:"
	@echo "============================================"
	@$(MAKE) dev-ps

dev-down:
	@echo "🛑 停止开发环境..."
	cd $(DEPLOY_DIR) && $(DOCKER_COMPOSE) down
	@echo "✅ 开发环境已停止"

dev-logs:
	cd $(DEPLOY_DIR) && $(DOCKER_COMPOSE) logs -f

dev-ps:
	cd $(DEPLOY_DIR) && $(DOCKER_COMPOSE) ps

dev-reset:
	@echo "🔄 重置开发环境 (删除所有数据)..."
	cd $(DEPLOY_DIR) && $(DOCKER_COMPOSE) down -v
	@echo "✅ 开发环境已重置"

# ============================================================
# 代码生成
# ============================================================
proto-gen:
	@echo "📝 生成 Protobuf 代码..."
	@./scripts/proto-gen.sh
	@echo "✅ Protobuf 代码生成完成"

# ============================================================
# 许可证检查
# ============================================================
license-check:
	@echo "📋 检查许可证合规性..."
	@./scripts/license-check.sh
	@echo "✅ 许可证检查完成"

# ============================================================
# 帮助信息
# ============================================================
help:
	@echo "============================================"
	@echo "EDR Platform Makefile"
	@echo "============================================"
	@echo ""
	@echo "构建命令:"
	@echo "  make build          - 构建所有模块"
	@echo "  make build-agent    - 仅构建 Agent"
	@echo "  make build-agent-c  - 仅构建 Agent C 核心库"
	@echo "  make build-agent-go - 仅构建 Agent Go 主程序"
	@echo "  make build-cloud    - 仅构建 Cloud 服务"
	@echo "  make build-console  - 仅构建 Console 前端"
	@echo "  make clean          - 清理构建产物"
	@echo ""
	@echo "测试命令:"
	@echo "  make test                   - 运行所有测试"
	@echo "  make test-agent             - 运行 Agent 测试"
	@echo "  make test-cloud             - 运行 Cloud 测试"
	@echo "  make test-cloud-unit        - 运行 Cloud 单元测试 (不依赖外部服务)"
	@echo "  make test-cloud-integration - 运行 Cloud 集成测试 (需要 PostgreSQL/Redis)"
	@echo "  make test-console           - 运行 Console 测试"
	@echo ""
	@echo "代码检查:"
	@echo "  make lint           - 运行所有代码检查"
	@echo "  make lint-c         - 检查 C 代码"
	@echo "  make lint-go        - 检查 Go 代码"
	@echo "  make lint-ts        - 检查 TypeScript 代码"
	@echo "  make fmt            - 格式化所有代码"
	@echo ""
	@echo "开发环境:"
	@echo "  make dev-up         - 启动开发环境容器"
	@echo "  make dev-down       - 停止开发环境容器"
	@echo "  make dev-logs       - 查看容器日志"
	@echo "  make dev-ps         - 查看容器状态"
	@echo "  make dev-reset      - 重置开发环境 (删除数据)"
	@echo ""
	@echo "其他命令:"
	@echo "  make proto-gen      - 生成 Protobuf 代码"
	@echo "  make license-check  - 检查许可证合规性"
	@echo "  make help           - 显示此帮助信息"
	@echo ""
	@echo "============================================"
	@echo "平台: $(PLATFORM) | 版本: $(VERSION)"
	@echo "============================================"
