package repository

import (
	"context"

	"github.com/abueno/go-auth-jwt/internal/domain"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update updates a user
	Update(ctx context.Context, user *domain.User) error

	// Delete deletes a user
	Delete(ctx context.Context, id string) error

	// ExistsByEmail checks if a user exists with the given email
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	// Create creates a new refresh token
	Create(ctx context.Context, token *domain.RefreshToken) error

	// GetByToken retrieves a refresh token by its token value
	GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error)

	// GetByUserID retrieves all refresh tokens for a user
	GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error)

	// Update updates a refresh token
	Update(ctx context.Context, token *domain.RefreshToken) error

	// Revoke revokes a refresh token
	Revoke(ctx context.Context, token string) error

	// RevokeAllForUser revokes all refresh tokens for a user
	RevokeAllForUser(ctx context.Context, userID string) error

	// DeleteExpired deletes all expired refresh tokens
	DeleteExpired(ctx context.Context) error

	// DeleteByToken deletes a refresh token by its token value
	DeleteByToken(ctx context.Context, token string) error
}
