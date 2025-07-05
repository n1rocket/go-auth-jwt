package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/config"
	"github.com/n1rocket/go-auth-jwt/internal/db"
	httpserver "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/repository/postgres"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

// App represents the application with all its dependencies
type App struct {
	Config       *config.Config
	DB           *db.DB
	Server       *http.Server
	AuthService  *service.AuthService
	TokenManager *token.Manager
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) (*App, error) {
	// Connect to database
	dbPool, err := db.Connect(cfg.Database.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbPool.TestConnection(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to test database connection: %w", err)
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepository(dbPool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(dbPool)

	// Initialize security components
	passwordHasher := security.NewDefaultPasswordHasher()

	tokenManager, err := token.NewManager(
		cfg.JWT.Algorithm,
		cfg.JWT.Secret,
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create token manager: %w", err)
	}

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		cfg.JWT.RefreshTokenTTL,
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      httpserver.Routes(authService, tokenManager),
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	return &App{
		Config:       cfg,
		DB:           dbPool,
		Server:       srv,
		AuthService:  authService,
		TokenManager: tokenManager,
	}, nil
}

// Close closes all resources
func (a *App) Close() error {
	if a.DB != nil {
		a.DB.Close()
	}
	return nil
}
