// Package interceptors 提供 gRPC 拦截器
package interceptors

import (
"context"
"runtime/debug"

"go.uber.org/zap"
"google.golang.org/grpc"
"google.golang.org/grpc/codes"
"google.golang.org/grpc/status"
)

// RecoveryInterceptor 创建 panic 恢复拦截器
func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
info *grpc.UnaryServerInfo,
handler grpc.UnaryHandler) (resp interface{}, err error) {

		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in gRPC handler",
zap.Any("panic", r),
zap.String("method", info.FullMethod),
zap.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor 创建流式 panic 恢复拦截器
func RecoveryStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
info *grpc.StreamServerInfo,
handler grpc.StreamHandler) (err error) {

		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in gRPC stream handler",
zap.Any("panic", r),
zap.String("method", info.FullMethod),
zap.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
