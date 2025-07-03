package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/http/middleware"
	"github.com/n1rocket/go-auth-jwt/internal/metrics"
	"github.com/n1rocket/go-auth-jwt/internal/monitoring"
)

// Example of how to integrate monitoring into the JWT auth service
func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create monitoring configuration
	monitorConfig := monitoring.Config{
		MetricsEnabled:   true,
		MetricsPath:      "/metrics",
		MetricsPort:      9090,
		HealthPath:       "/health",
		ReadyPath:        "/ready",
		ProfilingEnabled: false,
		ExportInterval:   10 * time.Second,
		PrometheusFormat: true,
	}

	// Create monitor
	monitor := monitoring.NewMonitor(monitorConfig, logger)

	// Get metrics instance
	metricsInstance := monitor.Metrics()

	// Create metrics collector for logging
	collector := monitoring.NewCollector(metricsInstance, logger)

	// Example: Create main application router using standard library
	mux := http.NewServeMux()

	// Add monitoring middleware
	handler := monitoringMiddleware(mux, collector)

	// Create rate limit middleware
	rateLimitConfig := middleware.RateLimitConfig{
		Rate:    100,
		Burst:   10,
		Window:  time.Minute,
		KeyFunc: middleware.IPKeyFunc(),
	}
	rateLimitMiddleware := middleware.RateLimit(rateLimitConfig, logger)

	// Create auth handlers
	authHandler := &AuthHandlerWithMetrics{
		metrics: metricsInstance,
		logger:  logger,
	}

	// Example routes with monitoring
	// Apply rate limiting to specific endpoints
	mux.Handle("/api/v1/auth/login", rateLimitMiddleware(
		http.HandlerFunc(authHandler.Login)))

	mux.Handle("/api/v1/auth/signup", rateLimitMiddleware(
		http.HandlerFunc(authHandler.Signup)))

	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// Create a service with metrics
	service := &ServiceWithMetrics{
		metrics:   metricsInstance,
		collector: collector,
		logger:    logger,
	}

	// Example: Process a request with metrics
	ctx := context.Background()
	if err := service.ProcessRequest(ctx); err != nil {
		logger.Error("Request processing failed", slog.String("error", err.Error()))
	}

	// Start monitoring server
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	go func() {
		if err := monitor.Start(monitorCtx); err != nil {
			logger.Error("Failed to start monitoring", slog.String("error", err.Error()))
		}
	}()

	// Set the service as ready
	monitor.SetReady(true)

	// Create main server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start main server
	go func() {
		logger.Info("Starting main server", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("Server started",
		slog.String("main_address", srv.Addr),
		slog.Int("metrics_port", monitorConfig.MetricsPort),
		slog.String("metrics_path", monitorConfig.MetricsPath),
	)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")

	// Shutdown contexts
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop main server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", slog.String("error", err.Error()))
	}

	// Stop monitoring
	monitorCancel()

	logger.Info("Server stopped")
}

// monitoringMiddleware adds metrics collection to HTTP requests
func monitoringMiddleware(next http.Handler, collector *monitoring.Collector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapper, r)

		// Record metrics
		collector.RecordHTTPRequest(
			r.Method,
			r.URL.Path,
			http.StatusText(wrapper.statusCode),
			time.Since(start),
			wrapper.size,
		)
	})
}

// AuthHandlerWithMetrics demonstrates how to add metrics to handlers
type AuthHandlerWithMetrics struct {
	metrics *metrics.Metrics
	logger  *slog.Logger
}

func (h *AuthHandlerWithMetrics) Login(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Simulate login processing
	time.Sleep(50 * time.Millisecond)

	// Record metrics using the actual API
	h.metrics.RecordHTTPRequest("POST", "/api/v1/auth/login", "200", time.Since(start), 100)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func (h *AuthHandlerWithMetrics) Signup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Simulate signup processing
	time.Sleep(100 * time.Millisecond)

	// Record metrics
	h.metrics.RecordHTTPRequest("POST", "/api/v1/auth/signup", "201", time.Since(start), 150)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"created"}`))
}

func (h *AuthHandlerWithMetrics) Logout(w http.ResponseWriter, r *http.Request) {
	// Record logout
	h.metrics.RecordHTTPRequest("POST", "/api/v1/auth/logout", "200", 5*time.Millisecond, 50)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"logged_out"}`))
}

// ServiceWithMetrics shows how to integrate metrics into a service
type ServiceWithMetrics struct {
	metrics   *metrics.Metrics
	collector *monitoring.Collector
	logger    *slog.Logger
}

func (s *ServiceWithMetrics) ProcessRequest(ctx context.Context) error {
	// Record DB query
	start := time.Now()
	err := s.simulateDBQuery()
	s.collector.RecordDBQuery("select_user", time.Since(start), err)

	if err != nil {
		return err
	}

	// Record email sending
	start = time.Now()
	err = s.simulateEmailSend()
	s.metrics.RecordEmailSent("welcome", time.Since(start), err)

	return err
}

func (s *ServiceWithMetrics) simulateDBQuery() error {
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (s *ServiceWithMetrics) simulateEmailSend() error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

// responseWriter wraps http.ResponseWriter to capture response details
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
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}