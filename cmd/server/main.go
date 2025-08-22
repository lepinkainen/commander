package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lepinkainen/commander/internal/api"
	"github.com/lepinkainen/commander/internal/assets"
	"github.com/lepinkainen/commander/internal/executor"
	"github.com/lepinkainen/commander/internal/files"
	"github.com/lepinkainen/commander/internal/storage"
	"github.com/lepinkainen/commander/internal/task"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "Server address")
		workers    = flag.Int("workers", 4, "Number of workers per tool")
		configPath = flag.String("config", "./config/tools.json", "Path to tools configuration")
		dbPath     = flag.String("db", "./data/commander.db", "Path to SQLite database")
		dev        = flag.Bool("dev", false, "Development mode - serve static files from filesystem instead of embedded")
	)
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	repo, err := storage.NewSQLiteRepository(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			log.Printf("Error closing database: %v", closeErr)
		}
	}()

	// Create task manager
	manager := task.NewManager(repo)

	// Create file manager
	fileManager := files.NewManager(repo)

	// Create file discovery service
	fileDiscovery := files.NewFileDiscovery(fileManager)

	// Wire file discovery to task manager
	manager.SetFileDiscovery(fileDiscovery)

	// Create executor with configured tools
	exec, err := executor.NewExecutor(*configPath, *workers, manager)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start the executor
	if err := exec.Start(); err != nil {
		log.Fatalf("Failed to start executor: %v", err)
	}

	// Create API server
	var staticFiles *embed.FS
	if !*dev {
		staticFiles = &assets.StaticFiles
	}
	server := api.NewServer(manager, exec, fileManager, staticFiles)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         *addr,
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on http://localhost%s", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exec.Stop()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := repo.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server exited")
}
