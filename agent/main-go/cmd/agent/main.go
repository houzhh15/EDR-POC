// Package main 是 EDR Agent 的入口点
//
// EDR Agent 负责终端事件采集、本地检测和与云端通信。
// 采用 C + Go 混合架构：
//   - C 核心库 (core-c): 平台相关采集、检测引擎
//   - Go 主程序: 业务逻辑、策略管理、gRPC 通信
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// 版本信息 (由编译时注入)
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("EDR Agent starting",
		zap.String("version", Version),
		zap.String("commit", GitCommit),
		zap.String("build_time", BuildTime),
	)

	// 创建上下文，监听退出信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel()
	}()

	// TODO: 初始化配置
	// cfg, err := config.Load()

	// TODO: 初始化 C 核心库
	// if err := cgo.Init(); err != nil {
	//     logger.Fatal("Failed to initialize core library", zap.Error(err))
	// }

	// TODO: 启动采集器
	// collector.Start(ctx)

	// TODO: 启动 gRPC 客户端连接云端
	// client.Connect(ctx, cfg.CloudEndpoint)

	// 等待退出
	<-ctx.Done()

	logger.Info("EDR Agent stopped")
}
