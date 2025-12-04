//go:build !windows
// +build !windows

package cgo

/*
#cgo CFLAGS: -I${SRCDIR}/../../../core-c/include -I${SRCDIR}/../../../core-c/src -I${SRCDIR}/../../../core-c/src/queue -I${SRCDIR}/../../../core-c/src/plugin
#cgo LDFLAGS: -L${SRCDIR}/../../../core-c/build -ledr_core -Wl,-rpath,${SRCDIR}/../../../core-c/build

#include "edr_core.h"
#include <stdlib.h>
*/
import "C"

// startCollectorPlatform 非 Windows 平台实现：使用原有占位实现
func startCollectorPlatform() error {
	// 调用 C 层采集器启动（当前为占位实现，不传递回调）
	err := C.edr_collector_start(nil, nil)
	if goErr := toGoError(err); goErr != nil {
		return goErr
	}
	return nil
}

// stopCollectorPlatform 非 Windows 平台实现：使用原有占位实现
func stopCollectorPlatform() error {
	err := C.edr_collector_stop()
	if goErr := toGoError(err); goErr != nil {
		return goErr
	}
	return nil
}
