package types

import (
	"time"
)

// Status represents the current state of a task
type Status string

const (
	StatusQueued   Status = "queued"
	StatusRunning  Status = "running"
	StatusComplete Status = "complete"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

// TaskData represents the data fields of a task
type TaskData struct {
	ID              string    `json:"id"`
	Tool            string    `json:"tool"`
	Command         string    `json:"command"`
	Args            []string  `json:"args"`
	Status          Status    `json:"status"`
	Output          []string  `json:"output"`
	Error           string    `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	EndedAt         time.Time `json:"ended_at,omitempty"`
	OutputDirectory *string   `json:"output_directory,omitempty"` // Directory where task outputs files
	AssociatedFiles []string  `json:"associated_files,omitempty"` // IDs of files created by this task
}

// Directory represents a download directory
type Directory struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	ToolName   *string   `json:"tool_name,omitempty"`
	DefaultDir bool      `json:"default_dir"`
	CreatedAt  time.Time `json:"created_at"`
}

// File represents a file in the system
type File struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	FilePath    string    `json:"file_path"`
	DirectoryID string    `json:"directory_id"`
	TaskID      *string   `json:"task_id,omitempty"`
	FileSize    int64     `json:"file_size"`
	MimeType    string    `json:"mime_type"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
}

// FileFilters represents filters for file listing
type FileFilters struct {
	DirectoryID string     `json:"directory_id,omitempty"`
	ToolName    string     `json:"tool_name,omitempty"`
	MimeType    string     `json:"mime_type,omitempty"`
	MinSize     int64      `json:"min_size,omitempty"`
	MaxSize     int64      `json:"max_size,omitempty"`
	CreatedFrom *time.Time `json:"created_from,omitempty"`
	CreatedTo   *time.Time `json:"created_to,omitempty"`
}
