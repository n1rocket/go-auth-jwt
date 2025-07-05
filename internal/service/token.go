package service

import (
	"context"
	"fmt"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/domain"
	"github.com/n1rocket/go-auth-jwt/internal/repository"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

// TokenService handles token-related operations
type TokenService struct {
	tokenManager     *token.Manager
	refreshTokenRepo repository.RefreshTokenRepository
	refreshTokenTTL  time.Duration
}

// NewTokenService creates a new token service
func NewTokenService(
	tokenManager *token.Manager,
	refreshTokenRepo repository.RefreshTokenRepository,
	refreshTokenTTL time.Duration,
) *TokenService {
	return &TokenService{
		tokenManager:     tokenManager,
		refreshTokenRepo: refreshTokenRepo,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// GenerateTokenPair generates a new access and refresh token pair for a user
func (s *TokenService) GenerateTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	// Generate access token
	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.EmailVerified)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Create refresh token
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
		CreatedAt: time.Now(),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

// RefreshTokenPair refreshes a token pair using a refresh token
func (s *TokenService) RefreshTokenPair(ctx context.Context, refreshTokenStr string) (*TokenPair, *domain.User, error) {
	// Get refresh token
	refreshToken, err := s.refreshTokenRepo.GetByToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid refresh token")
	}

	if refreshToken == nil || refreshToken.Revoked || time.Now().After(refreshToken.ExpiresAt) {
		return nil, nil, fmt.Errorf("invalid or expired refresh token")
	}

	// Revoke old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, refreshToken.Token); err != nil {
		return nil, nil, fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// Get user
	user := &domain.User{ID: refreshToken.UserID}

	// Generate new token pair
	tokenPair, err := s.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return tokenPair, user, nil
}

// RevokeRefreshToken revokes a specific refresh token
func (s *TokenService) RevokeRefreshToken(ctx context.Context, tokenStr string) error {
	return s.refreshTokenRepo.Revoke(ctx, tokenStr)
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *TokenService) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
}

// ValidateAccessToken validates an access token and returns the claims
func (s *TokenService) ValidateAccessToken(tokenStr string) (*token.Claims, error) {
	return s.tokenManager.ValidateAccessToken(tokenStr)
}

// GenerateVerificationToken generates an email verification token
func (s *TokenService) GenerateVerificationToken(user *domain.User) (string, error) {
	// For now, we'll use the access token mechanism with a short TTL
	// In a real implementation, you might want a separate verification token system
	return s.tokenManager.GenerateAccessToken(user.ID, user.Email, false)
}

// ValidateVerificationToken validates an email verification token
func (s *TokenService) ValidateVerificationToken(tokenStr string) (*token.Claims, error) {
	return s.tokenManager.ValidateAccessToken(tokenStr)
}
