package auth

import (
"testing"
"time"
)

func TestDefaultTokenConfig(t *testing.T) {
	secret := []byte("test-secret")
	config := DefaultTokenConfig(secret)

	if string(config.Secret) != "test-secret" {
		t.Error("Secret mismatch")
	}
	if config.Issuer != "edr-platform" {
		t.Errorf("Issuer = %s, want edr-platform", config.Issuer)
	}
	if config.ExpiresIn != 24*time.Hour {
		t.Errorf("ExpiresIn = %v, want 24h", config.ExpiresIn)
	}
}

func TestTokenManagerGenerateAndValidate(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	config := DefaultTokenConfig(secret)
	manager := NewTokenManager(config)

	// 生成 Token
	token, err := manager.GenerateToken("agent-123", "tenant-456")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	// 验证 Token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.AgentID != "agent-123" {
		t.Errorf("AgentID = %s, want agent-123", claims.AgentID)
	}
	if claims.TenantID != "tenant-456" {
		t.Errorf("TenantID = %s, want tenant-456", claims.TenantID)
	}
}

func TestTokenManagerInvalidToken(t *testing.T) {
	secret := []byte("test-secret-key")
	config := DefaultTokenConfig(secret)
	manager := NewTokenManager(config)

	// 验证无效 Token
	_, err := manager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken should fail for invalid token")
	}
}

func TestTokenManagerWrongSecret(t *testing.T) {
	secret1 := []byte("secret-key-1")
	secret2 := []byte("secret-key-2")

	manager1 := NewTokenManager(DefaultTokenConfig(secret1))
	manager2 := NewTokenManager(DefaultTokenConfig(secret2))

	// 用 secret1 生成 Token
	token, _ := manager1.GenerateToken("agent-123", "tenant-456")

	// 用 secret2 验证应该失败
	_, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken should fail with wrong secret")
	}
}

func TestTokenManagerRefresh(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	config := DefaultTokenConfig(secret)
	manager := NewTokenManager(config)

	// 生成原始 Token
	originalToken, _ := manager.GenerateToken("agent-123", "tenant-456")

	// 刷新 Token
	newToken, err := manager.RefreshToken(originalToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("RefreshToken returned empty token")
	}

	// 验证新 Token
	claims, err := manager.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("ValidateToken for refreshed token failed: %v", err)
	}
	if claims.AgentID != "agent-123" {
		t.Errorf("AgentID = %s, want agent-123", claims.AgentID)
	}
}

func TestTokenManagerGetSecret(t *testing.T) {
	secret := []byte("my-secret-key")
	config := DefaultTokenConfig(secret)
	manager := NewTokenManager(config)

	if string(manager.GetSecret()) != "my-secret-key" {
		t.Error("GetSecret returned wrong value")
	}
}

func TestTLSConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *TLSConfig
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  &TLSConfig{},
			wantErr: true,
		},
		{
			name: "missing cert",
			config: &TLSConfig{
				KeyFile: "/path/to/key",
				CAFile:  "/path/to/ca",
			},
			wantErr: true,
		},
		{
			name: "missing key",
			config: &TLSConfig{
				CertFile: "/path/to/cert",
				CAFile:   "/path/to/ca",
			},
			wantErr: true,
		},
		{
			name: "missing ca",
			config: &TLSConfig{
				CertFile: "/path/to/cert",
				KeyFile:  "/path/to/key",
			},
			wantErr: true,
		},
		{
			name: "files not exist",
			config: &TLSConfig{
				CertFile: "/nonexistent/cert.pem",
				KeyFile:  "/nonexistent/key.pem",
				CAFile:   "/nonexistent/ca.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
err := ValidateTLSConfig(tt.config)
if (err != nil) != tt.wantErr {
t.Errorf("ValidateTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
}
})
	}
}
