package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Metrics holds all application metrics
type Metrics struct {
	// HTTP metrics
	RequestsTotal    *Counter
	RequestDuration  *Histogram
	RequestsInFlight *Gauge
	ResponseSize     *Histogram

	// Authentication metrics
	LoginAttempts     *Counter
	LoginSuccess      *Counter
	LoginFailure      *Counter
	SignupAttempts    *Counter
	SignupSuccess     *Counter
	SignupFailure     *Counter
	TokensIssued      *Counter
	TokensRefreshed   *Counter
	TokensRevoked     *Counter
	ActiveSessions    *Gauge

	// Email metrics
	EmailsSent       *Counter
	EmailsFailed     *Counter
	EmailQueue       *Gauge
	EmailSendLatency *Histogram

	// Database metrics
	DBConnections      *Gauge
	DBQueriesTotal     *Counter
	DBQueryDuration    *Histogram
	DBErrors           *Counter

	// System metrics
	GoRoutines      *Gauge
	MemoryAllocated *Gauge
	MemoryTotal     *Gauge
	GCPauses        *Histogram

	// Business metrics
	UsersTotal          *Counter
	UsersActive         *Gauge
	UsersVerified       *Counter
	PasswordResets      *Counter
	VerificationsSent   *Counter

	// Rate limiting metrics
	RateLimitHits       *Counter
	RateLimitExceeded   *Counter

	// Custom registry
	registry map[string]Metric
	mu       sync.RWMutex

	// Update interval for system metrics
	stopCh chan struct{}
}

// Metric is the interface for all metric types
type Metric interface {
	Name() string
	Value() interface{}
	String() string
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		// HTTP metrics
		RequestsTotal:    NewCounter("http_requests_total", "Total number of HTTP requests"),
		RequestDuration:  NewHistogram("http_request_duration_seconds", "HTTP request latencies in seconds"),
		RequestsInFlight: NewGauge("http_requests_in_flight", "Number of HTTP requests currently being processed"),
		ResponseSize:     NewHistogram("http_response_size_bytes", "HTTP response sizes in bytes"),

		// Authentication metrics
		LoginAttempts:   NewCounter("auth_login_attempts_total", "Total number of login attempts"),
		LoginSuccess:    NewCounter("auth_login_success_total", "Total number of successful logins"),
		LoginFailure:    NewCounter("auth_login_failure_total", "Total number of failed logins"),
		SignupAttempts:  NewCounter("auth_signup_attempts_total", "Total number of signup attempts"),
		SignupSuccess:   NewCounter("auth_signup_success_total", "Total number of successful signups"),
		SignupFailure:   NewCounter("auth_signup_failure_total", "Total number of failed signups"),
		TokensIssued:    NewCounter("auth_tokens_issued_total", "Total number of tokens issued"),
		TokensRefreshed: NewCounter("auth_tokens_refreshed_total", "Total number of tokens refreshed"),
		TokensRevoked:   NewCounter("auth_tokens_revoked_total", "Total number of tokens revoked"),
		ActiveSessions:  NewGauge("auth_active_sessions", "Number of active user sessions"),

		// Email metrics
		EmailsSent:       NewCounter("email_sent_total", "Total number of emails sent"),
		EmailsFailed:     NewCounter("email_failed_total", "Total number of failed email attempts"),
		EmailQueue:       NewGauge("email_queue_size", "Number of emails in queue"),
		EmailSendLatency: NewHistogram("email_send_duration_seconds", "Email send latencies in seconds"),

		// Database metrics
		DBConnections:   NewGauge("db_connections_active", "Number of active database connections"),
		DBQueriesTotal:  NewCounter("db_queries_total", "Total number of database queries"),
		DBQueryDuration: NewHistogram("db_query_duration_seconds", "Database query latencies in seconds"),
		DBErrors:        NewCounter("db_errors_total", "Total number of database errors"),

		// System metrics
		GoRoutines:      NewGauge("go_goroutines", "Number of goroutines"),
		MemoryAllocated: NewGauge("go_memory_allocated_bytes", "Allocated memory in bytes"),
		MemoryTotal:     NewGauge("go_memory_total_bytes", "Total memory obtained from OS"),
		GCPauses:        NewHistogram("go_gc_pause_seconds", "GC pause durations in seconds"),

		// Business metrics
		UsersTotal:        NewCounter("users_total", "Total number of registered users"),
		UsersActive:       NewGauge("users_active", "Number of active users"),
		UsersVerified:     NewCounter("users_verified_total", "Total number of verified users"),
		PasswordResets:    NewCounter("password_resets_total", "Total number of password reset requests"),
		VerificationsSent: NewCounter("verifications_sent_total", "Total number of verification emails sent"),

		// Rate limiting metrics
		RateLimitHits:     NewCounter("rate_limit_hits_total", "Total number of rate limit checks"),
		RateLimitExceeded: NewCounter("rate_limit_exceeded_total", "Total number of rate limit exceeded events"),

		registry: make(map[string]Metric),
		stopCh:   make(chan struct{}),
	}

	// Register all metrics
	m.registerAll()

	// Start system metrics collector
	go m.collectSystemMetrics()

	return m
}

// Start starts the metrics collection
func (m *Metrics) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.updateSystemMetrics()
		}
	}
}

// Stop stops the metrics collection
func (m *Metrics) Stop() {
	select {
	case <-m.stopCh:
		// Already closed
	default:
		close(m.stopCh)
	}
}

// Handler returns an HTTP handler for metrics endpoint
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		metrics := make(map[string]interface{})
		
		m.mu.RLock()
		for name, metric := range m.registry {
			metrics[name] = metric.Value()
		}
		m.mu.RUnlock()

		// Add current timestamp
		metrics["timestamp"] = time.Now().Unix()
		
		// Encode metrics as JSON
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// PrometheusHandler returns a Prometheus-compatible metrics handler
func (m *Metrics) PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, metric := range m.registry {
			switch v := metric.(type) {
			case *Counter:
				fmt.Fprintf(w, "# HELP %s %s\n", v.name, v.help)
				fmt.Fprintf(w, "# TYPE %s counter\n", v.name)
				fmt.Fprintf(w, "%s %d\n", v.name, v.Value())
			case *Gauge:
				fmt.Fprintf(w, "# HELP %s %s\n", v.name, v.help)
				fmt.Fprintf(w, "# TYPE %s gauge\n", v.name)
				fmt.Fprintf(w, "%s %f\n", v.name, v.Value())
			case *Histogram:
				fmt.Fprintf(w, "# HELP %s %s\n", v.name, v.help)
				fmt.Fprintf(w, "# TYPE %s histogram\n", v.name)
				
				buckets := v.Buckets()
				sum := v.Sum()
				count := v.Count()
				
				for bound, count := range buckets {
					fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", v.name, bound, count)
				}
				fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", v.name, count)
				fmt.Fprintf(w, "%s_sum %f\n", v.name, sum)
				fmt.Fprintf(w, "%s_count %d\n", v.name, count)
			}
			fmt.Fprintln(w)
		}
	})
}

// registerAll registers all metrics
func (m *Metrics) registerAll() {
	// HTTP metrics
	m.register(m.RequestsTotal)
	m.register(m.RequestDuration)
	m.register(m.RequestsInFlight)
	m.register(m.ResponseSize)

	// Auth metrics
	m.register(m.LoginAttempts)
	m.register(m.LoginSuccess)
	m.register(m.LoginFailure)
	m.register(m.SignupAttempts)
	m.register(m.SignupSuccess)
	m.register(m.SignupFailure)
	m.register(m.TokensIssued)
	m.register(m.TokensRefreshed)
	m.register(m.TokensRevoked)
	m.register(m.ActiveSessions)

	// Email metrics
	m.register(m.EmailsSent)
	m.register(m.EmailsFailed)
	m.register(m.EmailQueue)
	m.register(m.EmailSendLatency)

	// Database metrics
	m.register(m.DBConnections)
	m.register(m.DBQueriesTotal)
	m.register(m.DBQueryDuration)
	m.register(m.DBErrors)

	// System metrics
	m.register(m.GoRoutines)
	m.register(m.MemoryAllocated)
	m.register(m.MemoryTotal)
	m.register(m.GCPauses)

	// Business metrics
	m.register(m.UsersTotal)
	m.register(m.UsersActive)
	m.register(m.UsersVerified)
	m.register(m.PasswordResets)
	m.register(m.VerificationsSent)

	// Rate limiting metrics
	m.register(m.RateLimitHits)
	m.register(m.RateLimitExceeded)
}

// register adds a metric to the registry
func (m *Metrics) register(metric Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry[metric.Name()] = metric
}

// collectSystemMetrics collects system metrics periodically
func (m *Metrics) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.updateSystemMetrics()
		}
	}
}

// updateSystemMetrics updates system metrics
func (m *Metrics) updateSystemMetrics() {
	// Goroutines
	m.GoRoutines.Set(float64(runtime.NumGoroutine()))

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.MemoryAllocated.Set(float64(memStats.Alloc))
	m.MemoryTotal.Set(float64(memStats.Sys))

	// GC pause times (convert to seconds)
	if len(memStats.PauseNs) > 0 {
		lastPause := float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e9
		m.GCPauses.Observe(lastPause)
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, size int) {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": status,
	}
	
	// Increment both base and labeled counters
	m.RequestsTotal.Inc()
	m.RequestsTotal.WithLabels(labels).Inc()
	m.RequestDuration.WithLabels(labels).Observe(duration.Seconds())
	m.ResponseSize.WithLabels(labels).Observe(float64(size))
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation string, duration time.Duration, err error) {
	labels := map[string]string{
		"operation": operation,
	}
	
	// Increment both base and labeled counters
	m.DBQueriesTotal.Inc()
	m.DBQueriesTotal.WithLabels(labels).Inc()
	m.DBQueryDuration.WithLabels(labels).Observe(duration.Seconds())
	
	if err != nil {
		m.DBErrors.Inc()
		m.DBErrors.WithLabels(labels).Inc()
	}
}

// RecordEmailSent records email metrics
func (m *Metrics) RecordEmailSent(emailType string, duration time.Duration, err error) {
	labels := map[string]string{
		"type": emailType,
	}
	
	if err != nil {
		m.EmailsFailed.WithLabels(labels).Inc()
	} else {
		m.EmailsSent.WithLabels(labels).Inc()
		m.EmailSendLatency.WithLabels(labels).Observe(duration.Seconds())
	}
}