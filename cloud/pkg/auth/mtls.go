package auth

import (
"crypto/tls"
"crypto/x509"
"fmt"
"os"
)

// TLSConfig mTLS 配置
type TLSConfig struct {
	CertFile string // 服务器证书文件路径
	KeyFile  string // 服务器私钥文件路径
	CAFile   string // CA 证书文件路径（用于验证客户端）
}

// LoadServerTLSConfig 加载服务端 mTLS 配置
// 配置 RequireAndVerifyClientCert 要求并验证客户端证书
// 配置 MinVersion 为 TLS 1.3
func LoadServerTLSConfig(config *TLSConfig) (*tls.Config, error) {
	// 加载服务器证书
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// 加载 CA 证书池
	caPool, err := loadCAPool(config.CAFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
		// 推荐的 TLS 1.3 密码套件（Go 会自动选择最佳）
		CipherSuites: nil, // TLS 1.3 不需要指定
	}, nil
}

// LoadClientTLSConfig 加载客户端 mTLS 配置
func LoadClientTLSConfig(config *TLSConfig) (*tls.Config, error) {
	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// 加载服务器 CA 证书池
	caPool, err := loadCAPool(config.CAFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// loadCAPool 加载 CA 证书池
func loadCAPool(caFile string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return pool, nil
}

// ValidateTLSConfig 验证 TLS 配置文件是否存在
func ValidateTLSConfig(config *TLSConfig) error {
	if config.CertFile == "" {
		return fmt.Errorf("cert_file is required")
	}
	if config.KeyFile == "" {
		return fmt.Errorf("key_file is required")
	}
	if config.CAFile == "" {
		return fmt.Errorf("ca_file is required")
	}

	// 检查文件是否存在
	if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("cert_file does not exist: %s", config.CertFile)
	}
	if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("key_file does not exist: %s", config.KeyFile)
	}
	if _, err := os.Stat(config.CAFile); os.IsNotExist(err) {
		return fmt.Errorf("ca_file does not exist: %s", config.CAFile)
	}

	return nil
}
