# Commander - Gemini Development Context

This document provides a comprehensive overview of the Commander project for the Gemini AI assistant. It outlines the project's purpose, architecture, and development conventions to ensure effective and consistent collaboration.

## Project Overview

Commander is a web-based interface for managing and monitoring command-line interface (CLI) tools. It allows users to run multiple tools in parallel, queue tasks, and view real-time output through a modern web UI.

**Core Technologies:**

* **Backend:** Go (Golang)
* **Frontend:** HTML, CSS, JavaScript (no framework)
* **API:** RESTful API with WebSocket for real-time updates
* **Build/Task Runner:** Taskfile (Task)
* **Containerization:** Docker

**Architecture:**

The project is structured into three main components:

1. **`cmd/server`**: The main application entry point. It initializes the task manager, executor, and API server.
2. **`internal`**: Contains the core business logic:
    * **`api`**: Handles HTTP and WebSocket connections, routing, and API endpoints.
    * **`executor`**: Manages the execution of CLI tools, including worker pools and command execution.
    * **`task`**: Provides task management, including queuing, status tracking, and event broadcasting.
3. **`web/static`**: Contains the frontend files (HTML, CSS, and JavaScript) for the web interface.

## Building and Running

The project uses `Taskfile.yml` for task automation. The following commands are essential for development:

* **Install development tools:**

    ```bash
    task install-tools
    ```

* **Run tests and linting:**

    ```bash
    task build
    ```

* **Run the application in development mode:**

    ```bash
    task dev
    ```

* **Run the application with auto-reload (requires `air`):**

    ```bash
    task dev-watch
    ```

* **Build and run the application for production:**

    ```bash
    task build
    task run
    ```

* **Build and run with Docker:**

    ```bash
    task docker-build
    task docker-run
    ```

The application will be available at `http://localhost:8080`.

## Development Conventions

* **Code Style:** Code is formatted with `goimports`. The `task fmt` command can be used to format the code.
* **Linting:** The project uses `golangci-lint` for linting. The linting rules are defined in `.golangci.yml`. The `task lint` command can be used to run the linter.
* **Testing:** Tests are written using the standard Go testing library. The `task test` command can be used to run the tests. Test files are located alongside the source files with a `_test.go` suffix.
* **CI/CD:** The project uses GitHub Actions for CI/CD. The workflow is defined in `.github/workflows/go-ci.yml`. The workflow runs tests, linting, and builds the application for multiple platforms.
* **Configuration:** Tool configurations are defined in `config/tools.json`. Each tool can have a name, command, description, number of workers, and default arguments.
* **Dependencies:** Go modules are used for dependency management. The `go.mod` file lists the project's dependencies.
