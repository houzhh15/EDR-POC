// Package main 是 Detection Engine 服务的入口点
//
// Detection Engine 负责：
//   - 从 Kafka 消费事件
//   - 应用检测规则（Sigma/YARA）
//   - 关联分析和攻击链检测
//   - 生成告警发送到 Alert Manager
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

	logger.Info("Detection Engine starting",
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

	// TODO: 加载检测规则
	// rules, err := loadRules(cfg.RulesPath)

	// TODO: 初始化 Kafka 消费者
	// consumer, err := kafka.NewConsumer(cfg.Kafka)

	// TODO: 初始化 Alert 生产者
	// alertProducer, err := kafka.NewProducer(cfg.Kafka, "alerts")

	// TODO: 启动检测循环
	// for {
	//     select {
	//     case <-ctx.Done():
	//         return
	//     case event := <-consumer.Events():
	//         if alerts := detect(event, rules); len(alerts) > 0 {
	//             alertProducer.Send(alerts)
	//         }
	//     }
	// }

	// 等待退出
	<-ctx.Done()

	logger.Info("Detection Engine stopped")
}
