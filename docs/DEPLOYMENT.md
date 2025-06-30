# Deployment Guide

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Variables](#environment-variables)
3. [Database Setup](#database-setup)
4. [Docker Deployment](#docker-deployment)
5. [Production Checklist](#production-checklist)
6. [Monitoring](#monitoring)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- Docker (optional)
- SMTP server for email notifications

## Environment Variables

Create a `.env` file or set these environment variables:

```bash
# Server Configuration
PORT=8080
ENVIRONMENT=production

# Database Configuration
DATABASE_DSN=postgres://username:password@localhost:5432/authdb?sslmode=require
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m
DATABASE_CONN_MAX_IDLE_TIME=1m

# JWT Configuration
JWT_SECRET=your-super-secret-key-minimum-32-characters-long
JWT_ISSUER=your-app-name
JWT_ACCESS_TOKEN_DURATION=15m
JWT_ALGORITHM=HS256

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-specific-password
SMTP_FROM_ADDRESS=noreply@yourdomain.com
SMTP_FROM_NAME=Your App Name

# Security Configuration
CORS_ALLOWED_ORIGINS=https://app.yourdomain.com,https://www.yourdomain.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=86400

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=10

# Worker Pool
WORKER_POOL_SIZE=10
WORKER_QUEUE_SIZE=100

# Application URLs
APP_BASE_URL=https://api.yourdomain.com
FRONTEND_URL=https://app.yourdomain.com
```

## Database Setup

### 1. Create Database

```sql
-- Create database
CREATE DATABASE authdb;

-- Create user
CREATE USER authuser WITH ENCRYPTED PASSWORD 'strongpassword';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE authdb TO authuser;
```

### 2. Run Migrations

Create migration files in `migrations/` directory:

```sql
-- migrations/001_create_users_table.up.sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_expires TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_email_verification_token ON users(email_verification_token);

-- migrations/002_create_refresh_tokens_table.up.sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP,
    user_agent TEXT,
    ip_address VARCHAR(45)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

Run migrations:

```bash
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./migrations -database "$DATABASE_DSN" up
```

## Docker Deployment

### 1. Create Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
```

### 2. Create docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: authuser
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: authdb
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U authuser -d authdb"]
      interval: 10s
      timeout: 5s
      retries: 5

  migrate:
    image: migrate/migrate
    volumes:
      - ./migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://authuser:${DB_PASSWORD}@postgres:5432/authdb?sslmode=disable",
      "up"
    ]
    depends_on:
      postgres:
        condition: service_healthy

  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      PORT: 8080
      ENVIRONMENT: production
      DATABASE_DSN: postgres://authuser:${DB_PASSWORD}@postgres:5432/authdb?sslmode=disable
      JWT_SECRET: ${JWT_SECRET}
      JWT_ISSUER: ${JWT_ISSUER}
      SMTP_HOST: ${SMTP_HOST}
      SMTP_PORT: ${SMTP_PORT}
      SMTP_USERNAME: ${SMTP_USERNAME}
      SMTP_PASSWORD: ${SMTP_PASSWORD}
      SMTP_FROM_ADDRESS: ${SMTP_FROM_ADDRESS}
      SMTP_FROM_NAME: ${SMTP_FROM_NAME}
      CORS_ALLOWED_ORIGINS: ${CORS_ALLOWED_ORIGINS}
      APP_BASE_URL: ${APP_BASE_URL}
      FRONTEND_URL: ${FRONTEND_URL}
    depends_on:
      migrate:
        condition: service_completed_successfully
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
```

### 3. Deploy with Docker Compose

```bash
# Create .env file with your configuration
cp .env.example .env
# Edit .env with your values

# Build and start services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down
```

## Production Checklist

### Security

- [ ] Use strong JWT secret (minimum 32 characters)
- [ ] Enable HTTPS/TLS
- [ ] Configure CORS properly (no wildcards in production)
- [ ] Enable rate limiting
- [ ] Use secure database connection (sslmode=require)
- [ ] Rotate JWT secrets periodically
- [ ] Implement proper logging (no sensitive data)
- [ ] Enable security headers
- [ ] Use environment variables for secrets
- [ ] Implement request timeout
- [ ] Enable CSRF protection if using cookies

### Performance

- [ ] Configure database connection pool
- [ ] Enable database query optimization
- [ ] Implement caching strategy
- [ ] Configure worker pool size based on load
- [ ] Enable gzip compression
- [ ] Implement graceful shutdown
- [ ] Configure appropriate timeouts

### Monitoring

- [ ] Set up health check endpoints
- [ ] Implement structured logging
- [ ] Configure error tracking (Sentry, etc.)
- [ ] Set up metrics collection
- [ ] Monitor database performance
- [ ] Track API response times
- [ ] Monitor rate limit hits
- [ ] Set up alerts for failures

### Backup & Recovery

- [ ] Regular database backups
- [ ] Test restore procedures
- [ ] Document recovery process
- [ ] Implement data retention policy

## Production Configuration Examples

### Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

### Systemd Service

```ini
[Unit]
Description=JWT Auth API
After=network.target postgresql.service

[Service]
Type=simple
User=authapi
Group=authapi
WorkingDirectory=/opt/authapi
ExecStart=/opt/authapi/main
Restart=always
RestartSec=5
StandardOutput=append:/var/log/authapi/api.log
StandardError=append:/var/log/authapi/error.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/authapi

# Environment
EnvironmentFile=/opt/authapi/.env

[Install]
WantedBy=multi-user.target
```

## Monitoring

### Prometheus Metrics

Add Prometheus metrics to your application:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "http_duration_seconds",
        Help: "Duration of HTTP requests.",
    }, []string{"path", "method", "status"})
    
    totalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total number of HTTP requests.",
    }, []string{"path", "method", "status"})
)

func init() {
    prometheus.MustRegister(httpDuration)
    prometheus.MustRegister(totalRequests)
}

// Add to your router
router.Handle("/metrics", promhttp.Handler())
```

### Logging Configuration

```go
// Structured logging with slog
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// Log with context
logger.Info("user logged in",
    slog.String("user_id", userID),
    slog.String("ip", clientIP),
    slog.Time("timestamp", time.Now()),
)
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   ```bash
   # Check PostgreSQL is running
   sudo systemctl status postgresql
   
   # Check connection
   psql -h localhost -U authuser -d authdb
   
   # Check logs
   tail -f /var/log/postgresql/postgresql-*.log
   ```

2. **JWT Token Issues**
   ```bash
   # Decode JWT token
   echo "your.jwt.token" | cut -d. -f2 | base64 -d | jq
   
   # Check token expiration
   # The 'exp' claim is Unix timestamp
   ```

3. **Email Sending Failures**
   ```bash
   # Test SMTP connection
   openssl s_client -connect smtp.gmail.com:587 -starttls smtp
   
   # Check email logs
   grep "email" /var/log/authapi/api.log
   ```

4. **Rate Limiting Issues**
   ```bash
   # Check rate limit headers
   curl -I https://api.yourdomain.com/api/v1/auth/login
   
   # Look for:
   # X-RateLimit-Limit: 60
   # X-RateLimit-Remaining: 59
   # X-RateLimit-Reset: 1634567890
   ```

### Debug Mode

Enable debug logging:

```bash
# Set log level
export LOG_LEVEL=debug

# Or in your application
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

### Health Check Script

```bash
#!/bin/bash
# healthcheck.sh

API_URL="https://api.yourdomain.com"

# Check health endpoint
health_response=$(curl -s -o /dev/null -w "%{http_code}" $API_URL/health)
if [ $health_response -ne 200 ]; then
    echo "Health check failed: HTTP $health_response"
    exit 1
fi

# Check ready endpoint
ready_response=$(curl -s $API_URL/ready)
if ! echo $ready_response | grep -q '"status":"ready"'; then
    echo "Ready check failed: $ready_response"
    exit 1
fi

echo "All health checks passed"
exit 0
```

## Scaling Considerations

### Horizontal Scaling

1. **Database Connections**
   - Use connection pooling
   - Consider read replicas
   - Implement database proxy (PgBouncer)

2. **Session Management**
   - Stateless JWT tokens scale well
   - Consider Redis for refresh token storage

3. **Load Balancing**
   ```nginx
   upstream api_backend {
       least_conn;
       server api1.internal:8080 weight=1;
       server api2.internal:8080 weight=1;
       server api3.internal:8080 weight=1;
   }
   ```

### Performance Optimization

1. **Database Indexes**
   ```sql
   -- Add indexes for common queries
   CREATE INDEX CONCURRENTLY idx_users_email_verified 
   ON users(email_verified) WHERE email_verified = false;
   
   CREATE INDEX CONCURRENTLY idx_refresh_tokens_user_expires 
   ON refresh_tokens(user_id, expires_at) WHERE revoked = false;
   ```

2. **Query Optimization**
   ```sql
   -- Use EXPLAIN ANALYZE
   EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com';
   ```

3. **Caching Strategy**
   - Cache user profiles
   - Cache JWT validation results
   - Use ETags for API responses