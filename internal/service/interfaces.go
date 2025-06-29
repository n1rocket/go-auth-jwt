package service

import (
	"context"

	"github.com/abueno/go-auth-jwt/internal/domain"
)

// AuthServiceInterface defines the authentication service interface
type AuthServiceInterface interface {
	Signup(ctx context.Context, input SignupInput) (*SignupOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*LoginOutput, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	VerifyEmail(ctx context.Context, input VerifyEmailInput) error
	ResendVerificationEmail(ctx context.Context, email string) (*ResendVerificationEmailOutput, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}