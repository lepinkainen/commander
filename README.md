# Commander - CLI Tool Manager

A web-based interface for managing and monitoring CLI tools with parallel execution and real-time output streaming.

## Features

- **Web UI**: Clean, modern interface for managing CLI tools
- **Parallel Execution**: Run multiple tools simultaneously with configurable worker pools
- **Task Queue**: Queue tasks for each tool and process them in order
- **Real-time Output**: Stream command output via WebSocket
- **Tool Configuration**: Easy JSON-based tool configuration
- **Status Monitoring**: Track task status (queued, running, complete, failed)
- **CI/CD Ready**: GitHub Actions workflow with testing and linting

## Quick Start

### Prerequisites

- Go 1.21 or higher
- [Task](https://taskfile.dev) (recommended) or make
- CLI tools you want to manage (yt-dlp, gallery-dl, wget, etc.)

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/lepinkainen/commander.git
cd commander
```

2. **Install development tools:**
```bash
task install-tools
```

3. **Build and run:**
```bash
task build
task run
```

4. **Or run in development mode:**
```bash
task dev
```

5. **Open the web interface:**
```
http://localhost:8080
```

## Development

### Available Tasks

Run `task` to see all available tasks:

```bash
task                # List all available tasks
task build          # Build the project (runs tests and linting first)
task test           # Run tests
task lint           # Run linters (goimports, go vet, golangci-lint)
task fmt            # Format code with goimports
task dev            # Run development server
task dev-watch      # Run with auto-reload (requires air)
task clean          # Clean build artifacts
task coverage       # Generate test coverage report
task docker-build   # Build Docker image
task docker-run     # Run Docker container
```

### Project Structure

```
commander/
├── cmd/server/         # Main application entry point
├── internal/
│   ├── task/          # Task management and queuing
│   ├── executor/      # Command execution engine
│   └── api/           # HTTP/WebSocket API handlers
├── web/static/        # Frontend files (HTML/CSS/JS)
├── config/            # Tool configurations
├── build/             # Build artifacts (generated)
├── .github/           # GitHub Actions CI/CD
├── Taskfile.yml       # Task automation
├── .golangci.yml      # Linter configuration
└── README.md          # This file
```

### Configuration

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

### API Endpoints

- `POST /api/tasks` - Create a new task
- `GET /api/tasks` - List all tasks
- `GET /api/tasks/{id}` - Get specific task
- `POST /api/tasks/{id}/cancel` - Cancel a task
- `GET /api/tools` - List available tools
- `GET /api/stats` - Get queue statistics
- `WS /api/ws` - WebSocket for real-time updates

### Command Line Flags

- `-addr` : Server address (default: ":8080")
- `-workers` : Default workers per tool (default: 4)
- `-config` : Path to tools configuration (default: "./config/tools.json")

Example:
```bash
./build/commander -addr :3000 -workers 8
```

## Testing

Run tests with coverage:
```bash
task test           # Run tests
task test-ci        # Run tests with coverage for CI
task coverage       # Generate HTML coverage report
```

## Docker Support

Build and run with Docker:
```bash
task docker-build   # Build Docker image
task docker-run     # Run container
```

Or manually:
```bash
docker build -t commander:latest .
docker run -p 8080:8080 -v $(pwd)/config:/app/config commander:latest
```

## CI/CD

The project includes GitHub Actions workflow that:
- Runs tests with coverage
- Performs linting with golangci-lint
- Builds for multiple platforms (Linux, macOS, Windows)
- Uploads build artifacts
- Can optionally upload coverage to Codecov

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

## Security Notes

- The server executes system commands - only run in trusted environments
- Consider adding authentication for production use
- Validate and sanitize all user inputs
- Use proper file path restrictions for download locations

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`task build`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

MIT

## Acknowledgments

- Built with [Gorilla Mux](https://github.com/gorilla/mux) and [Gorilla WebSocket](https://github.com/gorilla/websocket)
- Task automation with [Task](https://taskfile.dev)
- Code quality with [golangci-lint](https://golangci-lint.run)

## TODO

- [ ] Authentication and user management
- [ ] Task persistence (database/file storage)
- [ ] Task scheduling (cron-like functionality)
- [ ] Download progress parsing for specific tools
- [ ] File management integration
- [ ] Task templates and presets
- [ ] Rate limiting and resource management
- [ ] Metrics and monitoring (Prometheus/Grafana)
- [ ] REST API documentation (OpenAPI/Swagger)
- [ ] WebSocket protocol documentation
