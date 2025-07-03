package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/domain"
	"github.com/n1rocket/go-auth-jwt/internal/security"
	"github.com/n1rocket/go-auth-jwt/internal/token"
)

// Mock implementations for testing

type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if _, exists := m.users[user.Email]; exists {
		return domain.ErrDuplicateEmail
	}
	user.ID = "user-" + user.Email
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	for email, user := range m.users {
		if user.ID == id {
			delete(m.users, email)
			return nil
		}
	}
	return domain.ErrUserNotFound
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, exists := m.users[email]
	return exists, nil
}

type mockRefreshTokenRepository struct {
	tokens  map[string]*domain.RefreshToken
	counter int
}

func newMockRefreshTokenRepository() *mockRefreshTokenRepository {
	return &mockRefreshTokenRepository{
		tokens:  make(map[string]*domain.RefreshToken),
		counter: 0,
	}
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	m.counter++
	token.Token = fmt.Sprintf("refresh-%s-%d", token.UserID, m.counter)
	// Make a copy of the token to avoid pointer issues
	tokenCopy := *token
	m.tokens[token.Token] = &tokenCopy
	return nil
}

func (m *mockRefreshTokenRepository) GetByToken(ctx context.Context, tokenValue string) (*domain.RefreshToken, error) {
	token, exists := m.tokens[tokenValue]
	if !exists {
		return nil, domain.ErrInvalidToken
	}
	return token, nil
}

func (m *mockRefreshTokenRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	var tokens []*domain.RefreshToken
	for _, token := range m.tokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *mockRefreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, tokenValue string) error {
	token, exists := m.tokens[tokenValue]
	if !exists {
		return domain.ErrInvalidToken
	}
	token.Revoked = true
	now := time.Now()
	token.RevokedAt = &now
	m.tokens[tokenValue] = token // Update the token in the map
	return nil
}

func (m *mockRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	for _, token := range m.tokens {
		if token.UserID == userID {
			token.Revoke()
		}
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	return nil
}

func (m *mockRefreshTokenRepository) DeleteByToken(ctx context.Context, tokenValue string) error {
	delete(m.tokens, tokenValue)
	return nil
}

// Test helpers

func createTestAuthService(t *testing.T) (*AuthService, *mockUserRepository, *mockRefreshTokenRepository) {
	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	passwordHasher := security.NewDefaultPasswordHasher()
	tokenManager, err := token.NewManager("HS256", "test-secret", "", "", "test-issuer", 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create token manager: %v", err)
	}

	service := NewAuthService(
		userRepo,
		refreshTokenRepo,
		passwordHasher,
		tokenManager,
		7*24*time.Hour, // 7 days refresh token TTL
	)

	return service, userRepo, refreshTokenRepo
}

// Tests

func TestAuthService_Signup(t *testing.T) {
	service, userRepo, _ := createTestAuthService(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   SignupInput
		wantErr bool
		errType error
	}{
		{
			name: "valid signup",
			input: SignupInput{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			input: SignupInput{
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
			errType: domain.ErrInvalidEmail,
		},
		{
			name: "weak password",
			input: SignupInput{
				Email:    "weak@example.com",
				Password: "123",
			},
			wantErr: true,
			errType: domain.ErrWeakPassword,
		},
		{
			name: "duplicate email",
			input: SignupInput{
				Email:    "duplicate@example.com",
				Password: "password123",
			},
			wantErr: false, // First signup should succeed
		},
		{
			name: "duplicate email second attempt",
			input: SignupInput{
				Email:    "duplicate@example.com",
				Password: "password123",
			},
			wantErr: true,
			errType: domain.ErrDuplicateEmail,
		},
	}

	// Run the duplicate email test first
	if _, err := service.Signup(ctx, tests[3].input); err != nil {
		t.Fatalf("Failed to create user for duplicate test: %v", err)
	}

	for _, tt := range tests {
		if tt.name == "duplicate email" {
			continue // Skip the first duplicate test as we already ran it
		}

		t.Run(tt.name, func(t *testing.T) {
			output, err := service.Signup(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Signup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Signup() error = %v, want %v", err, tt.errType)
				}
				return
			}

			if !tt.wantErr {
				if output == nil {
					t.Error("Signup() returned nil output without error")
					return
				}

				if output.UserID == "" {
					t.Error("Signup() returned empty UserID")
				}

				if output.EmailVerificationToken == "" {
					t.Error("Signup() returned empty EmailVerificationToken")
				}

				// Verify user was created
				user, err := userRepo.GetByEmail(ctx, tt.input.Email)
				if err != nil {
					t.Errorf("Failed to get created user: %v", err)
				}

				if user.Email != tt.input.Email {
					t.Errorf("User email = %v, want %v", user.Email, tt.input.Email)
				}

				if user.PasswordHash == "" {
					t.Error("User password hash is empty")
				}

				if user.EmailVerified {
					t.Error("New user should not have verified email")
				}
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	service, _, _ := createTestAuthService(t)
	ctx := context.Background()

	// Create a test user
	signupOutput, err := service.Signup(ctx, SignupInput{
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name    string
		input   LoginInput
		wantErr bool
		errType error
	}{
		{
			name: "valid login",
			input: LoginInput{
				Email:    "login@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Email:    "login@example.com",
				Password: "wrongpassword",
			},
			wantErr: true,
			errType: domain.ErrInvalidCredentials,
		},
		{
			name: "non-existent user",
			input: LoginInput{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			wantErr: true,
			errType: domain.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := service.Login(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Login() error = %v, want %v", err, tt.errType)
				}
				return
			}

			if !tt.wantErr {
				if output == nil {
					t.Error("Login() returned nil output without error")
					return
				}

				if output.AccessToken == "" {
					t.Error("Login() returned empty AccessToken")
				}

				if output.RefreshToken == "" {
					t.Error("Login() returned empty RefreshToken")
				}

				if output.ExpiresIn <= 0 {
					t.Error("Login() returned invalid ExpiresIn")
				}
			}
		})
	}

	_ = signupOutput // Suppress unused variable warning
}

func TestAuthService_Refresh(t *testing.T) {
	service, _, refreshTokenRepo := createTestAuthService(t)
	ctx := context.Background()

	// Create a test user and login
	_, err := service.Signup(ctx, SignupInput{
		Email:    "refresh@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	loginOutput, err := service.Login(ctx, LoginInput{
		Email:    "refresh@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	tests := []struct {
		name    string
		input   RefreshInput
		setup   func()
		wantErr bool
		errType error
	}{
		{
			name: "valid refresh",
			input: RefreshInput{
				RefreshToken: loginOutput.RefreshToken,
			},
			setup: func() {
				// Ensure we start fresh
			},
			wantErr: false,
		},
		{
			name: "invalid refresh token",
			input: RefreshInput{
				RefreshToken: "invalid-token",
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
		{
			name: "revoked refresh token",
			input: RefreshInput{
				RefreshToken: loginOutput.RefreshToken,
			},
			setup: func() {
				// Revoke the token
				refreshTokenRepo.Revoke(ctx, loginOutput.RefreshToken)
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			output, err := service.Refresh(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Refresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Refresh() error = %v, want %v", err, tt.errType)
				}
				return
			}

			if !tt.wantErr {
				if output == nil {
					t.Error("Refresh() returned nil output without error")
					return
				}

				if output.AccessToken == "" {
					t.Error("Refresh() returned empty AccessToken")
				}

				if output.RefreshToken == "" {
					t.Error("Refresh() returned empty RefreshToken")
				}

				// The old token should be revoked
				// Let's wait a bit to ensure different timestamps
				time.Sleep(10 * time.Millisecond)
				
				// Check all tokens to debug
				allTokens, _ := refreshTokenRepo.GetByUserID(ctx, "user-refresh@example.com")
				t.Logf("Total tokens for user: %d", len(allTokens))
				t.Logf("Login token: %s", loginOutput.RefreshToken)
				t.Logf("New token: %s", output.RefreshToken)
				
				for i, tk := range allTokens {
					t.Logf("Token %d: %s, Revoked: %v", i, tk.Token, tk.Revoked)
				}
				
				// The count should be 2: old (revoked) and new
				if len(allTokens) != 2 {
					t.Errorf("Expected 2 tokens, got %d", len(allTokens))
				}
				
				// Check the old token specifically
				oldToken, err := refreshTokenRepo.GetByToken(ctx, loginOutput.RefreshToken)
				if err != nil {
					t.Logf("Error getting old token: %v", err)
				} else if oldToken != nil {
					t.Logf("Old token found: %s, Revoked: %v", oldToken.Token, oldToken.Revoked)
					if !oldToken.Revoked {
						t.Error("Old refresh token should be revoked")
					}
				}
			}
		})
	}
}

func TestAuthService_VerifyEmail(t *testing.T) {
	service, userRepo, _ := createTestAuthService(t)
	ctx := context.Background()

	// Create a test user
	signupOutput, err := service.Signup(ctx, SignupInput{
		Email:    "verify@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// First test invalid token (before valid verification clears the token)
	t.Run("invalid token", func(t *testing.T) {
		err := service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "verify@example.com",
			Token: "invalid-token",
		})
		if err == nil {
			t.Error("VerifyEmail() should return error for invalid token")
		}
		if !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("VerifyEmail() error = %v, want %v", err, domain.ErrInvalidToken)
		}
	})

	// Then test valid verification
	t.Run("valid verification", func(t *testing.T) {
		err := service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "verify@example.com",
			Token: signupOutput.EmailVerificationToken,
		})
		if err != nil {
			t.Errorf("VerifyEmail() error = %v", err)
		}

		// Check that email is verified
		user, err := userRepo.GetByEmail(ctx, "verify@example.com")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if !user.EmailVerified {
			t.Error("Email should be verified after VerifyEmail()")
		}

		if user.EmailVerificationToken != nil {
			t.Error("Email verification token should be cleared")
		}
	})

	// Finally test already verified
	t.Run("already verified", func(t *testing.T) {
		err := service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "verify@example.com",
			Token: "any-token", // Token doesn't matter for already verified
		})
		if err != nil {
			t.Errorf("VerifyEmail() error = %v", err)
		}
	})
}