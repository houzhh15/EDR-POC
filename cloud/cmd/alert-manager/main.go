// Package main 是 Alert Manager 服务的入口点
//
// Alert Manager 负责：
//   - 接收和聚合告警
//   - 告警去重和抑制
//   - 告警通知（邮件/Webhook/企业微信等）
//   - 告警生命周期管理
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

	logger.Info("Alert Manager starting",
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

	// TODO: 初始化数据库连接
	// db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})

	// TODO: 初始化 Kafka 消费者
	// consumer, err := kafka.NewConsumer(cfg.Kafka, "alerts")

	// TODO: 初始化通知渠道
	// notifiers := initNotifiers(cfg.Notifications)

	// TODO: 启动告警处理循环
	// for {
	//     select {
	//     case <-ctx.Done():
	//         return
	//     case alert := <-consumer.Alerts():
	//         processAlert(db, alert, notifiers)
	//     }
	// }

	// 等待退出
	<-ctx.Done()

	logger.Info("Alert Manager stopped")
}
