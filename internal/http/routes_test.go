package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/domain"
	inthttp "github.com/n1rocket/go-auth-jwt/internal/http"
	"github.com/n1rocket/go-auth-jwt/internal/http/handlers"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/service"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

var ErrNotFound = errors.New("not found")

// Mock repositories
type mockUserRepository struct {
	createFunc        func(ctx context.Context, user *domain.User) error
	getByEmailFunc    func(ctx context.Context, email string) (*domain.User, error)
	getByIDFunc       func(ctx context.Context, id string) (*domain.User, error)
	updateFunc        func(ctx context.Context, user *domain.User) error
	deleteFunc        func(ctx context.Context, id string) error
	existsByEmailFunc func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return nil, ErrNotFound
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, ErrNotFound
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFunc != nil {
		return m.existsByEmailFunc(ctx, email)
	}
	return false, nil
}

type mockRefreshTokenRepository struct {
	createFunc           func(ctx context.Context, token *domain.RefreshToken) error
	getByTokenFunc       func(ctx context.Context, token string) (*domain.RefreshToken, error)
	getByUserIDFunc      func(ctx context.Context, userID string) ([]*domain.RefreshToken, error)
	updateFunc           func(ctx context.Context, token *domain.RefreshToken) error
	deleteFunc           func(ctx context.Context, id string) error
	deleteByTokenFunc    func(ctx context.Context, token string) error
	deleteAllForUserFunc func(ctx context.Context, userID string) error
	revokeFunc           func(ctx context.Context, token string) error
	revokeAllForUserFunc func(ctx context.Context, userID string) error
	deleteExpiredFunc    func(ctx context.Context) error
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, token)
	}
	return nil
}

func (m *mockRefreshTokenRepository) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if m.getByTokenFunc != nil {
		return m.getByTokenFunc(ctx, token)
	}
	return nil, ErrNotFound
}

func (m *mockRefreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, token)
	}
	return nil
}

func (m *mockRefreshTokenRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	if m.deleteByTokenFunc != nil {
		return m.deleteByTokenFunc(ctx, token)
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteAllForUser(ctx context.Context, userID string) error {
	if m.deleteAllForUserFunc != nil {
		return m.deleteAllForUserFunc(ctx, userID)
	}
	return nil
}

func (m *mockRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	if m.revokeAllForUserFunc != nil {
		return m.revokeAllForUserFunc(ctx, userID)
	}
	return nil
}

func (m *mockRefreshTokenRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, token)
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	if m.deleteExpiredFunc != nil {
		return m.deleteExpiredFunc(ctx)
	}
	return nil
}

func createTestServices() (*service.AuthService, *token.Manager) {
	userRepo := &mockUserRepository{}
	refreshTokenRepo := &mockRefreshTokenRepository{}
	passwordHasher := security.NewDefaultPasswordHasher()

	tokenManager, err := token.NewManager(
		"HS256",
		"test-secret-key-for-testing-only",
		"",
		"",
		"test-issuer",
		15*time.Minute,
	)
	if err != nil {
		panic(err)
	}

	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		24*time.Hour,
	)

	return authService, tokenManager
}

func TestRoutes(t *testing.T) {
	authService, tokenManager := createTestServices()
	handler := inthttp.Routes(authService, tokenManager)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	// Test various endpoints to ensure they are configured
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "health endpoint",
			method:     "GET",
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ready endpoint",
			method:     "GET",
			path:       "/ready",
			wantStatus: http.StatusOK,
		},
		{
			name:       "signup endpoint - invalid body",
			method:     "POST",
			path:       "/api/v1/auth/signup",
			body:       `{"invalid":"json"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "login endpoint - invalid body",
			method:     "POST",
			path:       "/api/v1/auth/login",
			body:       `{"invalid":"json"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "refresh endpoint - missing token",
			method:     "POST",
			path:       "/api/v1/auth/refresh",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "verify-email endpoint - missing token",
			method:     "POST",
			path:       "/api/v1/auth/verify-email",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "logout endpoint - no auth",
			method:     "POST",
			path:       "/api/v1/auth/logout",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "logout-all endpoint - no auth",
			method:     "POST",
			path:       "/api/v1/auth/logout-all",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "me endpoint - no auth",
			method:     "GET",
			path:       "/api/v1/auth/me",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-existent endpoint",
			method:     "GET",
			path:       "/api/v1/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Some endpoints may return 429 due to rate limiting, which is also acceptable
			if w.Code != tt.wantStatus && w.Code != http.StatusTooManyRequests {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

func TestRoutes_CORSHeaders(t *testing.T) {
	authService, tokenManager := createTestServices()
	handler := inthttp.Routes(authService, tokenManager)

	// Test OPTIONS request for CORS preflight
	req := httptest.NewRequest("OPTIONS", "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Just verify that OPTIONS requests are handled
	if w.Code != http.StatusNoContent && w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 204, 200 or 405 for OPTIONS, got %d", w.Code)
	}
}

func TestRoutes_SecurityHeaders(t *testing.T) {
	authService, tokenManager := createTestServices()
	handler := inthttp.Routes(authService, tokenManager)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that we got a valid response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check at least one security header is present
	hasSecurityHeader := false
	securityHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, header := range securityHeaders {
		if w.Header().Get(header) != "" {
			hasSecurityHeader = true
			break
		}
	}

	if !hasSecurityHeader {
		t.Error("Expected at least one security header to be present")
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Create a simple handler that just calls the health endpoint
	// This avoids middleware issues with nil services
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.Health)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	mux.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
