package interceptors

import (
"context"
"fmt"
"strings"

"github.com/golang-jwt/jwt/v5"
"google.golang.org/grpc"
"google.golang.org/grpc/codes"
"google.golang.org/grpc/credentials"
"google.golang.org/grpc/metadata"
"google.golang.org/grpc/peer"
"google.golang.org/grpc/status"
)

// Context key 类型
type contextKey string

const (
AgentIDKey  contextKey = "agent_id"
TenantIDKey contextKey = "tenant_id"
)

// AgentClaims JWT Claims 结构
type AgentClaims struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// AgentStore Agent 存储接口
type AgentStore interface {
	IsRegistered(ctx context.Context, agentID string) bool
}

// GetAgentIDFromContext 从 context 获取 agent_id
func GetAgentIDFromContext(ctx context.Context) string {
	if v := ctx.Value(AgentIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetTenantIDFromContext 从 context 获取 tenant_id
func GetTenantIDFromContext(ctx context.Context) string {
	if v := ctx.Value(TenantIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// AuthInterceptor 创建 mTLS + JWT 认证拦截器
func AuthInterceptor(agentStore AgentStore, jwtSecret []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
info *grpc.UnaryServerInfo,
handler grpc.UnaryHandler) (interface{}, error) {

		// 跳过健康检查
		if strings.Contains(info.FullMethod, "Health") {
			return handler(ctx, req)
		}

		newCtx, err := authenticate(ctx, agentStore, jwtSecret)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// AuthStreamInterceptor 创建流式 mTLS + JWT 认证拦截器
func AuthStreamInterceptor(agentStore AgentStore, jwtSecret []byte) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
info *grpc.StreamServerInfo,
handler grpc.StreamHandler) error {

		// 跳过健康检查
		if strings.Contains(info.FullMethod, "Health") {
			return handler(srv, ss)
		}

		newCtx, err := authenticate(ss.Context(), agentStore, jwtSecret)
		if err != nil {
			return err
		}

		wrapped := &wrappedServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}

// authenticate 执行认证逻辑
func authenticate(ctx context.Context, agentStore AgentStore, jwtSecret []byte) (context.Context, error) {
	// 1. 获取 TLS 连接信息
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no peer info")
	}

	// 2. 提取客户端证书
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok || len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no client certificate")
	}

	// 3. 解析 Agent ID (证书 CN 格式: agent-{uuid})
	cert := tlsInfo.State.PeerCertificates[0]
	certAgentID := strings.TrimPrefix(cert.Subject.CommonName, "agent-")

	// 4. 提取并验证 JWT Token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 || !strings.HasPrefix(authHeader[0], "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	claims, err := validateJWT(tokenString, jwtSecret)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	// 5. 验证证书 agent_id 与 token agent_id 一致
	if claims.AgentID != certAgentID {
		return nil, status.Error(codes.PermissionDenied, "agent_id mismatch")
	}

	// 6. 验证 Agent 注册状态 (如果提供了 agentStore)
	if agentStore != nil && !agentStore.IsRegistered(ctx, claims.AgentID) {
		return nil, status.Error(codes.PermissionDenied, "agent not registered")
	}

	// 7. 注入身份信息到 Context
	ctx = context.WithValue(ctx, AgentIDKey, claims.AgentID)
	ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)

	return ctx, nil
}

// validateJWT 验证 JWT Token
func validateJWT(tokenString string, secret []byte) (*AgentClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AgentClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AgentClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// wrappedServerStream 包装 ServerStream 以注入新的 context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
