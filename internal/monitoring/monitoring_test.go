package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/metrics"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.MetricsEnabled {
		t.Error("Expected metrics to be enabled by default")
	}
	if config.MetricsPath != "/metrics" {
		t.Errorf("Expected metrics path to be /metrics, got %s", config.MetricsPath)
	}
	if config.MetricsPort != 9090 {
		t.Errorf("Expected metrics port to be 9090, got %d", config.MetricsPort)
	}
	if config.HealthPath != "/health" {
		t.Errorf("Expected health path to be /health, got %s", config.HealthPath)
	}
	if config.ReadyPath != "/ready" {
		t.Errorf("Expected ready path to be /ready, got %s", config.ReadyPath)
	}
	if config.ProfilingEnabled {
		t.Error("Expected profiling to be disabled by default")
	}
	if config.ExportInterval != 10*time.Second {
		t.Errorf("Expected export interval to be 10s, got %v", config.ExportInterval)
	}
	if !config.PrometheusFormat {
		t.Error("Expected Prometheus format to be enabled by default")
	}
}

func TestNewMonitor(t *testing.T) {
	config := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	monitor := NewMonitor(config, logger)

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}
	if monitor.config.MetricsPath != config.MetricsPath {
		t.Error("Expected config to be set correctly")
	}
	if monitor.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	if monitor.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
	if monitor.ready {
		t.Error("Expected ready to be false initially")
	}
}

func TestMonitor_Metrics(t *testing.T) {
	config := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	metrics := monitor.Metrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}
}

func TestMonitor_SetReady(t *testing.T) {
	config := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	// Initially not ready
	if monitor.ready {
		t.Error("Expected monitor to not be ready initially")
	}

	// Set ready
	monitor.SetReady(true)
	if !monitor.ready {
		t.Error("Expected monitor to be ready after SetReady(true)")
	}

	// Set not ready
	monitor.SetReady(false)
	if monitor.ready {
		t.Error("Expected monitor to not be ready after SetReady(false)")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	config := DefaultConfig()
	config.MetricsPort = 9999 // Use a different port to avoid conflicts
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start monitor in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- monitor.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", config.MetricsPort, config.HealthPath))
	if err != nil {
		t.Fatalf("Failed to reach health endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// Stop monitor
	cancel()

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start did not return after context cancellation")
	}
}

func TestMonitor_StartDisabled(t *testing.T) {
	config := DefaultConfig()
	config.MetricsEnabled = false
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when metrics disabled, got %v", err)
	}
}

func TestMonitor_HealthHandler(t *testing.T) {
	config := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	monitor.healthHandler(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}

	// Check response body
	body := rr.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("Handler returned unexpected body: %s", body)
	}
	if !strings.Contains(body, `"timestamp":`) {
		t.Errorf("Handler returned body without timestamp: %s", body)
	}
}

func TestMonitor_ReadyHandler(t *testing.T) {
	config := DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	monitor := NewMonitor(config, logger)

	tests := []struct {
		name           string
		ready          bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "not ready",
			ready:          false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `"status":"not_ready"`,
		},
		{
			name:           "ready",
			ready:          true,
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"ready"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor.SetReady(tt.ready)

			req, err := http.NewRequest("GET", "/ready", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			monitor.readyHandler(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Handler returned unexpected body: %s", body)
			}
		})
	}
}

func TestChecksToJSON(t *testing.T) {
	checks := []ReadinessCheck{
		{
			Name:    "database",
			Healthy: true,
			Message: "Connected",
			Latency: 5 * time.Millisecond,
		},
		{
			Name:    "cache",
			Healthy: false,
			Message: "Connection failed",
			Latency: 100 * time.Millisecond,
		},
	}

	result := checksToJSON(checks)

	// Verify JSON structure
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Error("Expected JSON array format")
	}

	// Verify content
	if !strings.Contains(result, `"name":"database"`) {
		t.Error("Expected database check in JSON")
	}
	if !strings.Contains(result, `"healthy":true`) {
		t.Error("Expected healthy:true in JSON")
	}
	if !strings.Contains(result, `"name":"cache"`) {
		t.Error("Expected cache check in JSON")
	}
	if !strings.Contains(result, `"healthy":false`) {
		t.Error("Expected healthy:false in JSON")
	}
}

func TestGetHostname(t *testing.T) {
	hostname := GetHostname()

	if hostname == "" {
		t.Error("Expected hostname to be non-empty")
	}

	// Verify it matches system hostname (if possible)
	expected, err := os.Hostname()
	if err == nil && hostname != expected {
		t.Errorf("Expected hostname %s, got %s", expected, hostname)
	}
}

func TestNewCollector(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	metrics := metrics.NewMetrics()

	collector := NewCollector(metrics, logger)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}
	if collector.metrics != metrics {
		t.Error("Expected metrics to be set correctly")
	}
	if collector.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

func TestCollector_RecordHTTPRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	m := metrics.NewMetrics()
	collector := NewCollector(m, logger)

	// Record normal request
	collector.RecordHTTPRequest("GET", "/api/users", "200", 100*time.Millisecond, 1024)

	// Verify metrics were recorded
	if v, ok := m.RequestsTotal().Value().(int64); !ok || v != 1 {
		t.Error("Expected request to be counted")
	}

	// Record slow request (should log warning)
	collector.RecordHTTPRequest("POST", "/api/upload", "201", 2*time.Second, 2048)

	if v, ok := m.RequestsTotal().Value().(int64); !ok || v != 2 {
		t.Error("Expected slow request to be counted")
	}
}

func TestCollector_RecordDBQuery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	m := metrics.NewMetrics()
	collector := NewCollector(m, logger)

	// Record normal query
	collector.RecordDBQuery("SELECT", 10*time.Millisecond, nil)

	// Verify metrics were recorded
	if v, ok := m.DBQueriesTotal().Value().(int64); !ok || v != 1 {
		t.Error("Expected query to be counted")
	}

	// Record slow query (should log warning)
	collector.RecordDBQuery("INSERT", 200*time.Millisecond, nil)

	if v, ok := m.DBQueriesTotal().Value().(int64); !ok || v != 2 {
		t.Error("Expected slow query to be counted")
	}

	// Record query with error (should log error)
	err := fmt.Errorf("connection timeout")
	collector.RecordDBQuery("UPDATE", 50*time.Millisecond, err)

	if v, ok := m.DBErrors().Value().(int64); !ok || v != 1 {
		t.Error("Expected error to be counted")
	}
}
