// Package main 是 API Gateway 服务的入口点
//
// API Gateway 负责：
//   - RESTful API 接口（Console 调用）
//   - Agent gRPC 接口（Agent 连接）
//   - 请求路由和负载均衡
//   - 认证鉴权
package main

import (
"context"
"fmt"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/redis/go-redis/v9"
"go.uber.org/zap"

"github.com/houzhh15/EDR-POC/cloud/internal/asset"
"github.com/houzhh15/EDR-POC/cloud/internal/config"
"github.com/houzhh15/EDR-POC/cloud/internal/event"
grpcserver "github.com/houzhh15/EDR-POC/cloud/internal/grpc"
"github.com/houzhh15/EDR-POC/cloud/pkg/auth"
pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// 版本信息
var (
Version   = "0.1.0"
GitCommit = "unknown"
BuildTime = "unknown"
)

// eventProducerAdapter 适配器：将 event.KafkaProducer 转换为 grpc.EventProducer
type eventProducerAdapter struct {
	producer *event.KafkaProducer
	logger   *zap.Logger
}

func (a *eventProducerAdapter) ProduceBatch(ctx context.Context, events []*pb.SecurityEvent) error {
	// 转换事件格式
	msgs := make([]*event.EventMessage, 0, len(events))
	for _, evt := range events {
		msg := &event.EventMessage{
			Events: []*event.SecurityEvent{
				{
					EventID:   evt.EventId,
					EventType: evt.EventType,
					Timestamp: evt.Timestamp.AsTime(),
					Severity:  int(evt.Severity),
				},
			},
			Timestamp:  time.Now(),
			ReceivedAt: time.Now(),
		}
		msgs = append(msgs, msg)
	}
	return a.producer.ProduceBatch(ctx, msgs)
}

func (a *eventProducerAdapter) Close() error {
	return a.producer.Close()
}

// statusManagerAdapter 适配器：将 asset.RedisAgentStatusManager 转换为 grpc.AgentStatusManager
type statusManagerAdapter struct {
	manager *asset.RedisAgentStatusManager
}

func (a *statusManagerAdapter) UpdateHeartbeat(ctx context.Context, agentID, tenantID, version, hostname, osType string) error {
	return a.manager.UpdateHeartbeat(ctx, agentID, tenantID, &asset.HeartbeatInfo{
		AgentVersion: version,
		Hostname:     hostname,
		OSFamily:     osType,
		Status:       "online",
	})
}

func (a *statusManagerAdapter) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return a.manager.IsOnline(ctx, agentID)
}

func main() {
	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("API Gateway starting",
zap.String("version", Version),
zap.String("commit", GitCommit),
)

	// 加载配置
	cfg, err := config.LoadGRPCConfig("configs/grpc.yaml")
	if err != nil {
		logger.Error("Failed to load gRPC config", zap.Error(err))
		// 使用默认配置继续运行
		cfg = &config.GRPCServerConfig{
			GRPC: config.GRPCConfig{
				ListenAddr: ":9090",
			},
			TLS: config.TLSConfig{
				Enabled: false,
			},
			Kafka: config.KafkaConfig{
				Brokers: []string{"localhost:19092"},
				Topic:   "edr-events",
			},
			Redis: config.RedisConfig{
				Addr: "localhost:16379",
			},
		}
		logger.Warn("Using default configuration")
	}

	// 初始化 Kafka 事件生产者
	kafkaProducer := event.NewKafkaProducer(
cfg.Kafka.Brokers,
cfg.Kafka.Topic,
logger.Named("kafka"),
)
	defer kafkaProducer.Close()
	logger.Info("Kafka producer initialized",
zap.Strings("brokers", cfg.Kafka.Brokers),
zap.String("topic", cfg.Kafka.Topic),
)

	// 创建事件生产者适配器
	producerAdapter := &eventProducerAdapter{
		producer: kafkaProducer,
		logger:   logger.Named("producer-adapter"),
	}

	// 初始化 Redis 客户端和 Agent 状态管理器
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	agentStatusManager := asset.NewAgentStatusManager(redisClient, logger.Named("asset"))
	logger.Info("Redis agent status manager initialized", zap.String("addr", cfg.Redis.Addr))

	// 创建状态管理适配器
	statusAdapter := &statusManagerAdapter{
		manager: agentStatusManager,
	}

	// 创建 AgentService
	agentService := grpcserver.NewAgentServiceServer(
logger.Named("agent-service"),
producerAdapter,
statusAdapter,
nil, // PolicyStore - TODO: 实现
nil, // CommandQueue - TODO: 实现
nil, // 使用默认配置
)
	logger.Info("AgentService created")

	// 创建 gRPC 服务配置
	grpcConfig := &grpcserver.ServerConfig{
		ListenAddr:           cfg.GRPC.ListenAddr,
		TLSCertFile:          cfg.TLS.CertFile,
		TLSKeyFile:           cfg.TLS.KeyFile,
		TLSCAFile:            cfg.TLS.CAFile,
		MaxRecvMsgSize:       cfg.Limits.MaxRecvMsgSize,
		MaxSendMsgSize:       cfg.Limits.MaxSendMsgSize,
		MaxConcurrentStreams: cfg.Limits.MaxConcurrentStreams,
		KeepaliveTime:        time.Duration(cfg.Keepalive.Time) * time.Second,
		KeepaliveTimeout:     time.Duration(cfg.Keepalive.Timeout) * time.Second,
	}

	// 设置默认值
	if grpcConfig.ListenAddr == "" {
		grpcConfig.ListenAddr = ":9090"
	}
	if grpcConfig.MaxRecvMsgSize == 0 {
		grpcConfig.MaxRecvMsgSize = 4 * 1024 * 1024
	}
	if grpcConfig.MaxSendMsgSize == 0 {
		grpcConfig.MaxSendMsgSize = 4 * 1024 * 1024
	}
	if grpcConfig.MaxConcurrentStreams == 0 {
		grpcConfig.MaxConcurrentStreams = 1000
	}

	// 配置 JWT 密钥
	if cfg.JWT.Secret != "" {
		grpcConfig.JWTSecret = []byte(cfg.JWT.Secret)
	}

	// 创建 gRPC 服务器
	grpcServer, err := grpcserver.NewServer(
grpcConfig,
logger.Named("grpc"),
agentService,
nil, // AgentStore for auth - 可选
)
	if err != nil {
		logger.Fatal("Failed to create gRPC server", zap.Error(err))
	}

	// 启动 gRPC 服务器
	go func() {
		logger.Info("Starting gRPC server", zap.String("addr", grpcConfig.ListenAddr))
		if err := grpcServer.Start(); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// 创建 Gin 路由
	router := gin.New()
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"status":  "healthy",
"version": Version,
"services": gin.H{
"http": "running",
"grpc": "running",
},
})
})

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// Agent 相关 API
		v1.GET("/agents", func(c *gin.Context) {
// TODO: 实现 Agent 列表查询
c.JSON(http.StatusOK, gin.H{"agents": []string{}})
})

		// 告警相关 API
		v1.GET("/alerts", func(c *gin.Context) {
// TODO: 实现告警查询
c.JSON(http.StatusOK, gin.H{"alerts": []string{}})
})

		// 事件相关 API
		v1.GET("/events", func(c *gin.Context) {
// TODO: 实现事件查询
c.JSON(http.StatusOK, gin.H{"events": []string{}})
})

		// Token 生成 API
		if cfg.JWT.Secret != "" {
			tokenConfig := &auth.TokenConfig{
				Secret:           []byte(cfg.JWT.Secret),
				Issuer:           cfg.JWT.Issuer,
				ExpiresIn:        time.Duration(cfg.JWT.ExpiresIn) * time.Second,
				RefreshExpiresIn: 7 * 24 * time.Hour,
			}
			tokenManager := auth.NewTokenManager(tokenConfig)

			v1.POST("/auth/token", func(c *gin.Context) {
var req struct {
AgentID  string `json:"agent_id"`
TenantID string `json:"tenant_id"`
}
if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				token, err := tokenManager.GenerateToken(req.AgentID, req.TenantID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
"token":      token,
"expires_in": cfg.JWT.ExpiresIn,
})
			})

			v1.POST("/auth/refresh", func(c *gin.Context) {
var req struct {
Token string `json:"token"`
}
if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				newToken, err := tokenManager.RefreshToken(req.Token)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
"token":      newToken,
"expires_in": cfg.JWT.ExpiresIn,
})
			})
		}
	}

	// 创建 HTTP 服务器
	httpAddr := ":8080"
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	// 启动 HTTP 服务器
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	logger.Info("API Gateway started successfully",
zap.String("http_addr", httpAddr),
zap.String("grpc_addr", grpcConfig.ListenAddr),
)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 gRPC 服务器
	grpcServer.StopWithContext(ctx)
	logger.Info("gRPC server stopped")

	// 关闭 HTTP 服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}
	logger.Info("HTTP server stopped")

	logger.Info("API Gateway stopped")
}
