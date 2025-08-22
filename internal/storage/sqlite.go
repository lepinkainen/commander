package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lepinkainen/commander/internal/types"
)

// SQLiteRepository implements TaskRepository and FileRepository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepository{db: db}

	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

// createTables creates the necessary database tables
func (r *SQLiteRepository) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		tool TEXT NOT NULL,
		command TEXT NOT NULL,
		args TEXT NOT NULL, -- JSON array
		status TEXT NOT NULL,
		error TEXT,
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		ended_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS task_outputs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		output TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks (id)
	);

	CREATE TABLE IF NOT EXISTS download_directories (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		path TEXT NOT NULL,
		tool_name TEXT,
		default_dir BOOLEAN DEFAULT false,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (tool_name) REFERENCES tools(name)
	);

	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		file_path TEXT NOT NULL,
		directory_id TEXT NOT NULL,
		task_id TEXT,
		file_size INTEGER NOT NULL,
		mime_type TEXT,
		created_at DATETIME NOT NULL,
		accessed_at DATETIME NOT NULL,
		FOREIGN KEY (directory_id) REFERENCES download_directories(id),
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);

	CREATE TABLE IF NOT EXISTS file_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		FOREIGN KEY (file_id) REFERENCES files(id),
		UNIQUE(file_id, tag)
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_tool ON tasks(tool);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_task_outputs_task_id ON task_outputs(task_id);
	CREATE INDEX IF NOT EXISTS idx_files_directory_id ON files(directory_id);
	CREATE INDEX IF NOT EXISTS idx_files_task_id ON files(task_id);
	CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_files_path ON files(file_path);
	CREATE INDEX IF NOT EXISTS idx_file_tags_file_id ON file_tags(file_id);
	`

	_, err := r.db.Exec(schema)
	return err
}

// Create adds a new task to storage
func (r *SQLiteRepository) Create(ctx context.Context, data types.TaskData) error {
	argsJSON, err := json.Marshal(data.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		INSERT INTO tasks (id, tool, command, args, status, error, created_at, started_at, ended_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var startedAt, endedAt interface{}
	if !data.StartedAt.IsZero() {
		startedAt = data.StartedAt
	}
	if !data.EndedAt.IsZero() {
		endedAt = data.EndedAt
	}

	_, err = r.db.ExecContext(ctx, query,
		data.ID, data.Tool, data.Command, string(argsJSON), string(data.Status),
		data.Error, data.CreatedAt, startedAt, endedAt)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Insert existing output if any
	for _, output := range data.Output {
		if err := r.AppendOutput(ctx, data.ID, output); err != nil {
			return fmt.Errorf("failed to insert existing output: %w", err)
		}
	}

	return nil
}

// GetByID retrieves a task by its ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (types.TaskData, error) {
	query := `
		SELECT id, tool, command, args, status, error, created_at, started_at, ended_at
		FROM tasks WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var data types.TaskData
	var argsJSON string
	var startedAt, endedAt sql.NullTime

	err := row.Scan(&data.ID, &data.Tool, &data.Command, &argsJSON, &data.Status,
		&data.Error, &data.CreatedAt, &startedAt, &endedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return types.TaskData{}, fmt.Errorf("task %s not found", id)
		}
		return types.TaskData{}, fmt.Errorf("failed to get task: %w", err)
	}

	if unmarshalErr := json.Unmarshal([]byte(argsJSON), &data.Args); unmarshalErr != nil {
		return types.TaskData{}, fmt.Errorf("failed to unmarshal args: %w", unmarshalErr)
	}

	if startedAt.Valid {
		data.StartedAt = startedAt.Time
	}
	if endedAt.Valid {
		data.EndedAt = endedAt.Time
	}

	// Get output
	outputQuery := `SELECT output FROM task_outputs WHERE task_id = ? ORDER BY timestamp`
	rows, err := r.db.QueryContext(ctx, outputQuery, id)
	if err != nil {
		return types.TaskData{}, fmt.Errorf("failed to get task output: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var output []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return types.TaskData{}, fmt.Errorf("failed to scan output: %w", err)
		}
		output = append(output, line)
	}
	data.Output = output

	return data, nil
}

// List retrieves all tasks
func (r *SQLiteRepository) List(ctx context.Context) ([]types.TaskData, error) {
	query := `
		SELECT id, tool, command, args, status, error, created_at, started_at, ended_at
		FROM tasks ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var tasks []types.TaskData
	for rows.Next() {
		var data types.TaskData
		var argsJSON string
		var startedAt, endedAt sql.NullTime

		err := rows.Scan(&data.ID, &data.Tool, &data.Command, &argsJSON, &data.Status,
			&data.Error, &data.CreatedAt, &startedAt, &endedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if unmarshalErr := json.Unmarshal([]byte(argsJSON), &data.Args); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", unmarshalErr)
		}

		if startedAt.Valid {
			data.StartedAt = startedAt.Time
		}
		if endedAt.Valid {
			data.EndedAt = endedAt.Time
		}

		// Get output for this task
		outputQuery := `SELECT output FROM task_outputs WHERE task_id = ? ORDER BY timestamp`
		outputRows, err := r.db.QueryContext(ctx, outputQuery, data.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task output: %w", err)
		}

		var output []string
		for outputRows.Next() {
			var line string
			if err := outputRows.Scan(&line); err != nil {
				if closeErr := outputRows.Close(); closeErr != nil {
					log.Printf("Error closing output rows: %v", closeErr)
				}
				return nil, fmt.Errorf("failed to scan output: %w", err)
			}
			output = append(output, line)
		}
		if err := outputRows.Close(); err != nil {
			log.Printf("Error closing output rows: %v", err)
		}
		data.Output = output

		tasks = append(tasks, data)
	}

	return tasks, nil
}

// ListByTool retrieves tasks for a specific tool
func (r *SQLiteRepository) ListByTool(ctx context.Context, tool string) ([]types.TaskData, error) {
	query := `
		SELECT id, tool, command, args, status, error, created_at, started_at, ended_at
		FROM tasks WHERE tool = ? ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tool)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by tool: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var tasks []types.TaskData
	for rows.Next() {
		var data types.TaskData
		var argsJSON string
		var startedAt, endedAt sql.NullTime

		err := rows.Scan(&data.ID, &data.Tool, &data.Command, &argsJSON, &data.Status,
			&data.Error, &data.CreatedAt, &startedAt, &endedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if unmarshalErr := json.Unmarshal([]byte(argsJSON), &data.Args); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", unmarshalErr)
		}

		if startedAt.Valid {
			data.StartedAt = startedAt.Time
		}
		if endedAt.Valid {
			data.EndedAt = endedAt.Time
		}

		// Get output for this task
		outputQuery := `SELECT output FROM task_outputs WHERE task_id = ? ORDER BY timestamp`
		outputRows, err := r.db.QueryContext(ctx, outputQuery, data.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task output: %w", err)
		}

		var output []string
		for outputRows.Next() {
			var line string
			if err := outputRows.Scan(&line); err != nil {
				if closeErr := outputRows.Close(); closeErr != nil {
					log.Printf("Error closing output rows: %v", closeErr)
				}
				return nil, fmt.Errorf("failed to scan output: %w", err)
			}
			output = append(output, line)
		}
		if err := outputRows.Close(); err != nil {
			log.Printf("Error closing output rows: %v", err)
		}
		data.Output = output

		tasks = append(tasks, data)
	}

	return tasks, nil
}

// Update updates an existing task
func (r *SQLiteRepository) Update(ctx context.Context, data types.TaskData) error {
	argsJSON, err := json.Marshal(data.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		UPDATE tasks 
		SET tool = ?, command = ?, args = ?, status = ?, error = ?, 
		    created_at = ?, started_at = ?, ended_at = ?
		WHERE id = ?
	`

	var startedAt, endedAt interface{}
	if !data.StartedAt.IsZero() {
		startedAt = data.StartedAt
	}
	if !data.EndedAt.IsZero() {
		endedAt = data.EndedAt
	}

	_, err = r.db.ExecContext(ctx, query,
		data.Tool, data.Command, string(argsJSON), string(data.Status),
		data.Error, data.CreatedAt, startedAt, endedAt, data.ID)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// AppendOutput adds output to a task
func (r *SQLiteRepository) AppendOutput(ctx context.Context, taskID string, output string) error {
	// Skip empty output
	if strings.TrimSpace(output) == "" {
		return nil
	}

	query := `INSERT INTO task_outputs (task_id, output) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, taskID, output)
	if err != nil {
		return fmt.Errorf("failed to append output: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// Directory operations

// CreateDirectory adds a new directory to storage
func (r *SQLiteRepository) CreateDirectory(ctx context.Context, dir *types.Directory) error {
	query := `
		INSERT INTO download_directories (id, name, path, tool_name, default_dir, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, dir.ID, dir.Name, dir.Path, dir.ToolName, dir.DefaultDir, dir.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// GetDirectory retrieves a directory by its ID
func (r *SQLiteRepository) GetDirectory(ctx context.Context, id string) (*types.Directory, error) {
	query := `
		SELECT id, name, path, tool_name, default_dir, created_at
		FROM download_directories WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var dir types.Directory
	var toolName sql.NullString

	err := row.Scan(&dir.ID, &dir.Name, &dir.Path, &toolName, &dir.DefaultDir, &dir.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("directory %s not found", id)
		}
		return nil, fmt.Errorf("failed to get directory: %w", err)
	}

	if toolName.Valid {
		dir.ToolName = &toolName.String
	}

	return &dir, nil
}

// ListDirectories retrieves all directories
func (r *SQLiteRepository) ListDirectories(ctx context.Context) ([]*types.Directory, error) {
	query := `
		SELECT id, name, path, tool_name, default_dir, created_at
		FROM download_directories ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var directories []*types.Directory
	for rows.Next() {
		var dir types.Directory
		var toolName sql.NullString

		err := rows.Scan(&dir.ID, &dir.Name, &dir.Path, &toolName, &dir.DefaultDir, &dir.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}

		if toolName.Valid {
			dir.ToolName = &toolName.String
		}

		directories = append(directories, &dir)
	}

	return directories, nil
}

// UpdateDirectory updates an existing directory
func (r *SQLiteRepository) UpdateDirectory(ctx context.Context, dir *types.Directory) error {
	query := `
		UPDATE download_directories 
		SET name = ?, path = ?, tool_name = ?, default_dir = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, dir.Name, dir.Path, dir.ToolName, dir.DefaultDir, dir.ID)
	if err != nil {
		return fmt.Errorf("failed to update directory: %w", err)
	}
	return nil
}

// DeleteDirectory removes a directory from storage
func (r *SQLiteRepository) DeleteDirectory(ctx context.Context, id string) error {
	query := `DELETE FROM download_directories WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}
	return nil
}

// File operations

// CreateFile adds a new file to storage
func (r *SQLiteRepository) CreateFile(ctx context.Context, file *types.File) error {
	query := `
		INSERT INTO files (id, filename, file_path, directory_id, task_id, file_size, mime_type, created_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, file.ID, file.Filename, file.FilePath, file.DirectoryID,
		file.TaskID, file.FileSize, file.MimeType, file.CreatedAt, file.AccessedAt)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Add tags if any
	for _, tag := range file.Tags {
		if err := r.AddFileTag(ctx, file.ID, tag); err != nil {
			return fmt.Errorf("failed to add file tag: %w", err)
		}
	}

	return nil
}

// GetFile retrieves a file by its ID
func (r *SQLiteRepository) GetFile(ctx context.Context, id string) (*types.File, error) {
	query := `
		SELECT id, filename, file_path, directory_id, task_id, file_size, mime_type, created_at, accessed_at
		FROM files WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var file types.File
	var taskID sql.NullString

	err := row.Scan(&file.ID, &file.Filename, &file.FilePath, &file.DirectoryID, &taskID,
		&file.FileSize, &file.MimeType, &file.CreatedAt, &file.AccessedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file %s not found", id)
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if taskID.Valid {
		file.TaskID = &taskID.String
	}

	// Get tags
	tags, err := r.GetFileTags(ctx, file.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file tags: %w", err)
	}
	file.Tags = tags

	return &file, nil
}

// ListFiles retrieves files based on filters
func (r *SQLiteRepository) ListFiles(ctx context.Context, filters types.FileFilters) ([]*types.File, error) {
	query := `
		SELECT id, filename, file_path, directory_id, task_id, file_size, mime_type, created_at, accessed_at
		FROM files
	`
	args := []interface{}{}
	conditions := []string{}

	if filters.DirectoryID != "" {
		conditions = append(conditions, "directory_id = ?")
		args = append(args, filters.DirectoryID)
	}
	if filters.MimeType != "" {
		conditions = append(conditions, "mime_type = ?")
		args = append(args, filters.MimeType)
	}
	if filters.MinSize > 0 {
		conditions = append(conditions, "file_size >= ?")
		args = append(args, filters.MinSize)
	}
	if filters.MaxSize > 0 {
		conditions = append(conditions, "file_size <= ?")
		args = append(args, filters.MaxSize)
	}
	if filters.CreatedFrom != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *filters.CreatedFrom)
	}
	if filters.CreatedTo != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *filters.CreatedTo)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var files []*types.File
	for rows.Next() {
		var file types.File
		var taskID sql.NullString

		err := rows.Scan(&file.ID, &file.Filename, &file.FilePath, &file.DirectoryID, &taskID,
			&file.FileSize, &file.MimeType, &file.CreatedAt, &file.AccessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if taskID.Valid {
			file.TaskID = &taskID.String
		}

		// Get tags for this file
		tags, err := r.GetFileTags(ctx, file.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get file tags: %w", err)
		}
		file.Tags = tags

		files = append(files, &file)
	}

	return files, nil
}

// UpdateFile updates an existing file
func (r *SQLiteRepository) UpdateFile(ctx context.Context, file *types.File) error {
	query := `
		UPDATE files 
		SET filename = ?, file_path = ?, directory_id = ?, task_id = ?, file_size = ?, mime_type = ?, accessed_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, file.Filename, file.FilePath, file.DirectoryID,
		file.TaskID, file.FileSize, file.MimeType, file.AccessedAt, file.ID)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	return nil
}

// DeleteFile removes a file from storage
func (r *SQLiteRepository) DeleteFile(ctx context.Context, id string) error {
	// Delete file tags first (due to foreign key constraint)
	if _, err := r.db.ExecContext(ctx, "DELETE FROM file_tags WHERE file_id = ?", id); err != nil {
		return fmt.Errorf("failed to delete file tags: %w", err)
	}

	// Delete the file record
	query := `DELETE FROM files WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// File tag operations

// AddFileTag adds a tag to a file
func (r *SQLiteRepository) AddFileTag(ctx context.Context, fileID, tag string) error {
	query := `INSERT OR IGNORE INTO file_tags (file_id, tag) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, fileID, tag)
	if err != nil {
		return fmt.Errorf("failed to add file tag: %w", err)
	}
	return nil
}

// RemoveFileTag removes a tag from a file
func (r *SQLiteRepository) RemoveFileTag(ctx context.Context, fileID, tag string) error {
	query := `DELETE FROM file_tags WHERE file_id = ? AND tag = ?`
	_, err := r.db.ExecContext(ctx, query, fileID, tag)
	if err != nil {
		return fmt.Errorf("failed to remove file tag: %w", err)
	}
	return nil
}

// GetFileTags retrieves all tags for a file
func (r *SQLiteRepository) GetFileTags(ctx context.Context, fileID string) ([]string, error) {
	query := `SELECT tag FROM file_tags WHERE file_id = ? ORDER BY tag`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file tags: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// SearchFiles searches for files by filename
func (r *SQLiteRepository) SearchFiles(ctx context.Context, query string) ([]*types.File, error) {
	searchQuery := `
		SELECT id, filename, file_path, directory_id, task_id, file_size, mime_type, created_at, accessed_at
		FROM files 
		WHERE filename LIKE ? OR file_path LIKE ?
		ORDER BY created_at DESC
	`
	searchTerm := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, searchQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var files []*types.File
	for rows.Next() {
		var file types.File
		var taskID sql.NullString

		err := rows.Scan(&file.ID, &file.Filename, &file.FilePath, &file.DirectoryID, &taskID,
			&file.FileSize, &file.MimeType, &file.CreatedAt, &file.AccessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if taskID.Valid {
			file.TaskID = &taskID.String
		}

		// Get tags for this file
		tags, err := r.GetFileTags(ctx, file.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get file tags: %w", err)
		}
		file.Tags = tags

		files = append(files, &file)
	}

	return files, nil
}
