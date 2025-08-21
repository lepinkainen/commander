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
