package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lepinkainen/commander/internal/executor"
	"github.com/lepinkainen/commander/internal/files"
	"github.com/lepinkainen/commander/internal/task"
	"github.com/lepinkainen/commander/internal/types"
	"github.com/rs/cors"
)

// Server represents the API server
type Server struct {
	manager     *task.Manager
	executor    *executor.Executor
	fileManager *files.Manager
	upgrader    websocket.Upgrader
}

// NewServer creates a new API server
func NewServer(manager *task.Manager, exec *executor.Executor, fileManager *files.Manager) *Server {
	return &Server{
		manager:     manager,
		executor:    exec,
		fileManager: fileManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// TODO: Configure this properly for production
				return true
			},
		},
	}
}

// Router creates and configures the HTTP router
func (s *Server) Router() http.Handler {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/tasks", s.createTask).Methods("POST")
	api.HandleFunc("/tasks", s.getTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", s.getTask).Methods("GET")
	api.HandleFunc("/tasks/{id}/cancel", s.cancelTask).Methods("POST")
	api.HandleFunc("/tools", s.getTools).Methods("GET")
	api.HandleFunc("/stats", s.getStats).Methods("GET")
	api.HandleFunc("/ws", s.handleWebSocket)

	// File management routes
	api.HandleFunc("/directories", s.getDirectories).Methods("GET")
	api.HandleFunc("/directories", s.createDirectory).Methods("POST")
	api.HandleFunc("/directories/{id}", s.getDirectory).Methods("GET")
	api.HandleFunc("/directories/{id}", s.updateDirectory).Methods("PUT")
	api.HandleFunc("/directories/{id}", s.deleteDirectory).Methods("DELETE")
	api.HandleFunc("/directories/{id}/scan", s.scanDirectory).Methods("POST")
	api.HandleFunc("/directories/{id}/files", s.getDirectoryFiles).Methods("GET")

	api.HandleFunc("/files", s.getFiles).Methods("GET")
	api.HandleFunc("/files/search", s.searchFiles).Methods("GET")
	api.HandleFunc("/files/{id}", s.getFile).Methods("GET")
	api.HandleFunc("/files/{id}", s.deleteFile).Methods("DELETE")
	api.HandleFunc("/files/{id}/download", s.downloadFile).Methods("GET")
	api.HandleFunc("/files/{id}/move", s.moveFile).Methods("POST")
	api.HandleFunc("/files/{id}/tags", s.updateFileTags).Methods("POST")

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(router)
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Tool    string   `json:"tool"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// createTask handles task creation
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate tool exists
	if !s.executor.IsToolAvailable(req.Tool) {
		http.Error(w, "Tool not available", http.StatusBadRequest)
		return
	}

	// Use tool's command if not specified
	if req.Command == "" {
		for _, tool := range s.executor.GetTools() {
			if tool.Name == req.Tool {
				req.Command = tool.Command
				break
			}
		}
	}

	// Create task
	newTask := task.NewTask(req.Tool, req.Command, req.Args)

	// Add to manager
	if err := s.manager.AddTask(newTask); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newTask); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getTasks returns all tasks
func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	tool := r.URL.Query().Get("tool")

	var tasks []*task.Task
	if tool != "" {
		tasks = s.manager.GetTasksByTool(tool)
	} else {
		tasks = s.manager.GetAllTasks()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getTask returns a specific task
func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	taskData, err := s.manager.GetTask(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(taskData); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// cancelTask cancels a task
func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := s.manager.UpdateTaskStatus(taskID, types.StatusCanceled); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "canceled"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getTools returns available tools
func (s *Server) getTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.executor.GetTools()); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getStats returns queue statistics
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	stats := s.manager.GetQueueStats()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleWebSocket handles WebSocket connections for real-time updates
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
	}()

	// Subscribe to task events
	events := s.manager.Subscribe()
	defer s.manager.Unsubscribe(events)

	// Send events to client
	for event := range events {
		if err := conn.WriteJSON(event); err != nil {
			log.Printf("WebSocket write failed: %v", err)
			break
		}
	}
}

// Directory management handlers

// CreateDirectoryRequest represents a directory creation request
type CreateDirectoryRequest struct {
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	ToolName   *string `json:"tool_name,omitempty"`
	DefaultDir bool    `json:"default_dir"`
}

// createDirectory handles directory creation
func (s *Server) createDirectory(w http.ResponseWriter, r *http.Request) {
	var req CreateDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dir, err := s.fileManager.CreateDirectory(r.Context(), req.Name, req.Path, req.ToolName, req.DefaultDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dir); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDirectories returns all directories
func (s *Server) getDirectories(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.fileManager.GetFileRepository().ListDirectories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dirs); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDirectory returns a specific directory
func (s *Server) getDirectory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dirID := vars["id"]

	dir, err := s.fileManager.GetFileRepository().GetDirectory(r.Context(), dirID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dir); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// updateDirectory updates a directory
func (s *Server) updateDirectory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dirID := vars["id"]

	var req CreateDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing directory first
	dir, err := s.fileManager.GetFileRepository().GetDirectory(r.Context(), dirID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Update fields
	dir.Name = req.Name
	dir.Path = req.Path
	dir.ToolName = req.ToolName
	dir.DefaultDir = req.DefaultDir

	if err := s.fileManager.GetFileRepository().UpdateDirectory(r.Context(), dir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dir); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// deleteDirectory deletes a directory
func (s *Server) deleteDirectory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dirID := vars["id"]

	if err := s.fileManager.GetFileRepository().DeleteDirectory(r.Context(), dirID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// scanDirectory scans a directory for files
func (s *Server) scanDirectory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dirID := vars["id"]

	if err := s.fileManager.ScanDirectory(r.Context(), dirID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "scanned"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDirectoryFiles returns files in a specific directory
func (s *Server) getDirectoryFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dirID := vars["id"]

	fileList, err := s.fileManager.GetFileRepository().ListFiles(r.Context(), types.FileFilters{
		DirectoryID: dirID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileList); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// File management handlers

// getFiles returns all files with optional filters
func (s *Server) getFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filters := types.FileFilters{
		DirectoryID: query.Get("directory_id"),
		MimeType:    query.Get("mime_type"),
	}

	if minSize := query.Get("min_size"); minSize != "" {
		if size, err := strconv.ParseInt(minSize, 10, 64); err == nil {
			filters.MinSize = size
		}
	}

	if maxSize := query.Get("max_size"); maxSize != "" {
		if size, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			filters.MaxSize = size
		}
	}

	fileList, err := s.fileManager.GetFileRepository().ListFiles(r.Context(), filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileList); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// searchFiles searches for files
func (s *Server) searchFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	fileList, err := s.fileManager.SearchFiles(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fileList); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getFile returns a specific file
func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	file, err := s.fileManager.GetFileRepository().GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(file); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// deleteFile deletes a file
func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	if err := s.fileManager.DeleteFile(r.Context(), fileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// downloadFile serves a file for download
func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	file, err := s.fileManager.GetFileRepository().GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Open the file
	fileHandle, err := os.Open(file.FilePath)
	if err != nil {
		http.Error(w, "File not found on filesystem", http.StatusNotFound)
		return
	}
	defer func() {
		if err := fileHandle.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Filename+"\"")
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.FileSize, 10))

	// Stream the file
	if _, err := io.Copy(w, fileHandle); err != nil {
		log.Printf("Error streaming file: %v", err)
	}
}

// MoveFileRequest represents a file move request
type MoveFileRequest struct {
	DirectoryID string `json:"directory_id"`
}

// moveFile moves a file to a different directory
func (s *Server) moveFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	var req MoveFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.fileManager.MoveFile(r.Context(), fileID, req.DirectoryID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "moved"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateFileTagsRequest represents a file tag update request
type UpdateFileTagsRequest struct {
	Tags []string `json:"tags"`
}

// updateFileTags updates tags for a file
func (s *Server) updateFileTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	var req UpdateFileTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.fileManager.TagFile(r.Context(), fileID, req.Tags); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "tagged"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
