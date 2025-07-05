package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Metrics holds all application metrics grouped by domain
type Metrics struct {
	// Grouped metrics
	HTTP      *HTTPMetrics
	Auth      *AuthMetrics
	Email     *EmailMetrics
	Database  *DatabaseMetrics
	System    *SystemMetrics
	Business  *BusinessMetrics
	RateLimit *RateLimitMetrics

	// Custom registry
	registry map[string]Metric
	mu       sync.RWMutex

	// Update interval for system metrics
	stopCh chan struct{}
}

// Legacy accessors for backward compatibility - expose as properties
var (
	_ = (*Metrics)(nil) // type check
)

// HTTP metrics getters
func (m *Metrics) RequestsTotal() *Counter      { return m.HTTP.RequestsTotal }
func (m *Metrics) RequestDuration() *Histogram  { return m.HTTP.RequestDuration }
func (m *Metrics) RequestsInFlight() *Gauge     { return m.HTTP.RequestsInFlight }
func (m *Metrics) ResponseSize() *Histogram     { return m.HTTP.ResponseSize }

// Auth metrics getters  
func (m *Metrics) LoginAttempts() *Counter      { return m.Auth.LoginAttempts }
func (m *Metrics) LoginSuccess() *Counter       { return m.Auth.LoginSuccess }
func (m *Metrics) LoginFailure() *Counter       { return m.Auth.LoginFailure }
func (m *Metrics) SignupAttempts() *Counter     { return m.Auth.SignupAttempts }
func (m *Metrics) SignupSuccess() *Counter      { return m.Auth.SignupSuccess }
func (m *Metrics) SignupFailure() *Counter      { return m.Auth.SignupFailure }
func (m *Metrics) TokensIssued() *Counter       { return m.Auth.TokensIssued }
func (m *Metrics) TokensRefreshed() *Counter    { return m.Auth.TokensRefreshed }
func (m *Metrics) TokensRevoked() *Counter      { return m.Auth.TokensRevoked }
func (m *Metrics) ActiveSessions() *Gauge       { return m.Auth.ActiveSessions }

// Email metrics getters
func (m *Metrics) EmailsSent() *Counter         { return m.Email.EmailsSent }
func (m *Metrics) EmailsFailed() *Counter       { return m.Email.EmailsFailed }
func (m *Metrics) EmailQueue() *Gauge           { return m.Email.EmailQueue }
func (m *Metrics) EmailSendLatency() *Histogram { return m.Email.EmailSendLatency }

// Database metrics getters
func (m *Metrics) DBConnections() *Gauge        { return m.Database.DBConnections }
func (m *Metrics) DBQueriesTotal() *Counter     { return m.Database.DBQueriesTotal }
func (m *Metrics) DBQueryDuration() *Histogram  { return m.Database.DBQueryDuration }
func (m *Metrics) DBErrors() *Counter           { return m.Database.DBErrors }

// System metrics getters
func (m *Metrics) GoRoutines() *Gauge           { return m.System.GoRoutines }
func (m *Metrics) MemoryAllocated() *Gauge      { return m.System.MemoryAllocated }
func (m *Metrics) MemoryTotal() *Gauge          { return m.System.MemoryTotal }
func (m *Metrics) GCPauses() *Histogram         { return m.System.GCPauses }

// Business metrics getters
func (m *Metrics) UsersTotal() *Counter         { return m.Business.UsersTotal }
func (m *Metrics) UsersActive() *Gauge          { return m.Business.UsersActive }
func (m *Metrics) UsersVerified() *Counter      { return m.Business.UsersVerified }
func (m *Metrics) PasswordResets() *Counter     { return m.Business.PasswordResets }
func (m *Metrics) VerificationsSent() *Counter  { return m.Business.VerificationsSent }

// Rate limit metrics getters
func (m *Metrics) RateLimitHits() *Counter      { return m.RateLimit.RateLimitHits }
func (m *Metrics) RateLimitExceeded() *Counter  { return m.RateLimit.RateLimitExceeded }

// Metric is the interface for all metric types
type Metric interface {
	Name() string
	Value() interface{}
	String() string
}

// Register implements the MetricRegistry interface
func (m *Metrics) Register(metric Metric) {
	m.mu.Lock()
	m.registry[metric.Name()] = metric
	m.mu.Unlock()
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		HTTP:      NewHTTPMetrics(),
		Auth:      NewAuthMetrics(),
		Email:     NewEmailMetrics(),
		Database:  NewDatabaseMetrics(),
		System:    NewSystemMetrics(),
		Business:  NewBusinessMetrics(),
		RateLimit: NewRateLimitMetrics(),
		registry:  make(map[string]Metric),
		stopCh:    make(chan struct{}),
	}

	// Register all metrics
	m.registerAll()

	// Start system metrics collector
	go m.System.StartCollector(10*time.Second, m.stopCh)

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
			m.System.Update()
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
	// Register all grouped metrics
	m.HTTP.Register(m)
	m.Auth.Register(m)
	m.Email.Register(m)
	m.Database.Register(m)
	m.System.Register(m)
	m.Business.Register(m)
	m.RateLimit.Register(m)
}



// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, size int) {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": status,
	}

	// Increment both base and labeled counters
	m.RequestsTotal().Inc()
	m.RequestsTotal().WithLabels(labels).Inc()
	m.RequestDuration().WithLabels(labels).Observe(duration.Seconds())
	m.ResponseSize().WithLabels(labels).Observe(float64(size))
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation string, duration time.Duration, err error) {
	labels := map[string]string{
		"operation": operation,
	}

	// Increment both base and labeled counters
	m.DBQueriesTotal().Inc()
	m.DBQueriesTotal().WithLabels(labels).Inc()
	m.DBQueryDuration().WithLabels(labels).Observe(duration.Seconds())

	if err != nil {
		m.DBErrors().Inc()
		m.DBErrors().WithLabels(labels).Inc()
	}
}

// RecordEmailSent records email metrics
func (m *Metrics) RecordEmailSent(emailType string, duration time.Duration, err error) {
	labels := map[string]string{
		"type": emailType,
	}

	if err != nil {
		m.EmailsFailed().WithLabels(labels).Inc()
	} else {
		m.EmailsSent().WithLabels(labels).Inc()
		m.EmailSendLatency().WithLabels(labels).Observe(duration.Seconds())
	}
}
