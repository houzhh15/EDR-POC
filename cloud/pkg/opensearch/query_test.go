package opensearch
package opensearch

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNewQuery(t *testing.T) {
	qb := NewQuery()
	if qb == nil {
		t.Error("NewQuery() returned nil")
	}
}

func TestQueryBuilderTerm(t *testing.T) {
	query := NewQuery().Term("status", "active").Build()

	expected := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"status": "active",
			},
		},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Errorf("Term query = %v, want %v", query, expected)
	}
}

func TestQueryBuilderTerms(t *testing.T) {
	query := NewQuery().Terms("status", "active", "pending").Build()

	queryJSON, _ := json.Marshal(query)
	t.Logf("Terms query: %s", queryJSON)

	if query["query"] == nil {
		t.Error("query should contain 'query' key")
	}
}

func TestQueryBuilderMatch(t *testing.T) {
	query := NewQuery().Match("message", "hello world").Build()

	expected := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"message": "hello world",
			},
		},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Errorf("Match query = %v, want %v", query, expected)
	}
}

func TestQueryBuilderRange(t *testing.T) {
	query := NewQuery().
		Range("@timestamp").
		Gte("now-24h").
		Lt("now").
		Done().
		Build()

	q := query["query"].(map[string]interface{})
	rangeQ := q["range"].(map[string]interface{})
	timestampRange := rangeQ["@timestamp"].(map[string]interface{})

	if timestampRange["gte"] != "now-24h" {
		t.Errorf("gte = %v, want %v", timestampRange["gte"], "now-24h")
	}
	if timestampRange["lt"] != "now" {
		t.Errorf("lt = %v, want %v", timestampRange["lt"], "now")
	}
}

func TestQueryBuilderBool(t *testing.T) {
	query := NewQuery().
		Bool().
		Must(TermQuery("event.category", "process")).
		Filter(
			TermQuery("event.type", "start"),
			RangeGte("@timestamp", "now-1h"),
		).
		Done().
		Build()

	q := query["query"].(map[string]interface{})
	boolQ := q["bool"].(map[string]interface{})

	if boolQ["must"] == nil {
		t.Error("bool query should contain 'must'")
	}
	if boolQ["filter"] == nil {
		t.Error("bool query should contain 'filter'")
	}

	must := boolQ["must"].([]interface{})
	if len(must) != 1 {
		t.Errorf("must length = %d, want 1", len(must))
	}

	filter := boolQ["filter"].([]interface{})
	if len(filter) != 2 {
		t.Errorf("filter length = %d, want 2", len(filter))
	}
}

func TestQueryBuilderSort(t *testing.T) {
	query := NewQuery().
		Term("status", "active").
		Sort("@timestamp", Desc).
		Sort("_score", Asc).
		Build()

	sorts := query["sort"].([]interface{})
	if len(sorts) != 2 {
		t.Errorf("sort length = %d, want 2", len(sorts))
	}

	firstSort := sorts[0].(map[string]interface{})
	timestampSort := firstSort["@timestamp"].(map[string]interface{})
	if timestampSort["order"] != "desc" {
		t.Errorf("first sort order = %v, want desc", timestampSort["order"])
	}
}

func TestQueryBuilderPagination(t *testing.T) {
	query := NewQuery().
		Term("status", "active").
		From(10).
		Size(20).
		Build()

	if query["from"] != 10 {
		t.Errorf("from = %v, want 10", query["from"])
	}
	if query["size"] != 20 {
		t.Errorf("size = %v, want 20", query["size"])
	}
}

func TestQueryBuilderSource(t *testing.T) {
	query := NewQuery().
		Term("status", "active").
		Source("field1", "field2").
		SourceExcludes("field3").
		Build()

	source := query["_source"].(map[string]interface{})
	includes := source["includes"].([]string)
	excludes := source["excludes"].([]string)

	if len(includes) != 2 {
		t.Errorf("includes length = %d, want 2", len(includes))
	}
	if len(excludes) != 1 {
		t.Errorf("excludes length = %d, want 1", len(excludes))
	}
}

func TestQueryBuilderAggregation(t *testing.T) {
	query := NewQuery().
		Term("status", "active").
		Aggregation("by_category", &TermsAgg{
			Field: "event.category",
			Size:  10,
		}).
		Aggregation("events_over_time", &DateHistogramAgg{
			Field:            "@timestamp",
			CalendarInterval: "1h",
		}).
		Build()

	aggs := query["aggs"].(map[string]interface{})

	if aggs["by_category"] == nil {
		t.Error("should have 'by_category' aggregation")
	}
	if aggs["events_over_time"] == nil {
		t.Error("should have 'events_over_time' aggregation")
	}
}

func TestQueryBuilderHighlight(t *testing.T) {
	query := NewQuery().
		Match("message", "error").
		Highlight("message", "description").
		Build()

	highlight := query["highlight"].(map[string]interface{})
	fields := highlight["fields"].(map[string]interface{})

	if fields["message"] == nil {
		t.Error("should have 'message' highlight field")
	}
	if fields["description"] == nil {
		t.Error("should have 'description' highlight field")
	}
}

func TestQueryBuilderTrackTotalHits(t *testing.T) {
	query := NewQuery().
		Term("status", "active").
		TrackTotalHits(true).
		Build()

	if query["track_total_hits"] != true {
		t.Errorf("track_total_hits = %v, want true", query["track_total_hits"])
	}
}

func TestTermsAgg(t *testing.T) {
	agg := &TermsAgg{
		Field: "event.category",
		Size:  10,
		Order: map[string]string{"_count": "desc"},
	}

	result := agg.Build()
	terms := result["terms"].(map[string]interface{})

	if terms["field"] != "event.category" {
		t.Errorf("field = %v, want event.category", terms["field"])
	}
	if terms["size"] != 10 {
		t.Errorf("size = %v, want 10", terms["size"])
	}
}

func TestDateHistogramAgg(t *testing.T) {
	agg := &DateHistogramAgg{
		Field:            "@timestamp",
		CalendarInterval: "1d",
		Format:           "yyyy-MM-dd",
		TimeZone:         "UTC",
	}

	result := agg.Build()
	histogram := result["date_histogram"].(map[string]interface{})

	if histogram["field"] != "@timestamp" {
		t.Errorf("field = %v, want @timestamp", histogram["field"])
	}
	if histogram["calendar_interval"] != "1d" {
		t.Errorf("calendar_interval = %v, want 1d", histogram["calendar_interval"])
	}
}

func TestCardinalityAgg(t *testing.T) {
	agg := &CardinalityAgg{
		Field:              "host.id",
		PrecisionThreshold: 1000,
	}

	result := agg.Build()
	cardinality := result["cardinality"].(map[string]interface{})

	if cardinality["field"] != "host.id" {
		t.Errorf("field = %v, want host.id", cardinality["field"])
	}
}

func TestNestedAggregations(t *testing.T) {
	agg := &TermsAgg{
		Field: "event.category",
		Size:  10,
		Aggs: map[string]Aggregation{
			"count_by_type": &TermsAgg{
				Field: "event.type",
				Size:  5,
			},
		},
	}

	result := agg.Build()

	if result["aggs"] == nil {
		t.Error("should have nested 'aggs'")
	}
}

func TestConvenienceQueryFunctions(t *testing.T) {
	// TermQuery
	term := TermQuery("status", "active")
	if term["term"] == nil {
		t.Error("TermQuery should produce term query")
	}

	// TermsQuery
	terms := TermsQuery("status", "active", "pending")
	if terms["terms"] == nil {
		t.Error("TermsQuery should produce terms query")
	}

	// MatchQuery
	match := MatchQuery("message", "hello")
	if match["match"] == nil {
		t.Error("MatchQuery should produce match query")
	}

	// RangeGte
	rangeGte := RangeGte("@timestamp", "now-1h")
	if rangeGte["range"] == nil {
		t.Error("RangeGte should produce range query")
	}

	// RangeBetween
	rangeBetween := RangeBetween("@timestamp", "now-1h", "now")
	rangeQ := rangeBetween["range"].(map[string]interface{})
	ts := rangeQ["@timestamp"].(map[string]interface{})
	if ts["gte"] == nil || ts["lte"] == nil {
		t.Error("RangeBetween should have gte and lte")
	}

	// ExistsQuery
	exists := ExistsQuery("field")
	if exists["exists"] == nil {
		t.Error("ExistsQuery should produce exists query")
	}

	// WildcardQuery
	wildcard := WildcardQuery("path", "/var/log/*")
	if wildcard["wildcard"] == nil {
		t.Error("WildcardQuery should produce wildcard query")
	}
}

func TestComplexQuery(t *testing.T) {
	// 构建一个复杂查询示例
	query := NewQuery().
		Bool().
		Must(
			TermQuery("event.category", "process"),
		).
		Filter(
			TermQuery("event.type", "start"),
			RangeGte("@timestamp", "now-24h"),
		).
		MustNot(
			TermQuery("process.name", "system"),
		).
		Done().
		Sort("@timestamp", Desc).
		From(0).
		Size(100).
		Source("@timestamp", "event", "process", "host").
		Aggregation("top_processes", &TermsAgg{
			Field: "process.name",
			Size:  10,
		}).
		TrackTotalHits(true).
		Build()

	// 验证结构
	if query["query"] == nil {
		t.Error("should have query")
	}
	if query["sort"] == nil {
		t.Error("should have sort")
	}
	if query["from"] == nil {
		t.Error("should have from")
	}
	if query["size"] == nil {
		t.Error("should have size")
	}
	if query["_source"] == nil {
		t.Error("should have _source")
	}
	if query["aggs"] == nil {
		t.Error("should have aggs")
	}
	if query["track_total_hits"] == nil {
		t.Error("should have track_total_hits")
	}

	// 输出 JSON 便于调试
	jsonBytes, _ := json.MarshalIndent(query, "", "  ")
	t.Logf("Complex query:\n%s", jsonBytes)
}
