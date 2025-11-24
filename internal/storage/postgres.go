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

	migrationManager := NewMigrationManager(db)
	if err := migrationManager.ApplyMigrations(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) GetTasks() ([]models.Task, error) {
	query := `
	SELECT id, title, project, status, is_active, 
	       start_time, 
	       COALESCE(total_time_seconds, 0) as total_time_seconds,
	       created_at, updated_at, created_by, assigned_to
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
			&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
		)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *PostgresStore) CreateTask(title, project, username string) (*models.Task, error) {
	query := `
	INSERT INTO tasks (title, project, created_by) 
	VALUES ($1, $2, $3) 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at, updated_at, created_by, assigned_to
	`

	var task models.Task
	err := s.db.QueryRow(query, title, project, username).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
	)

	return &task, err
}

func (s *PostgresStore) UpdateTaskStatus(id, status string) (*models.Task, error) {
	query := `
	UPDATE tasks 
	SET status = $1, updated_at = NOW() 
	WHERE id = $2 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at, updated_at, created_by, assigned_to
	`

	var task models.Task
	err := s.db.QueryRow(query, status, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
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
	SET is_active = true, start_time = NOW(), updated_at = NOW() 
	WHERE id = $1 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at, updated_at, created_by, assigned_to
	`

	var task models.Task
	err := s.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
	)

	return &task, err
}

func (s *PostgresStore) StopTimer(id string) (*models.Task, error) {
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

	query := `
	UPDATE tasks 
	SET is_active = false, 
	    start_time = NULL,
	    total_time_seconds = total_time_seconds + COALESCE(EXTRACT(EPOCH FROM (NOW() - start_time))::INTEGER, 0),
	    updated_at = NOW()
	WHERE id = $1 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at, updated_at, created_by, assigned_to
	`

	var task models.Task
	err = s.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
	)

	return &task, err
}

func (s *PostgresStore) AssignTask(taskID string, username *string) (*models.Task, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
	UPDATE tasks 
	SET assigned_to = $1, updated_at = NOW() 
	WHERE id = $2 
	RETURNING id, title, project, status, is_active, start_time, total_time_seconds, created_at, updated_at, created_by, assigned_to
	`

	var task models.Task
	err = tx.QueryRow(query, username, taskID).Scan(
		&task.ID, &task.Title, &task.Project, &task.Status,
		&task.IsActive, &task.StartTime, &task.TotalTimeSeconds, &task.CreatedAt,
		&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *PostgresStore) GetTasksByAssignedUser(username string) ([]models.Task, error) {
	query := `
	SELECT id, title, project, status, is_active, 
	       start_time, 
	       COALESCE(total_time_seconds, 0) as total_time_seconds,
	       created_at, updated_at, created_by, assigned_to
	FROM tasks 
	WHERE assigned_to = $1
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, username)
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
			&task.UpdatedAt, &task.CreatedBy, &task.AssignedTo,
		)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *PostgresStore) CreateUser(username, passwordHash string) (*models.User, error) {
	query := `
	INSERT INTO users (username, password_hash) 
	VALUES ($1, $2) 
	RETURNING username, password_hash, created_at, last_seen
	`

	var user models.User
	err := s.db.QueryRow(query, username, passwordHash).Scan(
		&user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastSeen,
	)

	return &user, err
}

// GetUserByUsername retrieves a user by username
func (s *PostgresStore) GetUserByUsername(username string) (*models.User, error) {
	query := `
	SELECT username, password_hash, created_at, last_seen
	FROM users 
	WHERE username = $1
	`

	var user models.User
	err := s.db.QueryRow(query, username).Scan(
		&user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &user, err
}

func (s *PostgresStore) UpdateUserLastSeen(username string) error {
	query := `UPDATE users SET last_seen = NOW() WHERE username = $1`
	_, err := s.db.Exec(query, username)
	return err
}

func (s *PostgresStore) GetTotalUsersCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (s *PostgresStore) GetTotalTasksCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	return count, err
}

func (s *PostgresStore) GetActiveTimersCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE is_active = true").Scan(&count)
	return count, err
}

func (s *PostgresStore) Ping() error {
	return s.db.Ping()
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
