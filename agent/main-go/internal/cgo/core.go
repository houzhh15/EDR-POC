// Package cgo 提供 Go 与 C 核心库的绑定层
package cgo

/*
#cgo CFLAGS: -I${SRCDIR}/../../../core-c/include -I${SRCDIR}/../../../core-c/src -I${SRCDIR}/../../../core-c/src/queue -I${SRCDIR}/../../../core-c/src/plugin
#cgo LDFLAGS: -L${SRCDIR}/../../../core-c/build -ledr_core -Wl,-rpath,${SRCDIR}/../../../core-c/build

#include "edr_core.h"
#include "edr_errors.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
)

// 错误定义
var (
	ErrUnknown            = errors.New("unknown error")
	ErrInvalidParam       = errors.New("invalid parameter")
	ErrNoMemory           = errors.New("out of memory")
	ErrNotInitialized     = errors.New("not initialized")
	ErrAlreadyInitialized = errors.New("already initialized")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrNotSupported       = errors.New("not supported")
	ErrTimeout            = errors.New("timeout")
)

// 内部状态
var (
	initOnce sync.Once
	initMu   sync.Mutex
	isInit   bool
)

// toGoError 将 C 错误码转换为 Go 错误
func toGoError(err C.edr_error_t) error {
	switch err {
	case C.EDR_OK:
		return nil
	case C.EDR_ERR_INVALID_PARAM:
		return ErrInvalidParam
	case C.EDR_ERR_NO_MEMORY:
		return ErrNoMemory
	case C.EDR_ERR_NOT_INITIALIZED:
		return ErrNotInitialized
	case C.EDR_ERR_ALREADY_INITIALIZED:
		return ErrAlreadyInitialized
	case C.EDR_ERR_PERMISSION:
		return ErrPermissionDenied
	case C.EDR_ERR_NOT_SUPPORTED:
		return ErrNotSupported
	case C.EDR_ERR_TIMEOUT:
		return ErrTimeout
	default:
		return ErrUnknown
	}
}

// Init 初始化 C 核心库
// 必须在调用其他 CGO 函数前调用
func Init() error {
	initMu.Lock()
	defer initMu.Unlock()

	if isInit {
		return ErrAlreadyInitialized
	}

	// 锁定到当前 OS 线程（某些 C 库要求）
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err := C.edr_core_init()
	if goErr := toGoError(err); goErr != nil {
		return goErr
	}

	isInit = true
	return nil
}

// Cleanup 清理 C 核心库
func Cleanup() {
	initMu.Lock()
	defer initMu.Unlock()

	if !isInit {
		return
	}

	C.edr_core_cleanup()
	isInit = false
}

// IsInitialized 检查是否已初始化
func IsInitialized() bool {
	initMu.Lock()
	defer initMu.Unlock()
	return isInit
}

// Version 获取 C 核心库版本
func Version() string {
	cVersion := C.edr_core_version()
	return C.GoString(cVersion)
}

// ErrorString 获取错误描述
func ErrorString(err error) string {
	var cErr C.edr_error_t
	switch err {
	case nil:
		cErr = C.EDR_OK
	case ErrInvalidParam:
		cErr = C.EDR_ERR_INVALID_PARAM
	case ErrNoMemory:
		cErr = C.EDR_ERR_NO_MEMORY
	case ErrNotInitialized:
		cErr = C.EDR_ERR_NOT_INITIALIZED
	case ErrAlreadyInitialized:
		cErr = C.EDR_ERR_ALREADY_INITIALIZED
	case ErrPermissionDenied:
		cErr = C.EDR_ERR_PERMISSION
	case ErrNotSupported:
		cErr = C.EDR_ERR_NOT_SUPPORTED
	case ErrTimeout:
		cErr = C.EDR_ERR_TIMEOUT
	default:
		cErr = C.EDR_ERR_UNKNOWN
	}
	return C.GoString(C.edr_error_string(cErr))
}
