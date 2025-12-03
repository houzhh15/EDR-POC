// Package main 是 EDR Agent 的入口点
//
// EDR Agent 负责终端事件采集、本地检测和与云端通信。
// 采用 C + Go 混合架构：
//   - C 核心库 (core-c): 平台相关采集、检测引擎
//   - Go 主程序: 业务逻辑、策略管理、gRPC 通信
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
	"github.com/houzhh15/EDR-POC/agent/main-go/internal/comm"
	"github.com/houzhh15/EDR-POC/agent/main-go/internal/config"
	"github.com/houzhh15/EDR-POC/agent/main-go/internal/log"
)

// 版本信息 (由编译时注入)
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// 命令行参数
var (
	configPath = flag.String("config", "/etc/edr/agent.yaml", "配置文件路径")
	showVer    = flag.Bool("version", false, "显示版本信息")
)

func main() {
	flag.Parse()

	// 显示版本
	if *showVer {
		fmt.Printf("EDR Agent %s (commit: %s, built: %s)\n", Version, GitCommit, BuildTime)
		fmt.Printf("Core Library: %s\n", cgo.Version())
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadAndValidate(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease check:\n")
		fmt.Fprintf(os.Stderr, "  - Config file exists: %s\n", *configPath)
		fmt.Fprintf(os.Stderr, "  - Config file is valid YAML format\n")
		fmt.Fprintf(os.Stderr, "  - All required fields are present\n")
		os.Exit(1)
	}

	// 初始化日志
	if err := log.Init(log.LogConfig{
		Level:      cfg.Log.Level,
		Output:     cfg.Log.Output,
		FilePath:   cfg.Log.FilePath,
		MaxSizeMB:  cfg.Log.MaxSizeMB,
		MaxBackups: cfg.Log.MaxBackups,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease check:\n")
		fmt.Fprintf(os.Stderr, "  - Log directory is writable\n")
		fmt.Fprintf(os.Stderr, "  - Log file path: %s\n", cfg.Log.FilePath)
		os.Exit(1)
	}

	logger := log.Global()
	defer logger.Sync()

	logger.Info("EDR Agent starting",
		zap.String("version", Version),
		zap.String("commit", GitCommit),
		zap.String("config_path", *configPath),
	)

	// 打印配置摘要
	logger.Info("Configuration loaded",
		zap.String("agent.id", cfg.Agent.ID),
		zap.String("log.level", cfg.Log.Level),
		zap.String("log.output", cfg.Log.Output),
		zap.String("log.file", cfg.Log.FilePath),
		zap.String("cloud.endpoint", cfg.Cloud.Endpoint),
		zap.Int("collector.buffer_size", cfg.Collector.BufferSize),
	)

	// 初始化 C 核心库
	logger.Info("Initializing C core library...")
	if err := cgo.Init(); err != nil {
		logger.Fatal("Failed to initialize core library", 
			zap.Error(err),
			zap.String("hint", "Please ensure libedr_core.dll is in the same directory or in PATH"),
		)
	}
	defer cgo.Cleanup()

	logger.Info("Core library initialized", zap.String("core_version", cgo.Version()))

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

	// 创建事件通道
	eventChan := make(chan cgo.Event, cfg.Collector.BufferSize)

	// 启动采集器
	if err := cgo.StartCollector(eventChan); err != nil {
		logger.Fatal("Failed to start collector", zap.Error(err))
	}
	logger.Info("Collector started")

	// 判断是否启用云端连接（独立模式 vs 云端模式）
	standAloneMode := cfg.Cloud.Endpoint == ""
	
	var conn *comm.Connection // 声明在外层作用域以便 defer 使用
	
	if standAloneMode {
		logger.Info("Running in standalone mode (no cloud connection)")
		logger.Info("Events will be logged locally only")
		
		// 独立模式：只采集和记录日志，不连接云端
		// 启动本地事件处理器
		go func() {
			for {
				select {
				case event := <-eventChan:
					// 记录事件到日志（简化输出）
					logger.Info("Event captured",
						zap.Uint32("type", event.Type),
						zap.Int64("timestamp", event.Timestamp),
						zap.String("data", string(event.Data)),
					)
				case <-ctx.Done():
					return
				}
			}
		}()
	} else {
		// 云端模式：连接到云端服务
		logger.Info("Running in cloud mode", zap.String("endpoint", cfg.Cloud.Endpoint))
		
		// 创建 gRPC 连接
		conn = comm.NewConnection(comm.ConnConfig{
			Endpoint:   cfg.Cloud.Endpoint,
			TLSEnabled: cfg.Cloud.TLS.Enabled,
			CACertPath: cfg.Cloud.TLS.CACert,
			Timeout:    10 * time.Second,
		})

		// 后台重连
		go func() {
			if err := conn.ConnectWithRetry(ctx); err != nil {
				if ctx.Err() == nil {
					logger.Error("Failed to connect to cloud", zap.Error(err))
				}
			} else {
				logger.Info("Connected to cloud", zap.String("endpoint", cfg.Cloud.Endpoint))
			}
		}()

		// 启动心跳客户端
		heartbeatClient := comm.NewHeartbeatClient(conn, comm.HeartbeatConfig{
			AgentID:      cfg.Agent.ID,
			AgentVersion: Version,
			Interval:     30 * time.Second,
		}, logger.WithModule("heartbeat"))
		go heartbeatClient.Start(ctx)

		// 创建事件客户端并启动批量发送
		eventClient := comm.NewEventClient(conn, 100, 5*time.Second)
		go eventClient.StartBatchSender(ctx, eventChan)
	}

	// 等待退出
	<-ctx.Done()

	// 优雅关闭
	logger.Info("Shutting down...")

	// 停止采集器
	if err := cgo.StopCollector(); err != nil {
		logger.Error("Failed to stop collector", zap.Error(err))
	}

	// 关闭云端连接（如果存在）
	if conn != nil {
		if err := conn.Close(); err != nil {
			logger.Error("Failed to close connection", zap.Error(err))
		}
	}

	logger.Info("EDR Agent stopped")
}
