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
