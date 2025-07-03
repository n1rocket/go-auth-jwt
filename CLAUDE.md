# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a JWT Authentication Provider built exclusively with Go's standard library (net/http). It includes user registration/login, JWT token generation/refresh, protected endpoints, asynchronous email sending, Prometheus observability, and Docker deployment.

## Key Architecture

### Clean Architecture Structure

- `cmd/`: Application entrypoints
  - `api/`: Main API server with graceful shutdown
  - `migrate/`: Database migration CLI tool
- `internal/`: Core business logic and internal packages
  - `config/`: Centralized configuration management with validation
  - `domain/`: Business entities (User)
  - `service/`: Business logic (AuthService)
  - `repository/`: Data access layer (UserRepo interface)
    - `postgres/`: PostgreSQL implementation with pgx/v5
  - `http/`: HTTP transport layer
    - `handlers/`: Request handlers
    - `middleware/`: Auth, CORS, rate limiting, security headers
    - `request/`: Request validation
    - `response/`: Response helpers and error handling
  - `token/`: JWT token management (HS256/RS256)
  - `worker/`: Asynchronous tasks (email dispatcher with worker pool)
  - `email/`: Email service with SMTP implementation
  - `metrics/`: Custom metrics implementation
  - `monitoring/`: Observability features
  - `db/`: Database connection and migration management
  - `security/`: Password hashing and security utilities
- `pkg/`: Reusable packages
- `deploy/`: Deployment configurations (Docker, docker-compose, k8s)
- `scripts/`: Utility scripts (migrations, k6 tests)
- `docs/`: Documentation (OpenAPI, architecture diagrams)
- `examples/`: Client examples (Go, JavaScript, Python, React)
- `migrations/`: SQL migration files

### Key Dependencies

- `golang-jwt/jwt v5`: JWT token handling
- `jackc/pgx/v5`: PostgreSQL driver
- `golang-migrate/migrate/v4`: Database migrations
- No web framework - uses standard library `net/http`

## Development Commands

```bash
# Setup and dependencies
make setup                  # Install development tools (golangci-lint, gosec, swag, migrate)
make mod                    # Download and tidy Go modules

# Run the application
make run                    # Start API on :8080
make dev                    # Run with hot reload (requires air)

# Testing
make test                   # Run unit tests with coverage report
make test-integration       # Run integration tests with dockertest
make test-e2e              # Run e2e tests with k6
make test-all              # Run all tests
make bench                  # Run benchmarks for JWT operations
go test -run TestSpecific   # Run a specific test

# Code quality
make lint                   # Run golangci-lint
make lint-fix              # Run golangci-lint with auto-fix
make security-scan         # Run gosec security analysis
make vuln-check           # Check for known vulnerabilities
make fmt                   # Format code
make vet                  # Run go vet

# Database
make migrate-up            # Apply all pending migrations
make migrate-down          # Rollback last migration
make migrate-create name=X # Create new migration file

# Docker
make docker-build          # Build Docker image
make compose-up            # Start all services (API, PostgreSQL, Prometheus, Grafana)
make compose-down          # Stop all services
make compose-logs          # View docker-compose logs
make dev-up               # Start development environment with MailHog
make dev-down             # Stop development environment

# Documentation
make docs                  # Generate OpenAPI documentation with swag

# Utilities
make keys                  # Generate RSA key pair for JWT (RS256)
make mock                  # Generate mocks with mockery
make clean                # Clean build artifacts and caches
make ci                   # Run CI checks locally (lint, test, security-scan)
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

1. Linting with golangci-lint v1.62.2
2. Unit and integration tests (Go 1.21+)
3. Security scanning with gosec v2.22.5
4. Docker image build and push to GHCR
5. Release notes generation with SemVer

Workflows:
- `.github/workflows/ci.yml`: Runs on PRs and pushes to main/develop
- `.github/workflows/deploy.yml`: Handles deployment to production

## Testing Patterns

### Unit Tests
- Use table-driven tests for comprehensive coverage
- Tests use `t.Parallel()` for faster execution
- Mock interfaces using mockery-generated mocks
- Test files follow `*_test.go` naming convention

### Integration Tests
- Use build tag `//go:build integration`
- Real PostgreSQL database using dockertest
- Located in `internal/test/integration/`
- Run with: `make test-integration`

### Running Specific Tests
```bash
# Run a single test
go test -run TestAuthService_Login ./internal/service

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/http/handlers/...

# Run with race detector
go test -race ./...
```

## Common Development Tasks

### Running Local Development Environment
```bash
# Start everything with docker-compose
make dev-up

# This starts:
# - API on http://localhost:8080
# - PostgreSQL on localhost:5432
# - MailHog on http://localhost:8025 (email testing)
# - Database is automatically migrated

# View logs
make dev-logs

# Stop everything
make dev-down
```

### Working with JWT Tokens
- Development uses HS256 with `JWT_SECRET` environment variable
- Production should use RS256 with key files
- Generate RSA keys: `make keys`
- Access tokens expire in 15 minutes by default
- Refresh tokens expire in 7 days by default

### Database Operations
```bash
# Connect to PostgreSQL
psql -h localhost -U auth -d authsvc

# Create a new migration
make migrate-create name=add_user_feature

# Check migration status
migrate -path migrations -database "${DB_DSN}" version
```

## Debugging Tips

- Check logs for structured JSON output (slog)
- Use `/metrics` endpoint for Prometheus metrics
- Grafana dashboard available at `localhost:3000` when using docker-compose
- Enable debug logging with `LOG_LEVEL=debug`
- All HTTP responses include request ID in `X-Request-ID` header
- Database queries can be logged by setting `DB_LOG_QUERIES=true`

## Important Notes

- The project uses Go's standard library `net/http` - no web framework
- All configuration is done through environment variables
- Passwords are hashed using bcrypt with cost factor 12
- Rate limiting uses token bucket algorithm (100 requests/minute by default)
- Email sending is asynchronous using a worker pool pattern
- All endpoints return JSON responses
- CORS is configured to support credentials and specific origins
