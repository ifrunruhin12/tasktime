package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/ifrunruhin12/tasktime/internal/models"
	"github.com/ifrunruhin12/tasktime/internal/storage"
)

type Server struct {
	store   *storage.PostgresStore
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func New() (*Server, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://tasktime:tasktime@localhost/tasktime?sslmode=disable"
	}

	store, err := storage.NewPostgresStore(databaseURL)
	if err != nil {
		return nil, err
	}

	return &Server{
		store:   store,
		clients: make(map[*websocket.Conn]bool),
	}, nil
}

func (s *Server) Start(port string) error {
	r := chi.NewRouter()

	// API routes
	r.Get("/api/v1/tasks", s.getTasks)
	r.Post("/api/v1/tasks", s.createTask)
	r.Put("/api/v1/tasks/{id}/status", s.updateTaskStatus)
	r.Delete("/api/v1/tasks/{id}", s.deleteTask)
	r.Post("/api/v1/tasks/{id}/time/start", s.startTimer)
	r.Post("/api/v1/tasks/{id}/time/stop", s.stopTimer)
	r.Get("/api/v1/ws", s.handleWebSocket)

	log.Printf("🚀 TaskTime server running on :%s", port)
	return http.ListenAndServe(":"+port, r)
}

func (s *Server) broadcast(message interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := json.Marshal(message)
	log.Printf("Broadcasting message to %d clients: %s", len(s.clients), string(data))

	// Create a list of clients to remove (to avoid modifying map while iterating)
	var toRemove []*websocket.Conn

	for client := range s.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send message to client: %v", err)
			toRemove = append(toRemove, client)
		}
	}

	// Remove failed clients
	for _, client := range toRemove {
		delete(s.clients, client)
		client.Close()
	}
}

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.GetTasks()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	task, err := s.store.CreateTask(req.Title, req.Project)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.broadcast(models.WSMessage{
		Type:    "task.created",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) updateTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	task, err := s.store.UpdateTaskStatus(taskID, req.Status)
	if err != nil {
		http.Error(w, "Task not found", 404)
		return
	}

	s.broadcast(models.WSMessage{
		Type:    "task.updated",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	err := s.store.DeleteTask(taskID)
	if err != nil {
		http.Error(w, "Task not found", 404)
		return
	}

	s.broadcast(models.WSMessage{
		Type:    "task.deleted",
		Payload: map[string]string{"id": taskID},
	})

	w.WriteHeader(204)
}

func (s *Server) startTimer(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	task, err := s.store.StartTimer(taskID)
	if err != nil {
		http.Error(w, "Task not found", 404)
		return
	}

	s.broadcast(models.WSMessage{
		Type:    "task.updated",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) stopTimer(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	task, err := s.store.StopTimer(taskID)
	if err != nil {
		http.Error(w, "Task not found", 404)
		return
	}

	s.broadcast(models.WSMessage{
		Type:    "task.updated",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	log.Printf("WebSocket client connected. Total clients: %d", clientCount)

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		clientCount := len(s.clients)
		s.mu.Unlock()
		log.Printf("WebSocket client disconnected. Total clients: %d", clientCount)
	}()

	// Set ping/pong handlers to keep connection alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send periodic pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
	}
}
