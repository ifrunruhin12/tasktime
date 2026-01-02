package server

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	config            *Config
	store             *storage.PostgresStore
	authManager       *auth.Manager
	connectionManager *ConnectionManager
	clients           map[*websocket.Conn]bool // Deprecated: use connectionManager instead
	onlineUsers       map[string]bool          // Deprecated: use connectionManager instead
	mu                sync.RWMutex
	startTime         time.Time
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func New() (*Server, error) {
	// Load configuration from environment variables
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize database store
	store, err := storage.NewPostgresStore(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database store: %w", err)
	}

	// Initialize auth manager with config values
	authManager, err := auth.NewManager(config.JWTSecret, config.JWTExpiryDays)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Initialize logger with config values
	err = InitLogger(config.LogLevel, config.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	LogInfo("Server initializing", 
		"log_level", config.LogLevel, 
		"log_file", config.LogFile,
		"port", config.Port,
		"host", config.Host,
		"jwt_expiry_days", config.JWTExpiryDays,
		"ws_ping_interval", config.WSPingInterval,
		"ws_pong_timeout", config.WSPongTimeout,
	)

	return &Server{
		config:            config,
		store:             store,
		authManager:       authManager,
		connectionManager: NewConnectionManager(),
		clients:           make(map[*websocket.Conn]bool),
		onlineUsers:       make(map[string]bool),
		startTime:         time.Now(),
	}, nil
}

func (s *Server) Start() error {
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

	address := s.config.Host + ":" + s.config.Port
	LogInfo("TaskTime server starting", "address", address)
	return http.ListenAndServe(address, r)
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

	LogDebug("Broadcasting message",
		"type", wsMsg.Type,
		"online_users", s.connectionManager.GetOnlineUserCount(),
	)

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
		LogWarn("Task creation failed: invalid request",
		"error", err.Error(),
	)
		http.Error(w, err.Error(), 400)
		return
	}

	username := getUsernameFromContext(r)

	task, err := s.store.CreateTask(req.Title, req.Project, username)
	if err != nil {
		LogError("Task creation failed: database error",
		"username", username,
		"title", req.Title,
		"project", req.Project,
		"error", err.Error(),
	)
		http.Error(w, err.Error(), 500)
		return
	}

	LogInfo("Task created",
		"task_id", task.ID,
		"username", username,
		"title", task.Title,
		"project", task.Project,
	)

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
		LogWarn("Task update failed: invalid request",
		"task_id", taskID,
		"error", err.Error(),
	)
		http.Error(w, err.Error(), 400)
		return
	}

	task, err := s.store.UpdateTaskStatus(taskID, req.Status)
	if err != nil {
		LogWarn("Task update failed: task not found",
		"task_id", taskID,
		"username", username,
		"error", err.Error(),
	)
		http.Error(w, "Task not found", 404)
		return
	}

	LogInfo("Task status updated",
		"task_id", task.ID,
		"username", username,
		"status", task.Status,
	)

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
		LogWarn("Task deletion failed: task not found",
		"task_id", taskID,
		"username", username,
		"error", err.Error(),
	)
		http.Error(w, "Task not found", 404)
		return
	}

	LogInfo("Task deleted",
		"task_id", taskID,
		"username", username,
	)

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
		LogWarn("Timer start failed: task not found",
		"task_id", taskID,
		"username", username,
		"error", err.Error(),
	)
		http.Error(w, "Task not found", 404)
		return
	}

	LogInfo("Timer started",
		"task_id", task.ID,
		"username", username,
		"title", task.Title,
	)

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
		LogWarn("Timer stop failed: task not found",
		"task_id", taskID,
		"username", username,
		"error", err.Error(),
	)
		http.Error(w, "Task not found", 404)
		return
	}

	LogInfo("Timer stopped",
		"task_id", task.ID,
		"username", username,
		"title", task.Title,
		"total_time_seconds", task.TotalTimeSeconds,
	)

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
		LogError("WebSocket upgrade failed",
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
		return
	}

	LogDebug("WebSocket connection established, waiting for authentication",
		"ip", r.RemoteAddr,
	)

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var authMsg models.WSMessage
	err = conn.ReadJSON(&authMsg)
	if err != nil {
		LogWarn("WebSocket auth failed: failed to read auth message",
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
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
		LogWarn("WebSocket auth failed: expected auth message",
		"received_type", authMsg.Type,
		"ip", r.RemoteAddr,
	)
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
		LogWarn("WebSocket auth failed: invalid payload format",
		"ip", r.RemoteAddr,
	)
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
		LogWarn("WebSocket auth failed: missing or invalid token",
		"ip", r.RemoteAddr,
	)
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
		LogWarn("WebSocket auth failed: token validation failed",
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
		conn.WriteJSON(models.WSMessage{
			Type: "auth.failed",
			Payload: map[string]string{
				"error": "Invalid or expired token",
			},
		})
		conn.Close()
		return
	}

	LogInfo("WebSocket connection authenticated",
		"username", username,
		"ip", r.RemoteAddr,
	)

	conn.SetReadDeadline(time.Time{})

	connectionID := s.connectionManager.AddUser(username, conn)

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

	// Handle connection cleanup on disconnect
	defer func() {
		s.connectionManager.RemoveUser(username, connectionID)
		s.connectionManager.BroadcastToAll(models.WSMessage{
			Type: "user.left",
			Payload: map[string]interface{}{
				"username":  username,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
		LogInfo("User disconnected",
		"username", username,
		"online_users", s.connectionManager.GetOnlineUserCount(),
	)
	}()

	LogInfo("User joined",
		"username", username,
		"online_users", len(onlineUsers),
	)

	s.store.UpdateUserLastSeen(username)

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(s.config.WSPongTimeout))
		s.connectionManager.UpdateLastPing(username)
		return nil
	})

	pingTicker := time.NewTicker(s.config.WSPingInterval)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				LogDebug("Failed to send ping",
		"username", username,
		"error", err.Error(),
	)
				return
			}
		}
	}()

	for {
		var msg models.WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				LogWarn("WebSocket unexpected close",
		"username", username,
		"error", err.Error(),
	)
			}
			break
		}

		LogDebug("Received WebSocket message",
		"username", username,
		"type", msg.Type,
	)
	}
}

func (s *Server) assignTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	assignedBy := getUsernameFromContext(r)

	var req models.AssignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogWarn("Task assignment failed: invalid request",
		"task_id", taskID,
		"error", err.Error(),
	)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AssignedTo != nil && *req.AssignedTo != "" {
		user, err := s.store.GetUserByUsername(*req.AssignedTo)
		if err != nil {
			LogError("Task assignment failed: database error",
		"task_id", taskID,
		"error", err.Error(),
	)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			LogWarn("Task assignment failed: user not found",
		"task_id", taskID,
		"assigned_to", *req.AssignedTo,
		"assigned_by", assignedBy,
	)
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}

		if !s.connectionManager.IsUserOnline(*req.AssignedTo) {
			LogWarn("Task assignment failed: user not online",
		"task_id", taskID,
		"assigned_to", *req.AssignedTo,
		"assigned_by", assignedBy,
	)
			http.Error(w, "User is not online", http.StatusBadRequest)
			return
		}
	}

	task, err := s.store.AssignTask(taskID, req.AssignedTo)
	if err != nil {
		LogWarn("Task assignment failed: task not found",
		"task_id", taskID,
		"assigned_by", assignedBy,
		"error", err.Error(),
	)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	assignedToStr := "unassigned"
	if task.AssignedTo != nil {
		assignedToStr = *task.AssignedTo
	}

	LogInfo("Task assignment changed",
		"task_id", task.ID,
		"assigned_to", assignedToStr,
		"assigned_by", assignedBy,
	)

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
		LogError("Health check: database ping failed", "error", err.Error())

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
		LogError("Failed to get total users count", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	totalTasks, err := s.store.GetTotalTasksCount()
	if err != nil {
		LogError("Failed to get total tasks count", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	activeTimers, err := s.store.GetActiveTimersCount()
	if err != nil {
		LogError("Failed to get active timers count", "error", err.Error())
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
