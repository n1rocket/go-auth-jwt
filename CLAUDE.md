# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a JWT Authentication Provider built exclusively with Go's standard library (net/http). It includes user registration/login, JWT token generation/refresh, protected endpoints, asynchronous email sending, Prometheus observability, and Docker deployment.

## Key Architecture

### Clean Architecture Structure

- `cmd/`: Application entrypoints (main.go for the API server)
- `internal/`: Core business logic and internal packages
  - `config/`: Centralized configuration management
  - `domain/`: Business entities (User)
  - `service/`: Business logic (AuthService)
  - `repository/`: Data access layer (UserRepo interface)
  - `http/`: HTTP transport layer (handlers, middleware)
  - `token/`: JWT token management (HS256/RS256)
  - `worker/`: Asynchronous tasks (email dispatcher)
- `pkg/`: Reusable packages
- `deploy/`: Deployment configurations (Docker, docker-compose)
- `scripts/`: Utility scripts (migrations, k6 tests)
- `docs/`: Documentation (OpenAPI, architecture diagrams)

### Key Dependencies

- `golang-jwt/jwt v5.2.2`: JWT token handling
- `pgx v5.7.5`: PostgreSQL driver
- `sqlx v1.4.0`: SQL extensions
- `prometheus/client_golang v1.22.0`: Metrics
- `opentelemetry-go v1.37.0`: Distributed tracing

## Development Commands

```bash
# Run the application
make run                    # Start API on :8080

# Testing
make test                   # Run unit tests with t.Parallel
make test-integration       # Run integration tests with dockertest
make bench                  # Run benchmarks for JWT operations

# Code quality
make lint                   # Run golangci-lint
gosec ./...                # Security analysis

# Docker
docker compose up           # Start all services (API, PostgreSQL, Prometheus, Grafana)
make docker-build          # Build multi-stage Docker image (<20MB)

# Documentation
swag init                  # Generate OpenAPI documentation
```

## Database Schema

### Users table

- `id`: UUID primary key
- `email`: Unique email address
- `password_hash`: Bcrypt hashed password
- `email_verified`: Boolean
- `created_at`, `updated_at`: Timestamps

### Refresh tokens table

- `token`: UUID primary key
- `user_id`: Foreign key to users
- `expires_at`: Token expiration
- `revoked`: Boolean for token invalidation
- Composite index on (user_id, token) for fast revocation

## Environment Variables

```bash
# Application
APP_PORT=8080                                              # API HTTP port
APP_ENV=development                                        # Environment (development/staging/production)
APP_READ_TIMEOUT=15s                                       # HTTP read timeout
APP_WRITE_TIMEOUT=15s                                      # HTTP write timeout
APP_IDLE_TIMEOUT=60s                                       # HTTP idle timeout
APP_SHUTDOWN_TIMEOUT=30s                                   # Graceful shutdown timeout

# Database
DB_DSN=postgres://auth:secret@db:5432/authsvc?sslmode=disable  # PostgreSQL connection
DB_MAX_OPEN_CONNS=25                                       # Max open connections
DB_MAX_IDLE_CONNS=5                                        # Max idle connections

# JWT
JWT_SECRET=insecure_demo                                   # HS256 secret (demo only)
JWT_PRIVATE_KEY_PATH=/path/to/private.pem                 # RS256 private key
JWT_PUBLIC_KEY_PATH=/path/to/public.pem                   # RS256 public key
JWT_ACCESS_TOKEN_TTL=15m                                   # Access token lifetime
JWT_REFRESH_TOKEN_TTL=168h                                 # Refresh token lifetime (7 days)
JWT_ISSUER=go-auth-jwt                                     # Token issuer
JWT_ALGORITHM=HS256                                        # HS256 or RS256

# Email (Optional for Phase 4)
SMTP_HOST=smtp.example.com                                # Email server
SMTP_USER=noreply@example.com                            # Email user
SMTP_PASS=change-me                                      # Email password
SMTP_PORT=587                                           # Email port
```

## API Endpoints

### Public endpoints

- `POST /api/v1/auth/signup`: User registration
- `POST /api/v1/auth/login`: User authentication
- `POST /api/v1/auth/refresh`: Token refresh
- `POST /api/v1/auth/verify-email`: Email verification
- `GET /health`: Basic health check
- `GET /ready`: Readiness probe with service checks

### Protected endpoints (require JWT)

- `GET /api/v1/auth/me`: Get current user profile
- `POST /api/v1/auth/logout`: Invalidate refresh token
- `POST /api/v1/auth/logout-all`: Logout from all devices

### Future endpoints

- `GET /.well-known/jwks.json`: Public keys for RS256 (Phase 6)
- `GET /metrics`: Prometheus metrics (Phase 5)

## Testing Strategy

1. **Unit tests**: Domain logic, utilities (aim for >80% coverage)
2. **Integration tests**: Real PostgreSQL using dockertest
3. **E2E tests**: k6 scripts for full auth flow testing
4. **Benchmarks**: JWT generation/verification performance

## Security Considerations

- Password hashing with bcrypt (configurable cost)
- JWT with HS256 (demo) or RS256 (production)
- Key rotation support via `kid` header and JWKS endpoint
- Refresh token rotation on each use
- Rate limiting middleware
- Security headers (CSP, HSTS, etc.)
- No secrets in code - use environment variables

## Development Workflow

1. Create feature branch from `main`
2. Write tests first (TDD approach)
3. Implement feature following Clean Architecture
4. Ensure `make lint` and `make test` pass
5. Update OpenAPI documentation if adding endpoints
6. Submit PR with conventional commit messages

## Common Tasks

### Adding a new endpoint

1. Define handler in `internal/http/handlers/`
2. Add route in `internal/http/routes.go`
3. Update OpenAPI annotations (if using swag)
4. Write handler tests
5. Add integration tests in `internal/test/integration/`

### Modifying database schema

1. Create migration file in `migrations/`
2. Update repository interfaces
3. Update domain entities if needed
4. Run migrations: `make migrate-up`

### Adding new JWT claims

1. Update token manager in `internal/token/`
2. Modify middleware to extract new claims
3. Update tests for token generation/validation

## CI/CD Pipeline

GitHub Actions workflow includes:

1. Linting with golangci-lint v2.2.0
2. Unit and integration tests (Go 1.24, 1.25-beta)
3. Security scanning with gosec v2.22.5
4. Docker image build and push to GHCR
5. Release notes generation with SemVer

## Debugging Tips

- Check logs for structured JSON output (slog)
- Use `/metrics` endpoint for Prometheus metrics
- Grafana dashboard available at `localhost:3000` when using docker-compose
- Enable debug logging with `LOG_LEVEL=debug`
