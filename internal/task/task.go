package task

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lepinkainen/commander/internal/types"
)

// Task represents a command to be executed
type Task struct {
	types.TaskData
	mu sync.RWMutex
}

// NewTask creates a new task
func NewTask(tool, command string, args []string) *Task {
	return &Task{
		TaskData: types.TaskData{
			ID:        uuid.New().String(),
			Tool:      tool,
			Command:   command,
			Args:      args,
			Status:    types.StatusQueued,
			Output:    make([]string, 0),
			CreatedAt: time.Now(),
		},
	}
}

// AppendOutput adds output to the task
func (t *Task) AppendOutput(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Output = append(t.Output, line)
}

// SetStatus updates the task status
func (t *Task) SetStatus(status types.Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status

	switch status {
	case types.StatusRunning:
		t.StartedAt = time.Now()
	case types.StatusComplete, types.StatusFailed, types.StatusCanceled:
		t.EndedAt = time.Now()
	}
}

// SetError sets an error message
func (t *Task) SetError(err string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = err
}

// GetStatus returns the current status
func (t *Task) GetStatus() types.Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// Clone returns a copy of the task data for safe reading
func (t *Task) Clone() types.TaskData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a copy of the task data
	clone := types.TaskData{
		ID:        t.ID,
		Tool:      t.Tool,
		Command:   t.Command,
		Args:      make([]string, len(t.Args)),
		Status:    t.Status,
		Output:    make([]string, len(t.Output)),
		Error:     t.Error,
		CreatedAt: t.CreatedAt,
		StartedAt: t.StartedAt,
		EndedAt:   t.EndedAt,
	}

	copy(clone.Output, t.Output)
	copy(clone.Args, t.Args)

	return clone
}
