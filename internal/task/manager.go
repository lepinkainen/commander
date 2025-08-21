package task

import (
	"fmt"
	"sync"
)

// Manager manages all tasks
type Manager struct {
	tasks     map[string]*Task
	queues    map[string]chan *Task
	mu        sync.RWMutex
	listeners []chan TaskEvent
}

// TaskEvent represents a task state change
type TaskEvent struct {
	TaskID string `json:"task_id"`
	Type   string `json:"type"`
	Data   string `json:"data"`
}

// NewManager creates a new task manager
func NewManager() *Manager {
	return &Manager{
		tasks:     make(map[string]*Task),
		queues:    make(map[string]chan *Task),
		listeners: make([]chan TaskEvent, 0),
	}
}

// CreateQueue creates a new queue for a tool
func (m *Manager) CreateQueue(tool string, bufferSize int) chan *Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.queues[tool]; !exists {
		m.queues[tool] = make(chan *Task, bufferSize)
	}
	return m.queues[tool]
}

// AddTask adds a new task to the manager
func (m *Manager) AddTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	m.tasks[task.ID] = task

	// Send to appropriate queue
	if queue, ok := m.queues[task.Tool]; ok {
		select {
		case queue <- task:
			m.broadcastEvent(TaskEvent{
				TaskID: task.ID,
				Type:   "created",
				Data:   fmt.Sprintf("Task %s queued for %s", task.ID, task.Tool),
			})
		default:
			return fmt.Errorf("queue for %s is full", task.Tool)
		}
	} else {
		return fmt.Errorf("no queue for tool %s", task.Tool)
	}

	return nil
}

// GetTask returns a task by ID
func (m *Manager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task %s not found", id)
	}

	return task, nil
}

// GetAllTasks returns all tasks
func (m *Manager) GetAllTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTasksByTool returns tasks for a specific tool
func (m *Manager) GetTasksByTool(tool string) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0)
	for _, task := range m.tasks {
		if task.Tool == tool {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// UpdateTaskStatus updates a task's status and broadcasts the change
func (m *Manager) UpdateTaskStatus(taskID string, status Status) error {
	task, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	task.SetStatus(status)

	m.broadcastEvent(TaskEvent{
		TaskID: taskID,
		Type:   "status",
		Data:   string(status),
	})

	return nil
}

// AppendTaskOutput appends output to a task and broadcasts it
func (m *Manager) AppendTaskOutput(taskID string, output string) error {
	task, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	task.AppendOutput(output)

	m.broadcastEvent(TaskEvent{
		TaskID: taskID,
		Type:   "output",
		Data:   output,
	})

	return nil
}

// Subscribe creates a new event listener channel
func (m *Manager) Subscribe() chan TaskEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan TaskEvent, 100)
	m.listeners = append(m.listeners, ch)
	return ch
}

// Unsubscribe removes an event listener
func (m *Manager) Unsubscribe(ch chan TaskEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, listener := range m.listeners {
		if listener == ch {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcastEvent sends an event to all listeners
func (m *Manager) broadcastEvent(event TaskEvent) {
	for _, listener := range m.listeners {
		select {
		case listener <- event:
		default:
			// Skip if listener is full
		}
	}
}

// GetQueueStats returns statistics about all queues
func (m *Manager) GetQueueStats() map[string]QueueStats {
m.mu.RLock()
defer m.mu.RUnlock()

stats := make(map[string]QueueStats)
for tool, queue := range m.queues {
// Create a local variable that we can modify
toolStats := QueueStats{
Tool:    tool,
 Pending: len(queue),
}

// Count running and completed tasks
for _, task := range m.tasks {
if task.Tool == tool {
switch task.GetStatus() {
case StatusRunning:
 toolStats.Running++
case StatusComplete:
 toolStats.Completed++
case StatusFailed:
 toolStats.Failed++
 }
 }
 }
 
 // Assign the completed stats struct to the map
		stats[tool] = toolStats
	}
	
	return stats
}

// QueueStats represents queue statistics
type QueueStats struct {
	Tool      string `json:"tool"`
	Pending   int    `json:"pending"`
	Running   int    `json:"running"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
}
