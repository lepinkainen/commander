package task

import (
	"sync"
	"time"

	"github.com/google/uuid"
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

// TaskData represents the data fields of a task (without mutex)
type TaskData struct {
	ID        string    `json:"id"`
	Tool      string    `json:"tool"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Status    Status    `json:"status"`
	Output    []string  `json:"output"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// Task represents a command to be executed
type Task struct {
	TaskData
	mu sync.RWMutex
}

// NewTask creates a new task
func NewTask(tool, command string, args []string) *Task {
	return &Task{
		TaskData: TaskData{
			ID:        uuid.New().String(),
			Tool:      tool,
			Command:   command,
			Args:      args,
			Status:    StatusQueued,
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
func (t *Task) SetStatus(status Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status

	switch status {
	case StatusRunning:
		t.StartedAt = time.Now()
	case StatusComplete, StatusFailed, StatusCanceled:
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
func (t *Task) GetStatus() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// Clone returns a copy of the task data for safe reading
func (t *Task) Clone() TaskData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a copy of the task data
	clone := TaskData{
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
