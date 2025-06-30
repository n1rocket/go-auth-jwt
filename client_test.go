package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

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
	Name     string `json:"name"`
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
	ExpiresIn    int    `json:"expires_in"`
}

// UserProfile represents the user profile response
type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Signup creates a new user account
func (c *AuthClient) Signup(ctx context.Context, req SignupRequest) error {
	return c.doRequest(ctx, "POST", "/auth/signup", req, nil, nil)
}

// Login authenticates a user and returns tokens
func (c *AuthClient) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/auth/login", req, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RefreshToken refreshes an access token
func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	req := map[string]string{"refresh_token": refreshToken}
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/auth/refresh", req, nil, &resp)
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
	err := c.doRequest(ctx, "GET", "/users/profile", nil, headers, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout logs out the user
func (c *AuthClient) Logout(ctx context.Context, accessToken string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	return c.doRequest(ctx, "POST", "/auth/logout", nil, headers, nil)
}

// VerifyEmail verifies a user's email address
func (c *AuthClient) VerifyEmail(ctx context.Context, token string) error {
	req := map[string]string{"token": token}
	return c.doRequest(ctx, "POST", "/auth/verify-email", req, nil, nil)
}

// ResendVerificationEmail resends the verification email
func (c *AuthClient) ResendVerificationEmail(ctx context.Context, email string) error {
	req := map[string]string{"email": email}
	return c.doRequest(ctx, "POST", "/auth/resend-verification", req, nil, nil)
}

// UpdatePassword updates the user's password
func (c *AuthClient) UpdatePassword(ctx context.Context, accessToken, currentPassword, newPassword string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	req := map[string]string{
		"current_password": currentPassword,
		"new_password":     newPassword,
	}
	return c.doRequest(ctx, "POST", "/users/update-password", req, headers, nil)
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

// Client Integration Tests

func TestClient_CompleteUserJourney(t *testing.T) {
	// Setup test server
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// 1. Sign up a new user
	err := client.Signup(ctx, SignupRequest{
		Email:    "client.test@example.com",
		Password: "SecurePassword123!",
		Name:     "Client Test User",
	})
	require.NoError(t, err)

	// 2. Verify email (in real scenario, user would click link in email)
	// For testing, we'll generate the token directly
	verificationToken, err := ts.authService.GenerateVerificationToken(ctx, "client.test@example.com")
	require.NoError(t, err)

	err = client.VerifyEmail(ctx, verificationToken)
	require.NoError(t, err)

	// 3. Login
	tokenResp, err := client.Login(ctx, LoginRequest{
		Email:    "client.test@example.com",
		Password: "SecurePassword123!",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.AccessToken)
	require.NotEmpty(t, tokenResp.RefreshToken)

	// 4. Get profile
	profile, err := client.GetProfile(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "client.test@example.com", profile.Email)
	assert.Equal(t, "Client Test User", profile.Name)

	// 5. Update password
	err = client.UpdatePassword(ctx, tokenResp.AccessToken, "SecurePassword123!", "NewSecurePassword456!")
	require.NoError(t, err)

	// 6. Login with new password
	newTokenResp, err := client.Login(ctx, LoginRequest{
		Email:    "client.test@example.com",
		Password: "NewSecurePassword456!",
	})
	require.NoError(t, err)
	require.NotEmpty(t, newTokenResp.AccessToken)

	// 7. Refresh token
	refreshedTokenResp, err := client.RefreshToken(ctx, tokenResp.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, refreshedTokenResp.AccessToken)

	// 8. Logout
	err = client.Logout(ctx, refreshedTokenResp.AccessToken)
	require.NoError(t, err)

	// 9. Verify token is invalid after logout
	_, err = client.GetProfile(ctx, refreshedTokenResp.AccessToken)
	assert.Error(t, err)
}

func TestClient_ErrorHandling(t *testing.T) {
	// Setup test server
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
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
		err := client.Signup(ctx, SignupRequest{
			Email:    "invalid-email",
			Password: "SecurePassword123!",
			Name:     "Test User",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("Duplicate registration", func(t *testing.T) {
		// First registration
		err := client.Signup(ctx, SignupRequest{
			Email:    "duplicate.client@example.com",
			Password: "SecurePassword123!",
			Name:     "Test User",
		})
		require.NoError(t, err)

		// Duplicate registration
		err = client.Signup(ctx, SignupRequest{
			Email:    "duplicate.client@example.com",
			Password: "SecurePassword123!",
			Name:     "Test User",
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
	// Setup test server
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// Create and verify a user for testing
	email := "concurrent.test@example.com"
	password := "SecurePassword123!"

	err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: password,
		Name:     "Concurrent Test User",
	})
	require.NoError(t, err)

	// Verify email
	verificationToken, err := ts.authService.GenerateVerificationToken(ctx, email)
	require.NoError(t, err)
	err = client.VerifyEmail(ctx, verificationToken)
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

func TestClient_TokenRefreshFlow(t *testing.T) {
	// Setup test server
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// Create and verify a user
	email := "refresh.test@example.com"
	password := "SecurePassword123!"

	err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: password,
		Name:     "Refresh Test User",
	})
	require.NoError(t, err)

	// Verify email
	verificationToken, err := ts.authService.GenerateVerificationToken(ctx, email)
	require.NoError(t, err)
	err = client.VerifyEmail(ctx, verificationToken)
	require.NoError(t, err)

	// Login
	tokenResp, err := client.Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)

	// Store original tokens
	originalAccessToken := tokenResp.AccessToken
	refreshToken := tokenResp.RefreshToken

	// Refresh token multiple times
	for i := 0; i < 3; i++ {
		newTokenResp, err := client.RefreshToken(ctx, refreshToken)
		require.NoError(t, err)
		require.NotEmpty(t, newTokenResp.AccessToken)
		
		// Verify new token works
		profile, err := client.GetProfile(ctx, newTokenResp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, email, profile.Email)

		// Use the same refresh token (some systems allow this)
		// If your system invalidates refresh tokens after use, update the refreshToken variable:
		// refreshToken = newTokenResp.RefreshToken
	}

	// Original access token should no longer work if enough time has passed
	// or if the system invalidates old tokens on refresh
}

func TestClient_ResendVerificationEmail(t *testing.T) {
	// Setup test server
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// Create unverified user
	email := "resend.test@example.com"
	err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: "SecurePassword123!",
		Name:     "Resend Test User",
	})
	require.NoError(t, err)

	// Resend verification email multiple times
	for i := 0; i < 3; i++ {
		err = client.ResendVerificationEmail(ctx, email)
		assert.NoError(t, err)
		
		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	// Verify that emails were sent
	emailSvc := ts.emailService.(*mockEmailService)
	emails := emailSvc.GetSentEmails(email)
	assert.GreaterOrEqual(t, len(emails), 3) // At least 3 emails (1 original + 3 resends)
}

// Benchmark tests

func BenchmarkClient_Login(b *testing.B) {
	// Setup test server
	ts := SetupTestServer(b)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// Create and verify a user
	email := "benchmark@example.com"
	password := "SecurePassword123!"

	err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: password,
		Name:     "Benchmark User",
	})
	require.NoError(b, err)

	// Verify email
	verificationToken, err := ts.authService.GenerateVerificationToken(ctx, email)
	require.NoError(b, err)
	err = client.VerifyEmail(ctx, verificationToken)
	require.NoError(b, err)

	// Reset timer
	b.ResetTimer()

	// Benchmark login
	for i := 0; i < b.N; i++ {
		_, err := client.Login(ctx, LoginRequest{
			Email:    email,
			Password: password,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClient_GetProfile(b *testing.B) {
	// Setup test server
	ts := SetupTestServer(b)
	defer ts.Cleanup()

	// Create client
	client := NewAuthClient(ts.server.URL)
	ctx := context.Background()

	// Create, verify, and login user
	email := "benchmark.profile@example.com"
	password := "SecurePassword123!"

	err := client.Signup(ctx, SignupRequest{
		Email:    email,
		Password: password,
		Name:     "Benchmark User",
	})
	require.NoError(b, err)

	verificationToken, err := ts.authService.GenerateVerificationToken(ctx, email)
	require.NoError(b, err)
	err = client.VerifyEmail(ctx, verificationToken)
	require.NoError(b, err)

	tokenResp, err := client.Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(b, err)

	// Reset timer
	b.ResetTimer()

	// Benchmark get profile
	for i := 0; i < b.N; i++ {
		_, err := client.GetProfile(ctx, tokenResp.AccessToken)
		if err != nil {
			b.Fatal(err)
		}
	}
}