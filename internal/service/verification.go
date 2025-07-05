package service

import (
	"context"
	"fmt"
)

// VerificationService handles email verification operations
type VerificationService struct {
	userService  *UserService
	tokenService *TokenService
}

// NewVerificationService creates a new verification service
func NewVerificationService(
	userService *UserService,
	tokenService *TokenService,
) *VerificationService {
	return &VerificationService{
		userService:  userService,
		tokenService: tokenService,
	}
}

// VerifyEmail verifies a user's email using a verification token
func (s *VerificationService) VerifyEmail(ctx context.Context, tokenStr string) error {
	// Validate verification token
	claims, err := s.tokenService.ValidateVerificationToken(tokenStr)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token")
	}

	// Get user
	user, err := s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Check if already verified
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Mark email as verified
	if err := s.userService.MarkEmailVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// ResendVerificationEmail generates a new verification token for a user
func (s *VerificationService) ResendVerificationEmail(ctx context.Context, email string) (string, error) {
	// Get user
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Check if already verified
	if user.EmailVerified {
		return "", fmt.Errorf("email already verified")
	}

	// Generate new verification token
	verificationToken, err := s.tokenService.GenerateVerificationToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}

	return verificationToken, nil
}
