package cgo

import (
	"testing"
)

func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Error("expected version to be non-empty")
	}
	t.Logf("EDR Core version: %s", version)
}

func TestInitCleanup(t *testing.T) {
	// 初始化
	err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// 验证已初始化
	if !IsInitialized() {
		t.Error("expected IsInitialized() to return true")
	}

	// 重复初始化应返回错误
	err = Init()
	if err != ErrAlreadyInitialized {
		t.Errorf("expected ErrAlreadyInitialized, got %v", err)
	}

	// 清理
	Cleanup()

	// 验证已清理
	if IsInitialized() {
		t.Error("expected IsInitialized() to return false after cleanup")
	}
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		err      error
		contains string
	}{
		{nil, "Success"},
		{ErrInvalidParam, "Invalid"},
		{ErrNoMemory, "memory"},
		{ErrNotInitialized, "initialized"},
		{ErrAlreadyInitialized, "initialized"},
		{ErrPermissionDenied, "Permission"},
		{ErrNotSupported, "supported"},
		{ErrTimeout, "Timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			msg := ErrorString(tt.err)
			if msg == "" {
				t.Error("expected error string to be non-empty")
			}
			t.Logf("%v -> %s", tt.err, msg)
		})
	}
}

func TestCollectorLifecycle(t *testing.T) {
	// 先初始化
	err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Cleanup()

	// 创建事件通道
	ch := make(chan Event, 100)

	// 启动采集器
	err = StartCollector(ch)
	if err != nil {
		t.Fatalf("StartCollector() error = %v", err)
	}

	// 验证运行状态
	if !IsCollectorRunning() {
		t.Error("expected IsCollectorRunning() to return true")
	}

	// 停止采集器
	err = StopCollector()
	if err != nil {
		t.Errorf("StopCollector() error = %v", err)
	}

	// 验证停止状态
	if IsCollectorRunning() {
		t.Error("expected IsCollectorRunning() to return false")
	}
}
