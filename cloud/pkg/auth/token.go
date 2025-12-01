package auth

import (
"fmt"
"time"

"github.com/golang-jwt/jwt/v5"
)

// AgentClaims Agent JWT Claims
type AgentClaims struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// TokenConfig JWT 配置
type TokenConfig struct {
	Secret           []byte        // JWT 签名密钥
	Issuer           string        // 签发者
	ExpiresIn        time.Duration // 过期时间
	RefreshExpiresIn time.Duration // 刷新 Token 过期时间
}

// DefaultTokenConfig 默认 Token 配置
func DefaultTokenConfig(secret []byte) *TokenConfig {
	return &TokenConfig{
		Secret:           secret,
		Issuer:           "edr-platform",
		ExpiresIn:        24 * time.Hour,
		RefreshExpiresIn: 7 * 24 * time.Hour,
	}
}

// TokenManager Token 管理器
type TokenManager struct {
	config *TokenConfig
}

// NewTokenManager 创建 Token 管理器
func NewTokenManager(config *TokenConfig) *TokenManager {
	return &TokenManager{config: config}
}

// GenerateToken 生成 Agent JWT Token
func (m *TokenManager) GenerateToken(agentID, tenantID string) (string, error) {
	now := time.Now()
	claims := &AgentClaims{
		AgentID:  agentID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   agentID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.config.Secret)
}

// ValidateToken 验证并解析 JWT Token
func (m *TokenManager) ValidateToken(tokenString string) (*AgentClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AgentClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.config.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*AgentClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken 刷新 Token
func (m *TokenManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 生成新 Token
	return m.GenerateToken(claims.AgentID, claims.TenantID)
}

// GetSecret 获取密钥（用于拦截器）
func (m *TokenManager) GetSecret() []byte {
	return m.config.Secret
}
