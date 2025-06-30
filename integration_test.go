package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"

	"github.com/abueno/go-auth-jwt/internal/config"
	"github.com/abueno/go-auth-jwt/internal/db"
	"github.com/abueno/go-auth-jwt/internal/email"
	"github.com/abueno/go-auth-jwt/internal/http/handlers"
	"github.com/abueno/go-auth-jwt/internal/http/middleware"
	"github.com/abueno/go-auth-jwt/internal/token"
	"github.com/abueno/go-auth-jwt/internal/metrics"
	"github.com/abueno/go-auth-jwt/internal/repository/postgres"
	"github.com/abueno/go-auth-jwt/internal/service"
	"github.com/abueno/go-auth-jwt/internal/worker"
)

// TestServer encapsulates all the components needed for integration testing
type TestServer struct {
	db             *sql.DB
	server         *httptest.Server
	userService    service.UserService
	authService    service.AuthService
	emailService   email.Service
	tokenService   token.Service
	metricsService *metrics.Metrics
	config         *config.Config
}

// SetupTestServer creates a new test server with all dependencies
func SetupTestServer(t *testing.T) *TestServer {
	// Create test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "test_auth",
			SSLMode:  "disable",
		},
		JWT: config.JWTConfig{
			Secret:                   "test-secret",
			AccessTokenExpiration:    15 * time.Minute,
			RefreshTokenExpiration:   7 * 24 * time.Hour,
			EmailVerificationExpiry:  24 * time.Hour,
			PasswordResetTokenExpiry: 1 * time.Hour,
		},
		Email: config.EmailConfig{
			From: "test@example.com",
			SMTP: config.SMTPConfig{
				Host:     "localhost",
				Port:     1025,
				Username: "test",
				Password: "test",
			},
			Templates: config.EmailTemplates{
				VerificationEmail: "templates/verification.html",
				WelcomeEmail:      "templates/welcome.html",
				PasswordReset:     "templates/password_reset.html",
			},
		},
		Server: config.ServerConfig{
			Port:            "8080",
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		Worker: config.WorkerConfig{
			MaxWorkers:     10,
			MaxQueueSize:   1000,
			MaxRetries:     3,
			RetryDelay:     time.Second,
			ProcessTimeout: 5 * time.Minute,
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		Security: config.SecurityConfig{
			BCryptCost:     10,
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		},
	}

	// Create database connection
	testDB := setupTestDatabase(t)

	// Run migrations
	if err := db.RunMigrations(testDB, "file://migrations"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create repositories
	userRepo := postgres.NewUserRepository(testDB)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(testDB)

	// Create services
	jwtService := token.NewService(cfg.JWT.Secret)
	
	// Create mock email service
	emailService := &mockEmailService{
		sentEmails: make(map[string][]mockEmail),
		mu:         sync.Mutex{},
	}

	// Create metrics service
	metricsService := metrics.NewMetrics()

	// Create auth service
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		jwtService,
		emailService,
		cfg.JWT.AccessTokenExpiration,
		cfg.JWT.RefreshTokenExpiration,
		cfg.Security.BCryptCost,
	)

	// Create user service
	userService := service.NewUserService(userRepo, cfg.Security.BCryptCost)

	// Create worker pool
	workerPool := worker.NewPool(cfg.Worker.MaxWorkers, cfg.Worker.MaxQueueSize)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, jwtService, cfg)
	userHandler := handlers.NewUserHandler(userService, jwtService)
	healthHandler := handlers.NewHealthHandler(testDB)

	// Create router
	router := http.NewServeMux()

	// Setup middleware
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.BurstSize)
	
	// Public routes
	router.HandleFunc("/health", healthHandler.Health)
	router.HandleFunc("/ready", healthHandler.Ready)
	router.HandleFunc("/auth/signup", withMiddleware(authHandler.Signup, rateLimiter.Limit))
	router.HandleFunc("/auth/login", withMiddleware(authHandler.Login, rateLimiter.Limit))
	router.HandleFunc("/auth/refresh", authHandler.RefreshToken)
	router.HandleFunc("/auth/verify-email", authHandler.VerifyEmail)
	router.HandleFunc("/auth/resend-verification", withMiddleware(authHandler.ResendVerificationEmail, rateLimiter.Limit))
	router.HandleFunc("/auth/forgot-password", withMiddleware(authHandler.ForgotPassword, rateLimiter.Limit))
	router.HandleFunc("/auth/reset-password", authHandler.ResetPassword)
	
	// Protected routes
	router.HandleFunc("/auth/logout", withAuth(authHandler.Logout, jwtService))
	router.HandleFunc("/users/profile", withAuth(userHandler.GetProfile, jwtService))
	router.HandleFunc("/users/update-password", withAuth(userHandler.UpdatePassword, jwtService))

	// Create test server
	server := httptest.NewServer(router)

	// Start worker pool
	go workerPool.Start(context.Background())

	return &TestServer{
		db:             testDB,
		server:         server,
		userService:    userService,
		authService:    authService,
		emailService:   emailService,
		tokenService:   jwtService,
		metricsService: metricsService,
		config:         cfg,
	}
}

// Helper function to wrap handlers with middleware
func withMiddleware(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Helper function to wrap handlers with auth middleware
func withAuth(handler http.HandlerFunc, jwtService token.Service) http.HandlerFunc {
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	return authMiddleware.Authenticate(handler)
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *sql.DB {
	// For integration tests, you might want to use a real test database
	// or a dockerized PostgreSQL instance
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/test_auth?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean up database before tests
	cleanupQueries := []string{
		"DROP TABLE IF EXISTS refresh_tokens CASCADE",
		"DROP TABLE IF EXISTS users CASCADE",
		"DROP TABLE IF EXISTS schema_migrations CASCADE",
	}

	for _, query := range cleanupQueries {
		if _, err := db.Exec(query); err != nil {
			t.Logf("Warning: Failed to clean up database: %v", err)
		}
	}

	return db
}

// Cleanup closes all resources
func (ts *TestServer) Cleanup() {
	ts.server.Close()
	ts.db.Close()
}

// Mock email service for testing
type mockEmailService struct {
	sentEmails map[string][]mockEmail
	mu         sync.Mutex
}

type mockEmail struct {
	to      string
	subject string
	body    string
	sentAt  time.Time
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, to, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sentEmails[to] = append(m.sentEmails[to], mockEmail{
		to:      to,
		subject: "Verify your email",
		body:    fmt.Sprintf("Verification token: %s", token),
		sentAt:  time.Now(),
	})
	return nil
}

func (m *mockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sentEmails[to] = append(m.sentEmails[to], mockEmail{
		to:      to,
		subject: "Welcome",
		body:    fmt.Sprintf("Welcome %s!", name),
		sentAt:  time.Now(),
	})
	return nil
}

func (m *mockEmailService) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sentEmails[to] = append(m.sentEmails[to], mockEmail{
		to:      to,
		subject: "Reset your password",
		body:    fmt.Sprintf("Reset token: %s", token),
		sentAt:  time.Now(),
	})
	return nil
}

func (m *mockEmailService) GetSentEmails(to string) []mockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.sentEmails[to]
}

// Test helpers

func makeRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}, headers map[string]string) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, server.URL+path, reqBody)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func parseResponse(t *testing.T, resp *http.Response, v interface{}) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if v != nil {
		err = json.Unmarshal(body, v)
		require.NoError(t, err)
	}
}

// Integration Tests

func TestIntegration_CompleteAuthFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// 1. Sign up a new user
	signupReq := map[string]string{
		"email":    "testuser@example.com",
		"password": "SecurePassword123!",
		"name":     "Test User",
	}

	resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var signupResp map[string]interface{}
	parseResponse(t, resp, &signupResp)
	assert.Contains(t, signupResp, "message")

	// 2. Verify that verification email was sent
	emailSvc := ts.emailService.(*mockEmailService)
	emails := emailSvc.GetSentEmails("testuser@example.com")
	require.Len(t, emails, 1)
	assert.Contains(t, emails[0].body, "Verification token:")

	// Extract verification token (in real scenario, this would be from email link)
	// For testing, we'll generate one directly
	verificationToken, err := ts.authService.GenerateVerificationToken(context.Background(), "testuser@example.com")
	require.NoError(t, err)

	// 3. Verify email
	verifyReq := map[string]string{
		"token": verificationToken,
	}

	resp = makeRequest(t, ts.server, "POST", "/auth/verify-email", verifyReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Login with verified account
	loginReq := map[string]string{
		"email":    "testuser@example.com",
		"password": "SecurePassword123!",
	}

	resp = makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]interface{}
	parseResponse(t, resp, &loginResp)
	assert.Contains(t, loginResp, "access_token")
	assert.Contains(t, loginResp, "refresh_token")

	accessToken := loginResp["access_token"].(string)
	refreshToken := loginResp["refresh_token"].(string)

	// 5. Access protected endpoint
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp = makeRequest(t, ts.server, "GET", "/users/profile", nil, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var profileResp map[string]interface{}
	parseResponse(t, resp, &profileResp)
	assert.Equal(t, "testuser@example.com", profileResp["email"])
	assert.Equal(t, "Test User", profileResp["name"])

	// 6. Refresh token
	refreshReq := map[string]string{
		"refresh_token": refreshToken,
	}

	resp = makeRequest(t, ts.server, "POST", "/auth/refresh", refreshReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var refreshResp map[string]interface{}
	parseResponse(t, resp, &refreshResp)
	assert.Contains(t, refreshResp, "access_token")
	newAccessToken := refreshResp["access_token"].(string)

	// 7. Use new access token
	headers["Authorization"] = "Bearer " + newAccessToken

	resp = makeRequest(t, ts.server, "GET", "/users/profile", nil, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Logout
	resp = makeRequest(t, ts.server, "POST", "/auth/logout", nil, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 9. Verify old tokens are invalid
	resp = makeRequest(t, ts.server, "GET", "/users/profile", nil, headers)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIntegration_ConcurrentUserRegistrations(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	const numUsers = 10
	var wg sync.WaitGroup
	errors := make(chan error, numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			signupReq := map[string]string{
				"email":    fmt.Sprintf("user%d@example.com", index),
				"password": "SecurePassword123!",
				"name":     fmt.Sprintf("User %d", index),
			}

			resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
			if resp.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("user %d: expected status 201, got %d", index, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all users were created
	for i := 0; i < numUsers; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		user, err := ts.userService.GetByEmail(context.Background(), email)
		assert.NoError(t, err)
		assert.Equal(t, email, user.Email)
	}
}

func TestIntegration_ErrorScenarios(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Signup with invalid email", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "invalid-email",
			"password": "SecurePassword123!",
			"name":     "Test User",
		}

		resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Signup with weak password", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "test@example.com",
			"password": "weak",
			"name":     "Test User",
		}

		resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Login with wrong password", func(t *testing.T) {
		// First create a user
		signupReq := map[string]string{
			"email":    "wrongpass@example.com",
			"password": "CorrectPassword123!",
			"name":     "Test User",
		}

		resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify email
		token, _ := ts.authService.GenerateVerificationToken(context.Background(), "wrongpass@example.com")
		verifyReq := map[string]string{"token": token}
		resp = makeRequest(t, ts.server, "POST", "/auth/verify-email", verifyReq, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Try to login with wrong password
		loginReq := map[string]string{
			"email":    "wrongpass@example.com",
			"password": "WrongPassword123!",
		}

		resp = makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access protected route without token", func(t *testing.T) {
		resp := makeRequest(t, ts.server, "GET", "/users/profile", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access protected route with invalid token", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer invalid-token",
		}

		resp := makeRequest(t, ts.server, "GET", "/users/profile", nil, headers)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Duplicate email registration", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "duplicate@example.com",
			"password": "SecurePassword123!",
			"name":     "Test User",
		}

		// First registration
		resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Duplicate registration
		resp = makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestIntegration_RateLimiting(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Make requests up to the rate limit
	for i := 0; i < ts.config.RateLimit.BurstSize; i++ {
		loginReq := map[string]string{
			"email":    fmt.Sprintf("test%d@example.com", i),
			"password": "password",
		}

		resp := makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
		// We expect 401 because credentials are invalid, but not rate limited
		assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode)
	}

	// Next request should be rate limited
	loginReq := map[string]string{
		"email":    "ratelimited@example.com",
		"password": "password",
	}

	resp := makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestIntegration_HealthChecks(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Health endpoint", func(t *testing.T) {
		resp := makeRequest(t, ts.server, "GET", "/health", nil, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		parseResponse(t, resp, &health)
		assert.Equal(t, "ok", health["status"])
	})

	t.Run("Ready endpoint", func(t *testing.T) {
		resp := makeRequest(t, ts.server, "GET", "/ready", nil, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ready map[string]interface{}
		parseResponse(t, resp, &ready)
		assert.Equal(t, "ok", ready["status"])
		assert.Contains(t, ready, "database")
	})
}

func TestIntegration_EmailVerificationFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create unverified user
	signupReq := map[string]string{
		"email":    "unverified@example.com",
		"password": "SecurePassword123!",
		"name":     "Unverified User",
	}

	resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try to login without verification
	loginReq := map[string]string{
		"email":    "unverified@example.com",
		"password": "SecurePassword123!",
	}

	resp = makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Resend verification email
	resendReq := map[string]string{
		"email": "unverified@example.com",
	}

	resp = makeRequest(t, ts.server, "POST", "/auth/resend-verification", resendReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that 2 emails were sent
	emailSvc := ts.emailService.(*mockEmailService)
	emails := emailSvc.GetSentEmails("unverified@example.com")
	assert.Len(t, emails, 2)

	// Verify email
	token, _ := ts.authService.GenerateVerificationToken(context.Background(), "unverified@example.com")
	verifyReq := map[string]string{"token": token}
	
	resp = makeRequest(t, ts.server, "POST", "/auth/verify-email", verifyReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Now login should work
	resp = makeRequest(t, ts.server, "POST", "/auth/login", loginReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_TokenExpiration(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create a custom JWT service with very short expiration for testing
	shortJWTService := token.NewService(ts.config.JWT.Secret)

	// Create and verify a user
	signupReq := map[string]string{
		"email":    "expiry@example.com",
		"password": "SecurePassword123!",
		"name":     "Test User",
	}

	resp := makeRequest(t, ts.server, "POST", "/auth/signup", signupReq, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	verificationToken, _ := ts.authService.GenerateVerificationToken(context.Background(), "expiry@example.com")
	verifyReq := map[string]string{"token": verificationToken}
	resp = makeRequest(t, ts.server, "POST", "/auth/verify-email", verifyReq, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Create a token that expires in 1 second
	claims := &token.Claims{
		UserID: "test-user-id",
		Email:  "expiry@example.com",
	}
	
	shortToken, err := shortJWTService.GenerateToken(claims, 1*time.Second)
	require.NoError(t, err)

	// Token should work immediately
	headers := map[string]string{
		"Authorization": "Bearer " + shortToken,
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Token should now be expired
	resp = makeRequest(t, ts.server, "GET", "/users/profile", nil, headers)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}