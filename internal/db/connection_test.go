package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/abueno/go-auth-jwt/internal/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.DatabaseConfig
		wantErr bool
	}{
		{
			name: "invalid DSN",
			cfg: &config.DatabaseConfig{
				DSN:             "invalid://dsn",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
			},
			wantErr: true,
		},
		// Note: Actual connection tests would require a test database
		// These would be in integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestDB_Health(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "healthy database",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
				rows := sqlmock.NewRows([]string{"result"}).AddRow(1)
				mock.ExpectQuery("SELECT 1").WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "ping fails",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
			errMsg:  "database health check failed",
		},
		{
			name: "query fails",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
				mock.ExpectQuery("SELECT 1").WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errMsg:  "database query check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("Failed to create mock: %v", err)
			}
			defer mockDB.Close()

			db := &DB{mockDB}
			tt.setupMock(mock)

			ctx := context.Background()
			err = db.Health(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Health() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDB_Close(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	db := &DB{mockDB}

	// Expect close to be called
	mock.ExpectClose()

	// Close the database
	err = db.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDB_Stats(t *testing.T) {
	// Create mock database
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mockDB.Close()

	db := &DB{mockDB}

	// Call Stats
	stats := db.Stats()

	// Basic validation - stats should be a valid struct
	if stats.MaxOpenConnections < 0 {
		t.Error("Invalid MaxOpenConnections")
	}
}

func TestDB_TestConnection(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful ping",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			wantErr: false,
		},
		{
			name: "ping fails",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("Failed to create mock: %v", err)
			}
			defer mockDB.Close()

			db := &DB{mockDB}
			tt.setupMock(mock)

			ctx := context.Background()
			err = db.TestConnection(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestConnection() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
