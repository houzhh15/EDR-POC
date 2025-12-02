package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/houzhh15/EDR-POC/cloud/internal/grpc/interceptors"
	"github.com/houzhh15/EDR-POC/cloud/pkg/auth"
	pb "github.com/houzhh15/EDR-POC/cloud/pkg/proto/edr/v1"
)

// ServerConfig gRPC 服务器配置
type ServerConfig struct {
	ListenAddr           string        // 监听地址，默认 :9090
	TLSCertFile          string        // 服务器证书文件路径
	TLSKeyFile           string        // 服务器私钥文件路径
	TLSCAFile            string        // CA 证书文件路径
	MaxRecvMsgSize       int           // 最大接收消息大小，默认 4MB
	MaxSendMsgSize       int           // 最大发送消息大小，默认 4MB
	MaxConcurrentStreams uint32        // 最大并发流数，默认 1000
	KeepaliveTime        time.Duration // Keepalive 时间，默认 5 分钟
	KeepaliveTimeout     time.Duration // Keepalive 超时，默认 20 秒
	MaxConnectionIdle    time.Duration // 最大空闲连接时间，默认 15 分钟
	MaxConnectionAge     time.Duration // 最大连接年龄，默认 30 分钟
	JWTSecret            []byte        // JWT 签名密钥
}

// DefaultServerConfig 返回默认配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ListenAddr:           ":9090",
		MaxRecvMsgSize:       4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:       4 * 1024 * 1024, // 4MB
		MaxConcurrentStreams: 1000,
		KeepaliveTime:        5 * time.Minute,
		KeepaliveTimeout:     20 * time.Second,
		MaxConnectionIdle:    15 * time.Minute,
		MaxConnectionAge:     30 * time.Minute,
	}
}

// Server gRPC 服务器
type Server struct {
	config       *ServerConfig
	logger       *zap.Logger
	grpcServer   *grpc.Server
	agentService *AgentServiceServer
	metrics      *interceptors.GRPCMetrics
}

// NewServer 创建 gRPC 服务器
func NewServer(
	config *ServerConfig,
	logger *zap.Logger,
	agentService *AgentServiceServer,
	agentStore interceptors.AgentStore,
) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// 创建指标收集器
	metrics := interceptors.NewGRPCMetrics()

	// 构建服务器选项
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(config.MaxSendMsgSize),
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    config.KeepaliveTime,
			Timeout: config.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
	}

	// 加载 TLS 配置（如果配置了证书）
	if config.TLSCertFile != "" && config.TLSKeyFile != "" && config.TLSCAFile != "" {
		tlsConfig, err := auth.LoadServerTLSConfig(&auth.TLSConfig{
			CertFile: config.TLSCertFile,
			KeyFile:  config.TLSKeyFile,
			CAFile:   config.TLSCAFile,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	// 配置拦截器链（顺序：Recovery -> Metrics -> Logging -> Auth）
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(logger),
			metrics.UnaryInterceptor(),
			interceptors.LoggingInterceptor(logger),
			interceptors.AuthInterceptor(agentStore, config.JWTSecret),
		),
		grpc.ChainStreamInterceptor(
			interceptors.RecoveryStreamInterceptor(logger),
			metrics.StreamInterceptor(),
			interceptors.LoggingStreamInterceptor(logger),
			interceptors.AuthStreamInterceptor(agentStore, config.JWTSecret),
		),
	) // 创建 gRPC 服务器
	grpcServer := grpc.NewServer(opts...)

	// 注册 AgentService
	if agentService != nil {
		pb.RegisterAgentServiceServer(grpcServer, agentService)
	}

	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("edr.v1.AgentService", grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册 gRPC reflection 服务（用于调试和 grpcurl）
	reflection.Register(grpcServer)

	return &Server{
		config:       config,
		logger:       logger,
		grpcServer:   grpcServer,
		agentService: agentService,
		metrics:      metrics,
	}, nil
}

// Start 启动 gRPC 服务器
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}

	s.logger.Info("starting gRPC server",
		zap.String("addr", s.config.ListenAddr),
		zap.Bool("tls_enabled", s.config.TLSCertFile != ""),
	)

	return s.grpcServer.Serve(listener)
}

// Stop 优雅停止 gRPC 服务器
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server gracefully")
	s.grpcServer.GracefulStop()
}

// StopWithContext 带超时的优雅停止
func (s *Server) StopWithContext(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("gRPC server graceful stop timeout, forcing stop")
		s.grpcServer.Stop()
	}
}

// GetMetrics 获取指标收集器
func (s *Server) GetMetrics() *interceptors.GRPCMetrics {
	return s.metrics
}

// GetGRPCServer 获取底层 gRPC 服务器（用于测试）
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}
