// Package ecs provides file event mappers.
package ecs

// FileCreateMapper 文件创建事件映射器
type FileCreateMapper struct{}

// Map 映射文件创建事件
func (m *FileCreateMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"file"}
	ecs.Event.Type = []string{"creation"}
	ecs.Event.Action = "file_created"
	ecs.Event.Dataset = "file"

	if evt.File != nil {
		ecs.File = &ECSFile{
			Path:      evt.File.Path,
			Name:      evt.File.Name,
			Extension: evt.File.Extension,
			Directory: evt.File.Directory,
			Size:      evt.File.Size,
			Mode:      evt.File.Mode,
			Owner:     evt.File.Owner,
			Hash:      convertHash(evt.File.Hash),
		}
	}

	// 关联进程信息（如果有）
	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:        evt.Process.PID,
			Name:       evt.Process.Name,
			Executable: evt.Process.Executable,
		}
	}

	return nil
}

// FileModifyMapper 文件修改事件映射器
type FileModifyMapper struct{}

// Map 映射文件修改事件
func (m *FileModifyMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"file"}
	ecs.Event.Type = []string{"change"}
	ecs.Event.Action = "file_modified"
	ecs.Event.Dataset = "file"

	if evt.File != nil {
		ecs.File = &ECSFile{
			Path:      evt.File.Path,
			Name:      evt.File.Name,
			Extension: evt.File.Extension,
			Directory: evt.File.Directory,
			Size:      evt.File.Size,
			Mode:      evt.File.Mode,
			Owner:     evt.File.Owner,
			Hash:      convertHash(evt.File.Hash),
		}
	}

	// 关联进程信息（如果有）
	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:        evt.Process.PID,
			Name:       evt.Process.Name,
			Executable: evt.Process.Executable,
		}
	}

	return nil
}

// FileDeleteMapper 文件删除事件映射器
type FileDeleteMapper struct{}

// Map 映射文件删除事件
func (m *FileDeleteMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"file"}
	ecs.Event.Type = []string{"deletion"}
	ecs.Event.Action = "file_deleted"
	ecs.Event.Dataset = "file"

	if evt.File != nil {
		ecs.File = &ECSFile{
			Path:      evt.File.Path,
			Name:      evt.File.Name,
			Extension: evt.File.Extension,
			Directory: evt.File.Directory,
		}
	}

	// 关联进程信息（如果有）
	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:        evt.Process.PID,
			Name:       evt.Process.Name,
			Executable: evt.Process.Executable,
		}
	}

	return nil
}
