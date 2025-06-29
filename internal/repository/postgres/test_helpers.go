package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestDB creates a test database connection
func TestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// CleanupTestData removes all test data from the database
func CleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		"DELETE FROM refresh_tokens",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("failed to cleanup test data: %v", err)
		}
	}
}

// MustExec executes a query and fails the test if there's an error
func MustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) sql.Result {
	t.Helper()

	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}

	return result
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *sql.DB, email, passwordHash string) string {
	t.Helper()

	var userID string
	query := `
		INSERT INTO users (id, email, password_hash, email_verified, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id`

	err := db.QueryRow(query, email, passwordHash).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return userID
}

// CreateTestRefreshToken creates a test refresh token in the database
func CreateTestRefreshToken(t *testing.T, db *sql.DB, userID string) string {
	t.Helper()

	var token string
	query := `
		INSERT INTO refresh_tokens (
			token, user_id, expires_at, revoked, 
			created_at, last_used_at
		) VALUES (
			gen_random_uuid(), $1, CURRENT_TIMESTAMP + INTERVAL '7 days', false,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		) RETURNING token`

	err := db.QueryRow(query, userID).Scan(&token)
	if err != nil {
		t.Fatalf("failed to create test refresh token: %v", err)
	}

	return token
}

// AssertRowCount asserts that a table has the expected number of rows
func AssertRowCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}

	if count != expected {
		t.Errorf("expected %d rows in %s, got %d", expected, table, count)
	}
}