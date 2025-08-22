package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/commander/internal/storage"
)

func TestFileDiscovery_DiscoverFilesFromOutput(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.mp4")
	testFile2 := filepath.Join(tempDir, "test2.mkv")

	// Create the test files
	if err := os.WriteFile(testFile1, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock file repository
	repo := storage.NewMockRepository()
	fileManager := NewManager(repo)
	discovery := NewFileDiscovery(fileManager)

	tests := []struct {
		name       string
		toolName   string
		output     []string
		expectLen  int
		expectFile string
	}{
		{
			name:     "yt-dlp destination detection",
			toolName: "yt-dlp",
			output: []string{
				"[download] Destination: " + testFile1,
				"Some other output",
			},
			expectLen:  1,
			expectFile: testFile1,
		},
		{
			name:     "yt-dlp ffmpeg merge detection",
			toolName: "yt-dlp",
			output: []string{
				"[ffmpeg] Merging formats into \"" + testFile2 + "\"",
				"Some other output",
			},
			expectLen:  1,
			expectFile: testFile2,
		},
		{
			name:     "wget save detection",
			toolName: "wget",
			output: []string{
				"saving to: '" + testFile1 + "'",
				"Progress output",
			},
			expectLen:  1,
			expectFile: testFile1,
		},
		{
			name:     "no matching patterns",
			toolName: "unknown-tool",
			output: []string{
				"Random output without file paths",
				"Another line",
			},
			expectLen: 0,
		},
		{
			name:     "file doesn't exist",
			toolName: "yt-dlp",
			output: []string{
				"[download] Destination: /nonexistent/file.mp4",
			},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			discovered, err := discovery.DiscoverFilesFromOutput(ctx, "task123", tt.toolName, tt.output)
			if err != nil {
				t.Fatalf("DiscoverFilesFromOutput() error = %v", err)
			}

			if len(discovered) != tt.expectLen {
				t.Errorf("Expected %d files, got %d", tt.expectLen, len(discovered))
			}

			if tt.expectLen > 0 && len(discovered) > 0 {
				if discovered[0] != tt.expectFile {
					t.Errorf("Expected file %s, got %s", tt.expectFile, discovered[0])
				}
			}
		})
	}
}

func TestFileDiscovery_DeduplicateFiles(t *testing.T) {
	repo := storage.NewMockRepository()
	fileManager := NewManager(repo)
	discovery := NewFileDiscovery(fileManager)

	input := []string{
		"/path/to/file1.mp4",
		"/path/to/file2.mp4",
		"/path/to/file1.mp4", // duplicate
		"/path/to/file3.mp4",
		"/path/to/file2.mp4", // duplicate
	}

	result := discovery.deduplicateFiles(input)

	expectedLen := 3
	if len(result) != expectedLen {
		t.Errorf("Expected %d unique files, got %d", expectedLen, len(result))
	}

	// Check that all expected files are present
	expected := map[string]bool{
		"/path/to/file1.mp4": false,
		"/path/to/file2.mp4": false,
		"/path/to/file3.mp4": false,
	}

	for _, file := range result {
		if _, ok := expected[file]; ok {
			expected[file] = true
		} else {
			t.Errorf("Unexpected file in result: %s", file)
		}
	}

	for file, found := range expected {
		if !found {
			t.Errorf("Expected file not found in result: %s", file)
		}
	}
}

func TestFileDiscovery_GetOrCreateToolDirectory(t *testing.T) {
	repo := storage.NewMockRepository()
	fileManager := NewManager(repo)
	discovery := NewFileDiscovery(fileManager)
	ctx := context.Background()

	// Test creating a new tool directory
	toolName := "test-tool"
	dir, err := discovery.GetOrCreateToolDirectory(ctx, toolName)
	if err != nil {
		t.Fatalf("GetOrCreateToolDirectory() error = %v", err)
	}

	if dir.ToolName == nil || *dir.ToolName != toolName {
		t.Errorf("Expected tool name %s, got %v", toolName, dir.ToolName)
	}

	expectedPath := filepath.Join("downloads", toolName)
	if dir.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, dir.Path)
	}

	// Test getting existing tool directory
	dir2, err := discovery.GetOrCreateToolDirectory(ctx, toolName)
	if err != nil {
		t.Fatalf("GetOrCreateToolDirectory() error = %v", err)
	}

	if dir.ID != dir2.ID {
		t.Errorf("Expected same directory ID, got different directories")
	}
}
