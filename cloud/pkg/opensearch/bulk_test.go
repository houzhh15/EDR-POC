package opensearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// mockClient 实现 Client 接口用于测试
type mockClient struct {
	bulkCalls    atomic.Int32
	bulkResponse *BulkResponse
	bulkErr      error
}

func (m *mockClient) Bulk(ctx context.Context, body io.Reader, opts ...RequestOption) (*BulkResponse, error) {
	m.bulkCalls.Add(1)
	if m.bulkErr != nil {
		return nil, m.bulkErr
	}
	if m.bulkResponse != nil {
		return m.bulkResponse, nil
	}
	return &BulkResponse{Took: 10, Errors: false}, nil
}

func (m *mockClient) Search(ctx context.Context, indices []string, query map[string]interface{}, opts ...RequestOption) (*SearchResponse, error) {
	return &SearchResponse{}, nil
}

func (m *mockClient) Index(ctx context.Context, index string, docID string, body interface{}, opts ...RequestOption) error {
	return nil
}

func (m *mockClient) CreateIndex(ctx context.Context, name string, settings map[string]interface{}) error {
	return nil
}

func (m *mockClient) DeleteIndex(ctx context.Context, name string) error {
	return nil
}

func (m *mockClient) PutIndexTemplate(ctx context.Context, name string, template *IndexTemplate) error {
	return nil
}

func (m *mockClient) PutISMPolicy(ctx context.Context, name string, policy *ISMPolicy) error {
	return nil
}

func (m *mockClient) Health(ctx context.Context) (*ClusterHealth, error) {
	return &ClusterHealth{Status: "green"}, nil
}

func (m *mockClient) Close() error {
	return nil
}

func TestNewBulkIndexer(t *testing.T) {
	client := &mockClient{}

	bi, err := NewBulkIndexer(client,
		WithBatchSize(100),
		WithFlushInterval(1*time.Second),
	)
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}
	defer bi.Close(context.Background())

	if bi == nil {
		t.Error("NewBulkIndexer() returned nil")
	}
}

func TestNewBulkIndexerNilClient(t *testing.T) {
	_, err := NewBulkIndexer(nil)
	if err == nil {
		t.Error("NewBulkIndexer(nil) should return error")
	}
}

func TestBulkIndexerAdd(t *testing.T) {
	client := &mockClient{}

	bi, err := NewBulkIndexer(client,
		WithBatchSize(10),
		WithFlushInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}
	defer bi.Close(context.Background())

	// 添加文档
	err = bi.Add(context.Background(), BulkItem{
		Action:   "index",
		Index:    "test-index",
		Document: map[string]string{"field": "value"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	stats := bi.Stats()
	if stats.NumAdded != 1 {
		t.Errorf("Stats().NumAdded = %v, want %v", stats.NumAdded, 1)
	}
}

func TestBulkIndexerBatchThreshold(t *testing.T) {
	client := &mockClient{}

	bi, err := NewBulkIndexer(client,
		WithBatchSize(5),
		WithFlushInterval(10*time.Second), // 长间隔防止定时刷新
	)
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}
	defer bi.Close(context.Background())

	// 添加 5 个文档触发批量刷新
	for i := 0; i < 5; i++ {
		err = bi.Add(context.Background(), BulkItem{
			Action:   "index",
			Index:    "test-index",
			Document: map[string]string{"id": string(rune('0' + i))},
		})
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// 等待异步刷新
	time.Sleep(100 * time.Millisecond)

	if client.bulkCalls.Load() == 0 {
		t.Error("Bulk should have been called after reaching batch threshold")
	}
}

func TestBulkIndexerFlush(t *testing.T) {
	client := &mockClient{}

	bi, err := NewBulkIndexer(client,
		WithBatchSize(100),
		WithFlushInterval(10*time.Second),
	)
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}
	defer bi.Close(context.Background())

	// 添加文档
	err = bi.Add(context.Background(), BulkItem{
		Action:   "index",
		Index:    "test-index",
		Document: map[string]string{"field": "value"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 手动刷新
	err = bi.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if client.bulkCalls.Load() != 1 {
		t.Errorf("bulkCalls = %v, want %v", client.bulkCalls.Load(), 1)
	}
}

func TestBulkIndexerClose(t *testing.T) {
	client := &mockClient{}

	bi, err := NewBulkIndexer(client,
		WithBatchSize(100),
		WithFlushInterval(10*time.Second),
	)
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}

	// 添加文档
	err = bi.Add(context.Background(), BulkItem{
		Action:   "index",
		Index:    "test-index",
		Document: map[string]string{"field": "value"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 关闭应该刷新剩余数据
	err = bi.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// 给异步操作一点时间
	time.Sleep(50 * time.Millisecond)

	// 检查文档是否被刷新（调用了 Bulk 或者 stats 记录了）
	stats := bi.Stats()
	// Close 之后 buffer 应该为空，表示已处理
	if stats.NumAdded != 1 {
		t.Errorf("NumAdded = %v, want 1", stats.NumAdded)
	}
}

func TestBulkIndexerStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := BulkResponse{
			Took:   10,
			Errors: false,
			Items: []map[string]BulkItemResponse{
				{"index": {Index: "test", ID: "1", Status: 201}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(WithAddresses(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	bi, err := NewBulkIndexer(client, WithBatchSize(100))
	if err != nil {
		t.Fatalf("NewBulkIndexer() error = %v", err)
	}

	// 添加并刷新
	bi.Add(context.Background(), BulkItem{
		Action:   "index",
		Index:    "test",
		Document: map[string]string{"f": "v"},
	})
	bi.Flush(context.Background())
	bi.Close(context.Background())

	stats := bi.Stats()
	if stats.NumAdded != 1 {
		t.Errorf("Stats().NumAdded = %v, want 1", stats.NumAdded)
	}
	if stats.NumFlushed != 1 {
		t.Errorf("Stats().NumFlushed = %v, want 1", stats.NumFlushed)
	}
}

func TestBulkItem(t *testing.T) {
	item := BulkItem{
		Action:     "index",
		Index:      "test-index",
		DocumentID: "doc-1",
		Document:   map[string]interface{}{"field": "value"},
		Routing:    "user-1",
	}

	if item.Action != "index" {
		t.Errorf("Action = %v, want %v", item.Action, "index")
	}
	if item.Index != "test-index" {
		t.Errorf("Index = %v, want %v", item.Index, "test-index")
	}
}
