package ecs

import (
	"testing"
)

func TestProcessCreateMapper_Map(t *testing.T) {
	mapper := &ProcessCreateMapper{}

	evt := &Event{
		ID:        "test-001",
		EventType: "process_create",
		Timestamp: 1701619200000000000,
		Process: &ProcessInfo{
			PID:         1234,
			PPID:        5678,
			Name:        "notepad.exe",
			Executable:  "C:\\Windows\\notepad.exe",
			CommandLine: "notepad.exe test.txt",
			Args:        []string{"notepad.exe", "test.txt"},
			Hash: &Hash{
				MD5:    "d41d8cd98f00b204e9800998ecf8427e",
				SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			User:      "Administrator",
			StartTime: 1701619200000000000,
		},
	}

	ecs := NewECSEvent()
	err := mapper.Map(evt, ecs)

	if err != nil {
		t.Fatalf("Map() error = %v", err)
	}

	if len(ecs.Event.Category) != 1 || ecs.Event.Category[0] != "process" {
		t.Errorf("Event.Category = %v, want [process]", ecs.Event.Category)
	}
	if len(ecs.Event.Type) != 1 || ecs.Event.Type[0] != "start" {
		t.Errorf("Event.Type = %v, want [start]", ecs.Event.Type)
	}
	if ecs.Event.Action != "process_created" {
		t.Errorf("Event.Action = %v, want process_created", ecs.Event.Action)
	}

	if ecs.Process == nil {
		t.Fatal("Process is nil")
	}
	if ecs.Process.PID != 1234 {
		t.Errorf("Process.PID = %d, want 1234", ecs.Process.PID)
	}
	if ecs.Process.Name != "notepad.exe" {
		t.Errorf("Process.Name = %s, want notepad.exe", ecs.Process.Name)
	}
	if ecs.Process.Parent == nil {
		t.Fatal("Process.Parent is nil")
	}
	if ecs.Process.Parent.PID != 5678 {
		t.Errorf("Process.Parent.PID = %d, want 5678", ecs.Process.Parent.PID)
	}
}

func TestProcessTerminateMapper_Map(t *testing.T) {
	mapper := &ProcessTerminateMapper{}

	evt := &Event{
		ID:        "test-002",
		EventType: "process_terminate",
		Timestamp: 1701619300000000000,
		Process: &ProcessInfo{
			PID:       1234,
			PPID:      5678,
			Name:      "notepad.exe",
			ExitCode:  0,
			StartTime: 1701619200000000000,
		},
	}

	ecs := NewECSEvent()
	err := mapper.Map(evt, ecs)

	if err != nil {
		t.Fatalf("Map() error = %v", err)
	}

	if len(ecs.Event.Type) != 1 || ecs.Event.Type[0] != "end" {
		t.Errorf("Event.Type = %v, want [end]", ecs.Event.Type)
	}
	if ecs.Event.Action != "process_terminated" {
		t.Errorf("Event.Action = %v, want process_terminated", ecs.Event.Action)
	}
}

func TestFileCreateMapper_Map(t *testing.T) {
	mapper := &FileCreateMapper{}

	evt := &Event{
		ID:        "test-003",
		EventType: "file_create",
		File: &FileInfo{
			Path:      "C:\\Users\\test\\document.txt",
			Name:      "document.txt",
			Extension: "txt",
			Size:      1024,
		},
	}

	ecs := NewECSEvent()
	err := mapper.Map(evt, ecs)

	if err != nil {
		t.Fatalf("Map() error = %v", err)
	}

	if len(ecs.Event.Category) != 1 || ecs.Event.Category[0] != "file" {
		t.Errorf("Event.Category = %v, want [file]", ecs.Event.Category)
	}
	if len(ecs.Event.Type) != 1 || ecs.Event.Type[0] != "creation" {
		t.Errorf("Event.Type = %v, want [creation]", ecs.Event.Type)
	}
	if ecs.File == nil {
		t.Fatal("File is nil")
	}
	if ecs.File.Name != "document.txt" {
		t.Errorf("File.Name = %s, want document.txt", ecs.File.Name)
	}
}

func TestNetworkConnectMapper_Map(t *testing.T) {
	mapper := &NetworkConnectMapper{}

	evt := &Event{
		ID:        "test-004",
		EventType: "network_connect",
		Network: &NetworkInfo{
			SourceIP:        "192.168.1.100",
			SourcePort:      54321,
			DestinationIP:   "8.8.8.8",
			DestinationPort: 443,
			Protocol:        "tcp",
			Direction:       "outbound",
			BytesSent:       1024,
			BytesReceived:   2048,
		},
		Enrichment: &EnrichmentData{
			GeoIP: &GeoIPInfo{
				CountryCode: "US",
				CountryName: "United States",
				Latitude:    37.386,
				Longitude:   -122.084,
			},
		},
	}

	ecs := NewECSEvent()
	err := mapper.Map(evt, ecs)

	if err != nil {
		t.Fatalf("Map() error = %v", err)
	}

	if ecs.Source == nil {
		t.Fatal("Source is nil")
	}
	if ecs.Source.IP != "192.168.1.100" {
		t.Errorf("Source.IP = %s, want 192.168.1.100", ecs.Source.IP)
	}

	if ecs.Destination == nil {
		t.Fatal("Destination is nil")
	}
	if ecs.Destination.IP != "8.8.8.8" {
		t.Errorf("Destination.IP = %s, want 8.8.8.8", ecs.Destination.IP)
	}

	if ecs.Destination.Geo == nil {
		t.Fatal("Destination.Geo is nil")
	}
	if ecs.Destination.Geo.CountryIsoCode != "US" {
		t.Errorf("Geo.CountryIsoCode = %s, want US", ecs.Destination.Geo.CountryIsoCode)
	}

	if ecs.Network == nil {
		t.Fatal("Network is nil")
	}
	if ecs.Network.Protocol != "tcp" {
		t.Errorf("Network.Protocol = %s, want tcp", ecs.Network.Protocol)
	}
	if ecs.Network.Bytes != 3072 {
		t.Errorf("Network.Bytes = %d, want 3072", ecs.Network.Bytes)
	}
}

func TestConvertHash(t *testing.T) {
	got := convertHash(nil)
	if got != nil {
		t.Errorf("convertHash(nil) = %v, want nil", got)
	}

	hash := &Hash{
		MD5:    "d41d8cd98f00b204e9800998ecf8427e",
		SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	got = convertHash(hash)
	if got == nil {
		t.Fatal("convertHash() = nil, want non-nil")
	}
	if got.MD5 != hash.MD5 {
		t.Errorf("MD5 = %s, want %s", got.MD5, hash.MD5)
	}
	if got.SHA256 != hash.SHA256 {
		t.Errorf("SHA256 = %s, want %s", got.SHA256, hash.SHA256)
	}
}
