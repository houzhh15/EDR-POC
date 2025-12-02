//go:build !windows
// +build !windows

package cgo

import "time"

// ProcessEvent 进程事件结构体(非Windows平台的stub定义)
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

// CollectorStats 采集器统计信息(非Windows平台的stub定义)
type CollectorStats struct {
	TotalEventsCollected uint64    `json:"total_events_collected"`
	TotalEventsProcessed uint64    `json:"total_events_processed"`
	TotalEventsDropped   uint64    `json:"total_events_dropped"`
	LastPollTime         time.Time `json:"last_poll_time"`
}
