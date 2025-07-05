package monitoring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/metrics"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestNewDashboard(t *testing.T) {
	tests := []struct {
		name            string
		config          DashboardConfig
		expectedRefresh time.Duration
		expectedTheme   string
	}{
		{
			name:            "default values",
			config:          DashboardConfig{Enabled: true, Path: "/dashboard"},
			expectedRefresh: 5 * time.Second,
			expectedTheme:   "light",
		},
		{
			name: "custom values",
			config: DashboardConfig{
				Enabled:         true,
				Path:            "/metrics-dashboard",
				RefreshInterval: 10 * time.Second,
				Theme:           "dark",
			},
			expectedRefresh: 10 * time.Second,
			expectedTheme:   "dark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.NewMetrics()
			dashboard := NewDashboard(tt.config, m)

			if dashboard == nil {
				t.Fatal("Expected dashboard to be created")
			}
			if dashboard.config.RefreshInterval != tt.expectedRefresh {
				t.Errorf("Expected refresh interval %v, got %v", tt.expectedRefresh, dashboard.config.RefreshInterval)
			}
			if dashboard.config.Theme != tt.expectedTheme {
				t.Errorf("Expected theme %s, got %s", tt.expectedTheme, dashboard.config.Theme)
			}
			if dashboard.metrics != m {
				t.Error("Expected metrics to be set correctly")
			}
		})
	}
}

func TestDashboard_Handler(t *testing.T) {
	config := DashboardConfig{
		Enabled:         true,
		Path:            "/dashboard",
		RefreshInterval: 5 * time.Second,
		Theme:           "light",
	}
	m := metrics.NewMetrics()
	dashboard := NewDashboard(config, m)

	tests := []struct {
		name       string
		path       string
		wantAPI    bool
		wantStatus int
	}{
		{
			name:       "dashboard HTML",
			path:       "/dashboard",
			wantAPI:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "metrics API",
			path:       "/dashboard/api/metrics",
			wantAPI:    true,
			wantStatus: http.StatusOK,
		},
	}

	handler := dashboard.Handler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}

			if tt.wantAPI {
				// Check API response
				contentType := rr.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected application/json, got %s", contentType)
				}

				var data DashboardData
				if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if data.Timestamp == 0 {
					t.Error("Expected timestamp to be set")
				}
			} else {
				// Check HTML response
				contentType := rr.Header().Get("Content-Type")
				if !strings.HasPrefix(contentType, "text/html") {
					t.Errorf("Expected text/html, got %s", contentType)
				}

				body := rr.Body.String()
				if !strings.Contains(body, "JWT Auth Service - Metrics Dashboard") {
					t.Error("Expected dashboard title in HTML")
				}
				if !strings.Contains(body, `data-theme="light"`) {
					t.Error("Expected theme to be set in HTML")
				}
			}
		})
	}
}

func TestDashboard_ServeDashboard(t *testing.T) {
	config := DashboardConfig{
		Enabled:         true,
		Path:            "/dashboard",
		RefreshInterval: 10 * time.Second,
		Theme:           "dark",
	}
	m := metrics.NewMetrics()
	dashboard := NewDashboard(config, m)

	req, err := http.NewRequest("GET", "/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	dashboard.serveDashboard(rr, req)

	// Check response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body := rr.Body.String()

	// Check that template variables are properly rendered
	if !strings.Contains(body, "JWT Auth Service - Metrics Dashboard") {
		t.Error("Expected title in rendered HTML")
	}
	if !strings.Contains(body, `data-theme="dark"`) {
		t.Error("Expected dark theme in rendered HTML")
	}
	// Check JavaScript variables (with flexible whitespace matching)
	if !strings.Contains(body, "const refreshInterval =") || !strings.Contains(body, "10") || !strings.Contains(body, "* 1000") {
		idx := strings.Index(body, "const refreshInterval")
		if idx >= 0 {
			t.Errorf("Expected refresh interval to be 10 seconds in JavaScript, body contains: %s", body[idx:min(idx+100, len(body))])
		} else {
			t.Error("Expected refresh interval in JavaScript not found")
		}
	}

	// Check metrics endpoint - the template escapes slashes as \/
	if !strings.Contains(body, "\\/dashboard\\/api\\/metrics") && !strings.Contains(body, "/dashboard/api/metrics") {
		idx := strings.Index(body, "const metricsEndpoint")
		if idx >= 0 {
			t.Errorf("Expected metrics endpoint in JavaScript, body contains: %s", body[idx:min(idx+100, len(body))])
		} else {
			t.Error("Expected metrics endpoint in JavaScript not found")
		}
	}
}

func TestDashboard_ServeMetricsAPI(t *testing.T) {
	config := DashboardConfig{
		Enabled: true,
		Path:    "/dashboard",
	}
	m := metrics.NewMetrics()

	// Set some test values
	m.RequestsTotal().Inc()
	m.LoginSuccess().Inc()
	m.EmailsSent().Inc()
	m.DBQueriesTotal().Inc()

	dashboard := NewDashboard(config, m)

	req, err := http.NewRequest("GET", "/dashboard/api/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	dashboard.serveMetricsAPI(rr, req)

	// Check response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var data DashboardData
	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify data structure
	if data.Timestamp == 0 {
		t.Error("Expected timestamp to be set")
	}
	if data.HTTP.RequestsTotal != 1 {
		t.Errorf("Expected requests total to be 1, got %d", data.HTTP.RequestsTotal)
	}
	if data.Auth.LoginSuccess != 1 {
		t.Errorf("Expected login success to be 1, got %d", data.Auth.LoginSuccess)
	}
	if data.Email.EmailsSent != 1 {
		t.Errorf("Expected emails sent to be 1, got %d", data.Email.EmailsSent)
	}
	if data.Database.QueriesTotal != 1 {
		t.Errorf("Expected queries total to be 1, got %d", data.Database.QueriesTotal)
	}
}

func TestDashboard_CollectDashboardData(t *testing.T) {
	config := DashboardConfig{
		Enabled: true,
		Path:    "/dashboard",
	}
	m := metrics.NewMetrics()

	// Set various metric values
	m.GoRoutines().Set(10)
	m.MemoryAllocated().Set(10485760.0) // 10 MB in bytes
	m.RequestsTotal().Add(100)
	m.RequestsInFlight().Set(5.0)
	m.ActiveSessions().Set(25.0)
	m.LoginSuccess().Add(50)
	m.LoginFailure().Add(10)
	m.EmailsSent().Add(200)
	m.EmailQueue().Set(3.0)
	m.DBConnections().Set(10.0)
	m.DBQueriesTotal().Add(1000)

	dashboard := NewDashboard(config, m)
	data := dashboard.collectDashboardData()

	// Verify system metrics
	if data.System.GoRoutines != 10 {
		t.Errorf("Expected 10 goroutines, got %d", data.System.GoRoutines)
	}
	if data.System.MemoryAllocated != 10.0 {
		t.Errorf("Expected 10 MB allocated, got %f", data.System.MemoryAllocated)
	}

	// Verify HTTP metrics
	if data.HTTP.RequestsTotal != 100 {
		t.Errorf("Expected 100 requests total, got %d", data.HTTP.RequestsTotal)
	}
	if data.HTTP.RequestsInFlight != 5.0 {
		t.Errorf("Expected 5 requests in flight, got %f", data.HTTP.RequestsInFlight)
	}

	// Verify auth metrics
	if data.Auth.ActiveSessions != 25.0 {
		t.Errorf("Expected 25 active sessions, got %f", data.Auth.ActiveSessions)
	}
	if data.Auth.LoginSuccess != 50 {
		t.Errorf("Expected 50 successful logins, got %d", data.Auth.LoginSuccess)
	}

	// Verify email metrics
	if data.Email.EmailsSent != 200 {
		t.Errorf("Expected 200 emails sent, got %d", data.Email.EmailsSent)
	}
	if data.Email.QueueSize != 3.0 {
		t.Errorf("Expected queue size 3, got %f", data.Email.QueueSize)
	}

	// Verify database metrics
	if data.Database.ActiveConnections != 10.0 {
		t.Errorf("Expected 10 active connections, got %f", data.Database.ActiveConnections)
	}
	if data.Database.QueriesTotal != 1000 {
		t.Errorf("Expected 1000 queries total, got %d", data.Database.QueriesTotal)
	}
}
