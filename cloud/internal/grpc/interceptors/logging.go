package interceptors

import (
"context"
"time"

"go.uber.org/zap"
"google.golang.org/grpc"
"google.golang.org/grpc/peer"
"google.golang.org/grpc/status"
)

// LoggingInterceptor 创建日志拦截器
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
info *grpc.UnaryServerInfo,
handler grpc.UnaryHandler) (interface{}, error) {

		start := time.Now()

		// 调用处理器
		resp, err := handler(ctx, req)

		// 获取状态码
		code := status.Code(err)
		latency := time.Since(start)

		// 获取客户端地址
		peerAddr := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			peerAddr = p.Addr.String()
		}

		// 获取认证信息
		agentID := GetAgentIDFromContext(ctx)
		tenantID := GetTenantIDFromContext(ctx)

		// 记录日志
		logger.Info("gRPC request completed",
zap.String("method", info.FullMethod),
zap.String("agent_id", agentID),
zap.String("tenant_id", tenantID),
zap.Duration("latency", latency),
zap.String("status", code.String()),
			zap.String("peer_addr", peerAddr),
		)

		return resp, err
	}
}

// LoggingStreamInterceptor 创建流式日志拦截器
func LoggingStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
info *grpc.StreamServerInfo,
handler grpc.StreamHandler) error {

		start := time.Now()

		// 获取客户端地址
		peerAddr := "unknown"
		if p, ok := peer.FromContext(ss.Context()); ok {
			peerAddr = p.Addr.String()
		}

		// 调用处理器
		err := handler(srv, ss)

		// 获取状态码
		code := status.Code(err)
		latency := time.Since(start)

		// 获取认证信息
		agentID := GetAgentIDFromContext(ss.Context())
		tenantID := GetTenantIDFromContext(ss.Context())

		// 记录日志
		logger.Info("gRPC stream completed",
zap.String("method", info.FullMethod),
zap.String("agent_id", agentID),
zap.String("tenant_id", tenantID),
zap.Duration("latency", latency),
zap.String("status", code.String()),
			zap.String("peer_addr", peerAddr),
			zap.Bool("client_stream", info.IsClientStream),
			zap.Bool("server_stream", info.IsServerStream),
		)

		return err
	}
}
