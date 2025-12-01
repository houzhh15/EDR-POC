package cgo

/*
#cgo CFLAGS: -I${SRCDIR}/../../../core-c/include -I${SRCDIR}/../../../core-c/src -I${SRCDIR}/../../../core-c/src/queue -I${SRCDIR}/../../../core-c/src/plugin
#cgo LDFLAGS: -L${SRCDIR}/../../../core-c/build -ledr_core -Wl,-rpath,${SRCDIR}/../../../core-c/build

#include "edr_core.h"
#include <stdlib.h>
*/
import "C"

import (
	"sync"
)

// Event 表示从 C 层接收的事件
type Event struct {
	Type      uint32 // 事件类型
	Timestamp int64  // 时间戳（毫秒）
	Data      []byte // 事件数据（JSON）
}

// 全局事件通道和回调状态
var (
	eventChan        chan Event
	eventChanMu      sync.RWMutex
	collectorRunning bool
)

// StartCollector 启动事件采集器
// ch: 用于接收事件的通道
func StartCollector(ch chan Event) error {
	eventChanMu.Lock()
	defer eventChanMu.Unlock()

	if collectorRunning {
		return ErrAlreadyInitialized
	}

	eventChan = ch

	// 调用 C 层采集器启动（当前为占位实现，不传递回调）
	err := C.edr_collector_start(nil, nil)
	if goErr := toGoError(err); goErr != nil {
		eventChan = nil
		return goErr
	}

	collectorRunning = true
	return nil
}

// StopCollector 停止事件采集器
func StopCollector() error {
	eventChanMu.Lock()
	defer eventChanMu.Unlock()

	if !collectorRunning {
		return nil
	}

	err := C.edr_collector_stop()
	if goErr := toGoError(err); goErr != nil {
		return goErr
	}

	eventChan = nil
	collectorRunning = false
	return nil
}

// IsCollectorRunning 检查采集器是否运行中
func IsCollectorRunning() bool {
	return bool(C.edr_collector_is_running())
}
