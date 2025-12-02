# Makefile for EDR Agent (Go)
# 用于编译Go层Agent和运行测试

# 变量定义
GO := go
BINARY_NAME := edr-agent
BUILD_DIR := build
C_LIB_DIR := ../core-c/build

# 平台检测
ifeq ($(OS),Windows_NT)
    BINARY_EXT := .exe
    C_LIB_NAME := edr_core.lib
else
    BINARY_EXT :=
    C_LIB_NAME := libedr_core.a
endif

# CGO配置
export CGO_ENABLED := 1
export CGO_CFLAGS := -I../core-c/include
export CGO_LDFLAGS := -L$(C_LIB_DIR) -ledr_core

# 默认目标
.PHONY: all
all: build

# 编译目标
.PHONY: build
build:
	@echo "Building EDR Agent..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) ./cmd/agent

# 测试目标
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# 测试(带覆盖率)
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# 清理目标
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...

# 静态检查
.PHONY: lint
lint:
	@echo "Running linters..."
	golangci-lint run ./...

# 运行Agent
.PHONY: run
run: build
	@echo "Starting EDR Agent..."
	$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)

# 帮助信息
.PHONY: help
help:
	@echo "EDR Agent Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all            - Build the agent (default)"
	@echo "  build          - Build the agent binary"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run static code analysis"
	@echo "  run            - Build and run the agent"
	@echo "  help           - Show this help message"
