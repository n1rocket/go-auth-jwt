package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abueno/go-auth-jwt/internal/http/handlers"
)

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