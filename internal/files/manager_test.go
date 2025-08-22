package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lepinkainen/commander/internal/storage"
	"github.com/lepinkainen/commander/internal/types"
)

func TestCreateDirectory(t *testing.T) {
	// Setup
	repo := storage.NewMockRepository()
	manager := NewManager(repo)
	ctx := context.Background()

	// Test creating a directory
	name := "Test Directory"
	path := "/tmp/test-dir"
	toolName := "test-tool"

	dir, err := manager.CreateDirectory(ctx, name, path, &toolName, true)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Verify directory properties
	if dir.Name != name {
		t.Errorf("Expected name %s, got %s", name, dir.Name)
	}
	if dir.Path != path {
		t.Errorf("Expected path %s, got %s", path, dir.Path)
	}
	if dir.ToolName == nil || *dir.ToolName != toolName {
		t.Errorf("Expected tool name %s, got %v", toolName, dir.ToolName)
	}
	if !dir.DefaultDir {
		t.Error("Expected directory to be marked as default")
	}

	// Clean up
	_ = os.RemoveAll(path)
}

func TestRegisterFileFromTask(t *testing.T) {
	// Setup
	repo := storage.NewMockRepository()
	manager := NewManager(repo)
	ctx := context.Background()

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory first
	dir, err := manager.CreateDirectory(ctx, "Test Dir", tempDir, nil, true)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Register file from task
	taskID := "test-task-123"
	err = manager.RegisterFileFromTask(ctx, taskID, testFile, &dir.ID)
	if err != nil {
		t.Fatalf("Failed to register file from task: %v", err)
	}

	// Verify the file was registered (this would need the mock to be more sophisticated)
	// For now, we just verify no error occurred
}

func TestFormatFileSize(t *testing.T) {
	repo := storage.NewMockRepository()
	manager := NewManager(repo)

	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 Bytes"},
		{512, "512 Bytes"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{1048576, "1 MB"},
		{1073741824, "1 GB"},
	}

	for _, test := range tests {
		result := manager.FormatFileSize(test.input)
		if result != test.expected {
			t.Errorf("FormatFileSize(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

// Helper method for testing
func (m *Manager) FormatFileSize(bytes int64) string {
	if bytes == 0 {
		return "0 Bytes"
	}
	const k = 1024
	sizes := []string{"Bytes", "KB", "MB", "GB"}

	if bytes < k {
		return fmt.Sprintf("%d %s", bytes, sizes[0])
	}

	div := int64(k)
	for i := 1; i < len(sizes); i++ {
		if bytes < div*k || i == len(sizes)-1 {
			value := float64(bytes) / float64(div)
			if value == float64(int64(value)) {
				return fmt.Sprintf("%.0f %s", value, sizes[i])
			}
			return fmt.Sprintf("%.1f %s", value, sizes[i])
		}
		div *= k
	}

	return fmt.Sprintf("%d %s", bytes, "Bytes")
}

func TestManager_BulkOperations(t *testing.T) {
	repo := storage.NewMockRepository()
	manager := NewManager(repo)
	ctx := context.Background()

	// Create test directory
	dir, err := manager.CreateDirectory(ctx, "Test Dir", "./test", nil, false)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create test files
	files := make([]*types.File, 3)
	fileIDs := make([]string, 3)

	for i := 0; i < 3; i++ {
		file := &types.File{
			ID:          fmt.Sprintf("file%d", i),
			Filename:    fmt.Sprintf("test%d.txt", i),
			FilePath:    fmt.Sprintf("./test/test%d.txt", i),
			DirectoryID: dir.ID,
			FileSize:    100,
			MimeType:    "text/plain",
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
			Tags:        []string{},
		}

		if err := repo.CreateFile(ctx, file); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		files[i] = file
		fileIDs[i] = file.ID
	}

	t.Run("BulkTagFiles", func(t *testing.T) {
		tags := []string{"test", "bulk"}
		err := manager.BulkTagFiles(ctx, fileIDs, tags)
		if err != nil {
			t.Errorf("BulkTagFiles() error = %v", err)
		}

		// Verify tags were added (note: this would need actual tag implementation in mock)
	})

	t.Run("BulkMoveFiles", func(t *testing.T) {
		// Create target directory
		targetDir, err := manager.CreateDirectory(ctx, "Target Dir", "./target", nil, false)
		if err != nil {
			t.Fatalf("Failed to create target directory: %v", err)
		}

		// Note: This test would fail because we can't actually move files in the mock
		// But it tests the interface
		err = manager.BulkMoveFiles(ctx, fileIDs[:2], targetDir.ID)
		if err == nil {
			t.Error("Expected error for mock file move, but got none")
		}
	})

	t.Run("BulkDeleteFiles", func(t *testing.T) {
		// Test with non-existent files to verify error handling
		nonExistentIDs := []string{"nonexistent1", "nonexistent2"}
		err := manager.BulkDeleteFiles(ctx, nonExistentIDs)
		if err == nil {
			t.Error("Expected error for deleting non-existent files, but got none")
		}
	})
}

func TestManager_GetTaskFiles(t *testing.T) {
	repo := storage.NewMockRepository()
	manager := NewManager(repo)
	ctx := context.Background()

	// Create test directory
	dir, err := manager.CreateDirectory(ctx, "Test Dir", "./test", nil, false)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	taskID := "test-task-123"

	// Create files with and without task association
	files := []*types.File{
		{
			ID:          "file1",
			Filename:    "task-file1.txt",
			FilePath:    "./test/task-file1.txt",
			DirectoryID: dir.ID,
			TaskID:      &taskID,
			FileSize:    100,
			MimeType:    "text/plain",
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
			Tags:        []string{},
		},
		{
			ID:          "file2",
			Filename:    "task-file2.txt",
			FilePath:    "./test/task-file2.txt",
			DirectoryID: dir.ID,
			TaskID:      &taskID,
			FileSize:    200,
			MimeType:    "text/plain",
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
			Tags:        []string{},
		},
		{
			ID:          "file3",
			Filename:    "other-file.txt",
			FilePath:    "./test/other-file.txt",
			DirectoryID: dir.ID,
			TaskID:      nil, // No task association
			FileSize:    300,
			MimeType:    "text/plain",
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
			Tags:        []string{},
		},
	}

	for _, file := range files {
		if createErr := repo.CreateFile(ctx, file); createErr != nil {
			t.Fatalf("Failed to create test file: %v", createErr)
		}
	}

	// Test getting task files
	taskFiles, err := manager.GetTaskFiles(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskFiles() error = %v", err)
	}

	expectedCount := 2
	if len(taskFiles) != expectedCount {
		t.Errorf("Expected %d task files, got %d", expectedCount, len(taskFiles))
	}

	// Verify all returned files have the correct task ID
	for _, file := range taskFiles {
		if file.TaskID == nil || *file.TaskID != taskID {
			t.Errorf("File %s has incorrect task ID: %v", file.ID, file.TaskID)
		}
	}
}
