package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/domain"
	"github.com/n1rocket/go-auth-jwt/internal/repository"
)

// RefreshTokenRepository implements repository.RefreshTokenRepository using PostgreSQL
type RefreshTokenRepository struct {
	db DBTX
}

// NewRefreshTokenRepository creates a new PostgreSQL refresh token repository
func NewRefreshTokenRepository(db DBTX) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token in the database
func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (
			token, user_id, expires_at, revoked, revoked_at,
			user_agent, ip_address, created_at, last_used_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING token`

	err := r.db.QueryRowContext(
		ctx,
		query,
		token.UserID,
		token.ExpiresAt,
		token.Revoked,
		token.RevokedAt,
		token.UserAgent,
		token.IPAddress,
		token.CreatedAt,
		token.LastUsedAt,
	).Scan(&token.Token)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetByToken retrieves a refresh token by its token value
func (r *RefreshTokenRepository) GetByToken(ctx context.Context, tokenValue string) (*domain.RefreshToken, error) {
	token := &domain.RefreshToken{}
	query := `
		SELECT 
			token, user_id, expires_at, revoked, revoked_at,
			user_agent, ip_address, created_at, last_used_at
		FROM refresh_tokens
		WHERE token = $1`

	err := r.db.QueryRowContext(ctx, query, tokenValue).Scan(
		&token.Token,
		&token.UserID,
		&token.ExpiresAt,
		&token.Revoked,
		&token.RevokedAt,
		&token.UserAgent,
		&token.IPAddress,
		&token.CreatedAt,
		&token.LastUsedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

// GetByUserID retrieves all refresh tokens for a user
func (r *RefreshTokenRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	query := `
		SELECT 
			token, user_id, expires_at, revoked, revoked_at,
			user_agent, ip_address, created_at, last_used_at
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh tokens by user id: %w", err)
	}
	defer rows.Close()

	var tokens []*domain.RefreshToken
	for rows.Next() {
		token := &domain.RefreshToken{}
		err := rows.Scan(
			&token.Token,
			&token.UserID,
			&token.ExpiresAt,
			&token.Revoked,
			&token.RevokedAt,
			&token.UserAgent,
			&token.IPAddress,
			&token.CreatedAt,
			&token.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating refresh tokens: %w", err)
	}

	return tokens, nil
}

// Update updates a refresh token in the database
func (r *RefreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		UPDATE refresh_tokens SET
			expires_at = $2,
			revoked = $3,
			revoked_at = $4,
			last_used_at = $5
		WHERE token = $1`

	result, err := r.db.ExecContext(
		ctx,
		query,
		token.Token,
		token.ExpiresAt,
		token.Revoked,
		token.RevokedAt,
		token.LastUsedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrInvalidToken
	}

	return nil
}

// Revoke revokes a refresh token
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenValue string) error {
	query := `
		UPDATE refresh_tokens SET
			revoked = true,
			revoked_at = $2
		WHERE token = $1 AND revoked = false`

	result, err := r.db.ExecContext(ctx, query, tokenValue, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrInvalidToken
	}

	return nil
}

// RevokeAllForUser revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_tokens SET
			revoked = true,
			revoked_at = $2
		WHERE user_id = $1 AND revoked = false`

	_, err := r.db.ExecContext(ctx, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens for user: %w", err)
	}

	return nil
}

// DeleteExpired deletes all expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1 OR (revoked = true AND revoked_at < $2)`

	// Delete tokens that have been expired or revoked for more than 30 days
	cutoffTime := time.Now().Add(-30 * 24 * time.Hour)

	_, err := r.db.ExecContext(ctx, query, time.Now(), cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	return nil
}

// DeleteByToken deletes a refresh token by its token value
func (r *RefreshTokenRepository) DeleteByToken(ctx context.Context, tokenValue string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`

	result, err := r.db.ExecContext(ctx, query, tokenValue)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrInvalidToken
	}

	return nil
}

// Ensure RefreshTokenRepository implements repository.RefreshTokenRepository
var _ repository.RefreshTokenRepository = (*RefreshTokenRepository)(nil)