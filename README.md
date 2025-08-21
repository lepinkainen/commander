# Commander - CLI Tool Manager

A web-based interface for managing and monitoring CLI tools with parallel execution and real-time output streaming.

## Features

- **Web UI**: Clean, modern interface for managing CLI tools
- **Parallel Execution**: Run multiple tools simultaneously with configurable worker pools
- **Task Queue**: Queue tasks for each tool and process them in order
- **Real-time Output**: Stream command output via WebSocket
- **Tool Configuration**: Easy JSON-based tool configuration
- **Status Monitoring**: Track task status (queued, running, complete, failed)

## Quick Start

1. **Install dependencies:**

```bash
cd /Users/shrike/projects/commander
go mod download
```

2. **Run the server:**

```bash
go run cmd/server/main.go
```

3. **Open the web interface:**

```plain
http://localhost:8080
```

## Configuration

Tools are configured in `config/tools.json`. Each tool can have:

- `name`: Tool identifier
- `command`: The actual command to execute
- `description`: Human-readable description
- `workers`: Number of parallel workers (optional, defaults to 4)
- `default_args`: Arguments always passed to the command

Example:

```json
{
  "name": "yt-dlp",
  "command": "yt-dlp",
  "description": "Download videos from YouTube",
  "workers": 2,
  "default_args": ["--no-warnings", "--progress"]
}
```

## API Endpoints

- `POST /api/tasks` - Create a new task
- `GET /api/tasks` - List all tasks
- `GET /api/tasks/{id}` - Get specific task
- `POST /api/tasks/{id}/cancel` - Cancel a task
- `GET /api/tools` - List available tools
- `GET /api/stats` - Get queue statistics
- `WS /api/ws` - WebSocket for real-time updates

## Architecture

```plain
commander/
├── cmd/server/         # Main application entry point
├── internal/
│   ├── task/          # Task management
│   ├── executor/      # Command execution
│   └── api/           # HTTP/WebSocket API
├── web/static/        # Frontend files
└── config/            # Tool configurations
```

## Adding New Tools

1. Edit `config/tools.json`
2. Add your tool configuration
3. Restart the server

Example for adding `aria2c`:

```json
{
  "name": "aria2c",
  "command": "aria2c",
  "description": "Multi-protocol download utility",
  "workers": 3,
  "default_args": ["--console-log-level=warn"]
}
```

## Command Line Flags

- `-addr` : Server address (default: ":8080")
- `-workers` : Default workers per tool (default: 4)
- `-config` : Path to tools configuration (default: "./config/tools.json")

Example:

```bash
go run cmd/server/main.go -addr :3000 -workers 8
```

## Development

### Building for Production

```bash
go build -o commander cmd/server/main.go
./commander
```

### Running Tests

```bash
go test ./...
```

## Requirements

- Go 1.21 or higher
- CLI tools you want to manage (yt-dlp, gallery-dl, wget, etc.)

## Security Notes

- The server executes system commands - only run in trusted environments
- Consider adding authentication for production use
- Validate and sanitize all user inputs
- Use proper file path restrictions for download locations

## License

MIT

## TODO

- [ ] Task persistence (database/file storage)
- [ ] Task scheduling (cron-like functionality)
- [ ] Download progress parsing for specific tools
- [ ] File management integration
- [ ] Docker container support
- [ ] Task templates and presets
- [ ] Rate limiting and resource management
