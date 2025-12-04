package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ClientOption
		wantErr bool
	}{
		{
			name: "valid config",
			opts: []ClientOption{
				WithAddresses("http://localhost:9200"),
			},
			wantErr: false,
		},
		{
			name: "multiple addresses",
			opts: []ClientOption{
				WithAddresses("http://node1:9200", "http://node2:9200"),
				WithBasicAuth("admin", "password"),
			},
			wantErr: false,
		},
		{
			name:    "empty addresses",
			opts:    []ClientOption{},
			wantErr: true,
		},
		{
			name: "with all options",
			opts: []ClientOption{
				WithAddresses("http://localhost:9200"),
				WithBasicAuth("admin", "password"),
				WithMaxRetries(5),
				WithRequestTimeout(60 * time.Second),
				WithConnectionPool(200, 20, 120*time.Second),
				WithMetrics(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Addresses: []string{"http://localhost:9200"},
			},
			wantErr: false,
		},
		{
			name: "empty addresses",
			config: &Config{
				Addresses: []string{},
			},
			wantErr: true,
		},
		{
			name: "empty address in list",
			config: &Config{
				Addresses: []string{"http://localhost:9200", ""},
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			config: &Config{
				Addresses:  []string{"http://localhost:9200"},
				MaxRetries: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cluster/health" {
			resp := ClusterHealth{
				ClusterName:                 "test-cluster",
				Status:                      "green",
				NumberOfNodes:               3,
				NumberOfDataNodes:           3,
				ActivePrimaryShards:         10,
				ActiveShards:                20,
				ActiveShardsPercentAsNumber: 100.0,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClient(WithAddresses(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if health.ClusterName != "test-cluster" {
		t.Errorf("Health().ClusterName = %v, want %v", health.ClusterName, "test-cluster")
	}
	if health.Status != "green" {
		t.Errorf("Health().Status = %v, want %v", health.Status, "green")
	}
	if !health.IsGreen() {
		t.Error("Health().IsGreen() = false, want true")
	}
}

func TestClientSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/test-index/_search" {
			resp := SearchResponse{
				Took:     5,
				TimedOut: false,
				Hits: SearchHits{
					Total: SearchTotal{Value: 100, Relation: "eq"},
					Hits: []SearchHit{
						{
							Index:  "test-index",
							ID:     "1",
							Source: json.RawMessage(`{"field": "value"}`),
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClient(WithAddresses(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	resp, err := client.Search(context.Background(), []string{"test-index"}, query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if resp.Hits.Total.Value != 100 {
		t.Errorf("Search().Hits.Total.Value = %v, want %v", resp.Hits.Total.Value, 100)
	}
	if len(resp.Hits.Hits) != 1 {
		t.Errorf("len(Search().Hits.Hits) = %v, want %v", len(resp.Hits.Hits), 1)
	}
}

func TestClientBulk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/_bulk" {
			resp := BulkResponse{
				Took:   10,
				Errors: false,
				Items: []map[string]BulkItemResponse{
					{
						"index": {
							Index:  "test-index",
							ID:     "1",
							Status: 201,
							Result: "created",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClient(WithAddresses(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	body := `{"index":{"_index":"test-index"}}
{"field":"value"}
`

	resp, err := client.Bulk(context.Background(), bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("Bulk() error = %v", err)
	}

	if resp.Errors {
		t.Error("Bulk().Errors = true, want false")
	}
	if len(resp.Items) != 1 {
		t.Errorf("len(Bulk().Items) = %v, want %v", len(resp.Items), 1)
	}
}

func TestClusterHealthStatus(t *testing.T) {
	tests := []struct {
		status   string
		isGreen  bool
		isYellow bool
		isRed    bool
	}{
		{"green", true, false, false},
		{"yellow", false, true, false},
		{"red", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			h := &ClusterHealth{Status: tt.status}
			if h.IsGreen() != tt.isGreen {
				t.Errorf("IsGreen() = %v, want %v", h.IsGreen(), tt.isGreen)
			}
			if h.IsYellow() != tt.isYellow {
				t.Errorf("IsYellow() = %v, want %v", h.IsYellow(), tt.isYellow)
			}
			if h.IsRed() != tt.isRed {
				t.Errorf("IsRed() = %v, want %v", h.IsRed(), tt.isRed)
			}
		})
	}
}
