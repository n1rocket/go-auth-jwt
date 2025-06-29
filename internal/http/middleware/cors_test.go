package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()
	
	if len(config.AllowedOrigins) != 1 || config.AllowedOrigins[0] != "*" {
		t.Errorf("Expected allowed origins [*], got %v", config.AllowedOrigins)
	}
	
	if !config.AllowCredentials {
		t.Error("Expected AllowCredentials to be true")
	}
	
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"}
	if len(config.AllowedMethods) != len(expectedMethods) {
		t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(config.AllowedMethods))
	}
	
	if config.MaxAge != 86400 {
		t.Errorf("Expected MaxAge 86400, got %d", config.MaxAge)
	}
}

func TestStrictCORSConfig(t *testing.T) {
	origins := []string{"https://example.com", "https://app.example.com"}
	config := StrictCORSConfig(origins)
	
	if len(config.AllowedOrigins) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(config.AllowedOrigins))
	}
	
	if !config.AllowCredentials {
		t.Error("Expected AllowCredentials to be true")
	}
	
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	if len(config.AllowedMethods) != len(expectedMethods) {
		t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(config.AllowedMethods))
	}
	
	if config.MaxAge != 3600 {
		t.Errorf("Expected MaxAge 3600, got %d", config.MaxAge)
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name               string
		config             CORSConfig
		requestOrigin      string
		requestMethod      string
		requestHeaders     map[string]string
		expectedStatus     int
		expectedHeaders    map[string]string
		notExpectedHeaders []string
		isPreflight        bool
	}{
		{
			name:          "simple request allowed origin",
			config:        DefaultCORSConfig(),
			requestOrigin: "https://example.com",
			requestMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			name: "simple request specific origin",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowCredentials: true,
			},
			requestOrigin: "https://example.com",
			requestMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "https://example.com",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "simple request disallowed origin",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			requestOrigin:       "https://evil.com",
			requestMethod:       "GET",
			expectedStatus:      http.StatusOK,
			notExpectedHeaders:  []string{"Access-Control-Allow-Origin"},
		},
		{
			name: "preflight request allowed",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST", "PUT"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:         3600,
			},
			requestOrigin: "https://example.com",
			requestMethod: "OPTIONS",
			requestHeaders: map[string]string{
				"Access-Control-Request-Method":  "PUT",
				"Access-Control-Request-Headers": "Content-Type,Authorization",
			},
			isPreflight: true,
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://example.com",
				"Access-Control-Allow-Methods": "GET,POST,PUT",
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Max-Age":       "3600",
			},
		},
		{
			name: "preflight request disallowed method",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			requestOrigin: "https://example.com",
			requestMethod: "OPTIONS",
			requestHeaders: map[string]string{
				"Access-Control-Request-Method": "DELETE",
			},
			isPreflight: true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "preflight request disallowed header",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type"},
			},
			requestOrigin: "https://example.com",
			requestMethod: "OPTIONS",
			requestHeaders: map[string]string{
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "X-Custom-Header",
			},
			isPreflight: true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "wildcard subdomain match",
			config: CORSConfig{
				AllowedOrigins: []string{"https://*.example.com"},
				AllowedMethods: []string{"GET"},
			},
			requestOrigin: "https://app.example.com",
			requestMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://app.example.com",
			},
		},
		{
			name: "wildcard subdomain no match",
			config: CORSConfig{
				AllowedOrigins: []string{"https://*.example.com"},
				AllowedMethods: []string{"GET"},
			},
			requestOrigin:      "https://example.org",
			requestMethod:      "GET",
			expectedStatus:     http.StatusOK,
			notExpectedHeaders: []string{"Access-Control-Allow-Origin"},
		},
		{
			name: "exposed headers",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
				ExposedHeaders: []string{"X-Total-Count", "X-Page-Size"},
			},
			requestOrigin: "https://example.com",
			requestMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Expose-Headers": "X-Total-Count,X-Page-Size",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
			
			corsMiddleware := NewCORS(tt.config)
			wrappedHandler := corsMiddleware(handler)
			
			req := httptest.NewRequest(tt.requestMethod, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}
			
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			
			// Check status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			// Check expected headers
			for header, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(header)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s = %s, got %s", header, expectedValue, actualValue)
				}
			}
			
			// Check not expected headers
			for _, header := range tt.notExpectedHeaders {
				if value := w.Header().Get(header); value != "" {
					t.Errorf("Expected header %s to be empty, got %s", header, value)
				}
			}
			
			// For preflight, check that handler was not called
			if tt.isPreflight && tt.expectedStatus == http.StatusNoContent {
				body := w.Body.String()
				if body != "" {
					t.Errorf("Expected empty body for preflight, got %s", body)
				}
			}
		})
	}
}

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		expected       bool
	}{
		{
			name:           "wildcard allows all",
			allowedOrigins: []string{"*"},
			origin:         "https://example.com",
			expected:       true,
		},
		{
			name:           "exact match",
			allowedOrigins: []string{"https://example.com", "https://app.example.com"},
			origin:         "https://example.com",
			expected:       true,
		},
		{
			name:           "no match",
			allowedOrigins: []string{"https://example.com"},
			origin:         "https://evil.com",
			expected:       false,
		},
		{
			name:           "subdomain wildcard match",
			allowedOrigins: []string{"https://*.example.com"},
			origin:         "https://app.example.com",
			expected:       true,
		},
		{
			name:           "subdomain wildcard no match",
			allowedOrigins: []string{"https://*.example.com"},
			origin:         "https://example.com",
			expected:       false,
		},
		{
			name:           "multiple wildcards",
			allowedOrigins: []string{"https://*.example.com", "https://*.test.com"},
			origin:         "https://api.test.com",
			expected:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to test through the middleware since isAllowedOrigin is private
			config := CORSConfig{
				AllowedOrigins: tt.allowedOrigins,
				AllowedMethods: []string{"GET"},
			}
			
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			
			corsMiddleware := NewCORS(config)
			wrappedHandler := corsMiddleware(handler)
			
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			
			hasOriginHeader := w.Header().Get("Access-Control-Allow-Origin") != ""
			if hasOriginHeader != tt.expected {
				t.Errorf("Expected origin allowed = %v, got %v", tt.expected, hasOriginHeader)
			}
		})
	}
}

func TestSimpleHeaders(t *testing.T) {
	// Test that simple headers are always allowed in preflight
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{}, // No custom headers allowed
	}
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	corsMiddleware := NewCORS(config)
	wrappedHandler := corsMiddleware(handler)
	
	// Test with simple headers
	simpleHeaders := []string{
		"Accept",
		"Accept-Language", 
		"Content-Language",
		"Content-Type",
	}
	
	for _, header := range simpleHeaders {
		t.Run(header, func(t *testing.T) {
			req := httptest.NewRequest("OPTIONS", "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", header)
			
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			
			// Simple headers should be allowed even if not in AllowedHeaders
			if w.Code != http.StatusNoContent {
				t.Errorf("Expected status 204 for simple header %s, got %d", header, w.Code)
			}
		})
	}
}