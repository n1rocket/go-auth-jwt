package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/abueno/go-auth-jwt/internal/http/response"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	rate    int           // tokens per interval
	burst   int           // max tokens in bucket
	window  time.Duration // time window
	keyFunc KeyFunc       // function to extract key from request
	logger  *slog.Logger
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens    float64
	lastFill  time.Time
	mu        sync.Mutex
}

// KeyFunc extracts a key from the request for rate limiting
type KeyFunc func(r *http.Request) string

// IPKeyFunc returns a key function that uses the client IP
func IPKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		return getClientIP(r)
	}
}

// UserKeyFunc returns a key function that uses the authenticated user ID
func UserKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		userID, ok := r.Context().Value("userID").(string)
		if !ok {
			return ""
		}
		return "user:" + userID
	}
}

// PathKeyFunc returns a key function that combines IP and path
func PathKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		return fmt.Sprintf("%s:%s", getClientIP(r), r.URL.Path)
	}
}

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	Rate     int           // tokens per window
	Burst    int           // max burst size
	Window   time.Duration // time window
	KeyFunc  KeyFunc       // key extraction function
	SkipFunc func(r *http.Request) bool // skip rate limiting for certain requests
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:    100,
		Burst:   10,
		Window:  time.Minute,
		KeyFunc: IPKeyFunc(),
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		rate:    config.Rate,
		burst:   config.Burst,
		window:  config.Window,
		keyFunc: config.KeyFunc,
		logger:  logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// RateLimit returns a middleware that enforces rate limiting
func RateLimit(config RateLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(config, logger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip rate limiting
			if config.SkipFunc != nil && config.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract key
			key := config.KeyFunc(r)
			if key == "" {
				// No key, skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			allowed, remaining, resetTime := limiter.Allow(key)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			if !allowed {
				// Rate limit exceeded
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(resetTime).Seconds())))
				
				response.WriteJSON(w, http.StatusTooManyRequests, map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
					"retry_after": int(time.Until(resetTime).Seconds()),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow(key string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:   float64(rl.burst),
			lastFill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Fill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastFill)
	tokensToAdd := elapsed.Seconds() * float64(rl.rate) / rl.window.Seconds()
	
	bucket.tokens = min(bucket.tokens+tokensToAdd, float64(rl.burst))
	bucket.lastFill = now

	// Check if we have tokens
	if bucket.tokens >= 1 {
		bucket.tokens--
		allowed = true
		remaining = int(bucket.tokens)
	} else {
		allowed = false
		remaining = 0
	}

	// Calculate reset time
	if bucket.tokens < float64(rl.burst) {
		tokensNeeded := float64(rl.burst) - bucket.tokens
		secondsToReset := tokensNeeded * rl.window.Seconds() / float64(rl.rate)
		resetTime = now.Add(time.Duration(secondsToReset) * time.Second)
	} else {
		resetTime = now.Add(rl.window)
	}

	return allowed, remaining, resetTime
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			bucket.mu.Lock()
			// Remove buckets that haven't been used for 2x the window
			if now.Sub(bucket.lastFill) > 2*rl.window {
				delete(rl.buckets, key)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff) - 1; idx > 0 {
			for i := idx; i >= 0; i-- {
				if xff[i] == ',' || xff[i] == ' ' {
					return xff[i+1:]
				}
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Common rate limit configurations
var (
	// AuthEndpointLimiter for authentication endpoints (strict)
	AuthEndpointLimiter = RateLimitConfig{
		Rate:    5,
		Burst:   2,
		Window:  time.Minute,
		KeyFunc: IPKeyFunc(),
	}

	// APIEndpointLimiter for general API endpoints (moderate)
	APIEndpointLimiter = RateLimitConfig{
		Rate:    100,
		Burst:   20,
		Window:  time.Minute,
		KeyFunc: UserKeyFunc(),
	}

	// PublicEndpointLimiter for public endpoints (relaxed)
	PublicEndpointLimiter = RateLimitConfig{
		Rate:    1000,
		Burst:   100,
		Window:  time.Minute,
		KeyFunc: IPKeyFunc(),
	}
)