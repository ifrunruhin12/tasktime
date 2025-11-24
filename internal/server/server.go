package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/ifrunruhin12/tasktime/internal/auth"
	"github.com/ifrunruhin12/tasktime/internal/models"
	"github.com/ifrunruhin12/tasktime/internal/storage"
)

type Server struct {
	store             *storage.PostgresStore
	authManager       *auth.Manager
	connectionManager *ConnectionManager
	clients           map[*websocket.Conn]bool // Deprecated: use connectionManager instead
	onlineUsers       map[string]bool          // Deprecated: use connectionManager instead
	mu                sync.RWMutex
	startTime         time.Time
	logger            *Logger
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

	authManager, err := auth.NewManager()
	if err != nil {
		return nil, err
	}

	logLevel := ParseLogLevel(os.Getenv("LOG_LEVEL"))
	logFile := os.Getenv("LOG_FILE")
	logger, err := NewLogger(logLevel, logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return &Server{
		store:             store,
		authManager:       authManager,
		connectionManager: NewConnectionManager(),
		clients:           make(map[*websocket.Conn]bool),
		onlineUsers:       make(map[string]bool),
		startTime:         time.Now(),
		logger:            logger,
	}, nil
}

func (s *Server) Start(port string) error {
	r := chi.NewRouter()

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Get("/api/v1/health", s.handleHealth)

	r.Post("/api/v1/auth/register", s.handleRegister)
	r.Post("/api/v1/auth/login", s.handleLogin)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/api/v1/users/online", s.handleGetOnlineUsers)
		r.Get("/api/v1/users/me", s.handleGetCurrentUser)

		r.Get("/api/v1/stats", s.handleStats)

		r.Get("/api/v1/tasks", s.getTasks)
		r.Post("/api/v1/tasks", s.createTask)
		r.Put("/api/v1/tasks/{id}/status", s.updateTaskStatus)
		r.Put("/api/v1/tasks/{id}/assign", s.assignTask)
		r.Get("/api/v1/tasks/assigned/{username}", s.getAssignedTasks)
		r.Delete("/api/v1/tasks/{id}", s.deleteTask)

		r.Post("/api/v1/tasks/{id}/time/start", s.startTimer)
		r.Post("/api/v1/tasks/{id}/time/stop", s.stopTimer)
	})

	r.Get("/api/v1/ws", s.handleWebSocket)

	log.Printf("TaskTime server running on :%s", port)
	return http.ListenAndServe(":"+port, r)
}

func (s *Server) broadcast(message interface{}) {
	var wsMsg models.WSMessage

	switch msg := message.(type) {
	case models.WSMessage:
		wsMsg = msg
	default:
		wsMsg = models.WSMessage{
			Type:    "unknown",
			Payload: message,
		}
	}

	s.logger.Debug("Broadcasting message", map[string]interface{}{
		"type":         wsMsg.Type,
		"online_users": s.connectionManager.GetOnlineUserCount(),
	})

	s.connectionManager.BroadcastToAll(wsMsg)
}

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")

	var tasks []models.Task
	var err error

	if filter == "my" {
		username := getUsernameFromContext(r)
		if username == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tasks, err = s.store.GetTasksByAssignedUser(username)
	} else {
		tasks, err = s.store.GetTasks()
	}

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
		s.logger.Warn("Task creation failed: invalid request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, err.Error(), 400)
		return
	}

	username := getUsernameFromContext(r)

	task, err := s.store.CreateTask(req.Title, req.Project, username)
	if err != nil {
		s.logger.Error("Task creation failed: database error", map[string]interface{}{
			"username": username,
			"title":    req.Title,
			"project":  req.Project,
			"error":    err.Error(),
		})
		http.Error(w, err.Error(), 500)
		return
	}

	s.logger.Info("Task created", map[string]interface{}{
		"task_id":  task.ID,
		"username": username,
		"title":    task.Title,
		"project":  task.Project,
	})

	s.broadcast(models.WSMessage{
		Type:    "task.created",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) updateTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	username := getUsernameFromContext(r)

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Task update failed: invalid request", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		http.Error(w, err.Error(), 400)
		return
	}

	task, err := s.store.UpdateTaskStatus(taskID, req.Status)
	if err != nil {
		s.logger.Warn("Task update failed: task not found", map[string]interface{}{
			"task_id":  taskID,
			"username": username,
			"error":    err.Error(),
		})
		http.Error(w, "Task not found", 404)
		return
	}

	s.logger.Info("Task status updated", map[string]interface{}{
		"task_id":  task.ID,
		"username": username,
		"status":   task.Status,
	})

	s.broadcast(models.WSMessage{
		Type:    "task.updated",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	username := getUsernameFromContext(r)

	err := s.store.DeleteTask(taskID)
	if err != nil {
		s.logger.Warn("Task deletion failed: task not found", map[string]interface{}{
			"task_id":  taskID,
			"username": username,
			"error":    err.Error(),
		})
		http.Error(w, "Task not found", 404)
		return
	}

	s.logger.Info("Task deleted", map[string]interface{}{
		"task_id":  taskID,
		"username": username,
	})

	s.broadcast(models.WSMessage{
		Type:    "task.deleted",
		Payload: map[string]string{"id": taskID},
	})

	w.WriteHeader(204)
}

func (s *Server) startTimer(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	username := getUsernameFromContext(r)

	task, err := s.store.StartTimer(taskID)
	if err != nil {
		s.logger.Warn("Timer start failed: task not found", map[string]interface{}{
			"task_id":  taskID,
			"username": username,
			"error":    err.Error(),
		})
		http.Error(w, "Task not found", 404)
		return
	}

	s.logger.Info("Timer started", map[string]interface{}{
		"task_id":  task.ID,
		"username": username,
		"title":    task.Title,
	})

	s.broadcast(models.WSMessage{
		Type:    "task.updated",
		Payload: task,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) stopTimer(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	username := getUsernameFromContext(r)

	task, err := s.store.StopTimer(taskID)
	if err != nil {
		s.logger.Warn("Timer stop failed: task not found", map[string]interface{}{
			"task_id":  taskID,
			"username": username,
			"error":    err.Error(),
		})
		http.Error(w, "Task not found", 404)
		return
	}

	s.logger.Info("Timer stopped", map[string]interface{}{
		"task_id":            task.ID,
		"username":           username,
		"title":              task.Title,
		"total_time_seconds": task.TotalTimeSeconds,
	})

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
		s.logger.Error("WebSocket upgrade failed", map[string]interface{}{
			"error": err.Error(),
			"ip":    r.RemoteAddr,
		})
		return
	}

	s.logger.Debug("WebSocket connection established, waiting for authentication", map[string]interface{}{
		"ip": r.RemoteAddr,
	})

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var authMsg models.WSMessage
	err = conn.ReadJSON(&authMsg)
	if err != nil {
		s.logger.Warn("WebSocket auth failed: failed to read auth message", map[string]interface{}{
			"error": err.Error(),
			"ip":    r.RemoteAddr,
		})
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Authentication timeout or invalid message",
			},
		})
		conn.Close()
		return
	}

	if authMsg.Type != "auth" {
		s.logger.Warn("WebSocket auth failed: expected auth message", map[string]interface{}{
			"received_type": authMsg.Type,
			"ip":            r.RemoteAddr,
		})
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Expected authentication message",
			},
		})
		conn.Close()
		return
	}

	payload, ok := authMsg.Payload.(map[string]interface{})
	if !ok {
		s.logger.Warn("WebSocket auth failed: invalid payload format", map[string]interface{}{
			"ip": r.RemoteAddr,
		})
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Invalid authentication payload",
			},
		})
		conn.Close()
		return
	}

	token, ok := payload["token"].(string)
	if !ok || token == "" {
		s.logger.Warn("WebSocket auth failed: missing or invalid token", map[string]interface{}{
			"ip": r.RemoteAddr,
		})
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Missing or invalid token",
			},
		})
		conn.Close()
		return
	}

	// Validate JWT token
	username, err := s.authManager.ValidateToken(token)
	if err != nil {
		s.logger.Warn("WebSocket auth failed: token validation failed", map[string]interface{}{
			"error": err.Error(),
			"ip":    r.RemoteAddr,
		})
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Invalid or expired token",
			},
		})
		conn.Close()
		return
	}

	s.logger.Info("WebSocket connection authenticated", map[string]interface{}{
		"username": username,
		"ip":       r.RemoteAddr,
	})

	conn.SetReadDeadline(time.Time{})

	s.connectionManager.AddUser(username, conn)

	onlineUsers := s.connectionManager.GetOnlineUsers()
	conn.WriteJSON(models.WSMessage{
		Type: "auth.success",
		Payload: map[string]interface{}{
			"username":     username,
			"online_users": onlineUsers,
		},
	})

	s.connectionManager.BroadcastToAll(models.WSMessage{
		Type: "user.joined",
		Payload: map[string]interface{}{
			"username":  username,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	s.logger.Info("User joined", map[string]interface{}{
		"username":     username,
		"online_users": len(onlineUsers),
	})

	s.store.UpdateUserLastSeen(username)

	defer func() {
		s.connectionManager.RemoveUser(username)

		s.connectionManager.BroadcastToAll(models.WSMessage{
			Type: "user.left",
			Payload: map[string]interface{}{
				"username":  username,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})

		s.logger.Info("User disconnected", map[string]interface{}{
			"username":     username,
			"online_users": s.connectionManager.GetOnlineUserCount(),
		})
	}()

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		s.connectionManager.UpdateLastPing(username)
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Debug("Failed to send ping", map[string]interface{}{
					"username": username,
					"error":    err.Error(),
				})
				return
			}
		}
	}()

	for {
		var msg models.WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Warn("WebSocket unexpected close", map[string]interface{}{
					"username": username,
					"error":    err.Error(),
				})
			}
			break
		}

		s.logger.Debug("Received WebSocket message", map[string]interface{}{
			"username": username,
			"type":     msg.Type,
		})
	}
}

func (s *Server) assignTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	assignedBy := getUsernameFromContext(r)

	var req models.AssignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Task assignment failed: invalid request", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AssignedTo != nil && *req.AssignedTo != "" {
		user, err := s.store.GetUserByUsername(*req.AssignedTo)
		if err != nil {
			s.logger.Error("Task assignment failed: database error", map[string]interface{}{
				"task_id": taskID,
				"error":   err.Error(),
			})
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			s.logger.Warn("Task assignment failed: user not found", map[string]interface{}{
				"task_id":     taskID,
				"assigned_to": *req.AssignedTo,
				"assigned_by": assignedBy,
			})
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}

		if !s.connectionManager.IsUserOnline(*req.AssignedTo) {
			s.logger.Warn("Task assignment failed: user not online", map[string]interface{}{
				"task_id":     taskID,
				"assigned_to": *req.AssignedTo,
				"assigned_by": assignedBy,
			})
			http.Error(w, "User is not online", http.StatusBadRequest)
			return
		}
	}

	task, err := s.store.AssignTask(taskID, req.AssignedTo)
	if err != nil {
		s.logger.Warn("Task assignment failed: task not found", map[string]interface{}{
			"task_id":     taskID,
			"assigned_by": assignedBy,
			"error":       err.Error(),
		})
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	assignedToStr := "unassigned"
	if task.AssignedTo != nil {
		assignedToStr = *task.AssignedTo
	}

	s.logger.Info("Task assignment changed", map[string]interface{}{
		"task_id":     task.ID,
		"assigned_to": assignedToStr,
		"assigned_by": assignedBy,
	})

	s.broadcast(models.WSMessage{
		Type: "task.assigned",
		Payload: map[string]interface{}{
			"task_id":     task.ID,
			"assigned_to": task.AssignedTo,
			"assigned_by": assignedBy,
			"task":        task, // Include full task for client updates
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) getAssignedTasks(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	tasks, err := s.store.GetTasksByAssignedUser(username)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	response := map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	err := s.store.Ping()
	if err != nil {
		dbStatus = "disconnected"
		log.Printf("Health check: database ping failed: %v", err)

		response := map[string]interface{}{
			"status":    "unhealthy",
			"database":  dbStatus,
			"error":     "Database connection failed",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	uptime := time.Since(s.startTime).Seconds()

	response := map[string]interface{}{
		"status":         "healthy",
		"database":       dbStatus,
		"uptime_seconds": int64(uptime),
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	activeUsers := s.connectionManager.GetOnlineUserCount()

	totalUsers, err := s.store.GetTotalUsersCount()
	if err != nil {
		log.Printf("Failed to get total users count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	totalTasks, err := s.store.GetTotalTasksCount()
	if err != nil {
		log.Printf("Failed to get total tasks count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	activeTimers, err := s.store.GetActiveTimersCount()
	if err != nil {
		log.Printf("Failed to get active timers count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	uptime := time.Since(s.startTime).Seconds()

	response := map[string]interface{}{
		"active_users":   activeUsers,
		"total_users":    totalUsers,
		"total_tasks":    totalTasks,
		"active_timers":  activeTimers,
		"uptime_seconds": int64(uptime),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
