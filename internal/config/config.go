package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Email    EmailConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
}

type AppConfig struct {
	Port            int
	Environment     string
	Name            string
	BaseURL         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ConnectionString returns the database connection string
func (d DatabaseConfig) ConnectionString() string {
	return d.DSN
}

type JWTConfig struct {
	Secret          string
	PrivateKeyPath  string
	PublicKeyPath   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
	Algorithm       string // HS256 or RS256
}

type EmailConfig struct {
	SMTPHost               string
	SMTPPort               int
	SMTPUser               string
	SMTPPassword           string
	FromAddress            string
	FromName               string
	SupportEmail           string
	WorkerCount            int
	QueueSize              int
	SendLoginNotifications bool
	TLSEnabled             bool
}

type LoggingConfig struct {
	Level  string
	Format string // json or text
}

type MetricsConfig struct {
	Port    string
	Enabled bool
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Port:            parseIntOrDefault("APP_PORT", 8080),
			Environment:     getEnvOrDefault("APP_ENV", "development"),
			Name:            getEnvOrDefault("APP_NAME", "Auth Service"),
			BaseURL:         getEnvOrDefault("APP_BASE_URL", "http://localhost:8080"),
			ReadTimeout:     parseDurationOrDefault("APP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    parseDurationOrDefault("APP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     parseDurationOrDefault("APP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: parseDurationOrDefault("APP_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             getEnvOrError("DB_DSN"),
			MaxOpenConns:    parseIntOrDefault("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    parseIntOrDefault("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: parseDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: parseDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
		},
		JWT: JWTConfig{
			Secret:          os.Getenv("JWT_SECRET"),
			PrivateKeyPath:  os.Getenv("JWT_PRIVATE_KEY_PATH"),
			PublicKeyPath:   os.Getenv("JWT_PUBLIC_KEY_PATH"),
			AccessTokenTTL:  parseDurationOrDefault("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: parseDurationOrDefault("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnvOrDefault("JWT_ISSUER", "go-auth-jwt"),
			Algorithm:       getEnvOrDefault("JWT_ALGORITHM", "HS256"),
		},
		Email: EmailConfig{
			SMTPHost:               os.Getenv("SMTP_HOST"),
			SMTPPort:               parseIntOrDefault("SMTP_PORT", 587),
			SMTPUser:               os.Getenv("SMTP_USER"),
			SMTPPassword:           os.Getenv("SMTP_PASS"),
			FromAddress:            getEnvOrDefault("EMAIL_FROM_ADDRESS", os.Getenv("SMTP_USER")),
			FromName:               getEnvOrDefault("EMAIL_FROM_NAME", "Auth Service"),
			SupportEmail:           getEnvOrDefault("EMAIL_SUPPORT", "support@example.com"),
			WorkerCount:            parseIntOrDefault("EMAIL_WORKER_COUNT", 5),
			QueueSize:              parseIntOrDefault("EMAIL_QUEUE_SIZE", 100),
			SendLoginNotifications: parseBoolOrDefault("EMAIL_SEND_LOGIN_NOTIFICATIONS", false),
			TLSEnabled:             parseBoolOrDefault("SMTP_TLS_ENABLED", true),
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", "info"),
			Format: getEnvOrDefault("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Port:    getEnvOrDefault("METRICS_PORT", "9090"),
			Enabled: parseBoolOrDefault("METRICS_ENABLED", true),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	// Validate JWT configuration
	if c.JWT.Algorithm == "HS256" {
		if c.JWT.Secret == "" {
			return fmt.Errorf("JWT_SECRET is required for HS256 algorithm")
		}
	} else if c.JWT.Algorithm == "RS256" {
		if c.JWT.PrivateKeyPath == "" || c.JWT.PublicKeyPath == "" {
			return fmt.Errorf("JWT_PRIVATE_KEY_PATH and JWT_PUBLIC_KEY_PATH are required for RS256 algorithm")
		}
	} else {
		return fmt.Errorf("unsupported JWT algorithm: %s", c.JWT.Algorithm)
	}

	// Validate database configuration
	if c.Database.DSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}

	// Validate email configuration
	if c.Email.SMTPHost == "" || c.Email.SMTPUser == "" || c.Email.SMTPPassword == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	// Validate logging level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrError(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return ""
}

func parseIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func parseBoolOrDefault(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}
