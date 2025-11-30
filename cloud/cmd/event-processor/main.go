// Package main 是 Event Processor 服务的入口点
//
// Event Processor 负责：
//   - 从 Kafka 消费事件
//   - 事件解析和标准化
//   - 事件存储（写入 OpenSearch）
//   - 事件关联和富化
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// 版本信息
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

	logger.Info("Event Processor starting",
		zap.String("version", Version),
		zap.String("commit", GitCommit),
	)

	// 创建上下文
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

	// TODO: 初始化 Kafka 消费者
	// consumer, err := kafka.NewConsumer(cfg.Kafka)

	// TODO: 初始化 OpenSearch 客户端
	// esClient, err := opensearch.NewClient(cfg.OpenSearch)

	// TODO: 启动事件处理循环
	// for {
	//     select {
	//     case <-ctx.Done():
	//         return
	//     case msg := <-consumer.Messages():
	//         processEvent(msg)
	//     }
	// }

	// 等待退出
	<-ctx.Done()

	logger.Info("Event Processor stopped")
}
