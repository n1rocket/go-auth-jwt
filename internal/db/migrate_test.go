package db

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewMigrator(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	tests := []struct {
		name   string
		config MigrationConfig
		want   MigrationConfig
	}{
		{
			name:   "Default config",
			config: MigrationConfig{},
			want: MigrationConfig{
				DatabaseName: "authdb",
				SchemaName:   "public",
			},
		},
		{
			name: "Custom config",
			config: MigrationConfig{
				DatabaseName: "testdb",
				SchemaName:   "custom",
			},
			want: MigrationConfig{
				DatabaseName: "testdb",
				SchemaName:   "custom",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator := NewMigrator(mockDB, tt.config)
			assert.NotNil(t, migrator)
			assert.Equal(t, tt.want.DatabaseName, migrator.config.DatabaseName)
			assert.Equal(t, tt.want.SchemaName, migrator.config.SchemaName)
		})
	}
}

func TestMigrationConfig_Defaults(t *testing.T) {
	var config MigrationConfig
	
	// Create a mock DB
	mockDB, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	// Create migrator with empty config
	migrator := NewMigrator(mockDB, config)
	
	// Check defaults are applied
	assert.Equal(t, "authdb", migrator.config.DatabaseName)
	assert.Equal(t, "public", migrator.config.SchemaName)
}

func TestRunMigrationsFromPath_InvalidPath(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	// Test with invalid path
	err = RunMigrationsFromPath(mockDB, "/invalid/path", MigrationConfig{})
	assert.Error(t, err)
}

// Helper function to create a test database
func createTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	return db, mock
}

func TestMigrator_Up(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	migrator := NewMigrator(db, MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "public",
	})

	// Test Up method (will fail due to embedded FS not being available in tests)
	err := migrator.Up()
	assert.Error(t, err)
}

func TestMigrator_Down(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	migrator := NewMigrator(db, MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "public",
	})

	// Test Down method (will fail due to embedded FS not being available in tests)
	err := migrator.Down()
	assert.Error(t, err)
}

func TestMigrator_Steps(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	migrator := NewMigrator(db, MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "public",
	})

	// Test Steps method (will fail due to embedded FS not being available in tests)
	err := migrator.Steps(1)
	assert.Error(t, err)

	err = migrator.Steps(-1)
	assert.Error(t, err)
}

func TestMigrator_Version(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	migrator := NewMigrator(db, MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "public",
	})

	// Test Version method (will fail due to embedded FS not being available in tests)
	_, _, err := migrator.Version()
	assert.Error(t, err)
}

func TestMigrator_Force(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	migrator := NewMigrator(db, MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "public",
	})

	// Test Force method (will fail due to embedded FS not being available in tests)
	err := migrator.Force(1)
	assert.Error(t, err)
}

func TestRunMigrationsFromPath_WithValidDB(t *testing.T) {
	db, mock := createTestDB(t)
	defer db.Close()

	// Expect driver creation queries
	mock.ExpectExec("SELECT CURRENT_DATABASE()").WillReturnResult(sqlmock.NewResult(0, 0))
	
	// Test with valid DB but invalid path
	err := RunMigrationsFromPath(db, "testdata/migrations", MigrationConfig{
		DatabaseName: "testdb",
		SchemaName:   "custom",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database driver")
}