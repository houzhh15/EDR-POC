package opensearch
package opensearch

import (
	"context"
	"testing"
	"time"
)

func TestNewIndexManager(t *testing.T) {
	client := &mockClient{}
	im := NewIndexManager(client)

	if im == nil {
		t.Error("NewIndexManager() returned nil")
	}
}

func TestCreateTimeBasedIndex(t *testing.T) {
	client := &mockClient{}
	im := NewIndexManager(client)

	timestamp := time.Date(2024, 12, 3, 10, 0, 0, 0, time.UTC)
	indexName, err := im.CreateTimeBasedIndex(context.Background(), "edr-events", timestamp)

	if err != nil {
		t.Fatalf("CreateTimeBasedIndex() error = %v", err)
	}

	expected := "edr-events-2024.12.03"
	if indexName != expected {
		t.Errorf("indexName = %v, want %v", indexName, expected)
	}
}

func TestNewEventsIndexTemplate(t *testing.T) {
	template := NewEventsIndexTemplate()

	if template == nil {
		t.Fatal("NewEventsIndexTemplate() returned nil")
	}

	if len(template.IndexPatterns) != 1 || template.IndexPatterns[0] != "edr-events-*" {
		t.Errorf("IndexPatterns = %v, want [edr-events-*]", template.IndexPatterns)
	}

	if template.Priority != 100 {
		t.Errorf("Priority = %v, want 100", template.Priority)
	}

	if template.Template == nil {
		t.Error("Template should not be nil")
	}

	settings := template.Template.Settings
	if settings["number_of_shards"] != 3 {
		t.Errorf("number_of_shards = %v, want 3", settings["number_of_shards"])
	}

	mappings := template.Template.Mappings
	if mappings["properties"] == nil {
		t.Error("mappings should have properties")
	}

	props := mappings["properties"].(map[string]interface{})
	if props["@timestamp"] == nil {
		t.Error("should have @timestamp field")
	}
	if props["event"] == nil {
		t.Error("should have event field")
	}
	if props["host"] == nil {
		t.Error("should have host field")
	}
	if props["process"] == nil {
		t.Error("should have process field")
	}
}

func TestNewAlertsIndexTemplate(t *testing.T) {
	template := NewAlertsIndexTemplate()

	if template == nil {
		t.Fatal("NewAlertsIndexTemplate() returned nil")
	}

	if len(template.IndexPatterns) != 1 || template.IndexPatterns[0] != "edr-alerts-*" {
		t.Errorf("IndexPatterns = %v, want [edr-alerts-*]", template.IndexPatterns)
	}

	settings := template.Template.Settings
	if settings["number_of_shards"] != 1 {
		t.Errorf("alerts should have 1 shard, got %v", settings["number_of_shards"])
	}
}

func TestNewAssetsIndexTemplate(t *testing.T) {
	template := NewAssetsIndexTemplate()

	if template == nil {
		t.Fatal("NewAssetsIndexTemplate() returned nil")
	}

	if len(template.IndexPatterns) != 1 || template.IndexPatterns[0] != "edr-assets" {
		t.Errorf("IndexPatterns = %v, want [edr-assets]", template.IndexPatterns)
	}
}

func TestNewEDRLifecyclePolicy(t *testing.T) {
	policy := NewEDRLifecyclePolicy()

	if policy == nil {
		t.Fatal("NewEDRLifecyclePolicy() returned nil")
	}

	if policy.DefaultState != "hot" {
		t.Errorf("DefaultState = %v, want hot", policy.DefaultState)
	}

	if len(policy.States) != 4 {
		t.Errorf("States length = %d, want 4 (hot, warm, cold, delete)", len(policy.States))
	}

	// 验证状态顺序
	expectedStates := []string{"hot", "warm", "cold", "delete"}
	for i, state := range policy.States {
		if state.Name != expectedStates[i] {
			t.Errorf("State[%d] = %v, want %v", i, state.Name, expectedStates[i])
		}
	}

	// 验证 hot 状态有 rollover
	hotState := policy.States[0]
	if len(hotState.Actions) == 0 || hotState.Actions[0].Rollover == nil {
		t.Error("hot state should have rollover action")
	}

	// 验证 delete 状态有 delete 动作
	deleteState := policy.States[3]
	if len(deleteState.Actions) == 0 || deleteState.Actions[0].Delete == nil {
		t.Error("delete state should have delete action")
	}

	// 验证 ISM 模板
	if len(policy.ISMTemplate) == 0 {
		t.Error("should have ISM template")
	}
	if policy.ISMTemplate[0].IndexPatterns[0] != "edr-events-*" {
		t.Errorf("ISMTemplate pattern = %v, want edr-events-*", policy.ISMTemplate[0].IndexPatterns[0])
	}
}

func TestRolloverConditions(t *testing.T) {
	cond := RolloverConditions{
		MaxAge:  "7d",
		MaxDocs: 10000000,
		MaxSize: "50gb",
	}

	m := cond.toMap()

	if m["max_age"] != "7d" {
		t.Errorf("max_age = %v, want 7d", m["max_age"])
	}
	if m["max_docs"] != int64(10000000) {
		t.Errorf("max_docs = %v, want 10000000", m["max_docs"])
	}
	if m["max_size"] != "50gb" {
		t.Errorf("max_size = %v, want 50gb", m["max_size"])
	}
}

func TestRolloverConditionsEmpty(t *testing.T) {
	cond := RolloverConditions{}
	m := cond.toMap()

	if len(m) != 0 {
		t.Errorf("empty conditions should produce empty map, got %v", m)
	}
}

func TestIndexStats(t *testing.T) {
	stats := &IndexStats{
		DocsCount:      1000000,
		DocsDeleted:    100,
		StoreSizeBytes: 1073741824, // 1GB
		IndexCount:     5,
		Indices: map[string]SingleIndexStats{
			"edr-events-2024.12.01": {DocsCount: 200000, StoreSizeBytes: 200000000},
			"edr-events-2024.12.02": {DocsCount: 300000, StoreSizeBytes: 300000000},
		},
	}

	if stats.DocsCount != 1000000 {
		t.Errorf("DocsCount = %v, want 1000000", stats.DocsCount)
	}

	if len(stats.Indices) != 2 {
		t.Errorf("Indices count = %d, want 2", len(stats.Indices))
	}
}

func TestParseIndexDate(t *testing.T) {
	tests := []struct {
		indexName string
		wantErr   bool
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{"edr-events-2024.12.03", false, 2024, 12, 3},
		{"edr-events-2024.01.15", false, 2024, 1, 15},
		{"logs-2024.12", false, 2024, 12, 1}, // monthly
		{"invalid-index", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.indexName, func(t *testing.T) {
			date, err := parseIndexDate(tt.indexName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIndexDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if date.Year() != tt.wantYear {
					t.Errorf("Year = %v, want %v", date.Year(), tt.wantYear)
				}
			}
		})
	}
}

func TestISMPolicyStructure(t *testing.T) {
	policy := &ISMPolicy{
		Description:  "Test policy",
		DefaultState: "hot",
		States: []ISMState{
			{
				Name: "hot",
				Actions: []ISMAction{
					{
						Rollover: &RolloverAction{
							MinDocCount: 1000000,
							MinSize:     "10gb",
						},
					},
				},
				Transitions: []ISMTransition{
					{
						StateName: "warm",
						Conditions: &ISMConditions{
							MinIndexAge: "7d",
						},
					},
				},
			},
		},
	}

	if policy.Description != "Test policy" {
		t.Errorf("Description = %v, want 'Test policy'", policy.Description)
	}

	if len(policy.States) != 1 {
		t.Errorf("States length = %d, want 1", len(policy.States))
	}

	hotState := policy.States[0]
	if hotState.Actions[0].Rollover.MinDocCount != 1000000 {
		t.Errorf("MinDocCount = %v, want 1000000", hotState.Actions[0].Rollover.MinDocCount)
	}
}
