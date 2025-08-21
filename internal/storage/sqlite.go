package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lepinkainen/commander/internal/types"
)

// SQLiteRepository implements TaskRepository using SQLite
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

	CREATE INDEX IF NOT EXISTS idx_tasks_tool ON tasks(tool);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_task_outputs_task_id ON task_outputs(task_id);
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

	if err := json.Unmarshal([]byte(argsJSON), &data.Args); err != nil {
		return types.TaskData{}, fmt.Errorf("failed to unmarshal args: %w", err)
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
	defer rows.Close()

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
	defer rows.Close()

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

		if err := json.Unmarshal([]byte(argsJSON), &data.Args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", err)
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
				outputRows.Close()
				return nil, fmt.Errorf("failed to scan output: %w", err)
			}
			output = append(output, line)
		}
		outputRows.Close()
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
	defer rows.Close()

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

		if err := json.Unmarshal([]byte(argsJSON), &data.Args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", err)
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
				outputRows.Close()
				return nil, fmt.Errorf("failed to scan output: %w", err)
			}
			output = append(output, line)
		}
		outputRows.Close()
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
