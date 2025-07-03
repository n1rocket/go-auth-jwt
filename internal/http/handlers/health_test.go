package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/n1rocket/go-auth-jwt/internal/http/handlers"
)

func TestReady(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	
	handlers.Ready(w, req)
	
	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
	
	// Check response body
	var response handlers.ReadyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if response.Status != "ready" {
		t.Errorf("Expected status 'ready', got %s", response.Status)
	}
	
	if response.Services == nil {
		t.Error("Expected services map to be non-nil")
	}
	
	// Check specific services
	if status, ok := response.Services["database"]; !ok || status != "ok" {
		t.Error("Expected database service to be 'ok'")
	}
	
	if status, ok := response.Services["auth"]; !ok || status != "ok" {
		t.Error("Expected auth service to be 'ok'")
	}
}

func TestHealth_MultipleCallsConsistent(t *testing.T) {
	// Test that multiple calls to Health return consistent results
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		
		handlers.Health(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Call %d: Expected status %d, got %d", i, http.StatusOK, w.Code)
		}
		
		var response handlers.HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Call %d: Failed to unmarshal response: %v", i, err)
		}
		
		if response.Status != "ok" {
			t.Errorf("Call %d: Expected status 'ok', got %s", i, response.Status)
		}
	}
}

func TestReady_ContentNegotiation(t *testing.T) {
	// Test that Ready responds with JSON even if client accepts different types
	tests := []struct {
		name   string
		accept string
	}{
		{
			name:   "accepts json",
			accept: "application/json",
		},
		{
			name:   "accepts any",
			accept: "*/*",
		},
		{
			name:   "accepts html",
			accept: "text/html",
		},
		{
			name:   "no accept header",
			accept: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			w := httptest.NewRecorder()
			
			handlers.Ready(w, req)
			
			// Should always return JSON
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
			
			// Should be valid JSON
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Response is not valid JSON: %v", err)
			}
		})
	}
}