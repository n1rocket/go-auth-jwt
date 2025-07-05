package service

import (
	"context"
	"fmt"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/domain"
	"github.com/n1rocket/go-auth-jwt/internal/repository"
	"github.com/n1rocket/go-auth-jwt/internal/security"
)

// UserService handles user-related operations
type UserService struct {
	userRepo       repository.UserRepository
	passwordHasher *security.PasswordHasher
}

// NewUserService creates a new user service
func NewUserService(
	userRepo repository.UserRepository,
	passwordHasher *security.PasswordHasher,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, email, password string) (*domain.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ValidateCredentials validates user credentials
func (s *UserService) ValidateCredentials(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := s.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// MarkEmailVerified marks a user's email as verified
func (s *UserService) MarkEmailVerified(ctx context.Context, userID string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, user)
}
