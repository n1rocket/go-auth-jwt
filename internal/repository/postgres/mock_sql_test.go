package postgres

import (
	"context"
	"database/sql"
)

// mockDB implements DBTX interface for testing
type mockDB struct {
	execFunc       func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	prepareFunc    func(ctx context.Context, query string) (*sql.Stmt, error)
	queryFunc      func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	queryRowFunc   func(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if m.prepareFunc != nil {
		return m.prepareFunc(ctx, query)
	}
	return nil, nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return nil
}

// mockResult implements sql.Result for testing
type mockResult struct {
	lastInsertID int64
	rowsAffected int64
	err          error
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, m.err
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.err
}