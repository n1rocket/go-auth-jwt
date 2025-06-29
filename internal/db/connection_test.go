package db

import (
	"testing"
	"time"

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
	// This test would require a real database connection
	// It should be part of integration tests
	t.Skip("Skipping test that requires database connection")
}

func TestDB_Close(t *testing.T) {
	// This test would require a real database connection
	// It should be part of integration tests
	t.Skip("Skipping test that requires database connection")
}

func TestDB_Stats(t *testing.T) {
	// This test would require a real database connection
	// It should be part of integration tests
	t.Skip("Skipping test that requires database connection")
}
