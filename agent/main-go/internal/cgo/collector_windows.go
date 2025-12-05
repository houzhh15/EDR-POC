//go:build windows
// +build windows

package cgo

/*
#cgo CFLAGS: -I../../../core-c/include
#cgo LDFLAGS: -L../../../core-c/build -ledr_core

#include "edr_events.h"
#include "edr_errors.h"
#include "event_buffer.h"
#include <stdlib.h>
#include <string.h>

// 声明C层API(实际会在edr_core.h中定义)
typedef void* edr_session_handle_t;

int edr_start_process_collector(edr_session_handle_t* out_handle);
int edr_stop_process_collector(edr_session_handle_t handle);
int edr_poll_process_events(
    edr_session_handle_t handle,
    edr_process_event_t* events,
    int max_count,
    int* out_count
);

// 获取 C 层事件结构体的真实大小
static inline size_t get_event_size(void) {
    return sizeof(edr_process_event_t);
}

// 分配事件数组 (确保使用 C 层的 sizeof)
static inline edr_process_event_t* alloc_events(int count) {
    return (edr_process_event_t*)calloc(count, sizeof(edr_process_event_t));
}

// 释放事件数组
static inline void free_events(edr_process_event_t* events) {
    free(events);
}
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// ProcessCollector Windows进程事件采集器
type ProcessCollector struct {
	handle  C.edr_session_handle_t // C层Session句柄
	stopCh  chan struct{}          // 停止信号
	eventCh chan *ProcessEvent     // 事件输出channel(容量1000)
	wg      sync.WaitGroup         // 等待goroutine退出

	statsLock sync.RWMutex   // 统计信息锁
	stats     CollectorStats // 统计信息
}

// ProcessEvent 进程事件结构体(ECS兼容格式)
type ProcessEvent struct {
	Timestamp          time.Time `json:"@timestamp"`
	EventType          string    `json:"event_type"` // "start" | "end"
	ProcessPID         uint32    `json:"process_pid"`
	ProcessPPID        uint32    `json:"process_ppid"`
	ProcessName        string    `json:"process_name"`
	ProcessPath        string    `json:"process_path"`
	ProcessCommandLine string    `json:"process_command_line"`
	ProcessHash        string    `json:"process_hash"` // SHA256 hex
	UserName           string    `json:"user_name"`
	UserDomain         string    `json:"user_domain"`
	ExitCode           *int32    `json:"exit_code,omitempty"` // 仅end事件有值
}

// CollectorStats 采集器统计信息
type CollectorStats struct {
	TotalEventsCollected uint64    `json:"total_events_collected"`
	TotalEventsProcessed uint64    `json:"total_events_processed"`
	TotalEventsDropped   uint64    `json:"total_events_dropped"`
	LastPollTime         time.Time `json:"last_poll_time"`
}

// StartProcessCollector 启动进程事件采集器
func StartProcessCollector() (*ProcessCollector, error) {
	var handle C.edr_session_handle_t

	// 调用C层API启动采集器
	ret := C.edr_start_process_collector(&handle)
	if ret != C.EDR_SUCCESS {
		return nil, fmt.Errorf("failed to start process collector: error_code=%d", int(ret))
	}

	pc := &ProcessCollector{
		handle:  handle,
		stopCh:  make(chan struct{}),
		eventCh: make(chan *ProcessEvent, 1000),
		stats:   CollectorStats{},
	}

	// 启动事件轮询goroutine
	pc.wg.Add(1)
	go pc.pollLoop()

	return pc, nil
}

// StopProcessCollector 停止采集器
func (pc *ProcessCollector) StopProcessCollector() error {
	// 发送停止信号
	close(pc.stopCh)

	// 等待goroutine退出
	pc.wg.Wait()

	// 调用C层API停止采集器
	ret := C.edr_stop_process_collector(pc.handle)
	if ret != C.EDR_SUCCESS {
		return fmt.Errorf("failed to stop process collector: error_code=%d", int(ret))
	}

	// 关闭事件channel
	close(pc.eventCh)

	return nil
}

// Events 返回事件channel(只读)
func (pc *ProcessCollector) Events() <-chan *ProcessEvent {
	return pc.eventCh
}

// GetStats 获取采集器统计信息
func (pc *ProcessCollector) GetStats() CollectorStats {
	pc.statsLock.RLock()
	defer pc.statsLock.RUnlock()
	return pc.stats
}

// pollLoop 事件轮询循环(内部方法)
func (pc *ProcessCollector) pollLoop() {
	defer pc.wg.Done()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	const maxBatch = 100
	// 使用 C 层的 calloc 分配内存,确保使用正确的 sizeof
	// 避免 Go 和 C 之间结构体大小不一致导致的内存越界
	cEventsPtr := C.alloc_events(C.int(maxBatch))
	if cEventsPtr == nil {
		return // 内存分配失败
	}
	defer C.free_events(cEventsPtr)
	
	// 创建 Go slice 来访问 C 数组(不复制内存)
	cEvents := unsafe.Slice(cEventsPtr, maxBatch)

	for {
		select {
		case <-pc.stopCh:
			return
		case <-ticker.C:
			// 批量获取事件
			var count C.int
			ret := C.edr_poll_process_events(
				pc.handle,
				cEventsPtr,
				C.int(maxBatch),
				&count,
			)

			if ret != C.EDR_SUCCESS {
				// 轮询失败,记录统计并继续
				continue
			}

			// 更新统计信息
			pc.statsLock.Lock()
			pc.stats.LastPollTime = time.Now()
			pc.stats.TotalEventsCollected += uint64(count)
			pc.statsLock.Unlock()

			// 转换并发送事件
			for i := 0; i < int(count); i++ {
				event := convertToGoEvent(&cEvents[i])

				// 非阻塞发送
				select {
				case pc.eventCh <- event:
					pc.statsLock.Lock()
					pc.stats.TotalEventsProcessed++
					pc.statsLock.Unlock()
				case <-pc.stopCh:
					return
				default:
					// channel满,丢弃事件
					pc.statsLock.Lock()
					pc.stats.TotalEventsDropped++
					pc.statsLock.Unlock()
				}
			}
		}
	}
}

// convertToGoEvent 将C事件结构体转换为Go事件结构体
func convertToGoEvent(cEvent *C.edr_process_event_t) *ProcessEvent {
	event := &ProcessEvent{
		// 时间戳转换(纳秒)
		Timestamp:          time.Unix(0, int64(cEvent.timestamp)),
		ProcessPID:         uint32(cEvent.pid),
		ProcessPPID:        uint32(cEvent.ppid),
		ProcessName:        C.GoString(&cEvent.process_name[0]),
		ProcessPath:        C.GoString(&cEvent.executable_path[0]),
		ProcessCommandLine: C.GoString(&cEvent.command_line[0]),
		UserName:           C.GoString(&cEvent.username[0]),
	}

	// 设置事件类型
	switch cEvent.event_type {
	case C.EDR_PROCESS_START:
		event.EventType = "start"
	case C.EDR_PROCESS_END:
		event.EventType = "end"
		// 仅end事件设置ExitCode
		exitCode := int32(cEvent.exit_code)
		event.ExitCode = &exitCode
	default:
		event.EventType = "unknown"
	}

	// 格式化SHA256哈希为hex字符串
	event.ProcessHash = fmt.Sprintf("%x", C.GoBytes(unsafe.Pointer(&cEvent.sha256[0]), 32))

	// 解析用户域(格式: DOMAIN\USER)
	// 简化处理,完整实现需要解析username字段
	event.UserDomain = ""

	return event
}
