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

// Task represents a command to be executed
type Task struct {
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
	mu        sync.RWMutex
}

// NewTask creates a new task
func NewTask(tool, command string, args []string) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Tool:      tool,
		Command:   command,
		Args:      args,
		Status:    StatusQueued,
		Output:    make([]string, 0),
		CreatedAt: time.Now(),
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

// Clone returns a copy of the task for safe reading
func (t *Task) Clone() Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	clone := *t
	clone.Output = make([]string, len(t.Output))
	copy(clone.Output, t.Output)
	clone.Args = make([]string, len(t.Args))
	copy(clone.Args, t.Args)
	
	return clone
}
