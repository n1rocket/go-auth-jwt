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