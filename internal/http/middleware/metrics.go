package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/abueno/go-auth-jwt/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.size += n
	return n, err
}

// Metrics returns a middleware that collects HTTP metrics
func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track in-flight requests
			m.RequestsInFlight.Inc()
			defer m.RequestsInFlight.Dec()

			// Start timer
			start := time.Now()

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Get route pattern for path label
			path := r.URL.Path
			if r.URL.Path != "" && r.URL.Path[0] == '/' {
				// Normalize path for metrics (remove IDs, etc.)
				path = normalizePath(r.URL.Path)
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start)
			status := strconv.Itoa(rw.statusCode)

			m.RecordHTTPRequest(r.Method, path, status, duration, rw.size)
		})
	}
}

// normalizePath normalizes URL paths for metrics to avoid high cardinality
func normalizePath(path string) string {
	// Common patterns to normalize
	patterns := map[string]string{
		// Auth endpoints
		"/api/v1/auth/verify-email": "/api/v1/auth/verify-email",
		"/api/v1/auth/signup":        "/api/v1/auth/signup",
		"/api/v1/auth/login":         "/api/v1/auth/login",
		"/api/v1/auth/logout":        "/api/v1/auth/logout",
		"/api/v1/auth/refresh":       "/api/v1/auth/refresh",
		"/api/v1/auth/me":            "/api/v1/auth/me",
		
		// Health endpoints
		"/health": "/health",
		"/ready":  "/ready",
		"/metrics": "/metrics",
	}

	// Check exact matches first
	if normalized, ok := patterns[path]; ok {
		return normalized
	}

	// For paths with IDs (UUIDs, numbers), normalize them
	// This is a simple implementation; could be enhanced with regex
	if len(path) > 36 { // Likely contains a UUID
		return "/api/v1/resource/:id"
	}

	return path
}

// MetricsCollector provides methods to record various metrics
type MetricsCollector struct {
	metrics *metrics.Metrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(m *metrics.Metrics) *MetricsCollector {
	return &MetricsCollector{metrics: m}
}

// RecordLogin records login metrics
func (mc *MetricsCollector) RecordLogin(success bool, duration time.Duration) {
	mc.metrics.LoginAttempts.Inc()
	if success {
		mc.metrics.LoginSuccess.Inc()
		mc.metrics.ActiveSessions.Inc()
	} else {
		mc.metrics.LoginFailure.Inc()
	}
}

// RecordSignup records signup metrics
func (mc *MetricsCollector) RecordSignup(success bool, duration time.Duration) {
	mc.metrics.SignupAttempts.Inc()
	if success {
		mc.metrics.SignupSuccess.Inc()
		mc.metrics.UsersTotal.Inc()
	} else {
		mc.metrics.SignupFailure.Inc()
	}
}

// RecordTokenIssued records token issuance
func (mc *MetricsCollector) RecordTokenIssued(tokenType string) {
	labels := map[string]string{"type": tokenType}
	mc.metrics.TokensIssued.WithLabels(labels).Inc()
}

// RecordTokenRefreshed records token refresh
func (mc *MetricsCollector) RecordTokenRefreshed() {
	mc.metrics.TokensRefreshed.Inc()
}

// RecordTokenRevoked records token revocation
func (mc *MetricsCollector) RecordTokenRevoked(reason string) {
	labels := map[string]string{"reason": reason}
	mc.metrics.TokensRevoked.WithLabels(labels).Inc()
	mc.metrics.ActiveSessions.Dec()
}

// RecordEmailQueued records email queuing
func (mc *MetricsCollector) RecordEmailQueued(emailType string) {
	mc.metrics.EmailQueue.Inc()
}

// RecordEmailProcessed records email processing
func (mc *MetricsCollector) RecordEmailProcessed(emailType string, duration time.Duration, err error) {
	mc.metrics.EmailQueue.Dec()
	mc.metrics.RecordEmailSent(emailType, duration, err)
}

// RecordUserVerified records user verification
func (mc *MetricsCollector) RecordUserVerified() {
	mc.metrics.UsersVerified.Inc()
}

// RecordPasswordReset records password reset request
func (mc *MetricsCollector) RecordPasswordReset() {
	mc.metrics.PasswordResets.Inc()
}

// RecordRateLimit records rate limit events
func (mc *MetricsCollector) RecordRateLimit(exceeded bool, endpoint string) {
	labels := map[string]string{"endpoint": endpoint}
	mc.metrics.RateLimitHits.WithLabels(labels).Inc()
	if exceeded {
		mc.metrics.RateLimitExceeded.WithLabels(labels).Inc()
	}
}