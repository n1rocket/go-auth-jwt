package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/abueno/go-auth-jwt/internal/metrics"
)

// Config holds monitoring configuration
type Config struct {
	// Metrics configuration
	MetricsEnabled bool
	MetricsPath    string
	MetricsPort    int

	// Health check configuration
	HealthPath string
	ReadyPath  string

	// Profiling configuration
	ProfilingEnabled bool
	ProfilingPath    string

	// Export configuration
	ExportInterval   time.Duration
	PrometheusFormat bool
}

// DefaultConfig returns default monitoring configuration
func DefaultConfig() Config {
	return Config{
		MetricsEnabled:   true,
		MetricsPath:      "/metrics",
		MetricsPort:      9090,
		HealthPath:       "/health",
		ReadyPath:        "/ready",
		ProfilingEnabled: false,
		ProfilingPath:    "/debug/pprof",
		ExportInterval:   10 * time.Second,
		PrometheusFormat: true,
	}
}

// Monitor handles application monitoring
type Monitor struct {
	config  Config
	metrics *metrics.Metrics
	logger  *slog.Logger
	server  *http.Server
	ready   bool
}

// NewMonitor creates a new monitor instance
func NewMonitor(config Config, logger *slog.Logger) *Monitor {
	return &Monitor{
		config:  config,
		metrics: metrics.NewMetrics(),
		logger:  logger,
	}
}

// Metrics returns the metrics instance
func (m *Monitor) Metrics() *metrics.Metrics {
	return m.metrics
}

// Start starts the monitoring server
func (m *Monitor) Start(ctx context.Context) error {
	if !m.config.MetricsEnabled {
		m.logger.Info("Metrics disabled")
		return nil
	}

	mux := http.NewServeMux()

	// Metrics endpoint
	if m.config.PrometheusFormat {
		mux.Handle(m.config.MetricsPath, m.metrics.PrometheusHandler())
	} else {
		mux.Handle(m.config.MetricsPath, m.metrics.Handler())
	}

	// Health endpoints
	mux.HandleFunc(m.config.HealthPath, m.healthHandler)
	mux.HandleFunc(m.config.ReadyPath, m.readyHandler)

	// Profiling endpoints (if enabled)
	if m.config.ProfilingEnabled {
		m.setupProfiling(mux)
	}

	// Create server
	m.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", m.config.MetricsPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start metrics collection
	go m.metrics.Start(ctx)

	// Start server
	go func() {
		m.logger.Info("Starting monitoring server", 
			slog.String("address", m.server.Addr),
			slog.String("metrics_path", m.config.MetricsPath),
		)
		
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Monitoring server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return m.Stop()
}

// Stop stops the monitoring server
func (m *Monitor) Stop() error {
	if m.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m.metrics.Stop()
	
	if err := m.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("monitoring server shutdown failed: %w", err)
	}

	m.logger.Info("Monitoring server stopped")
	return nil
}

// SetReady sets the readiness state
func (m *Monitor) SetReady(ready bool) {
	m.ready = ready
}

// healthHandler handles health check requests
func (m *Monitor) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":%d}`, time.Now().Unix())
}

// readyHandler handles readiness check requests
func (m *Monitor) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if !m.ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","timestamp":%d}`, time.Now().Unix())
		return
	}

	// Check various subsystems
	checks := m.performReadinessChecks()
	allHealthy := true
	
	for _, check := range checks {
		if !check.Healthy {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","checks":%s,"timestamp":%d}`, 
			checksToJSON(checks), time.Now().Unix())
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","checks":%s,"timestamp":%d}`, 
			checksToJSON(checks), time.Now().Unix())
	}
}

// ReadinessCheck represents a readiness check result
type ReadinessCheck struct {
	Name    string        `json:"name"`
	Healthy bool          `json:"healthy"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency_ms"`
}

// performReadinessChecks performs all readiness checks
func (m *Monitor) performReadinessChecks() []ReadinessCheck {
	checks := []ReadinessCheck{
		{
			Name:    "metrics",
			Healthy: true,
			Message: "Metrics collection active",
			Latency: 0,
		},
	}

	// Add more checks as needed (database, cache, etc.)
	
	return checks
}

// checksToJSON converts checks to JSON string
func checksToJSON(checks []ReadinessCheck) string {
	// Convert to JSON using json.Marshal
	data, err := json.Marshal(checks)
	if err != nil {
		// Fallback to manual JSON construction
		result := "["
		for i, check := range checks {
			if i > 0 {
				result += ","
			}
			result += fmt.Sprintf(`{"name":"%s","healthy":%t,"message":"%s","latency_ms":%d}`,
				check.Name, check.Healthy, check.Message, check.Latency.Milliseconds())
		}
		result += "]"
		return result
	}
	return string(data)
}

// setupProfiling sets up profiling endpoints
func (m *Monitor) setupProfiling(mux *http.ServeMux) {
	// Import pprof handlers
	mux.HandleFunc(m.config.ProfilingPath+"/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Profiling endpoint", http.StatusOK)
	})
}

// Collector provides a convenient interface for collecting metrics
type Collector struct {
	metrics *metrics.Metrics
	logger  *slog.Logger
}

// NewCollector creates a new metrics collector
func NewCollector(m *metrics.Metrics, logger *slog.Logger) *Collector {
	return &Collector{
		metrics: m,
		logger:  logger,
	}
}

// RecordHTTPRequest records an HTTP request
func (c *Collector) RecordHTTPRequest(method, path, status string, duration time.Duration, size int) {
	c.metrics.RecordHTTPRequest(method, path, status, duration, size)
	
	// Log slow requests
	if duration > 1*time.Second {
		c.logger.Warn("Slow HTTP request",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("status", status),
			slog.Duration("duration", duration),
		)
	}
}

// RecordDBQuery records a database query
func (c *Collector) RecordDBQuery(operation string, duration time.Duration, err error) {
	c.metrics.RecordDBQuery(operation, duration, err)
	
	// Log slow queries
	if duration > 100*time.Millisecond {
		c.logger.Warn("Slow database query",
			slog.String("operation", operation),
			slog.Duration("duration", duration),
		)
	}
	
	// Log errors
	if err != nil {
		c.logger.Error("Database query error",
			slog.String("operation", operation),
			slog.String("error", err.Error()),
		)
	}
}

// GetHostname returns the hostname for metrics labels
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}