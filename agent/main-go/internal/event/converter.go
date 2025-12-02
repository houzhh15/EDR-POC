package event

import (
	"time"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/cgo"
)

// ConvertToECS 将ProcessEvent转换为ECS格式
// 遵循Elastic Common Schema标准
func ConvertToECS(event *cgo.ProcessEvent) map[string]interface{} {
	ecs := map[string]interface{}{
		"@timestamp": event.Timestamp.Format(time.RFC3339Nano),
		"event": map[string]interface{}{
			"category": []string{"process"},
			"type":     []string{event.EventType},
			"created":  event.Timestamp.Format(time.RFC3339Nano),
		},
		"process": map[string]interface{}{
			"pid":  event.ProcessPID,
			"name": event.ProcessName,
		},
	}

	// 添加父进程信息
	if event.ProcessPPID != 0 {
		ecs["process"].(map[string]interface{})["parent"] = map[string]interface{}{
			"pid": event.ProcessPPID,
		}
	}

	// 添加可执行文件路径
	if event.ProcessPath != "" {
		ecs["process"].(map[string]interface{})["executable"] = event.ProcessPath
	}

	// 添加命令行
	if event.ProcessCommandLine != "" {
		ecs["process"].(map[string]interface{})["command_line"] = event.ProcessCommandLine
	}

	// 添加哈希
	if event.ProcessHash != "" && event.ProcessHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		ecs["process"].(map[string]interface{})["hash"] = map[string]interface{}{
			"sha256": event.ProcessHash,
		}
	}

	// 添加用户信息
	if event.UserName != "" {
		ecs["user"] = map[string]interface{}{
			"name": event.UserName,
		}
		if event.UserDomain != "" {
			ecs["user"].(map[string]interface{})["domain"] = event.UserDomain
		}
	}

	// 添加退出码(仅end事件)
	if event.ExitCode != nil {
		ecs["process"].(map[string]interface{})["exit_code"] = *event.ExitCode
	}

	return ecs
}

// ConvertToProtobuf 将ProcessEvent转换为Protobuf格式
// 注意: 这里提供接口框架,实际需要根据proto定义实现
func ConvertToProtobuf(event *cgo.ProcessEvent, agentID string, agentVersion string) map[string]interface{} {
	// 实际项目中应该返回 *pb.ProcessEvent
	// 简化实现,返回map格式
	pb := map[string]interface{}{
		"timestamp":       event.Timestamp.Unix(),
		"timestamp_nanos": event.Timestamp.UnixNano(),
		"event_type":      event.EventType,
		"pid":             event.ProcessPID,
		"ppid":            event.ProcessPPID,
		"process_name":    event.ProcessName,
		"executable_path": event.ProcessPath,
		"command_line":    event.ProcessCommandLine,
		"sha256":          event.ProcessHash,
		"username":        event.UserName,
		"agent_id":        agentID,
		"agent_version":   agentVersion,
	}

	// 添加退出码(仅end事件)
	if event.ExitCode != nil {
		pb["exit_code"] = *event.ExitCode
	}

	return pb
}

// ConvertBatchToECS 批量转换为ECS格式
func ConvertBatchToECS(events []*cgo.ProcessEvent) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, ConvertToECS(event))
	}
	return result
}

// ConvertBatchToProtobuf 批量转换为Protobuf格式
func ConvertBatchToProtobuf(events []*cgo.ProcessEvent, agentID string, agentVersion string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, ConvertToProtobuf(event, agentID, agentVersion))
	}
	return result
}
