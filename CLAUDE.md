# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make build` - Build the application to `bin/commander`
- `make run` - Run the development server directly
- `go run cmd/server/main.go` - Alternative way to run the server
- `make dev` - Run with auto-reload (requires `air` tool)

### Testing and Code Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make fmt` - Format Go code
- `make lint` - Run golangci-lint (requires installation)
- `goimports -w <files>` - Format imports (run on changed Go files)

### Dependencies
- `make install` - Download and tidy Go modules
- `go mod tidy` - Clean up module dependencies

## Architecture Overview

Commander is a web-based CLI tool manager with real-time task execution and monitoring. The architecture follows a clean separation of concerns:

### Core Components

**Main Entry Point** (`cmd/server/main.go`)
- Application bootstrap with graceful shutdown
- Command-line flag parsing for server configuration
- Coordinates task manager, executor, and API server

**Task Management** (`internal/task/`)
- `Manager`: Central task orchestrator with event broadcasting
- `Task`: Individual command execution unit with status tracking
- Thread-safe operations with mutex protection
- WebSocket event system for real-time updates

**Command Execution** (`internal/executor/`)
- `Executor`: Manages worker pools per tool with configurable concurrency
- Tool configuration loading from `config/tools.json`
- Context-based cancellation for clean shutdown
- Separate stdout/stderr streaming to task manager

**Web API** (`internal/api/`)
- REST API for task management (CRUD operations)
- WebSocket endpoint for real-time task updates
- CORS-enabled for development
- Serves static frontend files

**Frontend** (`web/static/`)
- Single-page JavaScript application
- Real-time task monitoring via WebSocket
- Tool selection and task submission interface

### Data Flow

1. **Task Creation**: Frontend → API → Manager → Tool Queue
2. **Task Execution**: Worker Pool → Task → Output Streaming → Manager → WebSocket → Frontend
3. **Status Updates**: All components communicate through Manager's event system

### Configuration

Tools are configured in `config/tools.json` with:
- `name`: Tool identifier for API calls
- `command`: Actual executable command
- `workers`: Concurrent execution limit (defaults to 4)
- `default_args`: Arguments always passed to command
- `description`: Human-readable tool description

### Key Patterns

- **Worker Pool Pattern**: Each tool has configurable worker concurrency
- **Observer Pattern**: Event broadcasting for real-time updates
- **Producer-Consumer**: Task queues with worker consumption
- **Context Cancellation**: Graceful shutdown and task cancellation

## Development Notes

- The server runs on `:8080` by default (configurable via `-addr` flag)
- Tool workers default to 4 per tool (configurable via `-workers` flag)
- WebSocket connections handle real-time task output streaming
- Task queues have a buffer size of 100 per tool
- All task data is kept in memory (no persistence)

## Security Considerations

This application executes arbitrary CLI commands and should only be used in trusted environments. The current implementation has no authentication or input sanitization - these would need to be added for production use.