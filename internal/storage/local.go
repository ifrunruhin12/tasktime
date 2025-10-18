package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ifrunruhin12/tasktime/internal/models"
)

type LocalStore struct {
	filePath string
}

func NewLocalStore() (*LocalStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Create .tasktime directory if it doesn't exist
	tasktimeDir := filepath.Join(homeDir, ".tasktime")
	if err := os.MkdirAll(tasktimeDir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(tasktimeDir, "personal_tasks.json")
	
	return &LocalStore{
		filePath: filePath,
	}, nil
}

func (s *LocalStore) GetTasks() ([]models.Task, error) {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return []models.Task{}, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var tasks []models.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *LocalStore) saveTasks(tasks []models.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *LocalStore) CreateTask(title, project string) (*models.Task, error) {
	tasks, err := s.GetTasks()
	if err != nil {
		return nil, err
	}

	task := &models.Task{
		ID:               generateID(),
		Title:            title,
		Project:          project,
		Status:           "todo",
		IsActive:         false,
		TotalTimeSeconds: 0,
		CreatedAt:        time.Now(),
		IsPersonal:       true,
	}

	tasks = append([]models.Task{*task}, tasks...)
	if err := s.saveTasks(tasks); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *LocalStore) UpdateTaskStatus(id, status string) (*models.Task, error) {
	tasks, err := s.GetTasks()
	if err != nil {
		return nil, err
	}

	for i, task := range tasks {
		if task.ID == id {
			tasks[i].Status = status
			if err := s.saveTasks(tasks); err != nil {
				return nil, err
			}
			return &tasks[i], nil
		}
	}

	return nil, os.ErrNotExist
}

func (s *LocalStore) DeleteTask(id string) error {
	tasks, err := s.GetTasks()
	if err != nil {
		return err
	}

	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			return s.saveTasks(tasks)
		}
	}

	return os.ErrNotExist
}

func (s *LocalStore) StartTimer(id string) (*models.Task, error) {
	tasks, err := s.GetTasks()
	if err != nil {
		return nil, err
	}

	for i, task := range tasks {
		if task.ID == id {
			now := time.Now()
			tasks[i].IsActive = true
			tasks[i].StartTime = &now
			if err := s.saveTasks(tasks); err != nil {
				return nil, err
			}
			return &tasks[i], nil
		}
	}

	return nil, os.ErrNotExist
}

func (s *LocalStore) StopTimer(id string) (*models.Task, error) {
	tasks, err := s.GetTasks()
	if err != nil {
		return nil, err
	}

	for i, task := range tasks {
		if task.ID == id && task.IsActive && task.StartTime != nil {
			duration := int(time.Since(*task.StartTime).Seconds())
			tasks[i].IsActive = false
			tasks[i].StartTime = nil
			tasks[i].TotalTimeSeconds += duration
			if err := s.saveTasks(tasks); err != nil {
				return nil, err
			}
			return &tasks[i], nil
		}
	}

	return nil, os.ErrNotExist
}

// Simple ID generator for local tasks
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}