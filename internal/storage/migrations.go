package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version     int
	Description string
	SQL         string
}

type MigrationManager struct {
	db *sql.DB
}

func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db}
}

func (m *MigrationManager) ensureMigrationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP DEFAULT NOW()
	);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *MigrationManager) GetCurrentVersion() (int, error) {
	if err := m.ensureMigrationTable(); err != nil {
		return 0, err
	}

	var version int
	err := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func (m *MigrationManager) loadMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var version int
		var description string
		_, err := fmt.Sscanf(entry.Name(), "%d_%s", &version, &description)
		if err != nil {
			continue
		}

		description = strings.TrimSuffix(description, ".sql")
		description = strings.ReplaceAll(description, "_", " ")

		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: description,
			SQL:         string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *MigrationManager) ApplyMigrations() error {
	if err := m.ensureMigrationTable(); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	currentVersion, err := m.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		fmt.Printf("Applying migration %d: %s\n", migration.Version, migration.Description)

		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		_, err = tx.Exec(
			"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
			migration.Version,
			migration.Description,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Successfully applied migration %d\n", migration.Version)
	}

	if len(migrations) == 0 || currentVersion >= migrations[len(migrations)-1].Version {
		fmt.Println("Database is up to date")
	}

	return nil
}
