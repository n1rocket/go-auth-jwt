// Package jwtauthclient provides a Go client for the JWT Authentication Service
package jwtauthclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client represents a JWT Auth Service client
type Client struct {
	baseURL      string
	apiPath      string
	httpClient   *http.Client
	accessToken  string
	refreshToken string
	tokenExpiry  time.Time
	autoRefresh  bool
	refreshTimer *time.Timer
	mu           sync.RWMutex
}

// Config holds client configuration
type Config struct {
	BaseURL     string
	APIPath     string
	Timeout     time.Duration
	AutoRefresh bool
}

// DefaultConfig returns default client configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:     "http://localhost:8080",
		APIPath:     "/api/v1",
		Timeout:     30 * time.Second,
		AutoRefresh: true,
	}
}

// NewClient creates a new JWT Auth client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}
	if config.APIPath == "" {
		config.APIPath = "/api/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL:     config.BaseURL,
		apiPath:     config.APIPath,
		autoRefresh: config.AutoRefresh,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// UserProfile represents user profile data
type UserProfile struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Signup registers a new user
func (c *Client) Signup(ctx context.Context, email, password string) error {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	_, err := c.request(ctx, "POST", "/auth/signup", payload, false)
	return err
}

// Login authenticates a user and stores tokens
func (c *Client) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.request(ctx, "POST", "/auth/login", payload, false)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Store tokens
	c.setTokens(authResp.AccessToken, authResp.RefreshToken, authResp.ExpiresIn)

	return &authResp, nil
}

// Refresh refreshes the access token
func (c *Client) Refresh(ctx context.Context) (*AuthResponse, error) {
	c.mu.RLock()
	refreshToken := c.refreshToken
	c.mu.RUnlock()

	if refreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	payload := map[string]string{
		"refresh_token": refreshToken,
	}

	resp, err := c.request(ctx, "POST", "/auth/refresh", payload, false)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Update tokens
	c.setTokens(authResp.AccessToken, authResp.RefreshToken, authResp.ExpiresIn)

	return &authResp, nil
}

// Logout revokes the refresh token
func (c *Client) Logout(ctx context.Context) error {
	c.mu.RLock()
	refreshToken := c.refreshToken
	c.mu.RUnlock()

	if refreshToken == "" {
		return nil
	}

	payload := map[string]string{
		"refresh_token": refreshToken,
	}

	_, err := c.request(ctx, "POST", "/auth/logout", payload, true)

	// Clear tokens regardless of error
	c.clearTokens()

	return err
}

// LogoutAll logs out from all devices
func (c *Client) LogoutAll(ctx context.Context) error {
	_, err := c.request(ctx, "POST", "/auth/logout-all", nil, true)
	c.clearTokens()
	return err
}

// GetProfile retrieves the current user's profile
func (c *Client) GetProfile(ctx context.Context) (*UserProfile, error) {
	resp, err := c.request(ctx, "GET", "/auth/me", nil, true)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(resp, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

// VerifyEmail verifies an email address with token
func (c *Client) VerifyEmail(ctx context.Context, email, token string) error {
	payload := map[string]string{
		"email": email,
		"token": token,
	}

	_, err := c.request(ctx, "POST", "/auth/verify-email", payload, false)
	return err
}

// ResendVerification resends the verification email
func (c *Client) ResendVerification(ctx context.Context) error {
	_, err := c.request(ctx, "POST", "/auth/resend-verification", nil, true)
	return err
}

// AuthenticatedRequest makes an authenticated request to any endpoint
func (c *Client) AuthenticatedRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	return c.request(ctx, method, endpoint, payload, true)
}

// IsAuthenticated checks if the client has valid tokens
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken != ""
}

// GetTokens returns the current tokens (for persistence)
func (c *Client) GetTokens() (accessToken, refreshToken string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken, c.refreshToken
}

// SetTokens sets the tokens (for restoration)
func (c *Client) SetTokens(accessToken, refreshToken string, expiresIn int) {
	c.setTokens(accessToken, refreshToken, expiresIn)
}

// request performs an HTTP request
func (c *Client) request(ctx context.Context, method, endpoint string, payload interface{}, authenticated bool) ([]byte, error) {
	url := c.baseURL + c.apiPath + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if authenticated {
		c.mu.RLock()
		token := c.accessToken
		c.mu.RUnlock()

		if token == "" {
			return nil, fmt.Errorf("no access token available")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    errResp.Message,
				Code:       errResp.Code,
				Details:    errResp.Details,
			}
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// setTokens updates the stored tokens
func (c *Client) setTokens(accessToken, refreshToken string, expiresIn int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Cancel existing refresh timer
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
	}

	// Schedule automatic refresh
	if c.autoRefresh && expiresIn > 30 {
		refreshTime := time.Duration(expiresIn-30) * time.Second
		c.refreshTimer = time.AfterFunc(refreshTime, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := c.Refresh(ctx); err != nil {
				// Log error or emit event
				fmt.Printf("Auto-refresh failed: %v\n", err)
			}
		})
	}
}

// clearTokens clears all stored tokens
func (c *Client) clearTokens() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = ""
	c.refreshToken = ""
	c.tokenExpiry = time.Time{}

	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
		c.refreshTimer = nil
	}
}

// Close cleans up client resources
func (c *Client) Close() {
	c.clearTokens()
}

// APIError represents an API error
type APIError struct {
	StatusCode int
	Message    string
	Code       string
	Details    map[string]interface{}
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (code: %s)", e.Message, e.Code)
	}
	return e.Message
}

// WithRetry creates a client with retry capabilities
func WithRetry(client *Client, maxRetries int, backoff time.Duration) *Client {
	// Wrap the HTTP client with retry logic
	transport := &retryTransport{
		base:       http.DefaultTransport,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
	client.httpClient.Transport = transport
	return client
}

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	backoff    time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= t.maxRetries; i++ {
		if i > 0 {
			time.Sleep(t.backoff * time.Duration(i))
		}

		resp, err = t.base.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	return resp, err
}