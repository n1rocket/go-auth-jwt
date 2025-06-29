package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abueno/go-auth-jwt/internal/domain"
	"github.com/abueno/go-auth-jwt/internal/repository"
	"github.com/abueno/go-auth-jwt/internal/security"
	"github.com/abueno/go-auth-jwt/internal/token"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	passwordHasher   *security.PasswordHasher
	tokenManager     *token.Manager
	refreshTokenTTL  time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	passwordHasher *security.PasswordHasher,
	tokenManager *token.Manager,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		passwordHasher:   passwordHasher,
		tokenManager:     tokenManager,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// SignupInput represents the input for signup
type SignupInput struct {
	Email    string
	Password string
}

// SignupOutput represents the output for signup
type SignupOutput struct {
	UserID                 string
	EmailVerificationToken string
}

// Signup creates a new user account
func (s *AuthService) Signup(ctx context.Context, input SignupInput) (*SignupOutput, error) {
	// Validate email
	if err := domain.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := domain.ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, domain.ErrDuplicateEmail
	}

	// Create new user
	user, err := domain.NewUser(input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Hash password
	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = passwordHash

	// Generate email verification token
	verificationToken, err := security.GenerateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Set verification token with 24-hour expiry
	user.SetEmailVerificationToken(verificationToken, time.Now().Add(24*time.Hour))

	// Save user to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &SignupOutput{
		UserID:                 user.ID,
		EmailVerificationToken: verificationToken,
	}, nil
}

// LoginInput represents the input for login
type LoginInput struct {
	Email     string
	Password  string
	UserAgent *string
	IPAddress *string
}

// LoginOutput represents the output for login
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := s.passwordHasher.Compare(input.Password, user.PasswordHash); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Check if email is verified (optional - depends on business requirements)
	// if !user.EmailVerified {
	//     return nil, domain.ErrEmailNotVerified
	// }

	// Generate access token
	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.EmailVerified)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Create refresh token
	refreshToken := domain.NewRefreshToken(user.ID, time.Now().Add(s.refreshTokenTTL))
	refreshToken.UserAgent = input.UserAgent
	refreshToken.IPAddress = input.IPAddress

	// Save refresh token
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    int64(s.refreshTokenTTL.Seconds()),
	}, nil
}

// RefreshInput represents the input for token refresh
type RefreshInput struct {
	RefreshToken string
	UserAgent    *string
	IPAddress    *string
}

// Refresh generates new tokens using a refresh token
func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (*LoginOutput, error) {
	// Get refresh token
	refreshToken, err := s.refreshTokenRepo.GetByToken(ctx, input.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Validate refresh token
	if !refreshToken.IsValid() {
		return nil, domain.ErrInvalidToken
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Rotate refresh token (create new, revoke old)
	if err := s.refreshTokenRepo.Revoke(ctx, input.RefreshToken); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Generate new access token
	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.EmailVerified)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Create new refresh token
	newRefreshToken := domain.NewRefreshToken(user.ID, time.Now().Add(s.refreshTokenTTL))
	newRefreshToken.UserAgent = input.UserAgent
	newRefreshToken.IPAddress = input.IPAddress

	// Save new refresh token
	if err := s.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
		ExpiresIn:    int64(s.refreshTokenTTL.Seconds()),
	}, nil
}

// Logout revokes the refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.refreshTokenRepo.Revoke(ctx, refreshToken); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			// Token already revoked or doesn't exist - not an error for logout
			return nil
		}
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// LogoutAll revokes all refresh tokens for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
	}

	return nil
}

// VerifyEmailInput represents the input for email verification
type VerifyEmailInput struct {
	Email string
	Token string
}

// VerifyEmail verifies a user's email address
func (s *AuthService) VerifyEmail(ctx context.Context, input VerifyEmailInput) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return nil // Already verified, not an error
	}

	// Validate token
	if !user.IsEmailVerificationTokenValid(input.Token) {
		return domain.ErrInvalidToken
	}

	// Mark email as verified
	user.MarkEmailVerified()

	// Update user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// ResendVerificationEmailOutput represents the output for resending verification email
type ResendVerificationEmailOutput struct {
	EmailVerificationToken string
}

// ResendVerificationEmail generates a new verification token and returns it
func (s *AuthService) ResendVerificationEmail(ctx context.Context, email string) (*ResendVerificationEmailOutput, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return nil, errors.New("email already verified")
	}

	// Generate new verification token
	verificationToken, err := security.GenerateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Set new token with 24-hour expiry
	user.SetEmailVerificationToken(verificationToken, time.Now().Add(24*time.Hour))

	// Update user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &ResendVerificationEmailOutput{
		EmailVerificationToken: verificationToken,
	}, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}