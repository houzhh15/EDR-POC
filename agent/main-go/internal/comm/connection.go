// Package comm 提供 Agent 与云端的通信功能
package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ConnConfig 连接配置
type ConnConfig struct {
	Endpoint   string        // 服务端地址
	TLSEnabled bool          // 是否启用 TLS
	CACertPath string        // CA 证书路径
	Timeout    time.Duration // 连接超时
}

// 重连配置
const (
	initialBackoff = 1 * time.Second  // 初始退避时间
	maxBackoff     = 30 * time.Second // 最大退避时间
	backoffFactor  = 2                // 退避倍数
)

// Connection 封装 gRPC 连接
type Connection struct {
	config ConnConfig
	conn   *grpc.ClientConn
	mu     sync.RWMutex

	// 连接状态
	connected bool
	stopCh    chan struct{}
}

// NewConnection 创建连接实例
func NewConnection(cfg ConnConfig) *Connection {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Connection{
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

// Connect 建立 gRPC 连接
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	// 构建 gRPC 连接选项
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // 每 30 秒发送一次 ping
			Timeout:             10 * time.Second, // ping 响应超时
			PermitWithoutStream: true,             // 无活动流时也发送 ping
		}),
	}

	// 配置 TLS
	if c.config.TLSEnabled {
		creds, err := c.loadTLSCredentials()
		if err != nil {
			return err
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
		return err
	}

	c.conn = conn
	c.connected = true
	return nil
}

// loadTLSCredentials 加载 TLS 证书
func (c *Connection) loadTLSCredentials() (credentials.TransportCredentials, error) {
	// 加载 CA 证书
	if c.config.CACertPath != "" {
		caCert, err := os.ReadFile(c.config.CACertPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}

		config := &tls.Config{
			RootCAs: certPool,
		}
		return credentials.NewTLS(config), nil
	}

	// 使用系统证书
	return credentials.NewTLS(&tls.Config{}), nil
}

// ConnectWithRetry 带指数退避的重连
func (c *Connection) ConnectWithRetry(ctx context.Context) error {
	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
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
			return ctx.Err()
		case <-c.stopCh:
			return errors.New("connection stopped")
		case <-time.After(backoff):
		}

		// 指数增长退避时间
		backoff *= backoffFactor
		if backoff > maxBackoff {
			backoff = maxBackoff
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
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.stopCh)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}
	return nil
}
