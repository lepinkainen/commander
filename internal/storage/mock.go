package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lepinkainen/commander/internal/types"
)

// MockRepository is a mock implementation of TaskRepository and FileRepository for testing
type MockRepository struct {
	tasks       map[string]types.TaskData
	directories map[string]*types.Directory
	files       map[string]*types.File
	fileTags    map[string][]string
	mu          sync.RWMutex
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		tasks:       make(map[string]types.TaskData),
		directories: make(map[string]*types.Directory),
		files:       make(map[string]*types.File),
		fileTags:    make(map[string][]string),
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

// FileRepository implementation

// CreateDirectory adds a new directory to storage
func (m *MockRepository) CreateDirectory(ctx context.Context, dir *types.Directory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.directories[dir.ID]; exists {
		return fmt.Errorf("directory %s already exists", dir.ID)
	}

	m.directories[dir.ID] = dir
	return nil
}

// GetDirectory retrieves a directory by its ID
func (m *MockRepository) GetDirectory(ctx context.Context, id string) (*types.Directory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dir, exists := m.directories[id]
	if !exists {
		return nil, fmt.Errorf("directory %s not found", id)
	}

	return dir, nil
}

// ListDirectories retrieves all directories
func (m *MockRepository) ListDirectories(ctx context.Context) ([]*types.Directory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var dirs []*types.Directory
	for _, dir := range m.directories {
		dirs = append(dirs, dir)
	}

	return dirs, nil
}

// UpdateDirectory updates an existing directory
func (m *MockRepository) UpdateDirectory(ctx context.Context, dir *types.Directory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.directories[dir.ID]; !exists {
		return fmt.Errorf("directory %s not found", dir.ID)
	}

	m.directories[dir.ID] = dir
	return nil
}

// DeleteDirectory removes a directory from storage
func (m *MockRepository) DeleteDirectory(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.directories[id]; !exists {
		return fmt.Errorf("directory %s not found", id)
	}

	delete(m.directories, id)
	return nil
}

// CreateFile adds a new file to storage
func (m *MockRepository) CreateFile(ctx context.Context, file *types.File) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[file.ID]; exists {
		return fmt.Errorf("file %s already exists", file.ID)
	}

	m.files[file.ID] = file
	if len(file.Tags) > 0 {
		m.fileTags[file.ID] = file.Tags
	}
	return nil
}

// GetFile retrieves a file by its ID
func (m *MockRepository) GetFile(ctx context.Context, id string) (*types.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, exists := m.files[id]
	if !exists {
		return nil, fmt.Errorf("file %s not found", id)
	}

	// Populate tags
	if tags, ok := m.fileTags[id]; ok {
		file.Tags = tags
	}

	return file, nil
}

// ListFiles retrieves files based on filters
func (m *MockRepository) ListFiles(ctx context.Context, filters types.FileFilters) ([]*types.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var files []*types.File
	for _, file := range m.files {
		// Apply filters
		if filters.DirectoryID != "" && file.DirectoryID != filters.DirectoryID {
			continue
		}
		if filters.MimeType != "" && file.MimeType != filters.MimeType {
			continue
		}
		if filters.MinSize > 0 && file.FileSize < filters.MinSize {
			continue
		}
		if filters.MaxSize > 0 && file.FileSize > filters.MaxSize {
			continue
		}

		// Populate tags
		if tags, ok := m.fileTags[file.ID]; ok {
			fileCopy := *file
			fileCopy.Tags = tags
			files = append(files, &fileCopy)
		} else {
			files = append(files, file)
		}
	}

	return files, nil
}

// UpdateFile updates an existing file
func (m *MockRepository) UpdateFile(ctx context.Context, file *types.File) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[file.ID]; !exists {
		return fmt.Errorf("file %s not found", file.ID)
	}

	m.files[file.ID] = file
	return nil
}

// DeleteFile removes a file from storage
func (m *MockRepository) DeleteFile(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[id]; !exists {
		return fmt.Errorf("file %s not found", id)
	}

	delete(m.files, id)
	delete(m.fileTags, id)
	return nil
}

// AddFileTag adds a tag to a file
func (m *MockRepository) AddFileTag(ctx context.Context, fileID, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[fileID]; !exists {
		return fmt.Errorf("file %s not found", fileID)
	}

	tags := m.fileTags[fileID]
	for _, existingTag := range tags {
		if existingTag == tag {
			return nil // Tag already exists
		}
	}

	m.fileTags[fileID] = append(tags, tag)
	return nil
}

// RemoveFileTag removes a tag from a file
func (m *MockRepository) RemoveFileTag(ctx context.Context, fileID, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[fileID]; !exists {
		return fmt.Errorf("file %s not found", fileID)
	}

	tags := m.fileTags[fileID]
	for i, existingTag := range tags {
		if existingTag == tag {
			m.fileTags[fileID] = append(tags[:i], tags[i+1:]...)
			return nil
		}
	}

	return nil // Tag not found, but not an error
}

// GetFileTags retrieves all tags for a file
func (m *MockRepository) GetFileTags(ctx context.Context, fileID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.files[fileID]; !exists {
		return nil, fmt.Errorf("file %s not found", fileID)
	}

	tags := m.fileTags[fileID]
	if tags == nil {
		return []string{}, nil
	}

	return tags, nil
}

// SearchFiles searches for files by filename
func (m *MockRepository) SearchFiles(ctx context.Context, query string) ([]*types.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var files []*types.File
	for _, file := range m.files {
		// Simple substring search
		if containsIgnoreCase(file.Filename, query) || containsIgnoreCase(file.FilePath, query) {
			// Populate tags
			if tags, ok := m.fileTags[file.ID]; ok {
				fileCopy := *file
				fileCopy.Tags = tags
				files = append(files, &fileCopy)
			} else {
				files = append(files, file)
			}
		}
	}

	return files, nil
}

func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
