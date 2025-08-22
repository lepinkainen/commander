package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lepinkainen/commander/internal/types"
)

// FileDiscovery handles automatic file discovery from task output
type FileDiscovery struct {
	fileManager *Manager
}

// NewFileDiscovery creates a new file discovery service
func NewFileDiscovery(fileManager *Manager) *FileDiscovery {
	return &FileDiscovery{
		fileManager: fileManager,
	}
}

// FilePattern represents patterns for detecting files in task output
type FilePattern struct {
	Tool        string
	Pattern     *regexp.Regexp
	Description string
}

// Common file detection patterns for different tools
var fileDetectionPatterns = []FilePattern{
	{
		Tool:        "yt-dlp",
		Pattern:     regexp.MustCompile(`\[download\] Destination: (.+)`),
		Description: "YouTube download destination",
	},
	{
		Tool:        "yt-dlp",
		Pattern:     regexp.MustCompile(`\[download\] (.+\.(?:mp4|mkv|webm|m4a|mp3|opus|flac))\s+has already been downloaded`),
		Description: "Already downloaded file",
	},
	{
		Tool:        "yt-dlp",
		Pattern:     regexp.MustCompile(`\[ffmpeg\] Merging formats into "(.+)"`),
		Description: "Merged output file",
	},
	{
		Tool:        "wget",
		Pattern:     regexp.MustCompile(`saving to: ['"](.+)['"]`),
		Description: "Wget download target",
	},
	{
		Tool:        "wget",
		Pattern:     regexp.MustCompile(`'(.+)' saved \[\d+/\d+\]`),
		Description: "Wget saved file",
	},
	{
		Tool:        "gallery-dl",
		Pattern:     regexp.MustCompile(`\[(.+)\] (.+\.[a-zA-Z0-9]+)$`),
		Description: "Gallery download",
	},
	{
		Tool:        "ffmpeg",
		Pattern:     regexp.MustCompile(`Output #0, .+, to '(.+)':`),
		Description: "FFmpeg output file",
	},
	{
		Tool:        "curl",
		Pattern:     regexp.MustCompile(`% Total.+\s+(.+)$`),
		Description: "Curl download output (if -o specified)",
	},
}

// DiscoverFilesFromOutput analyzes task output and discovers created files
func (fd *FileDiscovery) DiscoverFilesFromOutput(ctx context.Context, taskID, toolName string, output []string) ([]string, error) {
	var discoveredFiles []string

	// Get patterns for this tool
	toolPatterns := make([]*regexp.Regexp, 0)
	for _, pattern := range fileDetectionPatterns {
		if pattern.Tool == toolName {
			toolPatterns = append(toolPatterns, pattern.Pattern)
		}
	}

	if len(toolPatterns) == 0 {
		// No patterns for this tool, try generic file path detection
		toolPatterns = append(toolPatterns, regexp.MustCompile(`([/\w\-.]+\.[a-zA-Z0-9]{2,4})`))
	}

	// Analyze each output line
	for _, line := range output {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[ERROR]") {
			continue
		}

		for _, pattern := range toolPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				filePath := strings.Trim(matches[1], "\"'")

				// Validate file exists and is not a directory
				if fd.isValidFile(filePath) {
					discoveredFiles = append(discoveredFiles, filePath)
				}
			}
		}
	}

	return fd.deduplicateFiles(discoveredFiles), nil
}

// isValidFile checks if the path represents a valid file
func (fd *FileDiscovery) isValidFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// deduplicateFiles removes duplicate file paths
func (fd *FileDiscovery) deduplicateFiles(files []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}

	return result
}

// RegisterDiscoveredFiles registers discovered files with the file manager
func (fd *FileDiscovery) RegisterDiscoveredFiles(ctx context.Context, taskID string, filePaths []string) error {
	for _, filePath := range filePaths {
		// Try to register with appropriate directory
		if err := fd.fileManager.RegisterFileFromTask(ctx, taskID, filePath, nil); err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to register file %s for task %s: %v\n", filePath, taskID, err)
		}
	}
	return nil
}

// GetOrCreateToolDirectory gets or creates a directory for a specific tool
func (fd *FileDiscovery) GetOrCreateToolDirectory(ctx context.Context, toolName string) (*types.Directory, error) {
	// Check if tool-specific directory exists
	dirs, err := fd.fileManager.GetFileRepository().ListDirectories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}

	for _, dir := range dirs {
		if dir.ToolName != nil && *dir.ToolName == toolName {
			return dir, nil
		}
	}

	// Create tool-specific directory
	toolPath := filepath.Join("downloads", toolName)
	// Capitalize first letter of tool name for display
	displayName := strings.ToUpper(toolName[:1]) + toolName[1:]
	return fd.fileManager.CreateDirectory(ctx, fmt.Sprintf("%s Downloads", displayName), toolPath, &toolName, false)
}

// OrganizeFilesByPattern organizes files using tool/date patterns
func (fd *FileDiscovery) OrganizeFilesByPattern(ctx context.Context, taskID, toolName string, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	// Get or create tool directory
	toolDir, err := fd.GetOrCreateToolDirectory(ctx, toolName)
	if err != nil {
		return fmt.Errorf("failed to get/create tool directory: %w", err)
	}

	// Create date-based subdirectory
	dateStr := time.Now().Format("2006-01-02")
	datePath := filepath.Join(toolDir.Path, dateStr)

	// Ensure date directory exists
	if err := os.MkdirAll(datePath, 0o755); err != nil {
		return fmt.Errorf("failed to create date directory: %w", err)
	}

	// Move files to organized structure
	for _, filePath := range filePaths {
		filename := filepath.Base(filePath)
		targetPath := filepath.Join(datePath, filename)

		// Only move if not already in the target location
		if filePath != targetPath {
			if err := os.Rename(filePath, targetPath); err != nil {
				fmt.Printf("Warning: failed to move file %s to %s: %v\n", filePath, targetPath, err)
				continue
			}

			// Register the file in its new location
			if err := fd.fileManager.RegisterFileFromTask(ctx, taskID, targetPath, &toolDir.ID); err != nil {
				fmt.Printf("Warning: failed to register moved file %s: %v\n", targetPath, err)
			}
		}
	}

	return nil
}
