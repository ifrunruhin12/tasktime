package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/ifrunruhin12/tasktime/internal/auth"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 50
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func validateUsername(username string) error {
	if len(username) < MinUsernameLength {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > MaxUsernameLength {
		return errors.New("username must not exceed 50 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username must contain only alphanumeric characters and underscores")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < auth.MinPasswordLength {
		return auth.ErrWeakPassword
	}
	return nil
}

func sendErrorResponse(w http.ResponseWriter, code string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogWarn("Registration failed: invalid request body",
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	LogInfo("User registration attempt",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)

	if err := validateUsername(req.Username); err != nil {
		LogWarn("Registration failed: invalid username",
		"username", req.Username,
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}

	if err := validatePassword(req.Password); err != nil {
		LogWarn("Registration failed: weak password",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingUser, err := s.store.GetUserByUsername(req.Username)
	if err != nil && err != sql.ErrNoRows {
		LogError("Registration failed: database error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "DATABASE_ERROR", "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		LogWarn("Registration failed: username already exists",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "USER_EXISTS", "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	passwordHash, err := s.authManager.HashPassword(req.Password)
	if err != nil {
		LogError("Registration failed: password hashing error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create user
	user, err := s.store.CreateUser(req.Username, passwordHash)
	if err != nil {
		LogError("Registration failed: user creation error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "DATABASE_ERROR", "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := s.authManager.GenerateToken(user.Username)
	if err != nil {
		LogError("Registration failed: token generation error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to generate token", http.StatusInternalServerError)
		return
	}

	LogInfo("User registered successfully",
		"username", user.Username,
		"ip", r.RemoteAddr,
	)

	// Return success response
	response := models.AuthResponse{
		Token:     token,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleLogin handles user login
// @Summary Login user
// @Description Authenticate user and receive JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogWarn("Login failed: invalid request body",
		"error", err.Error(),
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	LogInfo("User login attempt",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)

	// Validate input
	if req.Username == "" || req.Password == "" {
		LogWarn("Login failed: missing credentials",
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "VALIDATION_ERROR", "Username and password are required", http.StatusBadRequest)
		return
	}

	// Get user from database
	user, err := s.store.GetUserByUsername(req.Username)
	if err != nil {
		LogError("Login failed: database error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "DATABASE_ERROR", "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		LogWarn("Login failed: user not found",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "INVALID_CREDENTIALS", "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := s.authManager.ComparePassword(user.PasswordHash, req.Password); err != nil {
		LogWarn("Login failed: invalid password",
		"username", req.Username,
		"ip", r.RemoteAddr,
	)
		sendErrorResponse(w, "INVALID_CREDENTIALS", "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Update last seen
	s.store.UpdateUserLastSeen(user.Username)

	// Generate JWT token
	token, err := s.authManager.GenerateToken(user.Username)
	if err != nil {
		LogError("Login failed: token generation error",
		"username", req.Username,
		"error", err.Error(),
	)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to generate token", http.StatusInternalServerError)
		return
	}

	LogInfo("User logged in successfully",
		"username", user.Username,
		"ip", r.RemoteAddr,
	)

	// Return success response
	response := models.AuthResponse{
		Token:    token,
		Username: user.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetOnlineUsers returns a list of currently online users
// @Summary Get online users
// @Description Get list of currently connected users
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "users: array of usernames, count: number"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /users/online [get]
func (s *Server) handleGetOnlineUsers(w http.ResponseWriter, r *http.Request) {
	users := s.connectionManager.GetOnlineUsers()

	response := map[string]interface{}{
		"users": users,
		"count": len(users),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetCurrentUser returns information about the currently authenticated user
// @Summary Get current user
// @Description Get information about the authenticated user
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "username, created_at, last_seen"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "User not found"
// @Router /users/me [get]
func (s *Server) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	username := getUsernameFromContext(r)
	if username == "" {
		sendErrorResponse(w, "UNAUTHORIZED", "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		sendErrorResponse(w, "DATABASE_ERROR", "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		sendErrorResponse(w, "USER_NOT_FOUND", "User not found", http.StatusNotFound)
		return
	}

	// Return user info (without password hash)
	response := map[string]interface{}{
		"username":   user.Username,
		"created_at": user.CreatedAt,
		"last_seen":  user.LastSeen,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
