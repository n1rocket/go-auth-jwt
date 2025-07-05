package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/config"
	"github.com/n1rocket/go-auth-jwt/internal/db"
	httpserver "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/repository/postgres"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to database
	dbPool, err := db.Connect(cfg.Database.ConnectionString())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := dbPool.TestConnection(ctx); err != nil {
		cancel()
		slog.Error("failed to test database connection", "error", err)
		os.Exit(1)
	}
	cancel()

	// Initialize dependencies
	userRepo := postgres.NewUserRepository(dbPool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(dbPool)
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
		slog.Error("failed to create token manager", "error", err)
		os.Exit(1)
	}

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

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server",
			"port", cfg.App.Port,
			"environment", cfg.App.Environment,
		)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		slog.Info("shutdown signal received", "signal", sig)

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
			// Force shutdown
			if err := srv.Close(); err != nil {
				slog.Error("forced shutdown failed", "error", err)
			}
		}
	}

	slog.Info("server stopped")
}
