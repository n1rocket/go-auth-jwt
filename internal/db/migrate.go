package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationConfig holds configuration for database migrations
type MigrationConfig struct {
	DatabaseName string
	SchemaName   string
}

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	config MigrationConfig
}

// NewMigrator creates a new Migrator instance
func NewMigrator(db *sql.DB, config MigrationConfig) *Migrator {
	if config.DatabaseName == "" {
		config.DatabaseName = "authdb"
	}
	if config.SchemaName == "" {
		config.SchemaName = "public"
	}
	return &Migrator{
		db:     db,
		config: config,
	}
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	migration, err := m.getMigration()
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	migration, err := m.getMigration()
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

// Steps runs N migrations (positive for up, negative for down)
func (m *Migrator) Steps(n int) error {
	migration, err := m.getMigration()
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migration steps: %w", err)
	}

	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	migration, err := m.getMigration()
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	return migration.Version()
}

// Force forces the migration to a specific version
func (m *Migrator) Force(version int) error {
	migration, err := m.getMigration()
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}

	return nil
}

// getMigration creates a new migrate instance
func (m *Migrator) getMigration() (*migrate.Migrate, error) {
	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	// Create database driver
	dbDriver, err := postgres.WithInstance(m.db, &postgres.Config{
		DatabaseName: m.config.DatabaseName,
		SchemaName:   m.config.SchemaName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	migration, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		m.config.DatabaseName, dbDriver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return migration, nil
}

// RunMigrationsFromPath runs migrations from a file path (for development)
func RunMigrationsFromPath(db *sql.DB, migrationsPath string, config MigrationConfig) error {
	if config.DatabaseName == "" {
		config.DatabaseName = "authdb"
	}
	if config.SchemaName == "" {
		config.SchemaName = "public"
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: config.DatabaseName,
		SchemaName:   config.SchemaName,
	})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	migration, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		config.DatabaseName,
		dbDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}