package db

import (
	"context"
	"database/sql"
	"fmt"
)

// TxFunc represents a function that will be executed within a transaction
type TxFunc func(context.Context, *sql.Tx) error

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(ctx context.Context, fn TxFunc) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer a rollback in case anything fails
	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, rollback and re-panic
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			// An error occurred, rollback
			_ = tx.Rollback()
		}
	}()

	// Execute the function
	err = fn(ctx, tx)
	if err != nil {
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionIsolation executes a function within a transaction with a specific isolation level
func (db *DB) WithTransactionIsolation(ctx context.Context, level sql.IsolationLevel, fn TxFunc) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: level,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction with isolation level %v: %w", level, err)
	}

	// Defer a rollback in case anything fails
	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, rollback and re-panic
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			// An error occurred, rollback
			_ = tx.Rollback()
		}
	}()

	// Execute the function
	err = fn(ctx, tx)
	if err != nil {
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
