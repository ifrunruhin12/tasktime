package models

import "time"

type Task struct {
	ID               string     `json:"id" db:"id"`
	Title            string     `json:"title" db:"title"`
	Project          string     `json:"project" db:"project"`
	Status           string     `json:"status" db:"status"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	StartTime        *time.Time `json:"start_time,omitempty" db:"start_time"`
	TotalTimeSeconds int        `json:"total_time_seconds" db:"total_time_seconds"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	IsPersonal       bool       `json:"is_personal" db:"is_personal"` // New field to distinguish personal vs team tasks

	CreatedBy  string  `json:"created_by" db:"created_by"`
	AssignedTo *string `json:"assigned_to,omitempty" db:"assigned_to"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type CreateTaskRequest struct {
	Title   string `json:"title"`
	Project string `json:"project"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

type AssignTaskRequest struct {
	AssignedTo *string `json:"assigned_to"` // Pointer to allow null for unassignment
}
