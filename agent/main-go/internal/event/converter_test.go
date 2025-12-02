package event

import (
	"testing"
	"time"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestEvent 创建测试用的ProcessEvent
func createTestEvent(eventType string) *cgo.ProcessEvent {
	exitCode := int32(0)
	event := &cgo.ProcessEvent{
		Timestamp:          time.Now(),
		EventType:          eventType,
		ProcessPID:         1234,
		ProcessPPID:        5678,
		ProcessName:        "test.exe",
		ProcessPath:        "C:\\test\\test.exe",
		ProcessCommandLine: "test.exe --arg1 --arg2",
		ProcessHash:        "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		UserName:           "testuser",
		UserDomain:         "TESTDOMAIN",
	}

	if eventType == "end" {
		event.ExitCode = &exitCode
	}

	return event
}

// TestConvertToECS_StartEvent 测试start事件转ECS
func TestConvertToECS_StartEvent(t *testing.T) {
	event := createTestEvent("start")

	ecs := ConvertToECS(event)

	// 验证必填字段
	assert.NotEmpty(t, ecs["@timestamp"])
	assert.NotNil(t, ecs["event"])
	assert.NotNil(t, ecs["process"])

	// 验证event字段
	eventMap := ecs["event"].(map[string]interface{})
	assert.Equal(t, []string{"process"}, eventMap["category"])
	assert.Equal(t, []string{"start"}, eventMap["type"])
	assert.NotEmpty(t, eventMap["created"])

	// 验证process字段
	processMap := ecs["process"].(map[string]interface{})
	assert.Equal(t, uint32(1234), processMap["pid"])
	assert.Equal(t, "test.exe", processMap["name"])
	assert.Equal(t, "C:\\test\\test.exe", processMap["executable"])
	assert.Equal(t, "test.exe --arg1 --arg2", processMap["command_line"])

	// 验证parent字段
	assert.NotNil(t, processMap["parent"])
	parentMap := processMap["parent"].(map[string]interface{})
	assert.Equal(t, uint32(5678), parentMap["pid"])

	// 验证hash字段
	assert.NotNil(t, processMap["hash"])
	hashMap := processMap["hash"].(map[string]interface{})
	assert.NotEmpty(t, hashMap["sha256"])

	// 验证user字段
	assert.NotNil(t, ecs["user"])
	userMap := ecs["user"].(map[string]interface{})
	assert.Equal(t, "testuser", userMap["name"])
	assert.Equal(t, "TESTDOMAIN", userMap["domain"])

	// start事件不应该有exit_code
	assert.Nil(t, processMap["exit_code"])
}

// TestConvertToECS_EndEvent 测试end事件转ECS
func TestConvertToECS_EndEvent(t *testing.T) {
	event := createTestEvent("end")

	ecs := ConvertToECS(event)

	// 验证event.type
	eventMap := ecs["event"].(map[string]interface{})
	assert.Equal(t, []string{"end"}, eventMap["type"])

	// end事件应该有exit_code
	processMap := ecs["process"].(map[string]interface{})
	assert.NotNil(t, processMap["exit_code"])
	assert.Equal(t, int32(0), processMap["exit_code"])
}

// TestConvertToECS_EmptyFields 测试空字段处理
func TestConvertToECS_EmptyFields(t *testing.T) {
	event := &cgo.ProcessEvent{
		Timestamp:   time.Now(),
		EventType:   "start",
		ProcessPID:  1234,
		ProcessName: "test.exe",
		// 其他字段为空
	}

	ecs := ConvertToECS(event)

	// 必填字段存在
	assert.NotNil(t, ecs["@timestamp"])
	assert.NotNil(t, ecs["event"])
	assert.NotNil(t, ecs["process"])

	processMap := ecs["process"].(map[string]interface{})

	// 空字段不应该出现
	assert.Nil(t, processMap["parent"])       // ppid=0时不添加
	assert.Nil(t, processMap["executable"])   // 空路径不添加
	assert.Nil(t, processMap["command_line"]) // 空命令行不添加
	assert.Nil(t, processMap["hash"])         // 空哈希不添加

	// user字段整个不应该存在
	assert.Nil(t, ecs["user"])
}

// TestConvertToECS_ZeroHash 测试零哈希处理
func TestConvertToECS_ZeroHash(t *testing.T) {
	event := createTestEvent("start")
	event.ProcessHash = "0000000000000000000000000000000000000000000000000000000000000000"

	ecs := ConvertToECS(event)

	processMap := ecs["process"].(map[string]interface{})
	// 零哈希不应该添加
	assert.Nil(t, processMap["hash"])
}

// TestConvertToProtobuf_StartEvent 测试Protobuf转换
func TestConvertToProtobuf_StartEvent(t *testing.T) {
	event := createTestEvent("start")
	agentID := "agent-001"
	agentVersion := "1.0.0"

	pb := ConvertToProtobuf(event, agentID, agentVersion)

	// 验证基本字段
	assert.NotZero(t, pb["timestamp"])
	assert.NotZero(t, pb["timestamp_nanos"])
	assert.Equal(t, "start", pb["event_type"])
	assert.Equal(t, uint32(1234), pb["pid"])
	assert.Equal(t, uint32(5678), pb["ppid"])
	assert.Equal(t, "test.exe", pb["process_name"])
	assert.Equal(t, "C:\\test\\test.exe", pb["executable_path"])
	assert.Equal(t, "test.exe --arg1 --arg2", pb["command_line"])
	assert.NotEmpty(t, pb["sha256"])
	assert.Equal(t, "testuser", pb["username"])
	assert.Equal(t, agentID, pb["agent_id"])
	assert.Equal(t, agentVersion, pb["agent_version"])

	// start事件不应该有exit_code
	assert.Nil(t, pb["exit_code"])
}

// TestConvertToProtobuf_EndEvent 测试Protobuf end事件
func TestConvertToProtobuf_EndEvent(t *testing.T) {
	event := createTestEvent("end")

	pb := ConvertToProtobuf(event, "agent-001", "1.0.0")

	// end事件应该有exit_code
	assert.NotNil(t, pb["exit_code"])
	assert.Equal(t, int32(0), pb["exit_code"])
}

// TestConvertBatchToECS 测试批量ECS转换
func TestConvertBatchToECS(t *testing.T) {
	events := []*cgo.ProcessEvent{
		createTestEvent("start"),
		createTestEvent("end"),
		createTestEvent("start"),
	}

	ecsArray := ConvertBatchToECS(events)

	require.Len(t, ecsArray, 3)

	// 验证第一个事件
	ecs1 := ecsArray[0]
	event1Map := ecs1["event"].(map[string]interface{})
	assert.Equal(t, []string{"start"}, event1Map["type"])

	// 验证第二个事件
	ecs2 := ecsArray[1]
	event2Map := ecs2["event"].(map[string]interface{})
	assert.Equal(t, []string{"end"}, event2Map["type"])
	process2Map := ecs2["process"].(map[string]interface{})
	assert.NotNil(t, process2Map["exit_code"])
}

// TestConvertBatchToProtobuf 测试批量Protobuf转换
func TestConvertBatchToProtobuf(t *testing.T) {
	events := []*cgo.ProcessEvent{
		createTestEvent("start"),
		createTestEvent("end"),
	}

	pbArray := ConvertBatchToProtobuf(events, "agent-001", "1.0.0")

	require.Len(t, pbArray, 2)

	// 验证所有事件都有agent信息
	for _, pb := range pbArray {
		assert.Equal(t, "agent-001", pb["agent_id"])
		assert.Equal(t, "1.0.0", pb["agent_version"])
	}
}

// TestConvertToECS_TimestampFormat 测试时间戳格式
func TestConvertToECS_TimestampFormat(t *testing.T) {
	event := createTestEvent("start")
	event.Timestamp = time.Date(2025, 12, 2, 15, 30, 45, 123456789, time.UTC)

	ecs := ConvertToECS(event)

	timestamp := ecs["@timestamp"].(string)
	// RFC3339Nano格式: 2025-12-02T15:30:45.123456789Z
	assert.Contains(t, timestamp, "2025-12-02T15:30:45")
	assert.Contains(t, timestamp, "Z")

	// 验证created字段也是相同格式
	eventMap := ecs["event"].(map[string]interface{})
	created := eventMap["created"].(string)
	assert.Equal(t, timestamp, created)
}

// BenchmarkConvertToECS 性能测试
func BenchmarkConvertToECS(b *testing.B) {
	event := createTestEvent("start")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToECS(event)
	}
}

// BenchmarkConvertToProtobuf 性能测试
func BenchmarkConvertToProtobuf(b *testing.B) {
	event := createTestEvent("start")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToProtobuf(event, "agent-001", "1.0.0")
	}
}

// BenchmarkConvertBatchToECS 批量转换性能测试
func BenchmarkConvertBatchToECS(b *testing.B) {
	events := make([]*cgo.ProcessEvent, 100)
	for i := 0; i < 100; i++ {
		events[i] = createTestEvent("start")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertBatchToECS(events)
	}
}
