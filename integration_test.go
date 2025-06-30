// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/abueno/go-auth-jwt/internal/config"
	"github.com/abueno/go-auth-jwt/internal/db"
	"github.com/abueno/go-auth-jwt/internal/email"
	"github.com/abueno/go-auth-jwt/internal/http/handlers"
	"github.com/abueno/go-auth-jwt/internal/http/middleware"
	"github.com/abueno/go-auth-jwt/internal/repository/postgres"
	"github.com/abueno/go-auth-jwt/internal/security"
	"github.com/abueno/go-auth-jwt/internal/service"
	"github.com/abueno/go-auth-jwt/internal/token"
	"github.com/abueno/go-auth-jwt/internal/worker"
	"github.com/gorilla/mux"
	"log/slog"
)

var (
	testServer *httptest.Server
	testDB     *db.DB
	testEmail  *email.MockService
)

func TestMain(m *testing.M) {
	// Setup
	setupTestEnvironment()
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	teardownTestEnvironment()
	
	os.Exit(code)
}

func setupTestEnvironment() {
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Load test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         "0",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: config.DatabaseConfig{
			DSN:             getEnvOrDefault("TEST_DATABASE_DSN", "postgres://postgres:password@localhost:5432/authdb_test?sslmode=disable"),
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		},
		JWT: config.JWTConfig{
			Secret:    "test-secret-key-for-integration-testing",
			Algorithm: "HS256",
			Issuer:    "auth-test",
			Duration:  15 * time.Minute,
		},
		Email: config.EmailConfig{
			SMTP: config.SMTPConfig{
				Host:        "localhost",
				Port:        1025,
				Username:    "test",
				Password:    "test",
				FromAddress: "test@example.com",
				FromName:    "Test App",
			},
		},
		Security: config.SecurityConfig{
			CORS: middleware.DefaultCORSConfig(),
		},
		Worker: config.WorkerConfig{
			PoolSize:  2,
			QueueSize: 10,
		},
		App: config.AppConfig{
			BaseURL:     "http://localhost:8080",
			FrontendURL: "http://localhost:3000",
		},
	}

	// Create database connection
	var err error
	testDB, err = db.New(&cfg.Database)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}

	// Clean database
	cleanDatabase()

	// Create repositories
	userRepo := postgres.NewUserRepository(testDB)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(testDB)

	// Create services
	passwordHasher := security.NewDefaultPasswordHasher()
	tokenManager, err := token.NewManager(
		cfg.JWT.Algorithm,
		cfg.JWT.Secret,
		"", "", // No RS256 keys for test
		cfg.JWT.Issuer,
		cfg.JWT.Duration,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create token manager: %v", err))
	}

	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		7*24*time.Hour, // 7 days refresh token TTL
	)

	// Create mock email service
	testEmail = email.NewMockService(logger)
	
	// Create email dispatcher
	emailDispatcher := worker.NewEmailDispatcher(testEmail, cfg.Worker.PoolSize, cfg.Worker.QueueSize, logger)
	emailDispatcher.Start()

	// Create auth service with email
	authServiceWithEmail := service.NewAuthServiceWithEmail(authService, emailDispatcher, cfg, logger)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Create router
	router := mux.NewRouter()
	
	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recover(router.NotFoundHandler))
	router.Use(middleware.Security(cfg.Security))
	router.Use(middleware.CORS(cfg.Security.CORS))

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Auth routes
	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/signup", authHandler.Signup).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")
	authRouter.HandleFunc("/refresh", authHandler.Refresh).Methods("POST")
	authRouter.HandleFunc("/verify-email", authHandler.VerifyEmail).Methods("POST")

	// Protected routes (need auth middleware)
	protected := authRouter.NewRoute().Subrouter()
	protected.Use(createAuthMiddleware(tokenManager))
	protected.HandleFunc("/logout", authHandler.Logout).Methods("POST")
	protected.HandleFunc("/logout-all", authHandler.LogoutAll).Methods("POST")
	protected.HandleFunc("/me", authHandler.GetCurrentUser).Methods("GET")

	// Health check routes
	router.HandleFunc("/health", handlers.Health).Methods("GET")
	router.HandleFunc("/ready", handlers.Ready).Methods("GET")

	// Create test server
	testServer = httptest.NewServer(router)
}

func teardownTestEnvironment() {
	if testServer != nil {
		testServer.Close()
	}
	if testDB != nil {
		cleanDatabase()
		testDB.Close()
	}
}

func cleanDatabase() {
	ctx := context.Background()
	queries := []string{
		"DELETE FROM refresh_tokens",
		"DELETE FROM users",
	}
	
	for _, query := range queries {
		if _, err := testDB.ExecContext(ctx, query); err != nil {
			fmt.Printf("Failed to clean database: %v\n", err)
		}
	}
}

func createAuthMiddleware(tm *token.Manager) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token
			const bearerPrefix = "Bearer "
			if !hasPrefix(authHeader, bearerPrefix) {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := authHeader[len(bearerPrefix):]
			
			// Validate token
			claims, err := tm.ValidateAccessToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := handlers.WithUserID(r.Context(), claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper functions for tests

func makeRequest(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, testServer.URL+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return client.Do(req)
}

func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// Actual integration tests

func TestCompleteAuthFlow(t *testing.T) {
	// 1. Signup
	signupReq := map[string]string{
		"email":    "integration@example.com",
		"password": "TestPassword123",
	}

	resp, err := makeRequest("POST", "/api/v1/auth/signup", signupReq, nil)
	if err != nil {
		t.Fatalf("Signup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var signupResp map[string]interface{}
	if err := parseResponse(resp, &signupResp); err != nil {
		t.Fatalf("Failed to parse signup response: %v", err)
	}

	// Check that email was sent
	emails := testEmail.GetSentEmails()
	if len(emails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(emails))
	}

	verificationEmail := emails[0]
	if verificationEmail.To != "integration@example.com" {
		t.Errorf("Expected email to integration@example.com, got %s", verificationEmail.To)
	}

	// Extract verification token from email (in real scenario, parse from email body)
	// For testing, we'll get it from the mock
	lastEmail, _ := testEmail.GetLastEmail()
	
	// 2. Verify email
	// In a real test, we'd parse the token from the email body
	// For now, we'll skip this step

	// 3. Login
	loginReq := map[string]string{
		"email":    "integration@example.com",
		"password": "TestPassword123",
	}

	resp, err = makeRequest("POST", "/api/v1/auth/login", loginReq, nil)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := parseResponse(resp, &loginResp); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Error("Expected access token")
	}
	if loginResp.RefreshToken == "" {
		t.Error("Expected refresh token")
	}

	// 4. Get current user
	headers := map[string]string{
		"Authorization": "Bearer " + loginResp.AccessToken,
	}

	resp, err = makeRequest("GET", "/api/v1/auth/me", nil, headers)
	if err != nil {
		t.Fatalf("Get user request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var userResp struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		CreatedAt     string `json:"created_at"`
	}
	if err := parseResponse(resp, &userResp); err != nil {
		t.Fatalf("Failed to parse user response: %v", err)
	}

	if userResp.Email != "integration@example.com" {
		t.Errorf("Expected email integration@example.com, got %s", userResp.Email)
	}

	// 5. Refresh token
	refreshReq := map[string]string{
		"refresh_token": loginResp.RefreshToken,
	}

	resp, err = makeRequest("POST", "/api/v1/auth/refresh", refreshReq, nil)
	if err != nil {
		t.Fatalf("Refresh request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var refreshResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := parseResponse(resp, &refreshResp); err != nil {
		t.Fatalf("Failed to parse refresh response: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Error("Expected new access token")
	}
	if refreshResp.RefreshToken == "" {
		t.Error("Expected new refresh token")
	}

	// 6. Logout
	logoutReq := map[string]string{
		"refresh_token": refreshResp.RefreshToken,
	}

	headers["Authorization"] = "Bearer " + refreshResp.AccessToken
	resp, err = makeRequest("POST", "/api/v1/auth/logout", logoutReq, headers)
	if err != nil {
		t.Fatalf("Logout request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 7. Verify token is invalid after logout
	resp, err = makeRequest("GET", "/api/v1/auth/me", nil, headers)
	if err != nil {
		t.Fatalf("Get user after logout request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 after logout, got %d", resp.StatusCode)
	}
}

func TestInvalidLogin(t *testing.T) {
	loginReq := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "WrongPassword",
	}

	resp, err := makeRequest("POST", "/api/v1/auth/login", loginReq, nil)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestDuplicateSignup(t *testing.T) {
	signupReq := map[string]string{
		"email":    "duplicate@example.com",
		"password": "TestPassword123",
	}

	// First signup
	resp, err := makeRequest("POST", "/api/v1/auth/signup", signupReq, nil)
	if err != nil {
		t.Fatalf("First signup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Clear sent emails
	testEmail.ClearEmails()

	// Duplicate signup
	resp, err = makeRequest("POST", "/api/v1/auth/signup", signupReq, nil)
	if err != nil {
		t.Fatalf("Duplicate signup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestInvalidRefreshToken(t *testing.T) {
	refreshReq := map[string]string{
		"refresh_token": "invalid-refresh-token",
	}

	resp, err := makeRequest("POST", "/api/v1/auth/refresh", refreshReq, nil)
	if err != nil {
		t.Fatalf("Refresh request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoints(t *testing.T) {
	// Health check
	resp, err := makeRequest("GET", "/health", nil, nil)
	if err != nil {
		t.Fatalf("Health request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Ready check
	resp, err = makeRequest("GET", "/ready", nil, nil)
	if err != nil {
		t.Fatalf("Ready request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var readyResp map[string]interface{}
	if err := parseResponse(resp, &readyResp); err != nil {
		t.Fatalf("Failed to parse ready response: %v", err)
	}

	if readyResp["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", readyResp["status"])
	}
}