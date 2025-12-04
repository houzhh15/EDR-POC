package writer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKafkaWriterConfig_Defaults(t *testing.T) {
	cfg := &KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
	}

	writer, err := NewKafkaWriter(cfg)
	if err != nil {
		t.Fatalf("NewKafkaWriter() error = %v", err)
	}
	defer writer.Close()

	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 100*time.Millisecond {
		t.Errorf("BatchTimeout = %v, want 100ms", cfg.BatchTimeout)
	}
}

func TestKafkaWriter_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *KafkaWriterConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "empty brokers",
			cfg:     &KafkaWriterConfig{Topic: "test"},
			wantErr: true,
		},
		{
			name:    "empty topic",
			cfg:     &KafkaWriterConfig{Brokers: []string{"localhost:9092"}},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &KafkaWriterConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewKafkaWriter(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKafkaWriter() error = %v, wantErr %v", err, tt.wantErr)
			}
			if writer != nil {
				writer.Close()
			}
		})
	}
}

func TestKafkaWriter_Compression(t *testing.T) {
	tests := []struct {
		compression string
	}{
		{""},
		{"gzip"},
		{"snappy"},
		{"lz4"},
		{"zstd"},
	}

	for _, tt := range tests {
		t.Run(tt.compression, func(t *testing.T) {
			cfg := &KafkaWriterConfig{
				Brokers:     []string{"localhost:9092"},
				Topic:       "test",
				Compression: tt.compression,
			}
			writer, err := NewKafkaWriter(cfg)
			if err != nil {
				t.Fatalf("NewKafkaWriter() error = %v", err)
			}
			defer writer.Close()
		})
	}
}

func TestKafkaWriter_ClosedWriter(t *testing.T) {
	cfg := &KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
	}
	writer, _ := NewKafkaWriter(cfg)
	writer.Close()

	err := writer.Write(context.Background(), []byte("test"))
	if err == nil {
		t.Error("Write() should return error for closed writer")
	}
}

func TestOpenSearchWriter_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *OpenSearchConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "empty addresses",
			cfg:     &OpenSearchConfig{Index: "test"},
			wantErr: true,
		},
		{
			name:    "empty index",
			cfg:     &OpenSearchConfig{Addresses: []string{"http://localhost:9200"}},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &OpenSearchConfig{
				Addresses:     []string{"http://localhost:9200"},
				Index:         "test",
				FlushInterval: 0, // 禁用自动刷新以避免测试中的goroutine
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewOpenSearchWriter(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenSearchWriter() error = %v, wantErr %v", err, tt.wantErr)
			}
			if writer != nil {
				writer.Close()
			}
		})
	}
}

func TestOpenSearchWriter_IndexRotation(t *testing.T) {
	tests := []struct {
		rotation string
		contains string
	}{
		{"daily", time.Now().UTC().Format("2006.01.02")},
		{"monthly", time.Now().UTC().Format("2006.01")},
		{"", "test-index"}, // 无轮转
	}

	for _, tt := range tests {
		t.Run(tt.rotation, func(t *testing.T) {
			cfg := &OpenSearchConfig{
				Addresses:     []string{"http://localhost:9200"},
				Index:         "test-index",
				IndexRotation: tt.rotation,
				FlushInterval: 0,
			}
			writer, err := NewOpenSearchWriter(cfg)
			if err != nil {
				t.Fatalf("NewOpenSearchWriter() error = %v", err)
			}
			defer writer.Close()

			index := writer.CurrentIndex()
			if !strings.Contains(index, tt.contains) {
				t.Errorf("CurrentIndex() = %s, want contains %s", index, tt.contains)
			}
		})
	}
}

func TestOpenSearchWriter_Buffering(t *testing.T) {
	cfg := &OpenSearchConfig{
		Addresses:     []string{"http://localhost:9200"},
		Index:         "test",
		BatchSize:     10,
		FlushInterval: 0, // 禁用自动刷新
	}
	writer, err := NewOpenSearchWriter(cfg)
	if err != nil {
		t.Fatalf("NewOpenSearchWriter() error = %v", err)
	}
	defer writer.Close()

	// 写入少于BatchSize的事件，应该缓冲
	for i := 0; i < 5; i++ {
		event := []byte(`{"test": "data"}`)
		// 忽略错误，因为没有真实的服务器
		writer.buffer = append(writer.buffer, event)
	}

	if writer.BufferSize() != 5 {
		t.Errorf("BufferSize() = %d, want 5", writer.BufferSize())
	}
}

func TestOpenSearchWriter_BulkRequest(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_bulk" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/x-ndjson" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BulkResponse{
			Took:   10,
			Errors: false,
			Items:  []BulkItem{},
		})
	}))
	defer server.Close()

	cfg := &OpenSearchConfig{
		Addresses:     []string{server.URL},
		Index:         "test",
		FlushInterval: 0,
	}
	writer, err := NewOpenSearchWriter(cfg)
	if err != nil {
		t.Fatalf("NewOpenSearchWriter() error = %v", err)
	}
	defer writer.Close()

	events := [][]byte{
		[]byte(`{"event":"test1"}`),
		[]byte(`{"event":"test2"}`),
	}

	err = writer.WriteBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	// 验证请求体格式
	lines := strings.Split(strings.TrimSpace(string(receivedBody)), "\n")
	if len(lines) != 4 { // 2 events * (1 meta + 1 data)
		t.Errorf("expected 4 lines in bulk request, got %d", len(lines))
	}
}

func TestOpenSearchWriter_BulkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(BulkResponse{
			Took:   10,
			Errors: true,
			Items: []BulkItem{
				{
					Index: BulkItemResult{
						Status: 400,
						Error: &BulkError{
							Type:   "mapper_parsing_exception",
							Reason: "failed to parse",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	cfg := &OpenSearchConfig{
		Addresses:     []string{server.URL},
		Index:         "test",
		FlushInterval: 0,
		MaxRetries:    0, // 不重试
	}
	writer, err := NewOpenSearchWriter(cfg)
	if err != nil {
		t.Fatalf("NewOpenSearchWriter() error = %v", err)
	}
	defer writer.Close()

	err = writer.WriteBatch(context.Background(), [][]byte{[]byte(`{"test":"data"}`)})
	if err == nil {
		t.Error("WriteBatch() should return error for bulk errors")
	}
}

func TestOpenSearchWriter_ClosedWriter(t *testing.T) {
	cfg := &OpenSearchConfig{
		Addresses:     []string{"http://localhost:9200"},
		Index:         "test",
		FlushInterval: 0,
	}
	writer, _ := NewOpenSearchWriter(cfg)
	writer.Close()

	err := writer.Write(context.Background(), []byte("test"))
	if err == nil {
		t.Error("Write() should return error for closed writer")
	}
}

func TestMultiWriter(t *testing.T) {
	// 创建mock writers
	var writes1, writes2 int

	mock1 := &mockWriter{
		writeFn: func(ctx context.Context, event []byte) error {
			writes1++
			return nil
		},
	}
	mock2 := &mockWriter{
		writeFn: func(ctx context.Context, event []byte) error {
			writes2++
			return nil
		},
	}

	multi := NewMultiWriter(mock1, mock2)

	err := multi.Write(context.Background(), []byte("test"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if writes1 != 1 || writes2 != 1 {
		t.Errorf("writes1=%d, writes2=%d, want 1, 1", writes1, writes2)
	}

	multi.Close()
}

type mockWriter struct {
	writeFn      func(ctx context.Context, event []byte) error
	writeBatchFn func(ctx context.Context, events [][]byte) error
}

func (m *mockWriter) Write(ctx context.Context, event []byte) error {
	if m.writeFn != nil {
		return m.writeFn(ctx, event)
	}
	return nil
}

func (m *mockWriter) WriteBatch(ctx context.Context, events [][]byte) error {
	if m.writeBatchFn != nil {
		return m.writeBatchFn(ctx, events)
	}
	return nil
}

func (m *mockWriter) Close() error {
	return nil
}

func TestSerializeEvent(t *testing.T) {
	event := EventMessage{
		ID:        "test-001",
		Timestamp: time.Now(),
		Type:      "process_create",
		Data:      json.RawMessage(`{"pid": 1234}`),
	}

	data, err := SerializeEvent(event)
	if err != nil {
		t.Fatalf("SerializeEvent() error = %v", err)
	}

	var decoded EventMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != event.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, event.ID)
	}
}
