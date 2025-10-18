package models

import "time"

// Task represents a task in the system
type Task struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Project          string     `json:"project"`
	Status           string     `json:"status"`
	IsActive         bool       `json:"is_active"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	TotalTimeSeconds int        `json:"total_time_seconds"`
	CreatedAt        time.Time  `json:"created_at"`
	IsPersonal       bool       `json:"is_personal"` // New field to distinguish personal vs team tasks
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Title   string `json:"title"`
	Project string `json:"project"`
}

// UpdateStatusRequest represents a request to update task status
type UpdateStatusRequest struct {
	Status string `json:"status"`
}