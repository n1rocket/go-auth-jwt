package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/config"
	"github.com/n1rocket/go-auth-jwt/internal/db"
	httpserver "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/repository/postgres"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AuthClient is a test client for the authentication API
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAuthClient creates a new authentication client
func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SignupRequest represents the signup request payload
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignupResponse represents the signup response
type SignupResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse represents the response containing tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserProfile represents the user profile response
type UserProfile struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Signup creates a new user account
func (c *AuthClient) Signup(ctx context.Context, req SignupRequest) (*SignupResponse, error) {
	var resp SignupResponse
	err := c.doRequest(ctx, "POST", "/api/v1/auth/signup", req, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Login authenticates a user and returns tokens
func (c *AuthClient) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/api/v1/auth/login", req, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RefreshToken refreshes an access token
func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	req := map[string]string{"refresh_token": refreshToken}
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/api/v1/auth/refresh", req, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProfile retrieves the user's profile
func (c *AuthClient) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	var resp UserProfile
	err := c.doRequest(ctx, "GET", "/api/v1/auth/me", nil, headers, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout logs out the user
func (c *AuthClient) Logout(ctx context.Context, accessToken, refreshToken string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	req := map[string]string{"refresh_token": refreshToken}
	return c.doRequest(ctx, "POST", "/api/v1/auth/logout", req, headers, nil)
}

// LogoutAll logs out from all devices
func (c *AuthClient) LogoutAll(ctx context.Context, accessToken string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	return c.doRequest(ctx, "POST", "/api/v1/auth/logout-all", nil, headers, nil)
}

// VerifyEmail verifies a user's email address
func (c *AuthClient) VerifyEmail(ctx context.Context, token string) error {
	req := map[string]string{"token": token}
	return c.doRequest(ctx, "POST", "/api/v1/auth/verify-email", req, nil, nil)
}

// doRequest performs an HTTP request
func (c *AuthClient) doRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string, response interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("server error %d: %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// SetupClientTestServer creates a test server for client integration tests
func SetupClientTestServer(t testing.TB) (*httptest.Server, func()) {
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

	return server, cleanup
}

// Client Integration Tests

func TestClient_CompleteUserJourney(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test server
	server, cleanup := SetupClientTestServer(t)
	defer cleanup()

	// Create client
	client := NewAuthClient(server.URL)
	ctx := context.Background()

	// 1. Sign up a new user
	signupResp, err := client.Signup(ctx, SignupRequest{
		Email:    "client.test@example.com",
		Password: "SecurePassword123!",
	})
	require.NoError(t, err)
	require.NotEmpty(t, signupResp.UserID)

	// 2. Login (email verification might be disabled in test)
	tokenResp, err := client.Login(ctx, LoginRequest{
		Email:    "client.test@example.com",
		Password: "SecurePassword123!",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.AccessToken)
	require.NotEmpty(t, tokenResp.RefreshToken)

	// 3. Get profile
	profile, err := client.GetProfile(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "client.test@example.com", profile.Email)

	// 4. Refresh token
	refreshedTokenResp, err := client.RefreshToken(ctx, tokenResp.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, refreshedTokenResp.AccessToken)

	// 5. Logout
	err = client.Logout(ctx, refreshedTokenResp.AccessToken, tokenResp.RefreshToken)
	require.NoError(t, err)

	// 6. Verify token is invalid after logout
	_, err = client.GetProfile(ctx, refreshedTokenResp.AccessToken)
	assert.Error(t, err)
}

func TestClient_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test server
	server, cleanup := SetupClientTestServer(t)
	defer cleanup()

	// Create client
	client := NewAuthClient(server.URL)
	ctx := context.Background()

	t.Run("Invalid credentials", func(t *testing.T) {
		_, err := client.Login(ctx, LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("Invalid email format", func(t *testing.T) {
		_, err := client.Signup(ctx, SignupRequest{
			Email:    "invalid-email",
			Password: "SecurePassword123!",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("Duplicate registration", func(t *testing.T) {
		// First registration
		_, err := client.Signup(ctx, SignupRequest{
			Email:    "duplicate.client@example.com",
			Password: "SecurePassword123!",
		})
		require.NoError(t, err)

		// Duplicate registration
		_, err = client.Signup(ctx, SignupRequest{
			Email:    "duplicate.client@example.com",
			Password: "SecurePassword123!",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "409")
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		_, err := client.GetProfile(ctx, "invalid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func TestClient_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test server
	server, cleanup := SetupClientTestServer(t)
	defer cleanup()

	// Create client
	client := NewAuthClient(server.URL)
	ctx := context.Background()

	// Create and login a user for testing
	email := "concurrent.test@example.com"
	password := "SecurePassword123!"

	_, err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Login to get tokens
	tokenResp, err := client.Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Make concurrent requests
	const numRequests = 20
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			_, err := client.GetProfile(ctx, tokenResp.AccessToken)
			errors <- err
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-errors
		assert.NoError(t, err)
	}
}
