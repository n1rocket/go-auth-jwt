# Monitoring Integration Example

This example demonstrates how to integrate monitoring and metrics collection into your Go application using the JWT auth service's monitoring package.

## Features Demonstrated

- Prometheus metrics endpoint setup
- Health and readiness checks
- HTTP request metrics collection
- Database query metrics
- Email sending metrics
- Custom metrics collectors
- Graceful shutdown

## Running the Example

```bash
# From the examples/monitoring directory
go run main.go

# The example will start:
# - Main server on :8080
# - Metrics server on :9090
```

## Endpoints

- `http://localhost:8080/api/v1/auth/login` - Example login endpoint with metrics
- `http://localhost:8080/api/v1/auth/signup` - Example signup endpoint with metrics
- `http://localhost:8080/api/v1/auth/logout` - Example logout endpoint
- `http://localhost:9090/metrics` - Prometheus metrics
- `http://localhost:9090/health` - Health check
- `http://localhost:9090/ready` - Readiness check

## Key Concepts

### 1. Creating a Monitor

```go
monitor := monitoring.NewMonitor(monitorConfig, logger)
metricsInstance := monitor.Metrics()
```

### 2. Recording Metrics

```go
// HTTP requests
metricsInstance.RecordHTTPRequest(method, path, status, duration, size)

// Database queries
collector.RecordDBQuery("select_user", duration, err)

// Email sending
metricsInstance.RecordEmailSent("welcome", duration, err)
```

### 3. Middleware Integration

The example shows how to create middleware that automatically collects metrics for all HTTP requests.

## Metrics Collected

- HTTP request duration (histogram)
- HTTP request count (counter)
- Database query duration (histogram)
- Database query errors (counter)
- Email send duration (histogram)
- Email send errors (counter)