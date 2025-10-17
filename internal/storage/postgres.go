package storage

import (
	"database/sql"

	"github.com/ifrunruhin12/tasktime/internal/models"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	store := &PostgresStore{db: db}
	if err := store.createTables(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title TEXT NOT NULL,
		project TEXT DEFAULT '',
		status TEXT DEFAULT 'todo',
		is_active BOOLEAN DEFAULT false,
		start_time TIMESTAMP,
		total_time_seconds INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS time_entries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		duration_seconds INTEGER,
		created_at TIMESTAMP DEFAULT NOW()
	);
	
	-- Add the new column if it doesn't exist (for existing databases)
	ALTER TABLE tasks ADD COLUMN IF NOT EXISTS total_time_seconds INTEGER DEFAULT 0;
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) GetTasks() ([]models.Task, error) {
	query := `
	SELECT id, title, project, status, is_active, 
	       start_time, 
	       COALESCE(total_time_seconds, 0) as total_time_seconds,
	       created_at 
	FROM tasks 
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID, &task.Title, &task.Project, &task.Status,
			&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *PostgresStore) CreateTask(title, project string) (*models.Task, error) {
	query := `
	INSERT INTO tasks (title, project) 
	VALUES ($1, $2) 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at
	`

	var task models.Task
	err := s.db.QueryRow(query, title, project).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
	)

	return &task, err
}

func (s *PostgresStore) UpdateTaskStatus(id, status string) (*models.Task, error) {
	query := `
	UPDATE tasks 
	SET status = $1 
	WHERE id = $2 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at
	`

	var task models.Task
	err := s.db.QueryRow(query, status, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
	)

	return &task, err
}

func (s *PostgresStore) DeleteTask(id string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = $1", id)
	return err
}

func (s *PostgresStore) StartTimer(id string) (*models.Task, error) {
	query := `
	UPDATE tasks 
	SET is_active = true, start_time = NOW() 
	WHERE id = $1 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at
	`

	var task models.Task
	err := s.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
	)

	return &task, err
}

func (s *PostgresStore) StopTimer(id string) (*models.Task, error) {
	// First, record the time entry and update total time
	_, err := s.db.Exec(`
		INSERT INTO time_entries (task_id, start_time, end_time, duration_seconds)
		SELECT id, start_time, NOW(), 
		       EXTRACT(EPOCH FROM (NOW() - start_time))::INTEGER
		FROM tasks 
		WHERE id = $1 AND is_active = true
	`, id)

	if err != nil {
		return nil, err
	}

	// Then update the task with accumulated time
	query := `
	UPDATE tasks 
	SET is_active = false, 
	    start_time = NULL,
	    total_time_seconds = total_time_seconds + COALESCE(EXTRACT(EPOCH FROM (NOW() - start_time))::INTEGER, 0)
	WHERE id = $1 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at
	`

	var task models.Task
	err = s.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
	)

	return &task, err
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
