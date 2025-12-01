package asset

import (
"context"
"testing"
"time"
)

func TestStatusKey(t *testing.T) {
	tests := []struct {
		agentID  string
		expected string
	}{
		{"agent-123", "agent:status:agent-123"},
		{"agent-abc", "agent:status:agent-abc"},
	}
	for _, tt := range tests {
		result := statusKey(tt.agentID)
		if result != tt.expected {
			t.Errorf("statusKey(%s) = %s, want %s", tt.agentID, result, tt.expected)
		}
	}
}

func TestOnlineKey(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"tenant-001", "agents:online:tenant-001"},
		{"tenant-abc", "agents:online:tenant-abc"},
	}
	for _, tt := range tests {
		result := onlineKey(tt.tenantID)
		if result != tt.expected {
			t.Errorf("onlineKey(%s) = %s, want %s", tt.tenantID, result, tt.expected)
		}
	}
}

func TestHeartbeatInfo(t *testing.T) {
	info := &HeartbeatInfo{
		Hostname:     "test-host",
		IPAddress:    "192.168.1.100",
		AgentVersion: "1.0.0",
		OSFamily:     "linux",
		Status:       "online",
	}
	if info.Hostname != "test-host" {
		t.Errorf("Hostname mismatch: got %s", info.Hostname)
	}
}

func TestAgentStatus(t *testing.T) {
	status := &AgentStatus{
		AgentID:       "agent-123",
		TenantID:      "tenant-001",
		Status:        "online",
		LastHeartbeat: time.Now(),
	}
	if status.AgentID != "agent-123" {
		t.Errorf("AgentID mismatch: got %s", status.AgentID)
	}
}

// MockAgentStatusManager 模拟实现
type MockAgentStatusManager struct {
	Statuses map[string]*AgentStatus
	Online   map[string][]string
}

func NewMockAgentStatusManager() *MockAgentStatusManager {
	return &MockAgentStatusManager{
		Statuses: make(map[string]*AgentStatus),
		Online:   make(map[string][]string),
	}
}

func (m *MockAgentStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *HeartbeatInfo) error {
	m.Statuses[agentID] = &AgentStatus{
		AgentID:      agentID,
		TenantID:     tenantID,
		Status:       info.Status,
		Hostname:     info.Hostname,
		IPAddress:    info.IPAddress,
		AgentVersion: info.AgentVersion,
		OSFamily:     info.OSFamily,
	}
	found := false
	for _, id := range m.Online[tenantID] {
		if id == agentID {
			found = true
			break
		}
	}
	if !found {
		m.Online[tenantID] = append(m.Online[tenantID], agentID)
	}
	return nil
}

func (m *MockAgentStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	_, ok := m.Statuses[agentID]
	return ok, nil
}

func (m *MockAgentStatusManager) GetStatus(ctx context.Context, agentID string) (*AgentStatus, error) {
	return m.Statuses[agentID], nil
}

func (m *MockAgentStatusManager) ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error) {
	return m.Online[tenantID], nil
}

func (m *MockAgentStatusManager) CountOnlineAgents(ctx context.Context, tenantID string) (int64, error) {
	return int64(len(m.Online[tenantID])), nil
}

func TestMockAgentStatusManager(t *testing.T) {
	mock := NewMockAgentStatusManager()
	ctx := context.Background()

	info := &HeartbeatInfo{
		Hostname:     "host-1",
		IPAddress:    "10.0.0.1",
		AgentVersion: "1.0.0",
		OSFamily:     "linux",
		Status:       "online",
	}

	err := mock.UpdateHeartbeat(ctx, "agent-1", "tenant-1", info)
	if err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	online, err := mock.IsOnline(ctx, "agent-1")
	if err != nil || !online {
		t.Error("agent-1 should be online")
	}

	status, err := mock.GetStatus(ctx, "agent-1")
	if err != nil || status == nil || status.Hostname != "host-1" {
		t.Errorf("GetStatus failed or wrong hostname")
	}

	count, err := mock.CountOnlineAgents(ctx, "tenant-1")
	if err != nil || count != 1 {
		t.Errorf("CountOnlineAgents mismatch: got %d", count)
	}
}
