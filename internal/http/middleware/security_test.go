package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if config.XContentTypeOptions != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options = nosniff, got %s", config.XContentTypeOptions)
	}

	if config.XFrameOptions != "DENY" {
		t.Errorf("Expected X-Frame-Options = DENY, got %s", config.XFrameOptions)
	}

	if config.XSSProtection != "1; mode=block" {
		t.Errorf("Expected X-XSS-Protection = 1; mode=block, got %s", config.XSSProtection)
	}

	if config.ReferrerPolicy != "strict-origin-when-cross-origin" {
		t.Errorf("Expected Referrer-Policy = strict-origin-when-cross-origin, got %s", config.ReferrerPolicy)
	}

	if !config.ForceHTTPS {
		t.Error("Expected ForceHTTPS to be true")
	}
}

func TestStrictSecurityConfig(t *testing.T) {
	config := StrictSecurityConfig()

	if config.StrictTransportSecurity != "max-age=31536000; includeSubDomains" {
		t.Errorf("Unexpected Strict-Transport-Security: %s", config.StrictTransportSecurity)
	}

	if config.ContentSecurityPolicy == "" {
		t.Error("Expected ContentSecurityPolicy to be set")
	}

	if !strings.Contains(config.ContentSecurityPolicy, "default-src 'self'") {
		t.Error("Expected CSP to contain default-src 'self'")
	}

	if config.ReferrerPolicy != "no-referrer" {
		t.Errorf("Expected Referrer-Policy = no-referrer, got %s", config.ReferrerPolicy)
	}

	if !config.ForceHTTPS {
		t.Error("Expected ForceHTTPS to be true")
	}
}

func TestAPISecurityConfig(t *testing.T) {
	config := APISecurityConfig()

	if config.XContentTypeOptions != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options = nosniff, got %s", config.XContentTypeOptions)
	}

	if config.XFrameOptions != "DENY" {
		t.Errorf("Expected X-Frame-Options = DENY, got %s", config.XFrameOptions)
	}

	if config.ContentSecurityPolicy != "" {
		t.Error("Expected no CSP for API config")
	}

	if config.ForceHTTPS {
		t.Error("Expected ForceHTTPS to be false for API")
	}
}

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name            string
		config          SecurityConfig
		scheme          string
		expectedHeaders map[string]string
		notExpected     []string
	}{
		{
			name:   "default security headers",
			config: DefaultSecurityConfig(),
			scheme: "https",
			expectedHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-Xss-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
		},
		{
			name:   "strict security headers",
			config: StrictSecurityConfig(),
			scheme: "https",
			expectedHeaders: map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Referrer-Policy":           "no-referrer",
			},
		},
		{
			name:   "API security headers",
			config: APISecurityConfig(),
			scheme: "https",
			expectedHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
			},
			notExpected: []string{"Content-Security-Policy", "Strict-Transport-Security"},
		},
		{
			name: "custom headers",
			config: SecurityConfig{
				CustomHeaders: map[string]string{
					"X-Custom-Header":  "value1",
					"X-Another-Header": "value2",
				},
			},
			scheme: "https",
			expectedHeaders: map[string]string{
				"X-Custom-Header":  "value1",
				"X-Another-Header": "value2",
			},
		},
		{
			name: "force HTTPS with HTTP request",
			config: SecurityConfig{
				ForceHTTPS: true,
			},
			scheme: "http",
			expectedHeaders: map[string]string{
				"Location": "https://example.com/test",
			},
		},
		{
			name: "empty header values not set",
			config: SecurityConfig{
				XContentTypeOptions: "nosniff",
				XFrameOptions:       "", // Empty, should not be set
				XSSProtection:       "1",
			},
			scheme: "https",
			expectedHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Xss-Protection":       "1",
			},
			notExpected: []string{"X-Frame-Options"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			securityMiddleware := SecurityHeaders(tt.config)
			wrappedHandler := securityMiddleware(handler)

			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			if tt.scheme == "https" {
				req.URL.Scheme = "https"
				req.TLS = &tls.ConnectionState{} // Mock TLS
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Check force HTTPS redirect
			if tt.config.ForceHTTPS && tt.scheme == "http" {
				if w.Code != http.StatusMovedPermanently {
					t.Errorf("Expected status 301 for force HTTPS, got %d", w.Code)
				}
				location := w.Header().Get("Location")
				if !strings.HasPrefix(location, "https://") {
					t.Errorf("Expected HTTPS redirect, got %s", location)
				}
				return
			}

			// Check expected headers
			for header, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(header)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s = %s, got %s", header, expectedValue, actualValue)
				}
			}

			// Check not expected headers
			for _, header := range tt.notExpected {
				if value := w.Header().Get(header); value != "" {
					t.Errorf("Expected header %s to be empty, got %s", header, value)
				}
			}
		})
	}
}

func TestCSPBuilder(t *testing.T) {
	tests := []struct {
		name     string
		build    func() string
		expected []string // Expected directives in CSP
	}{
		{
			name: "basic CSP",
			build: func() string {
				return NewCSPBuilder().
					DefaultSrc(CSPSelf).
					ScriptSrc(CSPSelf, CSPUnsafeInline).
					StyleSrc(CSPSelf).
					Build()
			},
			expected: []string{
				"default-src 'self'",
				"script-src 'self' 'unsafe-inline'",
				"style-src 'self'",
			},
		},
		{
			name: "complex CSP",
			build: func() string {
				return NewCSPBuilder().
					DefaultSrc(CSPNone).
					ScriptSrc(CSPSelf, "https://cdn.example.com").
					StyleSrc(CSPSelf, CSPUnsafeInline).
					ImgSrc(CSPSelf, "data:", "https:").
					ConnectSrc(CSPSelf, "https://api.example.com").
					FontSrc(CSPSelf, "https://fonts.gstatic.com").
					FrameAncestors(CSPNone).
					BaseURI(CSPSelf).
					FormAction(CSPSelf).
					UpgradeInsecureRequests().
					Build()
			},
			expected: []string{
				"default-src 'none'",
				"script-src 'self' https://cdn.example.com",
				"style-src 'self' 'unsafe-inline'",
				"img-src 'self' data: https:",
				"connect-src 'self' https://api.example.com",
				"font-src 'self' https://fonts.gstatic.com",
				"frame-ancestors 'none'",
				"base-uri 'self'",
				"form-action 'self'",
				"upgrade-insecure-requests",
			},
		},
		{
			name: "nonce support",
			build: func() string {
				nonce := "random-nonce-123"
				return NewCSPBuilder().
					DefaultSrc(CSPSelf).
					ScriptSrc(CSPSelf, "'nonce-"+nonce+"'").
					StyleSrc(CSPSelf, "'nonce-"+nonce+"'").
					Build()
			},
			expected: []string{
				"default-src 'self'",
				"script-src 'self' 'nonce-random-nonce-123'",
				"style-src 'self' 'nonce-random-nonce-123'",
			},
		},
		{
			name: "SHA hashes",
			build: func() string {
				return NewCSPBuilder().
					DefaultSrc(CSPSelf).
					ScriptSrc(CSPSelf, "'sha256-abc123'").
					Build()
			},
			expected: []string{
				"default-src 'self'",
				"script-src 'self' 'sha256-abc123'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csp := tt.build()

			// Check that all expected directives are present
			for _, directive := range tt.expected {
				if !strings.Contains(csp, directive) {
					t.Errorf("Expected CSP to contain '%s', got: %s", directive, csp)
				}
			}

			// Check format (directives should be separated by semicolons)
			parts := strings.Split(csp, ";")
			expectedParts := len(tt.expected)
			if len(parts) != expectedParts {
				t.Errorf("Expected %d CSP directives, got %d", expectedParts, len(parts))
			}
		})
	}
}

func TestSecurityHeadersIntegration(t *testing.T) {
	// Test that security headers work correctly with a full handler chain
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a custom header in the handler
		w.Header().Set("X-Custom", "from-handler")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	config := SecurityConfig{
		XContentTypeOptions: "nosniff",
		XFrameOptions:       "SAMEORIGIN",
		CustomHeaders: map[string]string{
			"X-Custom-Middleware": "from-middleware",
		},
	}

	securityMiddleware := SecurityHeaders(config)
	wrappedHandler := securityMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Check that both middleware and handler headers are set
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Security header not set")
	}

	if w.Header().Get("X-Custom-Middleware") != "from-middleware" {
		t.Error("Custom middleware header not set")
	}

	if w.Header().Get("X-Custom") != "from-handler" {
		t.Error("Handler header not set")
	}
}
