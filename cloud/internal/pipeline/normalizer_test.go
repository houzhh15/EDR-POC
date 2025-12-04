package pipeline

import (
	"context"
	"testing"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

func TestECSNormalizer_Normalize_ProcessCreate(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "test-001",
		AgentID:   "agent-001",
		EventType: "process_create",
		Timestamp: 1701619200000000000,
		Process: &ecs.ProcessInfo{
			PID:         1234,
			PPID:        5678,
			Name:        "notepad.exe",
			Executable:  "C:\\Windows\\notepad.exe",
			CommandLine: "notepad.exe test.txt",
		},
	}

	ecsEvent, err := normalizer.Normalize(context.Background(), evt)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if ecsEvent.ECS.Version != "8.11.0" {
		t.Errorf("ECS.Version = %s, want 8.11.0", ecsEvent.ECS.Version)
	}
	if ecsEvent.Event.ID != "test-001" {
		t.Errorf("Event.ID = %s, want test-001", ecsEvent.Event.ID)
	}
	if ecsEvent.Event.Kind != "event" {
		t.Errorf("Event.Kind = %s, want event", ecsEvent.Event.Kind)
	}
	if len(ecsEvent.Event.Category) != 1 || ecsEvent.Event.Category[0] != "process" {
		t.Errorf("Event.Category = %v, want [process]", ecsEvent.Event.Category)
	}
	if ecsEvent.Process == nil {
		t.Fatal("Process is nil")
	}
	if ecsEvent.Process.PID != 1234 {
		t.Errorf("Process.PID = %d, want 1234", ecsEvent.Process.PID)
	}
}

func TestECSNormalizer_Normalize_FileCreate(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "test-002",
		AgentID:   "agent-001",
		EventType: "file_create",
		Timestamp: 1701619200000000000,
		File: &ecs.FileInfo{
			Path:      "C:\\Users\\test\\document.txt",
			Name:      "document.txt",
			Extension: "txt",
			Size:      1024,
		},
	}

	ecsEvent, err := normalizer.Normalize(context.Background(), evt)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if len(ecsEvent.Event.Category) != 1 || ecsEvent.Event.Category[0] != "file" {
		t.Errorf("Event.Category = %v, want [file]", ecsEvent.Event.Category)
	}
	if ecsEvent.File == nil {
		t.Fatal("File is nil")
	}
	if ecsEvent.File.Name != "document.txt" {
		t.Errorf("File.Name = %s, want document.txt", ecsEvent.File.Name)
	}
}

func TestECSNormalizer_Normalize_NetworkConnect(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "test-003",
		AgentID:   "agent-001",
		EventType: "network_connect",
		Timestamp: 1701619200000000000,
		Network: &ecs.NetworkInfo{
			SourceIP:        "192.168.1.100",
			SourcePort:      54321,
			DestinationIP:   "8.8.8.8",
			DestinationPort: 443,
			Protocol:        "tcp",
		},
	}

	ecsEvent, err := normalizer.Normalize(context.Background(), evt)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if ecsEvent.Source == nil {
		t.Fatal("Source is nil")
	}
	if ecsEvent.Destination == nil {
		t.Fatal("Destination is nil")
	}
	if ecsEvent.Destination.IP != "8.8.8.8" {
		t.Errorf("Destination.IP = %s, want 8.8.8.8", ecsEvent.Destination.IP)
	}
}

func TestECSNormalizer_Normalize_WithEnrichment(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "test-004",
		AgentID:   "agent-001",
		EventType: "network_connect",
		Timestamp: 1701619200000000000,
		Network: &ecs.NetworkInfo{
			SourceIP:        "192.168.1.100",
			DestinationIP:   "8.8.8.8",
			DestinationPort: 443,
			Protocol:        "tcp",
		},
		Enrichment: &ecs.EnrichmentData{
			GeoIP: &ecs.GeoIPInfo{
				CountryCode: "US",
				CountryName: "United States",
				CityName:    "Mountain View",
				Latitude:    37.386,
				Longitude:   -122.084,
			},
			Asset: &ecs.AssetInfo{
				Hostname:   "workstation-001",
				OSFamily:   "windows",
				OSVersion:  "10.0.19041",
				Department: "Engineering",
				Tags:       []string{"critical"},
			},
			Agent: &ecs.AgentInfo{
				Version:  "1.0.0",
				Hostname: "workstation-001",
			},
		},
	}

	ecsEvent, err := normalizer.Normalize(context.Background(), evt)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	// 验证 GeoIP 被应用到 Destination
	if ecsEvent.Destination == nil || ecsEvent.Destination.Geo == nil {
		t.Fatal("Destination.Geo is nil")
	}
	if ecsEvent.Destination.Geo.CountryIsoCode != "US" {
		t.Errorf("Destination.Geo.CountryIsoCode = %s, want US", ecsEvent.Destination.Geo.CountryIsoCode)
	}

	// 验证 Asset 被应用到 Host
	if ecsEvent.Host.Hostname != "workstation-001" {
		t.Errorf("Host.Hostname = %s, want workstation-001", ecsEvent.Host.Hostname)
	}

	// 验证 Labels
	if ecsEvent.Labels == nil || ecsEvent.Labels["department"] != "Engineering" {
		t.Error("Labels[department] not set correctly")
	}

	// 验证 Agent
	if ecsEvent.Agent.Version != "1.0.0" {
		t.Errorf("Agent.Version = %s, want 1.0.0", ecsEvent.Agent.Version)
	}
}

func TestECSNormalizer_UnsupportedType(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	evt := &ecs.Event{
		ID:        "test-005",
		EventType: "unknown_type",
	}

	_, err := normalizer.Normalize(context.Background(), evt)
	if err == nil {
		t.Error("Normalize() should return error for unsupported type")
	}
}

func TestECSNormalizer_SupportedTypes(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	types := normalizer.SupportedTypes()
	if len(types) < 6 {
		t.Errorf("SupportedTypes() returned %d types, want at least 6", len(types))
	}

	expected := map[string]bool{
		"process_create":    true,
		"process_terminate": true,
		"file_create":       true,
		"file_modify":       true,
		"file_delete":       true,
		"network_connect":   true,
	}

	for _, typ := range types {
		if !expected[typ] {
			continue
		}
		delete(expected, typ)
	}

	if len(expected) > 0 {
		t.Errorf("missing expected types: %v", expected)
	}
}

func TestECSNormalizer_NilEvent(t *testing.T) {
	normalizer := NewECSNormalizer(nil)

	_, err := normalizer.Normalize(context.Background(), nil)
	if err == nil {
		t.Error("Normalize() should return error for nil event")
	}
}
