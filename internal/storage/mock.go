package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/lepinkainen/commander/internal/types"
)

// MockRepository is a mock implementation of TaskRepository for testing
type MockRepository struct {
	tasks map[string]types.TaskData
	mu    sync.RWMutex
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		tasks: make(map[string]types.TaskData),
	}
}

// Create adds a new task to storage
func (m *MockRepository) Create(ctx context.Context, data types.TaskData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[data.ID]; exists {
		return fmt.Errorf("task %s already exists", data.ID)
	}

	m.tasks[data.ID] = data
	return nil
}

// GetByID retrieves a task by its ID
func (m *MockRepository) GetByID(ctx context.Context, id string) (types.TaskData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tasks[id]
	if !exists {
		return types.TaskData{}, fmt.Errorf("task %s not found", id)
	}

	return data, nil
}

// List retrieves all tasks
func (m *MockRepository) List(ctx context.Context) ([]types.TaskData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []types.TaskData
	for _, data := range m.tasks {
		tasks = append(tasks, data)
	}

	return tasks, nil
}

// ListByTool retrieves tasks for a specific tool
func (m *MockRepository) ListByTool(ctx context.Context, tool string) ([]types.TaskData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []types.TaskData
	for _, data := range m.tasks {
		if data.Tool == tool {
			tasks = append(tasks, data)
		}
	}

	return tasks, nil
}

// Update updates an existing task
func (m *MockRepository) Update(ctx context.Context, data types.TaskData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[data.ID]; !exists {
		return fmt.Errorf("task %s not found", data.ID)
	}

	m.tasks[data.ID] = data
	return nil
}

// AppendOutput adds output to a task
func (m *MockRepository) AppendOutput(ctx context.Context, taskID string, output string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	data.Output = append(data.Output, output)
	m.tasks[taskID] = data
	return nil
}

// Close closes the storage connection
func (m *MockRepository) Close() error {
	return nil
}
