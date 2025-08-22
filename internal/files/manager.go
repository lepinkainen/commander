package files

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lepinkainen/commander/internal/storage"
	"github.com/lepinkainen/commander/internal/types"
)

// Manager handles file and directory operations
type Manager struct {
	fileRepo storage.FileRepository
}

// NewManager creates a new file manager
func NewManager(fileRepo storage.FileRepository) *Manager {
	return &Manager{
		fileRepo: fileRepo,
	}
}

// CreateDirectory creates a new download directory
func (m *Manager) CreateDirectory(ctx context.Context, name, path string, toolName *string, defaultDir bool) (*types.Directory, error) {
	dir := &types.Directory{
		ID:         uuid.New().String(),
		Name:       name,
		Path:       path,
		ToolName:   toolName,
		DefaultDir: defaultDir,
		CreatedAt:  time.Now(),
	}

	// Create the directory on filesystem if it doesn't exist
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.fileRepo.CreateDirectory(ctx, dir); err != nil {
		return nil, fmt.Errorf("failed to save directory: %w", err)
	}

	return dir, nil
}

// ScanDirectory scans a directory for files and adds them to the database
func (m *Manager) ScanDirectory(ctx context.Context, directoryID string) error {
	dir, err := m.fileRepo.GetDirectory(ctx, directoryID)
	if err != nil {
		return fmt.Errorf("failed to get directory: %w", err)
	}

	return filepath.WalkDir(dir.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file already exists in database
		existingFiles, err := m.fileRepo.ListFiles(ctx, types.FileFilters{
			DirectoryID: directoryID,
		})
		if err != nil {
			return err
		}

		// Check if this file path already exists
		for _, existing := range existingFiles {
			if existing.FilePath == path {
				return nil // File already tracked
			}
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Detect MIME type
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Create file record
		file := &types.File{
			ID:          uuid.New().String(),
			Filename:    d.Name(),
			FilePath:    path,
			DirectoryID: directoryID,
			FileSize:    info.Size(),
			MimeType:    mimeType,
			CreatedAt:   info.ModTime(),
			AccessedAt:  time.Now(),
			Tags:        []string{},
		}

		return m.fileRepo.CreateFile(ctx, file)
	})
}

// RegisterFileFromTask registers a file that was created by a task
func (m *Manager) RegisterFileFromTask(ctx context.Context, taskID, filePath string, directoryID *string) error {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// If no directory specified, use default or create one
	var targetDirID string
	if directoryID != nil {
		targetDirID = *directoryID
	} else {
		// Find or create default directory
		dirs, err := m.fileRepo.ListDirectories(ctx)
		if err != nil {
			return fmt.Errorf("failed to list directories: %w", err)
		}

		var defaultDir *types.Directory
		for _, dir := range dirs {
			if dir.DefaultDir {
				defaultDir = dir
				break
			}
		}

		if defaultDir == nil {
			// Create default directory
			defaultPath := "./downloads"
			defaultDir, err = m.CreateDirectory(ctx, "Default Downloads", defaultPath, nil, true)
			if err != nil {
				return fmt.Errorf("failed to create default directory: %w", err)
			}
		}

		targetDirID = defaultDir.ID
	}

	// Detect MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Create file record
	file := &types.File{
		ID:          uuid.New().String(),
		Filename:    filepath.Base(filePath),
		FilePath:    filePath,
		DirectoryID: targetDirID,
		TaskID:      &taskID,
		FileSize:    info.Size(),
		MimeType:    mimeType,
		CreatedAt:   info.ModTime(),
		AccessedAt:  time.Now(),
		Tags:        []string{},
	}

	return m.fileRepo.CreateFile(ctx, file)
}

// MoveFile moves a file from one directory to another
func (m *Manager) MoveFile(ctx context.Context, fileID, targetDirID string) error {
	file, err := m.fileRepo.GetFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	targetDir, err := m.fileRepo.GetDirectory(ctx, targetDirID)
	if err != nil {
		return fmt.Errorf("failed to get target directory: %w", err)
	}

	// Calculate new file path
	newPath := filepath.Join(targetDir.Path, file.Filename)

	// Move the actual file
	if err := os.Rename(file.FilePath, newPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Update database record
	file.DirectoryID = targetDirID
	file.FilePath = newPath
	file.AccessedAt = time.Now()

	return m.fileRepo.UpdateFile(ctx, file)
}

// DeleteFile removes a file from both filesystem and database
func (m *Manager) DeleteFile(ctx context.Context, fileID string) error {
	file, err := m.fileRepo.GetFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Remove from filesystem
	if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file from filesystem: %w", err)
	}

	// Remove from database
	return m.fileRepo.DeleteFile(ctx, fileID)
}

// FindDuplicateFiles finds files with the same content (by comparing file size and paths)
func (m *Manager) FindDuplicateFiles(ctx context.Context, directoryID string) ([][]*types.File, error) {
	files, err := m.fileRepo.ListFiles(ctx, types.FileFilters{
		DirectoryID: directoryID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Group files by size
	sizeGroups := make(map[int64][]*types.File)
	for _, file := range files {
		sizeGroups[file.FileSize] = append(sizeGroups[file.FileSize], file)
	}

	var duplicates [][]*types.File
	for _, group := range sizeGroups {
		if len(group) > 1 {
			// Further group by filename for potential duplicates
			nameGroups := make(map[string][]*types.File)
			for _, file := range group {
				nameGroups[file.Filename] = append(nameGroups[file.Filename], file)
			}

			for _, nameGroup := range nameGroups {
				if len(nameGroup) > 1 {
					duplicates = append(duplicates, nameGroup)
				}
			}
		}
	}

	return duplicates, nil
}

// GetDirectoryUsage calculates storage usage for a directory
func (m *Manager) GetDirectoryUsage(ctx context.Context, directoryID string) (totalSize int64, fileCount int, err error) {
	fileList, err := m.fileRepo.ListFiles(ctx, types.FileFilters{
		DirectoryID: directoryID,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list files: %w", err)
	}

	fileCount = len(fileList)

	for _, file := range fileList {
		totalSize += file.FileSize
	}

	return totalSize, fileCount, nil
}

// SearchFiles searches for files by name or content
func (m *Manager) SearchFiles(ctx context.Context, query string) ([]*types.File, error) {
	return m.fileRepo.SearchFiles(ctx, query)
}

// TagFile adds tags to a file
func (m *Manager) TagFile(ctx context.Context, fileID string, tags []string) error {
	for _, tag := range tags {
		if err := m.fileRepo.AddFileTag(ctx, fileID, strings.TrimSpace(tag)); err != nil {
			return fmt.Errorf("failed to add tag %s: %w", tag, err)
		}
	}
	return nil
}

// UntagFile removes tags from a file
func (m *Manager) UntagFile(ctx context.Context, fileID string, tags []string) error {
	for _, tag := range tags {
		if err := m.fileRepo.RemoveFileTag(ctx, fileID, strings.TrimSpace(tag)); err != nil {
			return fmt.Errorf("failed to remove tag %s: %w", tag, err)
		}
	}
	return nil
}

// BulkDeleteFiles deletes multiple files by their IDs
func (m *Manager) BulkDeleteFiles(ctx context.Context, fileIDs []string) error {
	var failures []string

	for _, fileID := range fileIDs {
		if err := m.DeleteFile(ctx, fileID); err != nil {
			failures = append(failures, fmt.Sprintf("file %s: %v", fileID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to delete some files: %s", strings.Join(failures, "; "))
	}

	return nil
}

// BulkMoveFiles moves multiple files to a target directory
func (m *Manager) BulkMoveFiles(ctx context.Context, fileIDs []string, targetDirID string) error {
	var failures []string

	for _, fileID := range fileIDs {
		if err := m.MoveFile(ctx, fileID, targetDirID); err != nil {
			failures = append(failures, fmt.Sprintf("file %s: %v", fileID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to move some files: %s", strings.Join(failures, "; "))
	}

	return nil
}

// BulkTagFiles adds tags to multiple files
func (m *Manager) BulkTagFiles(ctx context.Context, fileIDs []string, tags []string) error {
	var failures []string

	for _, fileID := range fileIDs {
		if err := m.TagFile(ctx, fileID, tags); err != nil {
			failures = append(failures, fmt.Sprintf("file %s: %v", fileID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to tag some files: %s", strings.Join(failures, "; "))
	}

	return nil
}

// GetTaskFiles returns all files associated with a specific task
func (m *Manager) GetTaskFiles(ctx context.Context, taskID string) ([]*types.File, error) {
	// Get all files from the database and filter by task ID
	allFiles, err := m.fileRepo.ListFiles(ctx, types.FileFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var taskFiles []*types.File
	for _, file := range allFiles {
		if file.TaskID != nil && *file.TaskID == taskID {
			taskFiles = append(taskFiles, file)
		}
	}

	return taskFiles, nil
}

// GetFileRepository returns the underlying file repository
func (m *Manager) GetFileRepository() storage.FileRepository {
	return m.fileRepo
}
