.PHONY: build run clean test dev install

# Build the application
build:
	go build -o bin/commander cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Development mode with auto-reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Please install air first: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

# Install dependencies
install:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "Please install golangci-lint first: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/commander-darwin-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/commander-darwin-arm64 cmd/server/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/commander-linux-amd64 cmd/server/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/commander-linux-arm64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/commander-windows-amd64.exe cmd/server/main.go

# Docker build
docker-build:
	docker build -t commander:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/config:/app/config commander:latest

# Help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make dev          - Run in development mode with auto-reload"
	@echo "  make install      - Install dependencies"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make build-all    - Build for multiple platforms"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
