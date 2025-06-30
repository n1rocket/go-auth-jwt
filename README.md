# go-auth-jwt

[![Go Version](https://img.shields.io/badge/Go-1.24.4-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI Status](https://github.com/youruser/go-auth-jwt/workflows/CI/badge.svg)](https://github.com/youruser/go-auth-jwt/actions)
[![Coverage](https://img.shields.io/badge/Coverage-80%25-brightgreen.svg)](https://github.com/youruser/go-auth-jwt)

JWT Authentication Provider built exclusively with Go's standard library (net/http). Enterprise-ready with production features including secure token management, asynchronous operations, comprehensive observability, and containerized deployment.

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/youruser/go-auth-jwt
cd go-auth-jwt

# Run with Make
make run         # Start API on :8080

# Or use Docker Compose (recommended)
docker compose up # Full stack with PostgreSQL, Prometheus, Grafana
```

## 📋 Features

### Core Authentication

- **User Management**: Registration, login, and profile management
- **JWT Tokens**: Support for both HS256 (demo) and RS256 (production) algorithms
- **Token Refresh**: Secure refresh token rotation with revocation support
- **Email Verification**: Asynchronous email sending via worker pool
- **Password Security**: Bcrypt hashing with configurable cost factor

### Architecture & Code Quality

- **Clean Architecture**: Separation of concerns with clear boundaries
- **Standard Library**: Built primarily with `net/http` and `http.ServeMux`
- **Minimal Dependencies**: Only essential, well-justified external libraries
- **Comprehensive Testing**: Unit, integration, and E2E test suites
- **Security First**: Regular vulnerability scanning, security headers, rate limiting

### Operations & Observability

- **Prometheus Metrics**: Request counts, latencies, and custom business metrics
- **Structured Logging**: JSON logs with `log/slog` for easy parsing
- **Health Checks**: Liveness and readiness probes for Kubernetes
- **Graceful Shutdown**: Proper connection draining and cleanup
- **Docker Ready**: Multi-stage builds producing <20MB images

## 🏗️ Architecture

```
go-auth-jwt/
├── cmd/
│   └── api/              # Application entrypoint
├── internal/
│   ├── config/          # Configuration management
│   ├── domain/          # Business entities
│   ├── service/         # Business logic
│   ├── repository/      # Data access layer
│   ├── http/           # HTTP transport
│   │   ├── handlers/   # Request handlers
│   │   └── middleware/ # Auth, logging, etc.
│   ├── token/          # JWT management
│   └── worker/         # Async tasks
├── pkg/                # Reusable packages
├── deploy/            # Deployment configs
├── migrations/        # Database migrations
├── scripts/          # Utility scripts
└── docs/            # Documentation
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_PORT` | API server port | `8080` | No |
| `DB_DSN` | PostgreSQL connection string | - | Yes |
| `JWT_SECRET` | HS256 secret (demo mode) | - | No* |
| `JWT_PRIVATE_KEY_PATH` | RS256 private key path | - | No* |
| `JWT_PUBLIC_KEY_PATH` | RS256 public key path | - | No* |
| `SMTP_HOST` | Email server hostname | - | Yes |
| `SMTP_PORT` | Email server port | `587` | No |
| `SMTP_USER` | Email username | - | Yes |
| `SMTP_PASS` | Email password | - | Yes |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |
| `METRICS_PORT` | Prometheus metrics port | `9090` | No |

*Either JWT_SECRET or JWT key paths must be provided

### Example `.env` file

```bash
# Database
DB_DSN=postgres://auth:secret@localhost:5432/authsvc?sslmode=disable

# JWT (Production - RS256)
JWT_PRIVATE_KEY_PATH=./certs/private.pem
JWT_PUBLIC_KEY_PATH=./certs/public.pem

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# Observability
LOG_LEVEL=info
METRICS_PORT=9090
```

## 📡 API Endpoints

### Authentication Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/signup` | Register new user | No |
| POST | `/login` | Authenticate user | No |
| POST | `/refresh` | Refresh access token | No |
| POST | `/logout` | Invalidate refresh token | Yes |
| GET | `/profile` | Get user profile | Yes |
| POST | `/verify-email` | Verify email address | No |
| POST | `/resend-verification` | Resend verification email | Yes |

### System Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/.well-known/jwks.json` | Public keys (RS256) |
| GET | `/docs` | OpenAPI documentation |

### Request/Response Examples

#### Signup

```bash
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'

# Response
{
  "message": "User created successfully. Please check your email for verification."
}
```

#### Login

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'

# Response
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
  "expires_in": 3600
}
```

## 🗄️ Database Migrations

The project includes a comprehensive migration system for managing database schema changes.

```bash
# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create a new migration
make migrate-create name=add_user_feature

# Check migration status
./scripts/migrate.sh status
```

### Migration Features

- **File-based migrations**: Development flexibility with SQL files
- **Embedded migrations**: Single binary deployment support
- **Version tracking**: Full migration history and rollback capability
- **CLI tool**: Standalone migration management tool
- **Programmatic API**: Run migrations from code

For detailed migration documentation, see [docs/MIGRATIONS.md](docs/MIGRATIONS.md).

## 🧪 Testing

```bash
# Unit tests with coverage
make test

# Integration tests (requires Docker)
make test-integration

# E2E tests with k6
make test-e2e

# Benchmarks
make bench

# Full test suite
make test-all
```

### Test Coverage Goals

- Unit Tests: >80% coverage
- Integration Tests: All API endpoints
- E2E Tests: Critical user flows

## 🚀 Deployment

### Docker

```bash
# Build image
make docker-build

# Run with docker-compose
docker compose up -d

# Check logs
docker compose logs -f api
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f deploy/k8s/

# Check deployment
kubectl get pods -n auth-system
```

### Production Checklist

- [ ] Use RS256 tokens with key rotation
- [ ] Enable TLS/HTTPS
- [ ] Configure rate limiting
- [ ] Set up monitoring alerts
- [ ] Enable distributed tracing
- [ ] Configure backup strategy
- [ ] Review security headers
- [ ] Set appropriate resource limits

## 📊 Monitoring

### Prometheus Metrics

- `http_requests_total`: Total HTTP requests by endpoint and status
- `http_request_duration_seconds`: Request latency histogram
- `jwt_tokens_generated_total`: JWT tokens created
- `jwt_verification_errors_total`: Failed token verifications
- `email_queue_size`: Pending emails in worker pool
- `db_connections_active`: Active database connections

### Grafana Dashboard

Import the dashboard from `docs/monitoring/grafana-dashboard.json` for:

- Request rate and error rate
- P50/P95/P99 latencies
- JWT operations monitoring
- Database connection pool stats
- Email queue performance

## 🔒 Security

### Features

- Password hashing with bcrypt (cost factor 12)
- JWT token expiration and refresh rotation
- Rate limiting per IP
- CORS configuration
- Security headers (CSP, HSTS, etc.)
- SQL injection prevention via prepared statements
- Input validation and sanitization

### Security Scanning

```bash
# Run gosec
make security-scan

# Vulnerability check
make vuln-check
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Write tests for new features
- Update documentation as needed
- Ensure CI passes before requesting review

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with Go's excellent standard library
- Inspired by clean architecture principles
- Thanks to all contributors

---

**Need help?** Create an [issue](https://github.com/youruser/go-auth-jwt/issues) or check our [documentation](./docs).
