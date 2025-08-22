package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lepinkainen/commander/internal/executor"
	"github.com/lepinkainen/commander/internal/task"
	"github.com/lepinkainen/commander/internal/types"
	"github.com/rs/cors"
)

// Server represents the API server
type Server struct {
	manager  *task.Manager
	executor *executor.Executor
	upgrader websocket.Upgrader
}

// NewServer creates a new API server
func NewServer(manager *task.Manager, exec *executor.Executor) *Server {
	return &Server{
		manager:  manager,
		executor: exec,
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
