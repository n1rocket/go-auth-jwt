package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDB_WithTransaction(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		fn        TxFunc
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful transaction",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO users").
					WithArgs("test@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, "INSERT INTO users", "test@example.com")
				return err
			},
			wantErr: false,
		},
		{
			name: "begin transaction error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return nil
			},
			wantErr: true,
			errMsg:  "failed to begin transaction",
		},
		{
			name: "function returns error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return errors.New("function error")
			},
			wantErr: true,
			errMsg:  "function error",
		},
		{
			name: "commit error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return nil
			},
			wantErr: true,
			errMsg:  "failed to commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock: %v", err)
			}
			defer mockDB.Close()

			db := &DB{mockDB}
			tt.setupMock(mock)

			err = db.WithTransaction(context.Background(), tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDB_WithTransaction_Panic(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mockDB.Close()

	db := &DB{mockDB}

	// Expect begin and rollback
	mock.ExpectBegin()
	mock.ExpectRollback()

	// Test that panic is propagated
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be propagated")
		}
	}()

	db.WithTransaction(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		panic("test panic")
	})
}

func TestDB_WithTransactionIsolation(t *testing.T) {
	tests := []struct {
		name      string
		level     sql.IsolationLevel
		setupMock func(sqlmock.Sqlmock)
		fn        TxFunc
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "successful transaction with isolation level",
			level: sql.LevelSerializable,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE users").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, "UPDATE users")
				return err
			},
			wantErr: false,
		},
		{
			name:  "begin transaction with isolation error",
			level: sql.LevelReadCommitted,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().
					WillReturnError(errors.New("isolation not supported"))
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return nil
			},
			wantErr: true,
			errMsg:  "failed to begin transaction with isolation level",
		},
		{
			name:  "function error with rollback",
			level: sql.LevelRepeatableRead,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return errors.New("operation failed")
			},
			wantErr: true,
			errMsg:  "operation failed",
		},
		{
			name:  "commit error",
			level: sql.LevelDefault,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			fn: func(ctx context.Context, tx *sql.Tx) error {
				return nil
			},
			wantErr: true,
			errMsg:  "failed to commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock: %v", err)
			}
			defer mockDB.Close()

			db := &DB{mockDB}
			tt.setupMock(mock)

			err = db.WithTransactionIsolation(context.Background(), tt.level, tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithTransactionIsolation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDB_WithTransactionIsolation_Panic(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mockDB.Close()

	db := &DB{mockDB}

	// Expect begin and rollback
	mock.ExpectBegin()
	mock.ExpectRollback()

	// Test that panic is propagated
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be propagated")
		}
	}()

	db.WithTransactionIsolation(context.Background(), sql.LevelDefault, func(ctx context.Context, tx *sql.Tx) error {
		panic("test panic")
	})
}

