# Database Migrations Guide

## Overview

This project uses a robust database migration system to manage schema changes. We support both file-based and embedded migrations for different deployment scenarios.

## Migration System Features

- **File-based migrations**: For development and flexible deployment
- **Embedded migrations**: For single binary deployments
- **Version control**: Track and manage database schema versions
- **Rollback support**: Safely revert changes when needed
- **Audit trail**: Complete history of applied migrations

## Migration Files Structure

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_refresh_tokens_table.up.sql
├── 000002_create_refresh_tokens_table.down.sql
├── 000003_add_user_profile.up.sql
├── 000003_add_user_profile.down.sql
├── 000004_add_audit_tables.up.sql
├── 000004_add_audit_tables.down.sql
├── 000005_add_roles_permissions.up.sql
├── 000005_add_roles_permissions.down.sql
└── README.md
```

## Current Schema

### Users Table
- Basic authentication fields (email, password_hash)
- Email verification fields
- Profile fields (first_name, last_name, phone, avatar, etc.)
- Activity tracking (last_login_at, login_count)
- Metadata JSONB field for extensibility

### Refresh Tokens Table
- Token management with expiration
- Device/session tracking (user_agent, ip_address)
- Revocation support

### Audit Tables
- `audit_logs`: General audit trail for all actions
- `login_attempts`: Track login attempts for security
- `password_reset_tokens`: Manage password reset flow

### RBAC Tables
- `roles`: Define system and custom roles
- `permissions`: Fine-grained permission definitions
- `role_permissions`: Map permissions to roles
- `user_roles`: Assign roles to users

## Usage Methods

### 1. Using Make Commands

```bash
# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create name=add_new_feature
```

### 2. Using Shell Script

```bash
# Run migrations
./scripts/migrate.sh up

# Rollback
./scripts/migrate.sh down

# Check status
./scripts/migrate.sh status

# Create migration
./scripts/migrate.sh create add_new_feature
```

### 3. Using Go CLI Tool

```bash
# Build the migrate tool
go build -o bin/migrate cmd/migrate/main.go

# Run migrations
./bin/migrate -command up

# Use embedded migrations
./bin/migrate -command up -embedded

# Run specific number of steps
./bin/migrate -command steps -steps 2

# Check version
./bin/migrate -command version
```

### 4. Programmatically in Code

```go
// Using embedded migrations
migrator := db.NewMigrator(database, db.MigrationConfig{
    DatabaseName: "authdb",
    SchemaName:   "public",
})

if err := migrator.Up(); err != nil {
    log.Fatal(err)
}

// Using file-based migrations
err := db.RunMigrationsFromPath(
    database, 
    "./migrations",
    db.MigrationConfig{},
)
```

## Docker Integration

### Development
```yaml
services:
  migrate:
    image: migrate/migrate
    volumes:
      - ./migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://user:pass@db:5432/authdb?sslmode=disable",
      "up"
    ]
```

### Production with Embedded Migrations
```dockerfile
# Migrations are embedded in the binary
RUN go build -o main cmd/api/main.go

# Run migrations on startup
ENTRYPOINT ["./main", "-migrate"]
```

## Best Practices

### 1. Migration Naming
- Use descriptive names: `add_user_profile`, not `update_users`
- Include table name when relevant
- Keep names concise but clear

### 2. Transaction Safety
```sql
BEGIN;
-- Your migration here
COMMIT;
```

### 3. Idempotent Operations
```sql
-- Good: Check if exists
CREATE TABLE IF NOT EXISTS users (...);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);

-- Bad: Will fail if already exists
CREATE TABLE users (...);
ALTER TABLE users ADD COLUMN email VARCHAR(255);
```

### 4. Data Migration
```sql
-- Migrate data in the same transaction
BEGIN;
ALTER TABLE users ADD COLUMN full_name VARCHAR(200);
UPDATE users SET full_name = CONCAT(first_name, ' ', last_name);
COMMIT;
```

### 5. Index Creation
```sql
-- Create indexes concurrently in production
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
```

## Troubleshooting

### Dirty Database State
```bash
# Check current state
./bin/migrate -command version

# Force to clean state (use carefully!)
./bin/migrate -command force -version 3
```

### Failed Migration
1. Check error logs
2. Manually fix the issue if needed
3. Force version if necessary
4. Re-run migrations

### Performance Issues
- Use `CONCURRENTLY` for index creation
- Split large data migrations
- Consider maintenance windows

## CI/CD Integration

### GitHub Actions
```yaml
- name: Run migrations
  env:
    DATABASE_DSN: ${{ secrets.DATABASE_DSN }}
  run: |
    go run cmd/migrate/main.go -command up
```

### Kubernetes Job
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: your-app:latest
        command: ["/app/migrate", "-command", "up", "-embedded"]
        env:
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: dsn
```

## Security Considerations

1. **Never commit sensitive data** in migrations
2. **Use environment variables** for connection strings
3. **Restrict migration permissions** in production
4. **Audit migration executions**
5. **Test migrations thoroughly** before production

## Future Enhancements

- [ ] Add migration testing framework
- [ ] Implement migration versioning API
- [ ] Add migration rollback hooks
- [ ] Create migration visualization tool
- [ ] Add automatic backup before migrations