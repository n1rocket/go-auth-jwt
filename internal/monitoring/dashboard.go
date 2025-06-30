package monitoring

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/abueno/go-auth-jwt/internal/metrics"
)

// DashboardConfig holds dashboard configuration
type DashboardConfig struct {
	Enabled       bool
	Path          string
	RefreshInterval time.Duration
	Theme         string // "light" or "dark"
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
	
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	
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
	Timestamp int64                  `json:"timestamp"`
	System    SystemMetrics          `json:"system"`
	HTTP      HTTPMetrics            `json:"http"`
	Auth      AuthMetrics            `json:"auth"`
	Email     EmailMetrics           `json:"email"`
	Database  DatabaseMetrics        `json:"database"`
	Rates     RateMetrics            `json:"rates"`
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
	ActiveSessions   float64 `json:"active_sessions"`
	LoginSuccess     int64   `json:"login_success"`
	LoginFailure     int64   `json:"login_failure"`
	SignupSuccess    int64   `json:"signup_success"`
	SignupFailure    int64   `json:"signup_failure"`
	TokensIssued     int64   `json:"tokens_issued"`
	TokensRefreshed  int64   `json:"tokens_refreshed"`
	TokensRevoked    int64   `json:"tokens_revoked"`
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
			GoRoutines:      int64(getGauge(d.metrics.GoRoutines)),
			MemoryAllocated: toMB(d.metrics.MemoryAllocated.Value()),
			MemoryTotal:     toMB(d.metrics.MemoryTotal.Value()),
			Uptime:          int64(time.Since(startTime).Seconds()),
		},
		HTTP: HTTPMetrics{
			RequestsTotal:    getCounter(d.metrics.RequestsTotal),
			RequestsInFlight: getGauge(d.metrics.RequestsInFlight),
			AvgResponseTime:  0, // Calculate from histogram
			ErrorRate:        0, // Calculate from counters
		},
		Auth: AuthMetrics{
			ActiveSessions:  getGauge(d.metrics.ActiveSessions),
			LoginSuccess:    getCounter(d.metrics.LoginSuccess),
			LoginFailure:    getCounter(d.metrics.LoginFailure),
			SignupSuccess:   getCounter(d.metrics.SignupSuccess),
			SignupFailure:   getCounter(d.metrics.SignupFailure),
			TokensIssued:    getCounter(d.metrics.TokensIssued),
			TokensRefreshed: getCounter(d.metrics.TokensRefreshed),
			TokensRevoked:   getCounter(d.metrics.TokensRevoked),
		},
		Email: EmailMetrics{
			EmailsSent:   getCounter(d.metrics.EmailsSent),
			EmailsFailed: getCounter(d.metrics.EmailsFailed),
			QueueSize:    getGauge(d.metrics.EmailQueue),
		},
		Database: DatabaseMetrics{
			ActiveConnections: getGauge(d.metrics.DBConnections),
			QueriesTotal:      getCounter(d.metrics.DBQueriesTotal),
			ErrorsTotal:       getCounter(d.metrics.DBErrors),
		},
		Rates: RateMetrics{
			RequestsPerSecond: 0, // Calculate from counters
			LoginsPerMinute:   0, // Calculate from counters
			SignupsPerHour:    0, // Calculate from counters
		},
	}
}

var startTime = time.Now()

// dashboardHTML is the dashboard template
const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        :root {
            --bg-primary: #f5f5f5;
            --bg-secondary: #ffffff;
            --text-primary: #333333;
            --text-secondary: #666666;
            --border-color: #e0e0e0;
            --success-color: #4caf50;
            --error-color: #f44336;
            --warning-color: #ff9800;
        }
        
        [data-theme="dark"] {
            --bg-primary: #1a1a1a;
            --bg-secondary: #2d2d2d;
            --text-primary: #ffffff;
            --text-secondary: #cccccc;
            --border-color: #404040;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background-color: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }
        
        header {
            background-color: var(--bg-secondary);
            padding: 20px;
            border-bottom: 1px solid var(--border-color);
            margin-bottom: 30px;
        }
        
        h1 {
            font-size: 24px;
            font-weight: 600;
        }
        
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .metric-card {
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 20px;
        }
        
        .metric-card h2 {
            font-size: 14px;
            font-weight: 500;
            color: var(--text-secondary);
            margin-bottom: 10px;
            text-transform: uppercase;
        }
        
        .metric-value {
            font-size: 32px;
            font-weight: 600;
            margin-bottom: 5px;
        }
        
        .metric-label {
            font-size: 12px;
            color: var(--text-secondary);
        }
        
        .status-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 5px;
        }
        
        .status-success {
            background-color: var(--success-color);
        }
        
        .status-error {
            background-color: var(--error-color);
        }
        
        .status-warning {
            background-color: var(--warning-color);
        }
        
        .chart-container {
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            height: 300px;
        }
        
        .refresh-indicator {
            position: fixed;
            top: 20px;
            right: 20px;
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 4px;
            padding: 8px 12px;
            font-size: 12px;
        }
        
        @media (max-width: 768px) {
            .metrics-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body data-theme="{{.Theme}}">
    <div class="container">
        <header>
            <h1>{{.Title}}</h1>
        </header>
        
        <div class="metrics-grid" id="metrics-container">
            <!-- Metrics will be populated here -->
        </div>
        
        <div class="chart-container" id="requests-chart">
            <!-- Request rate chart -->
        </div>
        
        <div class="chart-container" id="auth-chart">
            <!-- Authentication metrics chart -->
        </div>
    </div>
    
    <div class="refresh-indicator">
        Next refresh in <span id="refresh-countdown">{{.RefreshInterval}}</span>s
    </div>
    
    <script>
        const metricsEndpoint = '{{.MetricsEndpoint}}';
        const refreshInterval = {{.RefreshInterval}} * 1000;
        let countdownValue = {{.RefreshInterval}};
        
        async function fetchMetrics() {
            try {
                const response = await fetch(metricsEndpoint);
                const data = await response.json();
                updateDashboard(data);
            } catch (error) {
                console.error('Failed to fetch metrics:', error);
            }
        }
        
        function updateDashboard(data) {
            const container = document.getElementById('metrics-container');
            container.innerHTML = '';
            
            // System metrics
            addMetricCard(container, 'Goroutines', data.system.goroutines);
            addMetricCard(container, 'Memory', data.system.memory_allocated_mb.toFixed(2) + ' MB');
            addMetricCard(container, 'Uptime', formatUptime(data.system.uptime_seconds));
            
            // HTTP metrics
            addMetricCard(container, 'Total Requests', data.http.requests_total);
            addMetricCard(container, 'Active Requests', data.http.requests_in_flight);
            
            // Auth metrics
            addMetricCard(container, 'Active Sessions', data.auth.active_sessions);
            addMetricCard(container, 'Successful Logins', data.auth.login_success);
            addMetricCard(container, 'Failed Logins', data.auth.login_failure);
            
            // Email metrics
            addMetricCard(container, 'Emails Sent', data.email.emails_sent);
            addMetricCard(container, 'Email Queue', data.email.queue_size);
            
            // Database metrics
            addMetricCard(container, 'DB Connections', data.database.active_connections);
            addMetricCard(container, 'DB Queries', data.database.queries_total);
        }
        
        function addMetricCard(container, label, value) {
            const card = document.createElement('div');
            card.className = 'metric-card';
            card.innerHTML = ` + "`" + `
                <h2>${label}</h2>
                <div class="metric-value">${value}</div>
            ` + "`" + `;
            container.appendChild(card);
        }
        
        function formatUptime(seconds) {
            const days = Math.floor(seconds / 86400);
            const hours = Math.floor((seconds % 86400) / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            
            if (days > 0) return ` + "`${days}d ${hours}h`" + `;
            if (hours > 0) return ` + "`${hours}h ${minutes}m`" + `;
            return ` + "`${minutes}m`" + `;
        }
        
        // Countdown timer
        setInterval(() => {
            countdownValue--;
            if (countdownValue <= 0) {
                countdownValue = {{.RefreshInterval}};
                fetchMetrics();
            }
            document.getElementById('refresh-countdown').textContent = countdownValue;
        }, 1000);
        
        // Initial fetch
        fetchMetrics();
    </script>
</body>
</html>
`