package middleware

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get request ID from context
		requestID, ok := r.Context().Value("request_id").(string)
		if !ok || requestID == "" {
			t.Error("Expected request ID in context")
		}
		
		// Response should already have X-Request-ID header set by middleware
		w.WriteHeader(http.StatusOK)
	})
	
	requestIDMiddleware := RequestID(handler)
	
	t.Run("generates request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		requestIDMiddleware.ServeHTTP(w, req)
		
		// Check that request ID was set in response header
		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected X-Request-ID header")
		}
		
		// Check format (should be UUID-like)
		if len(requestID) < 20 {
			t.Errorf("Request ID seems too short: %s", requestID)
		}
	})
	
	t.Run("uses existing request ID", func(t *testing.T) {
		existingID := "existing-request-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		w := httptest.NewRecorder()
		
		requestIDMiddleware.ServeHTTP(w, req)
		
		// Should use the existing ID
		requestID := w.Header().Get("X-Request-ID")
		if requestID != existingID {
			t.Errorf("Expected request ID %s, got %s", existingID, requestID)
		}
	})
	
	t.Run("different requests get different IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			
			requestIDMiddleware.ServeHTTP(w, req)
			
			requestID := w.Header().Get("X-Request-ID")
			if ids[requestID] {
				t.Errorf("Duplicate request ID generated: %s", requestID)
			}
			ids[requestID] = true
		}
	})
}

func TestRecover(t *testing.T) {
	tests := []struct {
		name         string
		panicValue   interface{}
		expectLog    bool
		expectStatus int
	}{
		{
			name:         "string panic",
			panicValue:   "something went wrong",
			expectLog:    true,
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "error panic",
			panicValue:   http.ErrAbortHandler,
			expectLog:    false, // ErrAbortHandler should not be logged
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "integer panic",
			panicValue:   42,
			expectLog:    true,
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "nil panic",
			panicValue:   nil,
			expectLog:    true,
			expectStatus: http.StatusInternalServerError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			
			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.panicValue != nil || tt.name == "nil panic" {
					panic(tt.panicValue)
				}
				w.WriteHeader(http.StatusOK)
			})
			
			// Need to temporarily replace default logger to capture output
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)
			
			recoverMiddleware := Recover(panicHandler)
			
			req := httptest.NewRequest("GET", "/test", nil)
			// Add request ID for better logging
			ctx := context.WithValue(req.Context(), "request_id", "test-request-id")
			req = req.WithContext(ctx)
			
			w := httptest.NewRecorder()
			
			// Should not panic
			recoverMiddleware.ServeHTTP(w, req)
			
			// Check status
			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}
			
			// Check log output
			logOutput := buf.String()
			if tt.expectLog {
				if logOutput == "" {
					t.Error("Expected log output for panic")
				}
				
				// Parse log
				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(logOutput), &logEntry); err == nil {
					// Check that panic info is logged
					if _, ok := logEntry["panic"]; !ok {
						t.Error("Expected 'panic' field in log")
					}
					
					if _, ok := logEntry["stack"]; !ok {
						t.Error("Expected 'stack' field in log")
					}
					
					if level, ok := logEntry["level"].(string); ok {
						if level != "ERROR" {
							t.Errorf("Expected ERROR level, got %s", level)
						}
					}
				}
			} else {
				if logOutput != "" && strings.Contains(logOutput, "panic") {
					t.Error("Expected no panic log for ErrAbortHandler")
				}
			}
		})
	}
}

func TestRecover_DefaultLogger(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with default logger")
	})
	
	// Recover uses the handler directly
	recoverMiddleware := Recover(panicHandler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Should not panic and should use default logger
	recoverMiddleware.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecover_AfterWrite(t *testing.T) {
	// Test panic after headers have been written
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write header and some data
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial response"))
		
		// Then panic
		panic("panic after write")
	})
	
	// Replace default logger
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)
	
	recoverMiddleware := Recover(panicHandler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	recoverMiddleware.ServeHTTP(w, req)
	
	// Status should be 200 (already written)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (already written), got %d", w.Code)
	}
	
	// Should have partial response
	body := w.Body.String()
	if !strings.Contains(body, "partial response") {
		t.Error("Expected partial response in body")
	}
	
	// Should still log the panic
	if !strings.Contains(buf.String(), "panic") {
		t.Error("Expected panic to be logged")
	}
}

func TestCORS_Legacy(t *testing.T) {
	// Test the legacy CORS function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	allowedOrigins := []string{"https://example.com", "https://app.example.com"}
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE"}
	allowedHeaders := []string{"Content-Type", "Authorization"}
	
	corsMiddleware := CORS(allowedOrigins, allowedMethods, allowedHeaders)
	wrappedHandler := corsMiddleware(handler)
	
	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(w, req)
		
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
			t.Errorf("Expected CORS origin header, got %s", origin)
		}
		
		if creds := w.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
			t.Error("Expected credentials header")
		}
	})
	
	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(w, req)
		
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
		
		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected Allow-Methods header")
		}
	})
}

func TestSecurity_Legacy(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	securityMiddleware := Security(handler)
	
	t.Run("HTTP request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		securityMiddleware.ServeHTTP(w, req)
		
		// Check basic security headers
		if h := w.Header().Get("X-Content-Type-Options"); h != "nosniff" {
			t.Errorf("Expected X-Content-Type-Options header, got %s", h)
		}
		
		if h := w.Header().Get("X-Frame-Options"); h != "DENY" {
			t.Errorf("Expected X-Frame-Options header, got %s", h)
		}
		
		// Should not have HSTS for HTTP
		if h := w.Header().Get("Strict-Transport-Security"); h != "" {
			t.Error("Should not have HSTS for HTTP request")
		}
	})
	
	t.Run("HTTPS request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		req.TLS = &tls.ConnectionState{}
		w := httptest.NewRecorder()
		
		securityMiddleware.ServeHTTP(w, req)
		
		// Should have HSTS for HTTPS
		if h := w.Header().Get("Strict-Transport-Security"); h == "" {
			t.Error("Expected HSTS header for HTTPS request")
		}
	})
}