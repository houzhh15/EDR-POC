package grpc

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	if config.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %s, want :9090", config.ListenAddr)
	}
	if config.MaxRecvMsgSize != 4*1024*1024 {
		t.Errorf("MaxRecvMsgSize = %d, want 4MB", config.MaxRecvMsgSize)
	}
	if config.MaxSendMsgSize != 4*1024*1024 {
		t.Errorf("MaxSendMsgSize = %d, want 4MB", config.MaxSendMsgSize)
	}
	if config.MaxConcurrentStreams != 1000 {
		t.Errorf("MaxConcurrentStreams = %d, want 1000", config.MaxConcurrentStreams)
	}
	if config.KeepaliveTime != 5*time.Minute {
		t.Errorf("KeepaliveTime = %v, want 5m", config.KeepaliveTime)
	}
	if config.KeepaliveTimeout != 20*time.Second {
		t.Errorf("KeepaliveTimeout = %v, want 20s", config.KeepaliveTimeout)
	}
	if config.MaxConnectionIdle != 15*time.Minute {
		t.Errorf("MaxConnectionIdle = %v, want 15m", config.MaxConnectionIdle)
	}
	if config.MaxConnectionAge != 30*time.Minute {
		t.Errorf("MaxConnectionAge = %v, want 30m", config.MaxConnectionAge)
	}
}

func TestNewServerWithDefaultConfig(t *testing.T) {
	logger := zap.NewNop()

	server, err := NewServer(nil, logger, nil, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.config.ListenAddr != ":9090" {
		t.Errorf("default ListenAddr = %s, want :9090", server.config.ListenAddr)
	}
}

func TestNewServerWithCustomConfig(t *testing.T) {
	logger := zap.NewNop()
	config := &ServerConfig{
		ListenAddr:           ":9080",
		MaxRecvMsgSize:       8 * 1024 * 1024,
		MaxSendMsgSize:       8 * 1024 * 1024,
		MaxConcurrentStreams: 500,
		KeepaliveTime:        10 * time.Minute,
		KeepaliveTimeout:     30 * time.Second,
		MaxConnectionIdle:    20 * time.Minute,
		MaxConnectionAge:     60 * time.Minute,
		JWTSecret:            []byte("test-secret"),
	}

	server, err := NewServer(config, logger, nil, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if server.config.ListenAddr != ":9080" {
		t.Errorf("ListenAddr = %s, want :9080", server.config.ListenAddr)
	}
}

func TestNewServerWithAgentService(t *testing.T) {
	logger := zap.NewNop()
	agentService := NewAgentServiceServer(logger, nil, nil, nil, nil, nil, nil)

	server, err := NewServer(nil, logger, agentService, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if server.agentService != agentService {
		t.Error("agentService not set correctly")
	}
}

func TestServerGetMetrics(t *testing.T) {
	logger := zap.NewNop()

	server, _ := NewServer(nil, logger, nil, nil)
	metrics := server.GetMetrics()

	if metrics == nil {
		t.Error("GetMetrics returned nil")
	}
}

func TestServerGetGRPCServer(t *testing.T) {
	logger := zap.NewNop()

	server, _ := NewServer(nil, logger, nil, nil)
	grpcServer := server.GetGRPCServer()

	if grpcServer == nil {
		t.Error("GetGRPCServer returned nil")
	}
}

func TestServerWithInvalidTLS(t *testing.T) {
	logger := zap.NewNop()
	config := &ServerConfig{
		ListenAddr:  ":9090",
		TLSCertFile: "/nonexistent/cert.pem",
		TLSKeyFile:  "/nonexistent/key.pem",
		TLSCAFile:   "/nonexistent/ca.pem",
	}

	_, err := NewServer(config, logger, nil, nil)
	if err == nil {
		t.Error("NewServer should fail with invalid TLS config")
	}
}
