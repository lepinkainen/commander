# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

**CRITICAL**: Always run `task build` before claiming any task complete. This runs tests, linting, and builds successfully.

### Primary Workflow (Task-based)

- `task` - List all available tasks
- `task build` - **Main build command** (runs tests + lint + build)
- `task dev` - Start development server
- `task test` - Run tests only
- `task lint` - Run all linters (goimports + govet + golangci-lint)
- `task clean` - Clean build artifacts

### Code Quality (Always Required)

- `goimports -w .` - **MANDATORY** after any Go code changes (not gofmt!)
- `task test-ci` - Full CI test suite with coverage
- `task install-tools` - Install required dev tools (goimports, golangci-lint, air)

### Legacy Makefile (Still Supported)

- `make build/run/test` commands exist but `task` commands are preferred

## Architecture Overview

Commander is a web-based CLI tool manager with real-time task execution and monitoring. The architecture uses Go's **concurrency patterns** extensively for managing parallel CLI tool execution.

### Critical Design Patterns

**Worker Pool + Producer-Consumer**: Each tool (`yt-dlp`, `wget`, etc.) has its own worker pool with configurable concurrency. Tasks flow: `API → Manager → Tool-Specific Queue → Worker Pool → Execution`

**Event Broadcasting**: The Manager acts as a central event hub using Go channels, broadcasting task status changes to all WebSocket clients in real-time.

**Context-Based Cancellation**: All long-running operations use `context.Context` for graceful shutdown and task cancellation.

### Core Components

**Main Entry Point** (`cmd/server/main.go`)

- Application bootstrap with graceful shutdown
- Command-line flag parsing for server configuration
- Coordinates task manager, executor, and API server

**Task Management** (`internal/task/`)

- `Manager`: Central orchestrator with `map[string]*Task` + `map[string]chan *Task` for per-tool queues
- `Task`: Has embedded `TaskData` struct + `sync.RWMutex` for thread-safety
- **Key Pattern**: Uses Go channels for event broadcasting to WebSocket clients
- **Status Flow**: `StatusQueued → StatusRunning → StatusComplete/Failed/Canceled`
- **Threading**: All operations are mutex-protected; `Clone()` returns safe `TaskData` copies

**Command Execution** (`internal/executor/`)

- `Executor`: Creates per-tool worker goroutines (default 4, configurable per tool)
- **Critical Pattern**: Uses `exec.CommandContext()` with stdout/stderr pipes for real-time output streaming
- Tools loaded from `config/tools.json` with default args merged with runtime args
- **Error Handling**: Separate goroutines read stdout/stderr, prefix stderr with `[ERROR]`
- **Graceful Shutdown**: `context.WithCancel()` terminates all workers cleanly

**Web API** (`internal/api/`)

- REST API for task management (CRUD operations)
- WebSocket endpoint for real-time task updates
- CORS-enabled for development
- Serves static frontend files

**Frontend** (`web/static/`)

- Single-page JavaScript application
- Real-time task monitoring via WebSocket
- Tool selection and task submission interface

### Data Flow & Concurrency

1. **Task Creation**: Frontend `POST /api/tasks` → `Manager.AddTask()` → Tool-specific `chan *Task`
2. **Task Execution**: Worker goroutine → `exec.CommandContext()` → Live stdout/stderr → `Manager.AppendTaskOutput()` → WebSocket broadcast
3. **Real-time Updates**: Manager's `[]chan TaskEvent` broadcasts to all connected WebSocket clients
4. **Queue Management**: Each tool has buffered channel (size 100) with dedicated worker pool

### Configuration Patterns

**Tool Configuration** (`config/tools.json`):

```json
{
  "name": "yt-dlp",           // API identifier
  "command": "yt-dlp",        // Actual executable
  "workers": 2,               // Concurrency limit (default: 4)
  "default_args": ["--no-warnings", "--progress"],
  "description": "YouTube downloader"
}
```

**Runtime Behavior**:

- If config missing, creates default config automatically
- Args merged: `tool.default_args + request.args`
- Worker pools created at startup, not per-request

### Go-Specific Implementation Details

**Thread Safety**:

- `Manager` uses `sync.RWMutex` for task map access
- `Task` embeds `TaskData` + `sync.RWMutex` for status/output updates
- `Clone()` method provides safe concurrent reads

**Channel Usage**:

- Per-tool task queues: `make(chan *Task, 100)`
- Event broadcasting: `[]chan TaskEvent` with non-blocking sends
- WebSocket cleanup: Manager tracks and closes subscriber channels

**Error Patterns**:

- Context cancellation vs command errors distinguished in `executor.executeTask()`
- Stderr prefixed with `[ERROR]` for client-side filtering
- HTTP errors return appropriate status codes (400/404/500)

## Development Guidelines

### Code Quality Requirements

- **ALWAYS** run `goimports -w .` after Go code changes (not `gofmt`!)
- Use `task build` to ensure tests + linting pass before claiming completion
- Follow existing mutex patterns when adding concurrent operations
- New CLI tools: add to `config/tools.json`, restart server

### Architecture Constraints

- **In-Memory Only**: No persistence - all task data lost on restart
- **No Authentication**: Executes arbitrary commands - trusted environments only
- **Buffer Limits**: Task queues limited to 100 items per tool
- **WebSocket Scaling**: No connection limits implemented

### Testing Patterns

- Use `task test-ci` for coverage reports
- Focus on testing core logic, not external CLI tool integration
- `//go:build !ci` tag for tests requiring external dependencies

## Security & Operational Notes

**Security Warning**: Executes arbitrary system commands via `exec.CommandContext()`. No input sanitization or authentication. **Trusted environments only.**

**Operational Details**:

- Server defaults: `:8080` (via `-addr`), 4 workers/tool (via `-workers`)
- Graceful shutdown: 30s timeout for task completion
- CORS enabled for development (`AllowedOrigins: ["*"]`)
- Tool availability checked before task creation

**Extension Points**:

- Add tools via `config/tools.json` modification
- WebSocket protocol: `{"task_id": "...", "type": "output|status|created", "data": "..."}`
- API endpoints: RESTful `/api/tasks`, `/api/tools`, `/api/stats` + WebSocket `/api/ws`
