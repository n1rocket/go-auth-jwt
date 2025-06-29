package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save current env vars
	originalEnv := map[string]string{
		"DB_DSN":     os.Getenv("DB_DSN"),
		"SMTP_HOST":  os.Getenv("SMTP_HOST"),
		"SMTP_USER":  os.Getenv("SMTP_USER"),
		"SMTP_PASS":  os.Getenv("SMTP_PASS"),
		"JWT_SECRET": os.Getenv("JWT_SECRET"),
	}

	// Restore env vars after test
	t.Cleanup(func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})

	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid config with HS256",
			envVars: map[string]string{
				"DB_DSN":        "postgres://user:pass@localhost/db",
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_USER":     "user@example.com",
				"SMTP_PASS":     "password",
				"JWT_SECRET":    "secret",
				"JWT_ALGORITHM": "HS256",
			},
			wantErr: false,
		},
		{
			name: "valid config with RS256",
			envVars: map[string]string{
				"DB_DSN":               "postgres://user:pass@localhost/db",
				"SMTP_HOST":            "smtp.example.com",
				"SMTP_USER":            "user@example.com",
				"SMTP_PASS":            "password",
				"JWT_PRIVATE_KEY_PATH": "/path/to/private.pem",
				"JWT_PUBLIC_KEY_PATH":  "/path/to/public.pem",
				"JWT_ALGORITHM":        "RS256",
			},
			wantErr: false,
		},
		{
			name: "missing DB_DSN",
			envVars: map[string]string{
				"SMTP_HOST":  "smtp.example.com",
				"SMTP_USER":  "user@example.com",
				"SMTP_PASS":  "password",
				"JWT_SECRET": "secret",
			},
			wantErr: true,
		},
		{
			name: "missing JWT_SECRET for HS256",
			envVars: map[string]string{
				"DB_DSN":        "postgres://user:pass@localhost/db",
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_USER":     "user@example.com",
				"SMTP_PASS":     "password",
				"JWT_ALGORITHM": "HS256",
			},
			wantErr: true,
		},
		{
			name: "invalid JWT algorithm",
			envVars: map[string]string{
				"DB_DSN":        "postgres://user:pass@localhost/db",
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_USER":     "user@example.com",
				"SMTP_PASS":     "password",
				"JWT_ALGORITHM": "INVALID",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"DB_DSN":     "postgres://user:pass@localhost/db",
				"SMTP_HOST":  "smtp.example.com",
				"SMTP_USER":  "user@example.com",
				"SMTP_PASS":  "password",
				"JWT_SECRET": "secret",
				"LOG_LEVEL":  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			os.Clearenv()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg == nil {
				t.Error("Load() returned nil config without error")
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env var exists",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "actual",
			want:         "actual",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() { os.Unsetenv(tt.key) })
			}

			got := getEnvOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "20",
			want:         20,
		},
		{
			name:         "invalid int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "not-a-number",
			want:         10,
		},
		{
			name:         "empty value",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() { os.Unsetenv(tt.key) })
			}

			got := parseIntOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseIntOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDurationOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		want         time.Duration
	}{
		{
			name:         "valid duration",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "30s",
			want:         30 * time.Second,
		},
		{
			name:         "invalid duration",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "invalid",
			want:         10 * time.Second,
		},
		{
			name:         "empty value",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "",
			want:         10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() { os.Unsetenv(tt.key) })
			}

			got := parseDurationOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseDurationOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBoolOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "valid true",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "valid false",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "invalid bool",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			want:         true,
		},
		{
			name:         "empty value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				t.Cleanup(func() { os.Unsetenv(tt.key) })
			}

			got := parseBoolOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseBoolOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
