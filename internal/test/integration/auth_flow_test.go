//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/config"
	"github.com/n1rocket/go-auth-jwt/internal/db"
	httpserver "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/repository/postgres"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

func setupTestServer(t *testing.T) *httptest.Server {
	// Set test environment variables
	os.Setenv("DB_DSN", "postgres://postgres:postgres@localhost:5432/go_auth_jwt_test?sslmode=disable")
	os.Setenv("JWT_SECRET", "test-secret-key")
	
	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Connect to test database
	dbConn, err := db.Connect(cfg.Database.ConnectionString())
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Clean up test data
	cleanupTestData(t, dbConn)
	
	// Setup dependencies
	userRepo := postgres.NewUserRepository(dbConn)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(dbConn)
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
		t.Fatalf("Failed to create token manager: %v", err)
	}
	
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		cfg.JWT.RefreshTokenTTL,
	)
	
	// Create test server
	handler := httpserver.Routes(authService, tokenManager)
	server := httptest.NewServer(handler)
	
	// Cleanup function
	t.Cleanup(func() {
		server.Close()
		dbConn.Close()
	})
	
	return server
}

func cleanupTestData(t *testing.T, db *db.DB) {
	ctx := context.Background()
	queries := []string{
		"DELETE FROM refresh_tokens",
		"DELETE FROM users",
	}
	
	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Logf("Warning: failed to clean up test data: %v", err)
		}
	}
}

func TestAuthenticationFlow(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Test data
	email := "test@example.com"
	password := "password123"
	
	// 1. Test Signup
	t.Run("Signup", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(reqBody)
		
		resp, err := client.Post(
			server.URL+"/api/v1/auth/signup",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			t.Fatalf("Failed to signup: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
		
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		
		if result["user_id"] == nil {
			t.Error("Expected user_id in response")
		}
	})
	
	// 2. Test Login
	var accessToken, refreshToken string
	t.Run("Login", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(reqBody)
		
		resp, err := client.Post(
			server.URL+"/api/v1/auth/login",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			t.Fatalf("Failed to login: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		
		accessToken, _ = result["access_token"].(string)
		refreshToken, _ = result["refresh_token"].(string)
		
		if accessToken == "" || refreshToken == "" {
			t.Error("Expected tokens in response")
		}
	})
	
	// 3. Test Protected Endpoint
	t.Run("Protected Endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to access protected endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		
		if result["email"] != email {
			t.Errorf("Expected email %s, got %v", email, result["email"])
		}
	})
	
	// 4. Test Refresh Token
	t.Run("Refresh Token", func(t *testing.T) {
		reqBody := map[string]string{
			"refresh_token": refreshToken,
		}
		body, _ := json.Marshal(reqBody)
		
		resp, err := client.Post(
			server.URL+"/api/v1/auth/refresh",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			t.Fatalf("Failed to refresh token: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		
		newAccessToken, _ := result["access_token"].(string)
		newRefreshToken, _ := result["refresh_token"].(string)
		
		if newAccessToken == "" || newRefreshToken == "" {
			t.Error("Expected new tokens in response")
		}
		
		// Update tokens for next tests
		accessToken = newAccessToken
		refreshToken = newRefreshToken
	})
	
	// 5. Test Logout
	t.Run("Logout", func(t *testing.T) {
		reqBody := map[string]string{
			"refresh_token": refreshToken,
		}
		body, _ := json.Marshal(reqBody)
		
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to logout: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})
}

func TestInvalidRequests(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	t.Run("Invalid JSON", func(t *testing.T) {
		resp, err := client.Post(
			server.URL+"/api/v1/auth/signup",
			"application/json",
			bytes.NewBufferString(`{"email":`),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
	
	t.Run("Missing Content-Type", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(reqBody)
		
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", bytes.NewBuffer(body))
		// Intentionally not setting Content-Type
		
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
	
	t.Run("Invalid Email Format", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "invalid-email",
			"password": "password123",
		}
		body, _ := json.Marshal(reqBody)
		
		resp, err := client.Post(
			server.URL+"/api/v1/auth/signup",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
	
	t.Run("Weak Password", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "test2@example.com",
			"password": "123",
		}
		body, _ := json.Marshal(reqBody)
		
		resp, err := client.Post(
			server.URL+"/api/v1/auth/signup",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
}