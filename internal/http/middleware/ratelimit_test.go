package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("allows requests within limit", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    10,
			Burst:   5,
			Window:  time.Second,
			KeyFunc: IPKeyFunc(),
		}

		limiter := NewRateLimiter(config, logger)

		// Should allow burst requests
		for i := 0; i < config.Burst; i++ {
			allowed, remaining, _ := limiter.Allow("test-key")
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
			if remaining != config.Burst-i-1 {
				t.Errorf("Expected %d remaining, got %d", config.Burst-i-1, remaining)
			}
		}

		// Next request should be denied
		allowed, remaining, _ := limiter.Allow("test-key")
		if allowed {
			t.Error("Request should be denied after burst")
		}
		if remaining != 0 {
			t.Errorf("Expected 0 remaining, got %d", remaining)
		}
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    10,
			Burst:   2,
			Window:  time.Second,
			KeyFunc: IPKeyFunc(),
		}

		limiter := NewRateLimiter(config, logger)

		// Use all tokens
		limiter.Allow("test-key")
		limiter.Allow("test-key")

		// Should be denied
		allowed, _, _ := limiter.Allow("test-key")
		if allowed {
			t.Error("Should be denied after using all tokens")
		}

		// Wait for refill
		time.Sleep(200 * time.Millisecond) // Should refill 2 tokens (10 per second * 0.2s)

		// Should be allowed again
		allowed, _, _ = limiter.Allow("test-key")
		if !allowed {
			t.Error("Should be allowed after refill")
		}
	})

	t.Run("different keys have separate buckets", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    10,
			Burst:   1,
			Window:  time.Second,
			KeyFunc: IPKeyFunc(),
		}

		limiter := NewRateLimiter(config, logger)

		// Use token for key1
		allowed1, _, _ := limiter.Allow("key1")
		if !allowed1 {
			t.Error("First request for key1 should be allowed")
		}

		// key2 should still have tokens
		allowed2, _, _ := limiter.Allow("key2")
		if !allowed2 {
			t.Error("First request for key2 should be allowed")
		}

		// key1 should be denied
		allowed1Again, _, _ := limiter.Allow("key1")
		if allowed1Again {
			t.Error("Second request for key1 should be denied")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("sets rate limit headers", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    100,
			Burst:   10,
			Window:  time.Minute,
			KeyFunc: IPKeyFunc(),
		}

		middleware := RateLimit(config, logger)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		// Check headers
		limit := w.Header().Get("X-RateLimit-Limit")
		if limit != "100" {
			t.Errorf("Expected limit header 100, got %s", limit)
		}

		remaining := w.Header().Get("X-RateLimit-Remaining")
		remainingInt, _ := strconv.Atoi(remaining)
		if remainingInt < 0 || remainingInt >= config.Burst {
			t.Errorf("Unexpected remaining value: %d", remainingInt)
		}

		reset := w.Header().Get("X-RateLimit-Reset")
		if reset == "" {
			t.Error("Expected X-RateLimit-Reset header")
		}
	})

	t.Run("returns 429 when rate limit exceeded", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    10,
			Burst:   1,
			Window:  time.Minute,
			KeyFunc: IPKeyFunc(),
		}

		middleware := RateLimit(config, logger)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"

		// First request should succeed
		w1 := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w1, req)
		if w1.Code != http.StatusOK {
			t.Errorf("First request should succeed, got %d", w1.Code)
		}

		// Second request should be rate limited
		w2 := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w2, req)
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("Second request should be rate limited, got %d", w2.Code)
		}

		// Check Retry-After header
		retryAfter := w2.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("Expected Retry-After header")
		}
	})

	t.Run("skip function bypasses rate limiting", func(t *testing.T) {
		config := RateLimitConfig{
			Rate:    1,
			Burst:   1,
			Window:  time.Minute,
			KeyFunc: IPKeyFunc(),
			SkipFunc: func(r *http.Request) bool {
				return r.Header.Get("X-Skip-RateLimit") == "true"
			},
		}

		middleware := RateLimit(config, logger)
		wrappedHandler := middleware(handler)

		// Multiple requests with skip header should all succeed
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			req.Header.Set("X-Skip-RateLimit", "true")

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d should succeed with skip header, got %d", i+1, w.Code)
			}
		}
	})
}

func TestKeyFunctions(t *testing.T) {
	t.Run("IPKeyFunc extracts IP", func(t *testing.T) {
		keyFunc := IPKeyFunc()

		tests := []struct {
			name     string
			headers  map[string]string
			remoteAddr string
			expected string
		}{
			{
				name:       "RemoteAddr only",
				remoteAddr: "192.168.1.1:1234",
				expected:   "192.168.1.1:1234",
			},
			{
				name: "X-Forwarded-For",
				headers: map[string]string{
					"X-Forwarded-For": "10.0.0.1",
				},
				remoteAddr: "192.168.1.1:1234",
				expected:   "10.0.0.1",
			},
			{
				name: "X-Real-IP",
				headers: map[string]string{
					"X-Real-IP": "10.0.0.2",
				},
				remoteAddr: "192.168.1.1:1234",
				expected:   "10.0.0.2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = tt.remoteAddr
				for k, v := range tt.headers {
					req.Header.Set(k, v)
				}

				key := keyFunc(req)
				if key != tt.expected {
					t.Errorf("Expected key %s, got %s", tt.expected, key)
				}
			})
		}
	})

	t.Run("UserKeyFunc extracts user ID", func(t *testing.T) {
		keyFunc := UserKeyFunc()

		// Without user ID
		req := httptest.NewRequest("GET", "/", nil)
		key := keyFunc(req)
		if key != "" {
			t.Errorf("Expected empty key without user ID, got %s", key)
		}

		// With user ID
		ctx := req.Context()
		ctx = WithUserID(ctx, "user123")
		req = req.WithContext(ctx)

		key = keyFunc(req)
		if key != "user:user123" {
			t.Errorf("Expected key 'user:user123', got %s", key)
		}
	})
}

// WithUserID adds user ID to context for testing
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "userID", userID)
}