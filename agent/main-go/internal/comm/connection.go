// Package comm 提供 Agent 与云端的通信功能
package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota // 未连接
	StateConnecting                          // 连接中
	StateConnected                           // 已连接
	StateReconnecting                        // 重连中
)

// String 返回状态的字符串表示
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReconnecting:
		return "Reconnecting"
	default:
		return "Unknown"
	}
}

// ConnConfig 连接配置
type ConnConfig struct {
	Endpoint   string        // 服务端地址
	TLSEnabled bool          // 是否启用 TLS
	CACertPath string        // CA 证书路径
	Timeout    time.Duration // 连接超时

	// mTLS 配置
	ClientCertPath string // 客户端证书路径 (mTLS)
	ClientKeyPath  string // 客户端私钥路径 (mTLS)

	// 压缩配置
	CompressionEnabled bool   // 是否启用压缩
	CompressionType    string // 压缩类型: "gzip" 或 "zstd"

	// Keepalive 配置
	KeepaliveTime    time.Duration // Ping 间隔，默认 30s
	KeepaliveTimeout time.Duration // Ping 超时，默认 10s

	// 重试配置
	RetryInitialBackoff time.Duration // 初始退避，默认 1s
	RetryMaxBackoff     time.Duration // 最大退避，默认 30s
}

// 重连配置（默认值）
const (
	defaultInitialBackoff = 1 * time.Second  // 初始退避时间
	defaultMaxBackoff     = 30 * time.Second // 最大退避时间
	backoffFactor         = 2                // 退避倍数
)

// Connection 封装 gRPC 连接
type Connection struct {
	config ConnConfig
	conn   *grpc.ClientConn
	mu     sync.RWMutex

	// 连接状态
	state   ConnectionState
	stateMu sync.RWMutex
	stopCh  chan struct{}
}

// NewConnection 创建连接实例
func NewConnection(cfg ConnConfig) *Connection {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.KeepaliveTime == 0 {
		cfg.KeepaliveTime = 30 * time.Second
	}
	if cfg.KeepaliveTimeout == 0 {
		cfg.KeepaliveTimeout = 10 * time.Second
	}
	if cfg.RetryInitialBackoff == 0 {
		cfg.RetryInitialBackoff = defaultInitialBackoff
	}
	if cfg.RetryMaxBackoff == 0 {
		cfg.RetryMaxBackoff = defaultMaxBackoff
	}
	return &Connection{
		config: cfg,
		state:  StateDisconnected,
		stopCh: make(chan struct{}),
	}
}

// Connect 建立 gRPC 连接
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateConnected && c.conn != nil {
		return nil
	}

	c.setState(StateConnecting)

	// 构建 gRPC 连接选项
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.config.KeepaliveTime,
			Timeout:             c.config.KeepaliveTimeout,
			PermitWithoutStream: true, // 无活动流时也发送 ping
		}),
	}

	// 配置压缩
	if c.config.CompressionEnabled {
		compressor := c.config.CompressionType
		if compressor == "" {
			compressor = "gzip" // 默认使用 gzip
		}
		// 注册 gzip 压缩器（由 import 自动触发）
		_ = gzip.Name
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(compressor)))
	}

	// 配置 TLS
	if c.config.TLSEnabled {
		creds, err := c.loadTLSCredentials()
		if err != nil {
			c.setState(StateDisconnected)
			return fmt.Errorf("load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 建立连接
	dialCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, c.config.Endpoint, opts...)
	if err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("dial: %w", err)
	}

	c.conn = conn
	c.setState(StateConnected)
	return nil
}

// loadTLSCredentials 加载 TLS 证书
func (c *Connection) loadTLSCredentials() (credentials.TransportCredentials, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS13, // 强制 TLS 1.3
	}

	// 加载 CA 证书
	if c.config.CACertPath != "" {
		caCert, err := os.ReadFile(c.config.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}
		config.RootCAs = certPool
	}

	// 加载客户端证书 (mTLS)
	if c.config.ClientCertPath != "" && c.config.ClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(c.config.ClientCertPath, c.config.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(config), nil
}

// setState 设置连接状态
func (c *Connection) setState(state ConnectionState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state = state
}

// GetState 获取当前连接状态
func (c *Connection) GetState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// ConnectWithRetry 带指数退避的重连
func (c *Connection) ConnectWithRetry(ctx context.Context) error {
	backoff := c.config.RetryInitialBackoff

	c.setState(StateReconnecting)

	for {
		select {
		case <-ctx.Done():
			c.setState(StateDisconnected)
			return ctx.Err()
		case <-c.stopCh:
			c.setState(StateDisconnected)
			return errors.New("connection stopped")
		default:
		}

		err := c.Connect(ctx)
		if err == nil {
			return nil
		}

		// 等待退避时间后重试
		select {
		case <-ctx.Done():
			c.setState(StateDisconnected)
			return ctx.Err()
		case <-c.stopCh:
			c.setState(StateDisconnected)
			return errors.New("connection stopped")
		case <-time.After(backoff):
		}

		// 指数增长退避时间
		backoff *= backoffFactor
		if backoff > c.config.RetryMaxBackoff {
			backoff = c.config.RetryMaxBackoff
		}
	}
}

// GetConn 获取底层 gRPC 连接
func (c *Connection) GetConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// IsConnected 检查是否已连接
func (c *Connection) IsConnected() bool {
	return c.GetState() == StateConnected
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.stopCh)
	c.setState(StateDisconnected)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
