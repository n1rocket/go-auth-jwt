package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abueno/go-auth-jwt/internal/metrics"
)

func TestMetricsResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	mw := &metricsResponseWriter{ResponseWriter: rec}

	// Test WriteHeader
	mw.WriteHeader(http.StatusCreated)
	if mw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, mw.statusCode)
	}

	// Test Write
	data := []byte("test response")
	n, err := mw.Write(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	if mw.size != len(data) {
		t.Errorf("Expected size to be %d, got %d", len(data), mw.size)
	}

	// Verify status code is set on first write if not already set
	rec2 := httptest.NewRecorder()
	mw2 := &metricsResponseWriter{ResponseWriter: rec2, statusCode: 0}
	mw2.Write([]byte("test"))
	// When WriteHeader is not called, Write should not set the status code
	// The metricsResponseWriter tracks status code only when WriteHeader is called
	if mw2.statusCode != 0 {
		t.Errorf("Expected status code to remain 0, got %d", mw2.statusCode)
	}
}

func TestMetrics(t *testing.T) {
	metricsInstance := metrics.NewMetrics()

	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
	}{
		{
			name:       "successful GET request",
			method:     http.MethodGet,
			path:       "/api/users/123",
			statusCode: http.StatusOK,
		},
		{
			name:       "POST request with error",
			method:     http.MethodPost,
			path:       "/api/auth/login",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "PUT request",
			method:     http.MethodPut,
			path:       "/api/users/456",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "DELETE request",
			method:     http.MethodDelete,
			path:       "/api/users/789",
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Metrics(metricsInstance)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("response"))
			}))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "auth login",
			path:     "/api/auth/login",
			expected: "/api/auth/login",
		},
		{
			name:     "auth register",
			path:     "/api/auth/register",
			expected: "/api/auth/register",
		},
		{
			name:     "auth logout",
			path:     "/api/auth/logout",
			expected: "/api/auth/logout",
		},
		{
			name:     "auth refresh",
			path:     "/api/auth/refresh",
			expected: "/api/auth/refresh",
		},
		{
			name:     "auth verify email",
			path:     "/api/auth/verify-email",
			expected: "/api/auth/verify-email",
		},
		{
			name:     "user ID path",
			path:     "/api/users/123",
			expected: "/api/users/123",
		},
		{
			name:     "user ID with action",
			path:     "/api/users/456/profile",
			expected: "/api/users/456/profile",
		},
		{
			name:     "UUID in path",
			path:     "/api/resources/550e8400-e29b-41d4-a716-446655440000",
			expected: "/api/v1/resource/:id",
		},
		{
			name:     "multiple IDs",
			path:     "/api/users/123/posts/456",
			expected: "/api/users/123/posts/456",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "other path",
			path:     "/health",
			expected: "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewMetricsCollector(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mc := NewMetricsCollector(metricsInstance)
	if mc == nil {
		t.Error("Expected MetricsCollector to be created")
	}
}

func TestMetricsCollectorMethods(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mc := NewMetricsCollector(metricsInstance)

	// Test RecordLogin
	t.Run("RecordLogin", func(t *testing.T) {
		mc.RecordLogin(true, 100*time.Millisecond)
		mc.RecordLogin(false, 200*time.Millisecond)
	})

	// Test RecordSignup
	t.Run("RecordSignup", func(t *testing.T) {
		mc.RecordSignup(true, 150*time.Millisecond)
		mc.RecordSignup(false, 250*time.Millisecond)
	})

	// Test RecordTokenIssued
	t.Run("RecordTokenIssued", func(t *testing.T) {
		mc.RecordTokenIssued("access")
		mc.RecordTokenIssued("refresh")
	})

	// Test RecordTokenRefreshed
	t.Run("RecordTokenRefreshed", func(t *testing.T) {
		mc.RecordTokenRefreshed()
	})

	// Test RecordTokenRevoked
	t.Run("RecordTokenRevoked", func(t *testing.T) {
		mc.RecordTokenRevoked("user_action")
		mc.RecordTokenRevoked("expiration")
		mc.RecordTokenRevoked("logout")
	})

	// Test RecordEmailQueued
	t.Run("RecordEmailQueued", func(t *testing.T) {
		mc.RecordEmailQueued("verification")
		mc.RecordEmailQueued("password_reset")
	})

	// Test RecordEmailProcessed
	t.Run("RecordEmailProcessed", func(t *testing.T) {
		mc.RecordEmailProcessed("verification", 100*time.Millisecond, nil)
		mc.RecordEmailProcessed("password_reset", 200*time.Millisecond, http.ErrBodyNotAllowed)
	})

	// Test RecordUserVerified
	t.Run("RecordUserVerified", func(t *testing.T) {
		mc.RecordUserVerified()
	})

	// Test RecordPasswordReset
	t.Run("RecordPasswordReset", func(t *testing.T) {
		mc.RecordPasswordReset()
	})

	// Test RecordRateLimit
	t.Run("RecordRateLimit", func(t *testing.T) {
		mc.RecordRateLimit(true, "/api/auth/login")
		mc.RecordRateLimit(false, "/api/users/:id")
	})
}

func TestMetricsMiddlewareIntegration(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	
	// Create a handler that simulates different response times
	handler := Metrics(metricsInstance)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		if strings.Contains(r.URL.Path, "error") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))

	// Make several requests
	paths := []struct {
		path   string
		status int
	}{
		{"/api/users/123", http.StatusOK},
		{"/api/users/456", http.StatusOK},
		{"/api/error", http.StatusInternalServerError},
		{"/api/auth/login", http.StatusOK},
	}

	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		
		if rec.Code != p.status {
			t.Errorf("Path %s: expected status %d, got %d", p.path, p.status, rec.Code)
		}
	}
}