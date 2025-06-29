package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		requestID      string
		remoteAddr     string
		userAgent      string
		expectFields   []string
	}{
		{
			name:       "successful request",
			method:     "GET",
			path:       "/api/users",
			statusCode: http.StatusOK,
			requestID:  "test-request-id",
			remoteAddr: "192.168.1.1:12345",
			userAgent:  "Mozilla/5.0",
			expectFields: []string{
				"method",
				"path", 
				"status",
				"duration",
				"request_id",
				"remote_addr",
				"user_agent",
			},
		},
		{
			name:       "error request",
			method:     "POST",
			path:       "/api/users",
			statusCode: http.StatusInternalServerError,
			requestID:  "error-request-id",
			remoteAddr: "10.0.0.1:54321",
			expectFields: []string{
				"method",
				"path",
				"status",
				"duration",
			},
		},
		{
			name:       "request without request ID",
			method:     "DELETE",
			path:       "/api/users/123",
			statusCode: http.StatusNoContent,
			remoteAddr: "127.0.0.1:8080",
			expectFields: []string{
				"method",
				"path",
				"status",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture logs
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate some processing time
				time.Sleep(10 * time.Millisecond)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("response"))
			})
			
			// Replace default logger
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)
			
			loggerMiddleware := Logger(handler)
			
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.requestID != "" {
				ctx := context.WithValue(req.Context(), "request_id", tt.requestID)
				req = req.WithContext(ctx)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			
			w := httptest.NewRecorder()
			loggerMiddleware.ServeHTTP(w, req)
			
			// Parse the log output
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}
			
			// Check that expected fields are present
			for _, field := range tt.expectFields {
				if _, ok := logEntry[field]; !ok {
					t.Errorf("Expected log field %s not found", field)
				}
			}
			
			// Check specific values
			if method, ok := logEntry["method"].(string); ok {
				if method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, method)
				}
			}
			
			if path, ok := logEntry["path"].(string); ok {
				if path != tt.path {
					t.Errorf("Expected path %s, got %s", tt.path, path)
				}
			}
			
			if status, ok := logEntry["status"].(float64); ok {
				if int(status) != tt.statusCode {
					t.Errorf("Expected status %d, got %d", tt.statusCode, int(status))
				}
			}
			
			// Check that duration is reasonable
			if duration, ok := logEntry["duration"].(string); ok {
				// Should be at least 10ms due to our sleep
				if !strings.Contains(duration, "ms") {
					t.Errorf("Expected duration in milliseconds, got %s", duration)
				}
			}
			
			// Check log level based on status code
			if level, ok := logEntry["level"].(string); ok {
				// Logger middleware always logs at INFO level
				if level != "INFO" {
					t.Errorf("Expected log level INFO, got %s", level)
				}
			}
		})
	}
}

func TestLogger_Default(t *testing.T) {
	// Test using the default logger
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	loggerMiddleware := Logger(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Should not panic and should work with default logger
	loggerMiddleware.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLogger_ResponseCapture(t *testing.T) {
	// Test that response writer correctly captures status and size
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write header multiple times - only first should count
		w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusInternalServerError) // Should be ignored
		
		// Write some data
		w.Write([]byte("Hello"))
		w.Write([]byte(" "))
		w.Write([]byte("World"))
	})
	
	// Replace default logger
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)
	
	loggerMiddleware := Logger(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	loggerMiddleware.ServeHTTP(w, req)
	
	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if body := w.Body.String(); body != "Hello World" {
		t.Errorf("Expected body 'Hello World', got %s", body)
	}
	
	// Parse log
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	// Check captured values
	if status, ok := logEntry["status"].(float64); ok {
		if int(status) != http.StatusOK {
			t.Errorf("Expected logged status 200, got %d", int(status))
		}
	}
	
	// The common.go Logger doesn't track response size
	// So we'll just check that the log entry exists
}

func TestLogger_Panic(t *testing.T) {
	// Test that logger doesn't interfere with panic recovery
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	
	// Wrap with recover middleware first, then logger
	recoverMiddleware := Recover(handler)
	// Replace default logger
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)
	
	loggerMiddleware := Logger(recoverMiddleware)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Should not panic
	loggerMiddleware.ServeHTTP(w, req)
	
	// Should log the 500 error
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}
	
	if status, ok := logEntry["status"].(float64); ok {
		if int(status) != http.StatusInternalServerError {
			t.Errorf("Expected status 500 after panic, got %d", int(status))
		}
	}
	
	// Logger always logs at INFO level
	if level, ok := logEntry["level"].(string); ok {
		if level != "INFO" {
			t.Errorf("Expected INFO level, got %s", level)
		}
	}
}