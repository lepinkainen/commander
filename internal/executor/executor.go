package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/lepinkainen/commander/internal/task"
)

// Tool represents a CLI tool configuration
type Tool struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Workers     int      `json:"workers,omitempty"`
	Args        []string `json:"default_args,omitempty"`
}

// Config represents the tools configuration
type Config struct {
	Tools []Tool `json:"tools"`
}

// Executor manages command execution
type Executor struct {
	config  Config
	manager *task.Manager
	workers int
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewExecutor creates a new executor
func NewExecutor(configPath string, defaultWorkers int, manager *task.Manager) (*Executor, error) {
	// Load configuration
	file, err := os.Open(configPath)
	if err != nil {
		// Create default config if file doesn't exist
		if os.IsNotExist(err) {
			return createDefaultExecutor(configPath, defaultWorkers, manager)
		}
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Executor{
		config:  config,
		manager: manager,
		workers: defaultWorkers,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// createDefaultExecutor creates an executor with default configuration
func createDefaultExecutor(configPath string, defaultWorkers int, manager *task.Manager) (*Executor, error) {
	config := Config{
		Tools: []Tool{
			{
				Name:        "yt-dlp",
				Command:     "yt-dlp",
				Description: "YouTube downloader",
				Workers:     2,
			},
			{
				Name:        "gallery-dl",
				Command:     "gallery-dl",
				Description: "Gallery downloader",
				Workers:     2,
			},
			{
				Name:        "wget",
				Command:     "wget",
				Description: "Web downloader",
				Workers:     4,
			},
			{
				Name:        "ffmpeg",
				Command:     "ffmpeg",
				Description: "Media converter",
				Workers:     2,
			},
			{
				Name:        "curl",
				Command:     "curl",
				Description: "HTTP client",
				Workers:     4,
			},
		},
	}

	// Save default config
	os.MkdirAll("./config", 0755)
	file, err := os.Create(configPath)
	if err != nil {
		log.Printf("Warning: failed to save default config: %v", err)
	} else {
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encoder.Encode(config)
		file.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Executor{
		config:  config,
		manager: manager,
		workers: defaultWorkers,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start starts the executor workers
func (e *Executor) Start() error {
	for _, tool := range e.config.Tools {
		workers := tool.Workers
		if workers == 0 {
			workers = e.workers
		}

		// Create queue for this tool
		queue := e.manager.CreateQueue(tool.Name, 100)

		// Start workers for this tool
		for i := 0; i < workers; i++ {
			e.wg.Add(1)
			go e.worker(tool, queue)
		}

		log.Printf("Started %d workers for %s", workers, tool.Name)
	}

	return nil
}

// Stop stops all workers
func (e *Executor) Stop() {
	e.cancel()
	e.wg.Wait()
}

// worker processes tasks from a queue
func (e *Executor) worker(tool Tool, queue chan *task.Task) {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case t := <-queue:
			if t == nil {
				return
			}
			e.executeTask(tool, t)
		}
	}
}

// executeTask executes a single task
func (e *Executor) executeTask(tool Tool, t *task.Task) {
	log.Printf("Executing task %s with %s", t.ID, tool.Name)

	// Update status to running
	e.manager.UpdateTaskStatus(t.ID, task.StatusRunning)

	// Prepare command
	args := append(tool.Args, t.Args...)
	cmd := exec.CommandContext(e.ctx, t.Command, args...)

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.SetError(fmt.Sprintf("Failed to create stdout pipe: %v", err))
		e.manager.UpdateTaskStatus(t.ID, task.StatusFailed)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.SetError(fmt.Sprintf("Failed to create stderr pipe: %v", err))
		e.manager.UpdateTaskStatus(t.ID, task.StatusFailed)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		t.SetError(fmt.Sprintf("Failed to start command: %v", err))
		e.manager.UpdateTaskStatus(t.ID, task.StatusFailed)
		return
	}

	// Create a wait group for output readers
	var outputWg sync.WaitGroup
	outputWg.Add(2)

	// Read stdout
	go func() {
		defer outputWg.Done()
		e.readOutput(t.ID, stdout, false)
	}()

	// Read stderr
	go func() {
		defer outputWg.Done()
		e.readOutput(t.ID, stderr, true)
	}()

	// Wait for output readers to finish
	outputWg.Wait()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		if e.ctx.Err() != nil {
			// Context was canceled
			e.manager.UpdateTaskStatus(t.ID, task.StatusCanceled)
		} else {
			t.SetError(fmt.Sprintf("Command failed: %v", err))
			e.manager.UpdateTaskStatus(t.ID, task.StatusFailed)
		}
		return
	}

	e.manager.UpdateTaskStatus(t.ID, task.StatusComplete)
	log.Printf("Task %s completed successfully", t.ID)
}

// readOutput reads output from a pipe and sends it to the manager
func (e *Executor) readOutput(taskID string, pipe io.Reader, isError bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if isError {
			line = "[ERROR] " + line
		}
		e.manager.AppendTaskOutput(taskID, line)
	}
}

// GetTools returns the configured tools
func (e *Executor) GetTools() []Tool {
	return e.config.Tools
}

// IsToolAvailable checks if a tool is configured
func (e *Executor) IsToolAvailable(toolName string) bool {
	for _, tool := range e.config.Tools {
		if tool.Name == toolName {
			return true
		}
	}
	return false
}
