# Monitoring and Metrics Guide

## Overview

The JWT Authentication Service includes comprehensive monitoring and metrics collection capabilities to ensure observability in production environments.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│   Service   │────▶│   Metrics    │────▶│ Prometheus │
│  (Go App)   │     │  Collector   │     │            │
└─────────────┘     └──────────────┘     └────────────┘
                            │                    │
                            ▼                    ▼
                    ┌──────────────┐     ┌────────────┐
                    │  Dashboard   │     │  Grafana   │
                    │   (Built-in) │     │            │
                    └──────────────┘     └────────────┘
```

## Metrics Types

### HTTP Metrics
- `http_requests_total` - Total number of HTTP requests
- `http_request_duration_seconds` - Request latency histogram
- `http_requests_in_flight` - Current number of requests being processed
- `http_response_size_bytes` - Response size histogram

### Authentication Metrics
- `auth_login_attempts_total` - Total login attempts
- `auth_login_success_total` - Successful logins
- `auth_login_failure_total` - Failed logins
- `auth_signup_attempts_total` - Total signup attempts
- `auth_signup_success_total` - Successful signups
- `auth_signup_failure_total` - Failed signups
- `auth_tokens_issued_total` - Tokens issued
- `auth_tokens_refreshed_total` - Tokens refreshed
- `auth_tokens_revoked_total` - Tokens revoked
- `auth_active_sessions` - Current active sessions

### Email Metrics
- `email_sent_total` - Emails sent successfully
- `email_failed_total` - Failed email attempts
- `email_queue_size` - Current email queue size
- `email_send_duration_seconds` - Email send latency

### Database Metrics
- `db_connections_active` - Active database connections
- `db_queries_total` - Total database queries
- `db_query_duration_seconds` - Query execution time
- `db_errors_total` - Database errors

### System Metrics
- `go_goroutines` - Number of goroutines
- `go_memory_allocated_bytes` - Allocated memory
- `go_memory_total_bytes` - Total memory from OS
- `go_gc_pause_seconds` - GC pause durations

### Business Metrics
- `users_total` - Total registered users
- `users_active` - Active users
- `users_verified_total` - Verified users
- `password_resets_total` - Password reset requests
- `verifications_sent_total` - Verification emails sent

### Rate Limiting Metrics
- `rate_limit_hits_total` - Rate limit checks
- `rate_limit_exceeded_total` - Rate limit exceeded events

## Configuration

### Environment Variables

```bash
# Monitoring configuration
METRICS_ENABLED=true
METRICS_PORT=9090
METRICS_PATH=/metrics
PROMETHEUS_FORMAT=true

# Dashboard configuration
DASHBOARD_ENABLED=true
DASHBOARD_PATH=/dashboard
DASHBOARD_THEME=dark

# Export configuration
EXPORT_INTERVAL=30s
EXPORT_STDOUT=false
EXPORT_FILE=/var/log/metrics.json
```

### Code Configuration

```go
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

monitor := monitoring.NewMonitor(monitorConfig, logger)
```

## Integration

### 1. Add Metrics Middleware

```go
import "github.com/abueno/go-auth-jwt/internal/http/middleware"

// Get metrics instance
metrics := monitor.Metrics()

// Add to router
router.Use(middleware.Metrics(metrics))
```

### 2. Record Custom Metrics

```go
// In your handlers
collector := middleware.NewMetricsCollector(metrics)

// Record login
collector.RecordLogin(success, duration)

// Record signup
collector.RecordSignup(success, duration)

// Record token operations
collector.RecordTokenIssued("access")
collector.RecordTokenRefreshed()
collector.RecordTokenRevoked("logout")
```

### 3. Database Metrics

```go
// Wrap database queries
start := time.Now()
err := db.Query(ctx, query)
metrics.RecordDBQuery("select_user", time.Since(start), err)
```

## Deployment

### Docker Compose

```yaml
services:
  api:
    image: jwt-auth:latest
    environment:
      METRICS_ENABLED: "true"
      METRICS_PORT: "9090"
    ports:
      - "8080:8080"
      - "9090:9090"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9091:9090"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
```

### Kubernetes

```yaml
apiVersion: v1
kind: Service
metadata:
  name: jwt-auth-metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
spec:
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
```

## Monitoring Stack Setup

### 1. Start Monitoring Stack

```bash
cd deploy/monitoring
docker-compose -f docker-compose.monitoring.yml up -d
```

### 2. Access Services

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Alertmanager: http://localhost:9093
- Built-in Dashboard: http://localhost:8080/dashboard

### 3. Import Dashboards

1. Login to Grafana
2. Go to Dashboards → Import
3. Upload `jwt-auth-overview.json`

## Alerts

### Critical Alerts

1. **Service Down**
   - Condition: Service unreachable for 1 minute
   - Action: Check service health, restart if needed

2. **High Error Rate**
   - Condition: Error rate > 5% for 5 minutes
   - Action: Check logs, investigate errors

3. **Database Issues**
   - Condition: Connection pool exhausted
   - Action: Check queries, scale database

### Warning Alerts

1. **High Login Failure Rate**
   - Condition: > 50% login failures
   - Action: Check for attacks, review logs

2. **Email Queue Backup**
   - Condition: > 100 emails queued
   - Action: Check SMTP service

3. **High Memory Usage**
   - Condition: > 500MB allocated
   - Action: Profile application, check for leaks

## Best Practices

### 1. Metric Naming

Follow Prometheus naming conventions:
- Use lowercase with underscores
- Include unit suffix (_total, _seconds, _bytes)
- Be descriptive but concise

### 2. Label Usage

```go
labels := map[string]string{
    "method": "POST",
    "endpoint": "/login",
    "status": "success",
}
metrics.LoginAttempts.WithLabels(labels).Inc()
```

### 3. Cardinality Control

Avoid high-cardinality labels:
- ❌ User ID, Session ID, Request ID
- ✅ Status code, Method, Endpoint pattern

### 4. Histogram Buckets

Choose appropriate buckets for your use case:

```go
// For HTTP latencies (in seconds)
buckets := []float64{
    0.001, 0.005, 0.01, 0.025, 0.05,
    0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
}

// For response sizes (in bytes)
buckets := []float64{
    100, 1000, 10000, 100000, 1000000,
}
```

## Performance Impact

The monitoring system is designed for minimal overhead:

- Metrics collection: < 1% CPU overhead
- Memory usage: ~10MB for typical workload
- Network: Configurable export interval

### Optimization Tips

1. **Batch Exports**: Use buffered channels for metric exports
2. **Sampling**: For high-volume metrics, consider sampling
3. **Async Collection**: Non-blocking metric updates
4. **Local Aggregation**: Pre-aggregate before export

## Troubleshooting

### Metrics Not Appearing

1. Check service is running: `curl http://localhost:9090/metrics`
2. Verify Prometheus scrape config
3. Check network connectivity
4. Review service logs

### High Memory Usage

1. Check histogram bucket count
2. Review label cardinality
3. Verify metric retention settings

### Dashboard Not Loading

1. Check Grafana datasource configuration
2. Verify Prometheus is accessible
3. Review dashboard JSON for errors

## Examples

### Custom Business Metrics

```go
// Track feature usage
metrics.FeatureUsage.WithLabels(map[string]string{
    "feature": "two-factor-auth",
    "plan": "premium",
}).Inc()

// Track revenue metrics
metrics.Revenue.Add(amount)

// Track user engagement
metrics.DailyActiveUsers.Set(float64(count))
```

### SLO Tracking

```go
// Track SLI for availability
available := metrics.SuccessfulRequests.Value() / 
             metrics.TotalRequests.Value()

// Track SLI for latency
p95Latency := metrics.RequestDuration.Percentile(95)
```

### Export to External Systems

```go
// Export to CloudWatch
exporter := monitoring.NewExporter(monitoring.ExporterConfig{
    HTTPPush: "https://monitoring.amazonaws.com/",
    Format: "cloudwatch",
    Interval: 60 * time.Second,
})

// Export to DataDog
exporter := monitoring.NewExporter(monitoring.ExporterConfig{
    HTTPPush: "https://api.datadoghq.com/api/v1/series",
    Format: "datadog",
    GlobalLabels: map[string]string{
        "env": "production",
        "service": "jwt-auth",
    },
})
```