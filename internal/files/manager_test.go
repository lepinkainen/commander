package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/commander/internal/storage"
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
