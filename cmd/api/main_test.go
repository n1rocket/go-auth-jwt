package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/config"
)

// Mock os.Exit for testing
var osExit = os.Exit

func TestNewApp(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid database connection",
			config: &config.Config{
				Database: config.DatabaseConfig{
					DSN: "postgres://invalid:invalid@invalid:5432/test?sslmode=disable",
				},
				JWT: config.JWTConfig{
					Algorithm:       "HS256",
					Secret:          "test-secret",
					AccessTokenTTL:  15 * time.Minute,
					RefreshTokenTTL: 7 * 24 * time.Hour,
					Issuer:          "test",
				},
				App: config.AppConfig{
					Port:            8080,
					ReadTimeout:     15 * time.Second,
					WriteTimeout:    15 * time.Second,
					IdleTimeout:     60 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "failed to connect to database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewApp() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if app != nil {
				defer app.Close()
			}
		})
	}
}

func TestAppClose(t *testing.T) {
	// Test that Close doesn't panic with nil DB
	app := &App{}
	err := app.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestMainFunction(t *testing.T) {
	// Test main function with invalid config
	// This will exit quickly due to missing environment variables
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping main function test in CI")
	}

	// Save original args and environment
	origArgs := os.Args
	origEnv := os.Environ()

	// Set minimal environment to make it fail fast
	os.Clearenv()
	os.Setenv("DATABASE_URL", "")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("DB_DSN", "postgres://test:test@invalid:5432/test?sslmode=disable")
	os.Setenv("EMAIL_ENABLED", "false")
	os.Setenv("SMTP_HOST", "localhost")
	os.Setenv("SMTP_PORT", "1025")
	os.Setenv("SMTP_USER", "test")
	os.Setenv("SMTP_PASS", "test")

	// Capture exit
	oldExit := osExit
	osExit = func(code int) {
		panic("os.Exit called")
	}

	defer func() {
		// Restore
		osExit = oldExit
		os.Args = origArgs
		for _, env := range origEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
		recover() // Recover from panic
	}()

	// Run main - it should exit
	main()

	// Should not reach here
	t.Error("Expected main to exit")
}

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
