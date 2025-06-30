package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abueno/go-auth-jwt/internal/config"
	"github.com/abueno/go-auth-jwt/internal/http/handlers"
	"github.com/abueno/go-auth-jwt/internal/http/middleware"
	"github.com/abueno/go-auth-jwt/internal/monitoring"
	"github.com/gorilla/mux"
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
	metrics := monitor.Metrics()

	// Create metrics collector for handlers
	collector := middleware.NewMetricsCollector(metrics)

	// Example: Create main application router
	router := mux.NewRouter()

	// Add metrics middleware
	router.Use(middleware.Metrics(metrics))

	// Example auth handler with metrics
	authHandler := &AuthHandlerWithMetrics{
		collector: collector,
		logger:    logger,
	}

	// Setup routes
	router.HandleFunc("/api/v1/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/api/v1/auth/signup", authHandler.Signup).Methods("POST")
	router.HandleFunc("/api/v1/auth/logout", authHandler.Logout).Methods("POST")

	// Health check endpoints (served by main app)
	router.HandleFunc("/health", handlers.Health).Methods("GET")
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		monitor.SetReady(true) // Set ready when app is initialized
		handlers.Ready(w, r)
	}).Methods("GET")

	// Create dashboard if needed
	dashboardConfig := monitoring.DashboardConfig{
		Enabled:         true,
		Path:            "/dashboard",
		RefreshInterval: 5 * time.Second,
		Theme:           "dark",
	}
	dashboard := monitoring.NewDashboard(dashboardConfig, metrics)
	router.PathPrefix("/dashboard").Handler(dashboard.Handler())

	// Create exporter for external systems
	exporterConfig := monitoring.ExporterConfig{
		Stdout:       false,
		File:         "/var/log/metrics.json",
		HTTPPush:     "", // e.g., "http://metrics-collector:8080/push"
		PushGateway:  "", // e.g., "http://pushgateway:9091"
		Interval:     30 * time.Second,
		Format:       "json",
		BatchSize:    100,
		BufferSize:   1000,
		GlobalLabels: map[string]string{
			"service": "jwt-auth",
			"env":     "production",
		},
	}
	exporter := monitoring.NewExporter(exporterConfig, metrics, logger)

	// Start monitoring in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := monitor.Start(ctx); err != nil {
			logger.Error("Failed to start monitoring", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := exporter.Start(ctx); err != nil {
			logger.Error("Failed to start exporter", slog.String("error", err.Error()))
		}
	}()

	// Create main server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		logger.Info("Starting server", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", slog.String("error", err.Error()))
	}

	// Stop monitoring
	cancel()
	if err := monitor.Stop(); err != nil {
		logger.Error("Monitor shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("Server stopped")
}

// AuthHandlerWithMetrics is an example auth handler that records metrics
type AuthHandlerWithMetrics struct {
	collector *middleware.MetricsCollector
	logger    *slog.Logger
}

func (h *AuthHandlerWithMetrics) Login(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Simulate login logic
	success := r.Header.Get("X-Test-Success") == "true"
	
	// Record metrics
	h.collector.RecordLogin(success, time.Since(start))
	
	if success {
		h.collector.RecordTokenIssued("access")
		h.collector.RecordTokenIssued("refresh")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid credentials"}`))
	}
}

func (h *AuthHandlerWithMetrics) Signup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Simulate signup logic
	success := r.Header.Get("X-Test-Success") == "true"
	
	// Record metrics
	h.collector.RecordSignup(success, time.Since(start))
	
	if success {
		h.collector.RecordEmailQueued("verification")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	} else {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"email already exists"}`))
	}
}

func (h *AuthHandlerWithMetrics) Logout(w http.ResponseWriter, r *http.Request) {
	// Record metrics
	h.collector.RecordTokenRevoked("logout")
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"logged out"}`))
}

// Example usage of monitoring in services
type ServiceWithMetrics struct {
	metrics *monitoring.Collector
	logger  *slog.Logger
}

func (s *ServiceWithMetrics) ProcessRequest(ctx context.Context) error {
	// Record DB query
	start := time.Now()
	err := s.simulateDBQuery()
	s.metrics.RecordDBQuery("select_user", time.Since(start), err)
	
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