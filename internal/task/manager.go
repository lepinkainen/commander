package task

import (
	"context"
	"fmt"
	"sync"

	"github.com/lepinkainen/commander/internal/storage"
	"github.com/lepinkainen/commander/internal/types"
)

// Manager manages all tasks
type Manager struct {
	repo      storage.TaskRepository
	tasks     map[string]*Task // In-memory cache for active tasks
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
func NewManager(repo storage.TaskRepository) *Manager {
	return &Manager{
		repo:      repo,
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

	// Save to database
	ctx := context.Background()
	if err := m.repo.Create(ctx, task.Clone()); err != nil {
		return fmt.Errorf("failed to save task to database: %w", err)
	}

	// Add to in-memory cache
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
	// First check in-memory cache for active tasks
	task, exists := m.tasks[id]
	m.mu.RUnlock()

	if exists {
		return task, nil
	}

	// If not in cache, try to load from database
	ctx := context.Background()
	data, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert TaskData back to Task
	dbTask := &Task{TaskData: data}
	return dbTask, nil
}

// GetAllTasks returns all tasks
func (m *Manager) GetAllTasks() []*Task {
	// Load all tasks from database
	ctx := context.Background()
	data, err := m.repo.List(ctx)
	if err != nil {
		// Fallback to in-memory tasks if database fails
		m.mu.RLock()
		defer m.mu.RUnlock()
		memoryTasks := make([]*Task, 0, len(m.tasks))
		for _, task := range m.tasks {
			memoryTasks = append(memoryTasks, task)
		}
		return memoryTasks
	}

	// Convert TaskData slice to Task slice
	tasks := make([]*Task, len(data))
	for i, d := range data {
		tasks[i] = &Task{TaskData: d}
	}
	return tasks
}

// GetTasksByTool returns tasks for a specific tool
func (m *Manager) GetTasksByTool(tool string) []*Task {
	// Load tasks from database
	ctx := context.Background()
	data, err := m.repo.ListByTool(ctx, tool)
	if err != nil {
		// Fallback to in-memory tasks if database fails
		m.mu.RLock()
		defer m.mu.RUnlock()
		memoryTasks := make([]*Task, 0)
		for _, task := range m.tasks {
			if task.Tool == tool {
				memoryTasks = append(memoryTasks, task)
			}
		}
		return memoryTasks
	}

	// Convert TaskData slice to Task slice
	tasks := make([]*Task, len(data))
	for i, d := range data {
		tasks[i] = &Task{TaskData: d}
	}
	return tasks
}

// UpdateTaskStatus updates a task's status and broadcasts the change
func (m *Manager) UpdateTaskStatus(taskID string, status types.Status) error {
	task, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	task.SetStatus(status)

	// Update in database
	ctx := context.Background()
	if err := m.repo.Update(ctx, task.Clone()); err != nil {
		// Log error but don't fail - we can continue with in-memory
		fmt.Printf("Warning: failed to update task in database: %v\n", err)
	}

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

	// Save output to database
	ctx := context.Background()
	if err := m.repo.AppendOutput(ctx, taskID, output); err != nil {
		// Log error but don't fail - we can continue with in-memory
		fmt.Printf("Warning: failed to save output to database: %v\n", err)
	}

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

		// Count running tasks from in-memory cache (active tasks)
		for _, task := range m.tasks {
			if task.Tool == tool {
				switch task.GetStatus() {
				case types.StatusRunning:
					toolStats.Running++
				}
			}
		}

		// Count completed/failed from database
		ctx := context.Background()
		allTasks, err := m.repo.ListByTool(ctx, tool)
		if err == nil {
			for _, taskData := range allTasks {
				switch taskData.Status {
				case types.StatusComplete:
					toolStats.Completed++
				case types.StatusFailed:
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
