//go:build windows
// +build windows

package cgo

import (
	"encoding/json"
	"sync"
	"time"
)

// Windows 平台特定变量
var (
	processCollector *ProcessCollector // ETW 进程采集器
	eventBridgeStop  chan struct{}     // 停止桥接goroutine
	eventBridgeWg    sync.WaitGroup    // 等待桥接goroutine退出
)

// startCollectorPlatform Windows 平台实现：启动 ETW 进程采集器
func startCollectorPlatform() error {
	var err error

	// 启动 ETW 进程采集器
	processCollector, err = StartProcessCollector()
	if err != nil {
		return err
	}

	// 启动事件桥接 goroutine
	eventBridgeStop = make(chan struct{})
	eventBridgeWg.Add(1)
	go eventBridgeLoop()

	return nil
}

// stopCollectorPlatform Windows 平台实现：停止 ETW 进程采集器
func stopCollectorPlatform() error {
	// 停止桥接 goroutine
	if eventBridgeStop != nil {
		close(eventBridgeStop)
		eventBridgeWg.Wait()
		eventBridgeStop = nil
	}

	// 停止 ETW 采集器
	if processCollector != nil {
		if err := processCollector.StopProcessCollector(); err != nil {
			return err
		}
		processCollector = nil
	}

	return nil
}

// eventBridgeLoop 将 ProcessEvent 转换为通用 Event 格式并发送到 eventChan
func eventBridgeLoop() {
	defer eventBridgeWg.Done()

	if processCollector == nil {
		return
	}

	events := processCollector.Events()

	for {
		select {
		case <-eventBridgeStop:
			return
		case processEvent, ok := <-events:
			if !ok {
				return
			}

			// 将 ProcessEvent 转换为通用 Event 格式
			event := convertProcessEventToEvent(processEvent)

			// 发送到 eventChan
			eventChanMu.RLock()
			ch := eventChan
			eventChanMu.RUnlock()

			if ch != nil {
				select {
				case ch <- event:
					// 成功发送
				case <-eventBridgeStop:
					return
				default:
					// channel 满，丢弃事件（避免阻塞）
				}
			}
		}
	}
}

// convertProcessEventToEvent 将 ProcessEvent 转换为通用 Event 格式
func convertProcessEventToEvent(pe *ProcessEvent) Event {
	// 构造 ECS 兼容的 JSON 数据
	data := map[string]interface{}{
		"@timestamp": pe.Timestamp.Format(time.RFC3339Nano),
		"event": map[string]interface{}{
			"kind":     "event",
			"category": []string{"process"},
			"type":     []string{pe.EventType},
		},
		"process": map[string]interface{}{
			"pid":          pe.ProcessPID,
			"ppid":         pe.ProcessPPID,
			"name":         pe.ProcessName,
			"executable":   pe.ProcessPath,
			"command_line": pe.ProcessCommandLine,
			"hash": map[string]interface{}{
				"sha256": pe.ProcessHash,
			},
		},
		"user": map[string]interface{}{
			"name":   pe.UserName,
			"domain": pe.UserDomain,
		},
	}

	// 进程结束事件添加 exit_code
	if pe.EventType == "end" && pe.ExitCode != nil {
		data["process"].(map[string]interface{})["exit_code"] = *pe.ExitCode
	}

	jsonData, _ := json.Marshal(data)

	// 事件类型映射
	var eventType uint32
	switch pe.EventType {
	case "start":
		eventType = 1 // EDR_EVENT_PROCESS_CREATE
	case "end":
		eventType = 2 // EDR_EVENT_PROCESS_EXIT
	default:
		eventType = 0
	}

	return Event{
		Type:      eventType,
		Timestamp: pe.Timestamp.UnixMilli(),
		Data:      jsonData,
	}
}
