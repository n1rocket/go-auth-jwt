# Database Migrations

This directory contains database migrations for the JWT Authentication Service.

## Overview

We use [golang-migrate](https://github.com/golang-migrate/migrate) for managing database migrations. Each migration consists of two files:
- `.up.sql` - Contains SQL to apply the migration
- `.down.sql` - Contains SQL to rollback the migration

## Installation

Install the migrate CLI tool:

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Using Go
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Usage

### Create a new migration

```bash
# Using make
make migrate-create name=add_user_roles

# Using migrate directly
migrate create -ext sql -dir migrations -seq add_user_roles
```

### Run migrations

```bash
# Run all pending migrations
make migrate-up

# Or using migrate directly
migrate -path ./migrations -database "$DATABASE_DSN" up

# Run a specific number of migrations
migrate -path ./migrations -database "$DATABASE_DSN" up 2
```

### Rollback migrations

```bash
# Rollback the last migration
make migrate-down

# Or using migrate directly
migrate -path ./migrations -database "$DATABASE_DSN" down 1

# Rollback all migrations
migrate -path ./migrations -database "$DATABASE_DSN" down
```

### Check migration status

```bash
migrate -path ./migrations -database "$DATABASE_DSN" version
```

### Force a specific version

```bash
# Use with caution!
migrate -path ./migrations -database "$DATABASE_DSN" force 2
```

## Migration Files

### 000001_create_users_table

Creates the users table with the following columns:
- `id` (UUID) - Primary key
- `email` (VARCHAR) - Unique user email
- `password_hash` (VARCHAR) - Bcrypt password hash
- `email_verified` (BOOLEAN) - Email verification status
- `email_verification_token` (VARCHAR) - Token for email verification
- `email_verification_expires` (TIMESTAMP) - Token expiration time
- `created_at` (TIMESTAMP) - User creation time
- `updated_at` (TIMESTAMP) - Last update time

### 000002_create_refresh_tokens_table

Creates the refresh_tokens table with the following columns:
- `id` (UUID) - Primary key
- `user_id` (UUID) - Foreign key to users table
- `token` (VARCHAR) - Unique refresh token
- `expires_at` (TIMESTAMP) - Token expiration time
- `created_at` (TIMESTAMP) - Token creation time
- `revoked` (BOOLEAN) - Token revocation status
- `revoked_at` (TIMESTAMP) - Revocation time
- `user_agent` (TEXT) - Client user agent
- `ip_address` (VARCHAR) - Client IP address

## Best Practices

1. **Always test migrations locally first**
   ```bash
   # Test on a local database
   DATABASE_DSN="postgres://localhost/authdb_test?sslmode=disable" make migrate-up
   ```

2. **Review both up and down migrations**
   - Ensure down migrations properly reverse the up migrations
   - Be careful with data loss in down migrations

3. **Use transactions when possible**
   ```sql
   BEGIN;
   -- Your migration SQL here
   COMMIT;
   ```

4. **Add comments to complex migrations**
   ```sql
   -- Add index to improve query performance for email lookups
   CREATE INDEX idx_users_email ON users(email);
   ```

5. **Version control**
   - Always commit migration files to version control
   - Never modify existing migration files in production
   - Create new migrations to fix issues

## Environment Variables

The migration commands use the following environment variable:
- `DATABASE_DSN` - PostgreSQL connection string

Example:
```bash
export DATABASE_DSN="postgres://user:password@localhost:5432/authdb?sslmode=disable"
```

## Troubleshooting

### "Dirty database" error

If migrations fail partway through, the database may be left in a "dirty" state:

```bash
# Check current version
migrate -path ./migrations -database "$DATABASE_DSN" version

# Force to a specific version (use with caution!)
migrate -path ./migrations -database "$DATABASE_DSN" force <version>

# Then run migrations again
migrate -path ./migrations -database "$DATABASE_DSN" up
```

### Connection issues

Ensure your DATABASE_DSN is correct:
```bash
# Test connection
psql "$DATABASE_DSN" -c "SELECT 1"
```

### Migration lock

If migrations appear stuck, check for locks:
```sql
SELECT * FROM pg_locks WHERE NOT granted;
```