package event

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  []kafka.Header
		expected *MessageHeaders
	}{
		{
			name: "all fields present",
			headers: []kafka.Header{
				{Key: "tenant_id", Value: []byte("tenant-001")},
				{Key: "schema_version", Value: []byte("v1")},
				{Key: "content_type", Value: []byte("application/json")},
				{Key: "trace_id", Value: []byte("trace-123")},
				{Key: "source_service", Value: []byte("agent-service")},
			},
			expected: &MessageHeaders{
				TenantID:      "tenant-001",
				SchemaVersion: "v1",
				ContentType:   "application/json",
				TraceID:       "trace-123",
				SourceService: "agent-service",
			},
		},
		{
			name: "required fields only",
			headers: []kafka.Header{
				{Key: "tenant_id", Value: []byte("tenant-002")},
				{Key: "schema_version", Value: []byte("v2")},
				{Key: "content_type", Value: []byte("application/protobuf")},
			},
			expected: &MessageHeaders{
				TenantID:      "tenant-002",
				SchemaVersion: "v2",
				ContentType:   "application/protobuf",
				TraceID:       "",
				SourceService: "",
			},
		},
		{
			name:    "empty headers",
			headers: []kafka.Header{},
			expected: &MessageHeaders{
				TenantID:      "",
				SchemaVersion: "",
				ContentType:   "",
				TraceID:       "",
				SourceService: "",
			},
		},
		{
			name: "unknown headers ignored",
			headers: []kafka.Header{
				{Key: "tenant_id", Value: []byte("tenant-003")},
				{Key: "unknown_key", Value: []byte("ignored")},
				{Key: "schema_version", Value: []byte("v1")},
				{Key: "content_type", Value: []byte("application/json")},
			},
			expected: &MessageHeaders{
				TenantID:      "tenant-003",
				SchemaVersion: "v1",
				ContentType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseHeaders(tt.headers)

			if result.TenantID != tt.expected.TenantID {
				t.Errorf("TenantID = %q, want %q", result.TenantID, tt.expected.TenantID)
			}
			if result.SchemaVersion != tt.expected.SchemaVersion {
				t.Errorf("SchemaVersion = %q, want %q", result.SchemaVersion, tt.expected.SchemaVersion)
			}
			if result.ContentType != tt.expected.ContentType {
				t.Errorf("ContentType = %q, want %q", result.ContentType, tt.expected.ContentType)
			}
			if result.TraceID != tt.expected.TraceID {
				t.Errorf("TraceID = %q, want %q", result.TraceID, tt.expected.TraceID)
			}
			if result.SourceService != tt.expected.SourceService {
				t.Errorf("SourceService = %q, want %q", result.SourceService, tt.expected.SourceService)
			}
		})
	}
}

func TestToKafkaHeaders(t *testing.T) {
	tests := []struct {
		name           string
		headers        *MessageHeaders
		expectedCount  int
		checkTraceID   bool
		checkSourceSvc bool
	}{
		{
			name: "all fields",
			headers: &MessageHeaders{
				TenantID:      "tenant-001",
				SchemaVersion: "v1",
				ContentType:   "application/json",
				TraceID:       "trace-123",
				SourceService: "agent-service",
			},
			expectedCount:  5,
			checkTraceID:   true,
			checkSourceSvc: true,
		},
		{
			name: "required fields only",
			headers: &MessageHeaders{
				TenantID:      "tenant-002",
				SchemaVersion: "v2",
				ContentType:   "application/protobuf",
			},
			expectedCount:  3,
			checkTraceID:   false,
			checkSourceSvc: false,
		},
		{
			name: "with trace_id only",
			headers: &MessageHeaders{
				TenantID:      "tenant-003",
				SchemaVersion: "v1",
				ContentType:   "application/json",
				TraceID:       "trace-456",
			},
			expectedCount:  4,
			checkTraceID:   true,
			checkSourceSvc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.headers.ToKafkaHeaders()

			if len(result) != tt.expectedCount {
				t.Errorf("header count = %d, want %d", len(result), tt.expectedCount)
			}

			// 验证必填字段
			headerMap := make(map[string]string)
			for _, h := range result {
				headerMap[h.Key] = string(h.Value)
			}

			if headerMap["tenant_id"] != tt.headers.TenantID {
				t.Errorf("tenant_id = %q, want %q", headerMap["tenant_id"], tt.headers.TenantID)
			}
			if headerMap["schema_version"] != tt.headers.SchemaVersion {
				t.Errorf("schema_version = %q, want %q", headerMap["schema_version"], tt.headers.SchemaVersion)
			}
			if headerMap["content_type"] != tt.headers.ContentType {
				t.Errorf("content_type = %q, want %q", headerMap["content_type"], tt.headers.ContentType)
			}

			// 验证可选字段
			if tt.checkTraceID {
				if headerMap["trace_id"] != tt.headers.TraceID {
					t.Errorf("trace_id = %q, want %q", headerMap["trace_id"], tt.headers.TraceID)
				}
			}
			if tt.checkSourceSvc {
				if headerMap["source_service"] != tt.headers.SourceService {
					t.Errorf("source_service = %q, want %q", headerMap["source_service"], tt.headers.SourceService)
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// 测试 ToKafkaHeaders -> ParseHeaders 往返转换
	original := &MessageHeaders{
		TenantID:      "tenant-roundtrip",
		SchemaVersion: "v1",
		ContentType:   "application/json",
		TraceID:       "trace-rt-123",
		SourceService: "test-service",
	}

	kafkaHeaders := original.ToKafkaHeaders()
	parsed := ParseHeaders(kafkaHeaders)

	if parsed.TenantID != original.TenantID {
		t.Errorf("TenantID roundtrip failed: got %q, want %q", parsed.TenantID, original.TenantID)
	}
	if parsed.SchemaVersion != original.SchemaVersion {
		t.Errorf("SchemaVersion roundtrip failed: got %q, want %q", parsed.SchemaVersion, original.SchemaVersion)
	}
	if parsed.ContentType != original.ContentType {
		t.Errorf("ContentType roundtrip failed: got %q, want %q", parsed.ContentType, original.ContentType)
	}
	if parsed.TraceID != original.TraceID {
		t.Errorf("TraceID roundtrip failed: got %q, want %q", parsed.TraceID, original.TraceID)
	}
	if parsed.SourceService != original.SourceService {
		t.Errorf("SourceService roundtrip failed: got %q, want %q", parsed.SourceService, original.SourceService)
	}
}

func TestDefaultMessageHeaders(t *testing.T) {
	tenantID := "test-tenant"
	headers := DefaultMessageHeaders(tenantID)

	if headers.TenantID != tenantID {
		t.Errorf("TenantID = %q, want %q", headers.TenantID, tenantID)
	}
	if headers.SchemaVersion != "v1" {
		t.Errorf("SchemaVersion = %q, want %q", headers.SchemaVersion, "v1")
	}
	if headers.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want %q", headers.ContentType, "application/json")
	}
}

func TestEventMessageKafkaFields(t *testing.T) {
	msg := &EventMessage{
		AgentID:   "agent-001",
		TenantID:  "tenant-001",
		Timestamp: time.Now(),
	}

	// 初始状态
	if msg.GetKafkaMessage() != nil {
		t.Error("initial kafkaMsg should be nil")
	}
	if msg.GetPartition() != 0 {
		t.Errorf("initial partition = %d, want 0", msg.GetPartition())
	}
	if msg.GetOffset() != 0 {
		t.Errorf("initial offset = %d, want 0", msg.GetOffset())
	}

	// 设置 Kafka 消息
	kafkaMsg := &kafka.Message{
		Partition: 5,
		Offset:    12345,
		Value:     []byte("test"),
	}
	msg.SetKafkaMessage(kafkaMsg)

	if msg.GetKafkaMessage() != kafkaMsg {
		t.Error("kafkaMsg not set correctly")
	}
	if msg.GetPartition() != 5 {
		t.Errorf("partition = %d, want 5", msg.GetPartition())
	}
	if msg.GetOffset() != 12345 {
		t.Errorf("offset = %d, want 12345", msg.GetOffset())
	}

	// 设置 nil
	msg.SetKafkaMessage(nil)
	if msg.GetKafkaMessage() != nil {
		t.Error("kafkaMsg should be nil after SetKafkaMessage(nil)")
	}
}
