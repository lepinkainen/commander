package storage

import (
	"context"

	"github.com/lepinkainen/commander/internal/types"
)

// TaskRepository defines the interface for task persistence
type TaskRepository interface {
	// Create adds a new task to storage
	Create(ctx context.Context, data types.TaskData) error

	// GetByID retrieves a task by its ID
	GetByID(ctx context.Context, id string) (types.TaskData, error)

	// List retrieves all tasks
	List(ctx context.Context) ([]types.TaskData, error)

	// ListByTool retrieves tasks for a specific tool
	ListByTool(ctx context.Context, tool string) ([]types.TaskData, error)

	// Update updates an existing task
	Update(ctx context.Context, data types.TaskData) error

	// AppendOutput adds output to a task
	AppendOutput(ctx context.Context, taskID string, output string) error

	// Close closes the storage connection
	Close() error
}

// FileRepository defines the interface for file and directory management
type FileRepository interface {
	// Directory operations
	CreateDirectory(ctx context.Context, dir *types.Directory) error
	GetDirectory(ctx context.Context, id string) (*types.Directory, error)
	ListDirectories(ctx context.Context) ([]*types.Directory, error)
	UpdateDirectory(ctx context.Context, dir *types.Directory) error
	DeleteDirectory(ctx context.Context, id string) error

	// File operations
	CreateFile(ctx context.Context, file *types.File) error
	GetFile(ctx context.Context, id string) (*types.File, error)
	ListFiles(ctx context.Context, filters types.FileFilters) ([]*types.File, error)
	UpdateFile(ctx context.Context, file *types.File) error
	DeleteFile(ctx context.Context, id string) error

	// File tag operations
	AddFileTag(ctx context.Context, fileID, tag string) error
	RemoveFileTag(ctx context.Context, fileID, tag string) error
	GetFileTags(ctx context.Context, fileID string) ([]string, error)

	// Search operations
	SearchFiles(ctx context.Context, query string) ([]*types.File, error)
}
