package enricher

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

func TestAssetEnricher_Enrich(t *testing.T) {
	cfg := &AssetEnricherConfig{Enabled: true, CacheTTL: 5 * time.Minute}
	enricher, err := NewAssetEnricher(cfg, nil)
	if err != nil {
		t.Fatalf("NewAssetEnricher() error = %v", err)
	}

	asset := &ecs.AssetInfo{Hostname: "workstation-001", OSFamily: "windows", Department: "Engineering"}
	enricher.SetAsset("agent-001", asset)

	evt := &ecs.Event{ID: "test-001", AgentID: "agent-001"}
	err = enricher.Enrich(context.Background(), evt)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}

	if evt.Enrichment == nil || evt.Enrichment.Asset == nil {
		t.Fatal("Enrichment.Asset is nil")
	}
	if evt.Enrichment.Asset.Hostname != "workstation-001" {
		t.Errorf("Asset.Hostname = %s, want workstation-001", evt.Enrichment.Asset.Hostname)
	}
}

func TestAssetEnricher_Disabled(t *testing.T) {
	cfg := &AssetEnricherConfig{Enabled: false}
	enricher, _ := NewAssetEnricher(cfg, nil)
	if enricher.Enabled() {
		t.Error("Enabled() = true, want false")
	}
}

func TestAssetCache_GetSet(t *testing.T) {
	cache := NewAssetCache(100 * time.Millisecond)
	asset := &ecs.AssetInfo{Hostname: "test-host"}
	cache.Set("agent-001", asset)

	got, ok := cache.Get("agent-001")
	if !ok || got.Hostname != "test-host" {
		t.Error("cache Get failed")
	}

	time.Sleep(150 * time.Millisecond)
	_, ok = cache.Get("agent-001")
	if ok {
		t.Error("expired entry should not be returned")
	}
}

func TestAgentEnricher_Enrich(t *testing.T) {
	cfg := &AgentEnricherConfig{Enabled: true, CacheTTL: 5 * time.Minute}
	enricher, _ := NewAgentEnricher(cfg, nil)

	agent := &ecs.AgentInfo{Version: "1.0.0", Hostname: "workstation-001"}
	enricher.SetAgent("agent-001", agent)

	evt := &ecs.Event{ID: "test-001", AgentID: "agent-001"}
	err := enricher.Enrich(context.Background(), evt)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}

	if evt.Enrichment == nil || evt.Enrichment.Agent == nil {
		t.Fatal("Enrichment.Agent is nil")
	}
	if evt.Enrichment.Agent.Version != "1.0.0" {
		t.Errorf("Agent.Version = %s, want 1.0.0", evt.Enrichment.Agent.Version)
	}
}

func TestEnricherChain(t *testing.T) {
	assetEnricher, _ := NewAssetEnricher(&AssetEnricherConfig{Enabled: true, CacheTTL: 5 * time.Minute}, nil)
	agentEnricher, _ := NewAgentEnricher(&AgentEnricherConfig{Enabled: true, CacheTTL: 5 * time.Minute}, nil)

	assetEnricher.SetAsset("agent-001", &ecs.AssetInfo{Hostname: "workstation-001"})
	agentEnricher.SetAgent("agent-001", &ecs.AgentInfo{Version: "1.0.0"})

	chain := NewEnricherChain(assetEnricher, agentEnricher)

	evt := &ecs.Event{ID: "test-001", AgentID: "agent-001"}
	chain.Enrich(context.Background(), evt)

	if evt.Enrichment == nil || evt.Enrichment.Asset == nil || evt.Enrichment.Agent == nil {
		t.Fatal("Enrichment data is incomplete")
	}
	chain.Close()
}

func TestGeoIPEnricher_Disabled(t *testing.T) {
	cfg := &GeoIPEnricherConfig{Enabled: false}
	enricher, _ := NewGeoIPEnricher(cfg, nil)

	if enricher.Enabled() {
		t.Error("Enabled() = true, want false")
	}
	if enricher.Name() != "geoip" {
		t.Errorf("Name() = %s, want geoip", enricher.Name())
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := isPrivateIP(ip)
		if got != tt.private {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
		}
	}
}
