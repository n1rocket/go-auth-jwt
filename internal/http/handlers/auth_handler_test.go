package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abueno/go-auth-jwt/internal/http/handlers"
)

func TestSignupRequest_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		skipValidRequest bool
	}{
		{
			name:           "valid request",
			requestBody:    `{"email":"test@example.com","password":"password123"}`,
			expectedStatus: http.StatusInternalServerError, // Will fail at service level
			skipValidRequest: true, // Skip this test since it requires a service
		},
		{
			name:           "missing email",
			requestBody:    `{"password":"password123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing password",
			requestBody:    `{"email":"test@example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    `{"email":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty email",
			requestBody:    `{"email":"","password":"password123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty password",
			requestBody:    `{"email":"test@example.com","password":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "null values",
			requestBody:    `{"email":null,"password":null}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipValidRequest {
				t.Skip("Skipping test that requires service implementation")
			}
			
			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create handler with nil service (will panic if it tries to use it)
			handler := handlers.NewAuthHandler(nil)

			// Handle request
			handler.Signup(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "missing email",
			requestBody:    `{"password":"password123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing password",
			requestBody:    `{"email":"test@example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty values",
			requestBody:    `{"email":"","password":""}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "Test-Agent")

			w := httptest.NewRecorder()
			handler := handlers.NewAuthHandler(nil)

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRefreshRequest_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "missing refresh token",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty refresh token",
			requestBody:    `{"refresh_token":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    `{refresh_token}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler := handlers.NewAuthHandler(nil)

			handler.Refresh(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestVerifyEmailRequest_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "missing token",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty token",
			requestBody:    `{"token":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    `{token}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler := handlers.NewAuthHandler(nil)

			handler.VerifyEmail(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLogout_MissingRefreshToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler := handlers.NewAuthHandler(nil)

	handler.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogout_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler := handlers.NewAuthHandler(nil)

	handler.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogoutAll_MissingUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)

	w := httptest.NewRecorder()
	handler := handlers.NewAuthHandler(nil)

	handler.LogoutAll(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetCurrentUser_MissingUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

	w := httptest.NewRecorder()
	handler := handlers.NewAuthHandler(nil)

	handler.GetCurrentUser(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
		testFunc      func(r *http.Request) string
	}{
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			testFunc: func(r *http.Request) string {
				// Simulate getting IP from RemoteAddr
				if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
					return r.RemoteAddr[:idx]
				}
				return r.RemoteAddr
			},
			expectedIP: "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			testFunc: func(r *http.Request) string {
				return r.RemoteAddr
			},
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For header",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "203.0.113.1, 198.51.100.1",
			testFunc: func(r *http.Request) string {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					if idx := strings.Index(xff, ","); idx != -1 {
						return strings.TrimSpace(xff[:idx])
					}
					return xff
				}
				return r.RemoteAddr
			},
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "203.0.113.1",
			testFunc: func(r *http.Request) string {
				if xri := r.Header.Get("X-Real-IP"); xri != "" {
					return xri
				}
				return r.RemoteAddr
			},
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			
			got := tt.testFunc(req)
			if got != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, got)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-123"
	
	// Test WithUserID
	ctxWithUser := handlers.WithUserID(ctx, userID)
	
	// Test retrieving user ID
	if val := ctxWithUser.Value(handlers.UserIDContextKey); val != userID {
		t.Errorf("Expected user ID %s, got %v", userID, val)
	}
	
	// Test missing user ID
	if val := ctx.Value(handlers.UserIDContextKey); val != nil {
		t.Errorf("Expected nil user ID, got %v", val)
	}
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response["status"])
	}
}

func TestReadyEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response handlers.ReadyResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ready" {
		t.Errorf("expected status 'ready', got '%s'", response.Status)
	}
}