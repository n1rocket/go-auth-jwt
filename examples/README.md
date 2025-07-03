# Examples

This directory contains various examples demonstrating how to use and integrate with the Go Auth JWT service. Each example is in its own subdirectory to avoid package conflicts.

## Directory Structure

```
examples/
├── clients/                  # Client implementations
│   ├── go/                  # Go client library
│   │   ├── client.go        # Reusable client package
│   │   └── example/         # Example usage
│   │       └── main.go
│   └── cli/                 # Command-line interface
│       └── jwt-auth-cli.go
├── monitoring/              # Monitoring integration example
│   ├── main.go
│   └── README.md
└── advanced/                # Advanced features example
    ├── main.go
    └── README.md
```

## Quick Start

Each example can be run independently:

```bash
# Go client example
cd examples/clients/go/example
go run main.go

# CLI client
cd examples/clients/cli
go run jwt-auth-cli.go

# Monitoring integration
cd examples/monitoring
go run main.go

# Advanced features
cd examples/advanced
go run main.go
```

## Examples Overview

### 1. Client Libraries (`clients/`)

#### Go Client (`clients/go/`)
- **client.go**: A reusable client library (package `jwtauthclient`)
- **example/main.go**: Demonstrates how to use the client library

#### CLI Client (`clients/cli/`)
- **jwt-auth-cli.go**: Interactive command-line client with:
  - User-friendly prompts
  - Token persistence
  - All auth operations

### 2. Monitoring Integration (`monitoring/`)

Demonstrates metrics and monitoring:
- Prometheus metrics endpoint
- Health/readiness checks
- Request/response metrics
- Database query metrics
- Custom metric collectors

### 3. Advanced Features (`advanced/`)

Shows production-ready patterns:
- Email service with SMTP
- Worker pools for async tasks
- Rate limiting strategies
- CORS configuration
- Security headers

## Prerequisites

1. **Running Auth Service**:
   ```bash
   # Start the main service
   docker compose up -d
   ```

2. **Environment Variables**:
   ```bash
   export AUTH_SERVICE_URL=http://localhost:8080
   export SMTP_HOST=localhost
   export SMTP_PORT=1025
   ```

3. **Optional Services**:
   ```bash
   # MailHog for email testing
   docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog
   ```

## Common Patterns

All examples follow these best practices:

1. **Error Handling**: Comprehensive error checking and logging
2. **Context Usage**: Proper context propagation for cancellation
3. **Structured Logging**: Using `slog` for consistent log output
4. **Graceful Shutdown**: Clean resource cleanup on exit
5. **Configuration**: Environment-based configuration

## Creating Your Own Integration

To integrate with the auth service:

1. **Use the Go Client Library**:
   ```go
   import "github.com/n1rocket/go-auth-jwt/examples/clients/go"
   
   client := jwtauthclient.NewClient(baseURL)
   ```

2. **Implement the REST API**:
   - `POST /api/v1/auth/signup`
   - `POST /api/v1/auth/login`
   - `POST /api/v1/auth/refresh`
   - `POST /api/v1/auth/logout`
   - `GET /api/v1/auth/me`

3. **Add Monitoring** (optional):
   - See `monitoring/` example for metrics integration
   - Use provided middleware for automatic collection

## Testing

Each example includes error scenarios and demonstrates proper error handling:

```bash
# Test with invalid credentials
AUTH_SERVICE_URL=http://localhost:9999 go run main.go

# Test with network issues
# (Stop the auth service and run examples)
```

## Contributing

When adding new examples:
1. Create a new subdirectory
2. Add a `main.go` file
3. Include a `README.md` with usage instructions
4. Follow the existing patterns for consistency