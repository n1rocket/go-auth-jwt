package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/metrics"
)

// DashboardConfig holds dashboard configuration
type DashboardConfig struct {
	Enabled         bool
	Path            string
	RefreshInterval time.Duration
	Theme           string // "light" or "dark"
}

// Dashboard provides a web-based metrics dashboard
type Dashboard struct {
	config  DashboardConfig
	metrics *metrics.Metrics
}

// NewDashboard creates a new dashboard
func NewDashboard(config DashboardConfig, m *metrics.Metrics) *Dashboard {
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 5 * time.Second
	}
	if config.Theme == "" {
		config.Theme = "light"
	}

	return &Dashboard{
		config:  config,
		metrics: m,
	}
}

// Handler returns the dashboard HTTP handler
func (d *Dashboard) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == d.config.Path+"/api/metrics" {
			d.serveMetricsAPI(w, r)
			return
		}

		d.serveDashboard(w, r)
	})
}

// serveDashboard serves the dashboard HTML
func (d *Dashboard) serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := MustLoadTemplate("dashboard.html")

	data := struct {
		Title           string
		RefreshInterval int
		Theme           string
		MetricsEndpoint string
	}{
		Title:           "JWT Auth Service - Metrics Dashboard",
		RefreshInterval: int(d.config.RefreshInterval.Seconds()),
		Theme:           d.config.Theme,
		MetricsEndpoint: d.config.Path + "/api/metrics",
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
	}
}

// serveMetricsAPI serves metrics data as JSON
func (d *Dashboard) serveMetricsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := d.collectDashboardData()

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
	}
}

// DashboardData holds all dashboard metrics
type DashboardData struct {
	Timestamp int64           `json:"timestamp"`
	System    SystemMetrics   `json:"system"`
	HTTP      HTTPMetrics     `json:"http"`
	Auth      AuthMetrics     `json:"auth"`
	Email     EmailMetrics    `json:"email"`
	Database  DatabaseMetrics `json:"database"`
	Rates     RateMetrics     `json:"rates"`
}

// SystemMetrics holds system metrics
type SystemMetrics struct {
	GoRoutines      int64   `json:"goroutines"`
	MemoryAllocated float64 `json:"memory_allocated_mb"`
	MemoryTotal     float64 `json:"memory_total_mb"`
	Uptime          int64   `json:"uptime_seconds"`
}

// HTTPMetrics holds HTTP metrics
type HTTPMetrics struct {
	RequestsTotal    int64   `json:"requests_total"`
	RequestsInFlight float64 `json:"requests_in_flight"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	ErrorRate        float64 `json:"error_rate"`
}

// AuthMetrics holds authentication metrics
type AuthMetrics struct {
	ActiveSessions  float64 `json:"active_sessions"`
	LoginSuccess    int64   `json:"login_success"`
	LoginFailure    int64   `json:"login_failure"`
	SignupSuccess   int64   `json:"signup_success"`
	SignupFailure   int64   `json:"signup_failure"`
	TokensIssued    int64   `json:"tokens_issued"`
	TokensRefreshed int64   `json:"tokens_refreshed"`
	TokensRevoked   int64   `json:"tokens_revoked"`
}

// EmailMetrics holds email metrics
type EmailMetrics struct {
	EmailsSent   int64   `json:"emails_sent"`
	EmailsFailed int64   `json:"emails_failed"`
	QueueSize    float64 `json:"queue_size"`
}

// DatabaseMetrics holds database metrics
type DatabaseMetrics struct {
	ActiveConnections float64 `json:"active_connections"`
	QueriesTotal      int64   `json:"queries_total"`
	ErrorsTotal       int64   `json:"errors_total"`
}

// RateMetrics holds rate metrics
type RateMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	LoginsPerMinute   float64 `json:"logins_per_minute"`
	SignupsPerHour    float64 `json:"signups_per_hour"`
}

// collectDashboardData collects all dashboard metrics
func (d *Dashboard) collectDashboardData() DashboardData {
	// Convert bytes to MB
	toMB := func(bytes interface{}) float64 {
		if v, ok := bytes.(float64); ok {
			return v / (1024 * 1024)
		}
		return 0
	}

	// Get counter values
	getCounter := func(c *metrics.Counter) int64 {
		if v, ok := c.Value().(int64); ok {
			return v
		}
		return 0
	}

	// Get gauge values
	getGauge := func(g *metrics.Gauge) float64 {
		if v, ok := g.Value().(float64); ok {
			return v
		}
		return 0
	}

	return DashboardData{
		Timestamp: time.Now().Unix(),
		System: SystemMetrics{
			GoRoutines:      int64(getGauge(d.metrics.GoRoutines())),
			MemoryAllocated: toMB(d.metrics.MemoryAllocated().Value()),
			MemoryTotal:     toMB(d.metrics.MemoryTotal().Value()),
			Uptime:          int64(time.Since(startTime).Seconds()),
		},
		HTTP: HTTPMetrics{
			RequestsTotal:    getCounter(d.metrics.RequestsTotal()),
			RequestsInFlight: getGauge(d.metrics.RequestsInFlight()),
			AvgResponseTime:  0, // Calculate from histogram
			ErrorRate:        0, // Calculate from counters
		},
		Auth: AuthMetrics{
			ActiveSessions:  getGauge(d.metrics.ActiveSessions()),
			LoginSuccess:    getCounter(d.metrics.LoginSuccess()),
			LoginFailure:    getCounter(d.metrics.LoginFailure()),
			SignupSuccess:   getCounter(d.metrics.SignupSuccess()),
			SignupFailure:   getCounter(d.metrics.SignupFailure()),
			TokensIssued:    getCounter(d.metrics.TokensIssued()),
			TokensRefreshed: getCounter(d.metrics.TokensRefreshed()),
			TokensRevoked:   getCounter(d.metrics.TokensRevoked()),
		},
		Email: EmailMetrics{
			EmailsSent:   getCounter(d.metrics.EmailsSent()),
			EmailsFailed: getCounter(d.metrics.EmailsFailed()),
			QueueSize:    getGauge(d.metrics.EmailQueue()),
		},
		Database: DatabaseMetrics{
			ActiveConnections: getGauge(d.metrics.DBConnections()),
			QueriesTotal:      getCounter(d.metrics.DBQueriesTotal()),
			ErrorsTotal:       getCounter(d.metrics.DBErrors()),
		},
		Rates: RateMetrics{
			RequestsPerSecond: 0, // Calculate from counters
			LoginsPerMinute:   0, // Calculate from counters
			SignupsPerHour:    0, // Calculate from counters
		},
	}
}

var startTime = time.Now()
