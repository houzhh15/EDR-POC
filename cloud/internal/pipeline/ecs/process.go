// Package ecs provides process event mappers.
package ecs

import "time"

// ProcessCreateMapper 进程创建事件映射器
type ProcessCreateMapper struct{}

// Map 映射进程创建事件
func (m *ProcessCreateMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"process"}
	ecs.Event.Type = []string{"start"}
	ecs.Event.Action = "process_created"
	ecs.Event.Dataset = "process"

	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:         evt.Process.PID,
			Name:        evt.Process.Name,
			Executable:  evt.Process.Executable,
			CommandLine: evt.Process.CommandLine,
			Args:        evt.Process.Args,
			WorkingDir:  evt.Process.WorkingDir,
			Hash:        convertHash(evt.Process.Hash),
		}

		if evt.Process.Args != nil {
			ecs.Process.ArgsCount = len(evt.Process.Args)
		}

		if evt.Process.StartTime > 0 {
			ecs.Process.Start = time.Unix(0, evt.Process.StartTime)
		}

		if evt.Process.PPID > 0 {
			ecs.Process.Parent = &ECSProcessParent{
				PID: evt.Process.PPID,
			}
		}

		if evt.Process.User != "" {
			ecs.Process.User = &ECSUser{
				Name: evt.Process.User,
			}
		}
	}

	return nil
}

// ProcessTerminateMapper 进程终止事件映射器
type ProcessTerminateMapper struct{}

// Map 映射进程终止事件
func (m *ProcessTerminateMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"process"}
	ecs.Event.Type = []string{"end"}
	ecs.Event.Action = "process_terminated"
	ecs.Event.Dataset = "process"

	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:         evt.Process.PID,
			Name:        evt.Process.Name,
			Executable:  evt.Process.Executable,
			CommandLine: evt.Process.CommandLine,
			ExitCode:    evt.Process.ExitCode,
		}

		if evt.Process.StartTime > 0 {
			ecs.Process.Start = time.Unix(0, evt.Process.StartTime)
		}

		ecs.Process.End = time.Unix(0, evt.Timestamp)

		if evt.Process.PPID > 0 {
			ecs.Process.Parent = &ECSProcessParent{
				PID: evt.Process.PPID,
			}
		}
	}

	return nil
}
