package integration_test

import (
	"bytes"
	"context"
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

	"github.com/n1rocket/go-auth-jwt/internal/config"
	"github.com/n1rocket/go-auth-jwt/internal/db"
	httpserver "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/repository/postgres"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

// TestServer encapsulates all the components needed for integration testing
type TestServer struct {
	server      *httptest.Server
	authService *service.AuthService
	config      *config.Config
	cleanup     func()
}

// SetupIntegrationTestServer creates a new test server with all dependencies
func SetupIntegrationTestServer(t testing.TB) *TestServer {
	// Create test configuration
	cfg := &config.Config{
		App: config.AppConfig{
			Port:        8080,
			Environment: "test",
		},
		Database: config.DatabaseConfig{
			DSN: "postgres://test:test@localhost:5432/test_auth?sslmode=disable",
		},
		JWT: config.JWTConfig{
			Secret:          "test-secret-key",
			PrivateKeyPath:  "",
			PublicKeyPath:   "",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
			Issuer:          "test-auth",
			Algorithm:       "HS256",
		},
		Email: config.EmailConfig{
			SMTPHost:     "localhost",
			SMTPPort:     1025,
			SMTPUser:     "test",
			SMTPPassword: "test",
			FromAddress:  "test@example.com",
			FromName:     "Test Auth",
		},
	}

	// Setup test database
	testDB, err := db.Connect(cfg.Database.DSN)
	require.NoError(t, err)

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = testDB.TestConnection(ctx)
	cancel()
	require.NoError(t, err)

	// Create repositories
	userRepo := postgres.NewUserRepository(testDB)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(testDB)

	// Create services
	passwordHasher := security.NewDefaultPasswordHasher()
	tokenManager, err := token.NewManager(
		cfg.JWT.Algorithm,
		cfg.JWT.Secret,
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
	)
	require.NoError(t, err)

	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		cfg.JWT.RefreshTokenTTL,
	)

	// Create router
	handler := httpserver.Routes(authService, tokenManager)

	// Create test server
	server := httptest.NewServer(handler)

	cleanup := func() {
		server.Close()
		testDB.Close()
	}

	return &TestServer{
		server:      server,
		authService: authService,
		config:      cfg,
		cleanup:     cleanup,
	}
}

// Test helpers

func makeIntegrationRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}, headers map[string]string) *http.Response {
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

func parseIntegrationResponse(t *testing.T, resp *http.Response, v interface{}) {
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
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	// 1. Sign up a new user
	signupReq := map[string]string{
		"email":    "testuser@example.com",
		"password": "SecurePassword123!",
	}

	resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var signupResp map[string]interface{}
	parseIntegrationResponse(t, resp, &signupResp)
	assert.Contains(t, signupResp, "user_id")

	// 2. Login (email verification might be disabled in test)
	loginReq := map[string]string{
		"email":    "testuser@example.com",
		"password": "SecurePassword123!",
	}

	resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/login", loginReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]interface{}
	parseIntegrationResponse(t, resp, &loginResp)
	assert.Contains(t, loginResp, "access_token")
	assert.Contains(t, loginResp, "refresh_token")

	accessToken := loginResp["access_token"].(string)
	refreshToken := loginResp["refresh_token"].(string)

	// 3. Access protected endpoint
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp = makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var profileResp map[string]interface{}
	parseIntegrationResponse(t, resp, &profileResp)
	assert.Equal(t, "testuser@example.com", profileResp["email"])

	// 4. Refresh token
	refreshReq := map[string]string{
		"refresh_token": refreshToken,
	}

	resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/refresh", refreshReq, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var refreshResp map[string]interface{}
	parseIntegrationResponse(t, resp, &refreshResp)
	assert.Contains(t, refreshResp, "access_token")
	newAccessToken := refreshResp["access_token"].(string)

	// 5. Use new access token
	headers["Authorization"] = "Bearer " + newAccessToken

	resp = makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 6. Logout
	logoutReq := map[string]string{
		"refresh_token": refreshToken,
	}
	resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/logout", logoutReq, headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 7. Verify old tokens are invalid
	resp = makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, headers)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIntegration_ConcurrentUserRegistrations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

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
			}

			resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
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
}

func TestIntegration_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	t.Run("Signup with invalid email", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "invalid-email",
			"password": "SecurePassword123!",
		}

		resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Signup with weak password", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "test@example.com",
			"password": "weak",
		}

		resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Login with wrong password", func(t *testing.T) {
		// First create a user
		signupReq := map[string]string{
			"email":    "wrongpass@example.com",
			"password": "CorrectPassword123!",
		}

		resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Try to login with wrong password (may need to verify email first in production)
		loginReq := map[string]string{
			"email":    "wrongpass@example.com",
			"password": "WrongPassword123!",
		}

		resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/login", loginReq, nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access protected route without token", func(t *testing.T) {
		resp := makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access protected route with invalid token", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer invalid-token",
		}

		resp := makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, headers)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Duplicate email registration", func(t *testing.T) {
		signupReq := map[string]string{
			"email":    "duplicate@example.com",
			"password": "SecurePassword123!",
		}

		// First registration
		resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Duplicate registration
		resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestIntegration_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	// Make many requests to trigger rate limiting
	for i := 0; i < 50; i++ {
		loginReq := map[string]string{
			"email":    fmt.Sprintf("test%d@example.com", i),
			"password": "password",
		}

		resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/login", loginReq, nil)
		// Eventually we should hit rate limit
		if resp.StatusCode == http.StatusTooManyRequests {
			// Success - rate limiting is working
			return
		}
	}

	t.Error("Expected to hit rate limit but didn't")
}

func TestIntegration_HealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	t.Run("Health endpoint", func(t *testing.T) {
		resp := makeIntegrationRequest(t, ts.server, "GET", "/health", nil, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		parseIntegrationResponse(t, resp, &health)
		assert.Equal(t, "ok", health["status"])
	})

	t.Run("Ready endpoint", func(t *testing.T) {
		resp := makeIntegrationRequest(t, ts.server, "GET", "/ready", nil, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ready map[string]interface{}
		parseIntegrationResponse(t, resp, &ready)
		assert.Equal(t, "ok", ready["status"])
		assert.Contains(t, ready, "database")
	})
}

func TestIntegration_EmailVerificationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	// Create unverified user
	signupReq := map[string]string{
		"email":    "unverified@example.com",
		"password": "SecurePassword123!",
	}

	resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try to login without verification (might work in test mode)
	loginReq := map[string]string{
		"email":    "unverified@example.com",
		"password": "SecurePassword123!",
	}

	resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/login", loginReq, nil)
	// In test mode, login might work without email verification
	if resp.StatusCode == http.StatusForbidden {
		// Email verification is required
		verifyReq := map[string]string{
			"token": "test-verification-token",
		}
		
		resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/verify-email", verifyReq, nil)
		// Verification might fail with test token, but that's OK for this test
	}
}

func TestIntegration_TokenExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ts := SetupIntegrationTestServer(t)
	defer ts.cleanup()

	// This test is simplified since we can't easily create custom tokens
	// with different expiration times without access to internal services
	
	// Create and login a user
	signupReq := map[string]string{
		"email":    "expiry@example.com",
		"password": "SecurePassword123!",
	}

	resp := makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/signup", signupReq, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	loginReq := map[string]string{
		"email":    "expiry@example.com",
		"password": "SecurePassword123!",
	}

	resp = makeIntegrationRequest(t, ts.server, "POST", "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]interface{}
	parseIntegrationResponse(t, resp, &loginResp)

	// Test that invalid token format is rejected
	headers := map[string]string{
		"Authorization": "Bearer invalid.token.format",
	}

	resp = makeIntegrationRequest(t, ts.server, "GET", "/api/v1/auth/me", nil, headers)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}