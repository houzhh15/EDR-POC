package interceptors

import (
"context"
"testing"
"time"

"github.com/golang-jwt/jwt/v5"
"go.uber.org/zap"
)

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, AgentIDKey, "agent-123")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-456")

	if got := GetAgentIDFromContext(ctx); got != "agent-123" {
		t.Errorf("GetAgentIDFromContext() = %v, want %v", got, "agent-123")
	}

	if got := GetTenantIDFromContext(ctx); got != "tenant-456" {
		t.Errorf("GetTenantIDFromContext() = %v, want %v", got, "tenant-456")
	}
}

func TestContextHelpersEmpty(t *testing.T) {
	ctx := context.Background()

	if got := GetAgentIDFromContext(ctx); got != "" {
		t.Errorf("GetAgentIDFromContext() = %v, want empty", got)
	}

	if got := GetTenantIDFromContext(ctx); got != "" {
		t.Errorf("GetTenantIDFromContext() = %v, want empty", got)
	}
}

func TestValidateJWT(t *testing.T) {
	secret := []byte("test-secret-key")

	// 创建有效的 JWT
	claims := &AgentClaims{
		AgentID:  "agent-123",
		TenantID: "tenant-456",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证 token
	result, err := validateJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("validateJWT() error = %v", err)
	}

	if result.AgentID != "agent-123" {
		t.Errorf("AgentID = %v, want %v", result.AgentID, "agent-123")
	}

	if result.TenantID != "tenant-456" {
		t.Errorf("TenantID = %v, want %v", result.TenantID, "tenant-456")
	}
}

func TestValidateJWTInvalidToken(t *testing.T) {
	secret := []byte("test-secret-key")
	wrongSecret := []byte("wrong-secret-key")

	// 用错误的密钥签名
	claims := &AgentClaims{
		AgentID:  "agent-123",
		TenantID: "tenant-456",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(wrongSecret)

	// 验证应该失败
	_, err := validateJWT(tokenString, secret)
	if err == nil {
		t.Error("validateJWT() should fail with wrong secret")
	}
}

func TestValidateJWTExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key")

	// 创建过期的 JWT
	claims := &AgentClaims{
		AgentID:  "agent-123",
		TenantID: "tenant-456",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 过期
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	// 验证应该失败
	_, err := validateJWT(tokenString, secret)
	if err == nil {
		t.Error("validateJWT() should fail with expired token")
	}
}

func TestGRPCMetricsNewGRPCMetrics(t *testing.T) {
	metrics := NewGRPCMetrics()
	if metrics == nil {
		t.Error("NewGRPCMetrics() returned nil")
	}
}

func TestRecoveryInterceptorCreation(t *testing.T) {
	logger := zap.NewNop()
	interceptor := RecoveryInterceptor(logger)
	if interceptor == nil {
		t.Error("RecoveryInterceptor() returned nil")
	}
}

func TestRecoveryStreamInterceptorCreation(t *testing.T) {
	logger := zap.NewNop()
	interceptor := RecoveryStreamInterceptor(logger)
	if interceptor == nil {
		t.Error("RecoveryStreamInterceptor() returned nil")
	}
}

func TestLoggingInterceptorCreation(t *testing.T) {
	logger := zap.NewNop()
	interceptor := LoggingInterceptor(logger)
	if interceptor == nil {
		t.Error("LoggingInterceptor() returned nil")
	}
}

func TestLoggingStreamInterceptorCreation(t *testing.T) {
	logger := zap.NewNop()
	interceptor := LoggingStreamInterceptor(logger)
	if interceptor == nil {
		t.Error("LoggingStreamInterceptor() returned nil")
	}
}

func TestAuthInterceptorCreation(t *testing.T) {
	interceptor := AuthInterceptor(nil, []byte("secret"))
	if interceptor == nil {
		t.Error("AuthInterceptor() returned nil")
	}
}

func TestAuthStreamInterceptorCreation(t *testing.T) {
	interceptor := AuthStreamInterceptor(nil, []byte("secret"))
	if interceptor == nil {
		t.Error("AuthStreamInterceptor() returned nil")
	}
}

// MockAgentStore 模拟 AgentStore
type MockAgentStore struct {
	registered map[string]bool
}

func NewMockAgentStore() *MockAgentStore {
	return &MockAgentStore{
		registered: make(map[string]bool),
	}
}

func (m *MockAgentStore) IsRegistered(ctx context.Context, agentID string) bool {
	return m.registered[agentID]
}

func (m *MockAgentStore) Register(agentID string) {
	m.registered[agentID] = true
}

func TestMockAgentStore(t *testing.T) {
	store := NewMockAgentStore()

	if store.IsRegistered(context.Background(), "agent-123") {
		t.Error("agent-123 should not be registered initially")
	}

	store.Register("agent-123")

	if !store.IsRegistered(context.Background(), "agent-123") {
		t.Error("agent-123 should be registered after Register()")
	}
}
