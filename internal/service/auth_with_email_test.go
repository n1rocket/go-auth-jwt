package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/abueno/go-auth-jwt/internal/config"
	"github.com/abueno/go-auth-jwt/internal/domain"
	"github.com/abueno/go-auth-jwt/internal/email"
	"github.com/abueno/go-auth-jwt/internal/repository"
	"github.com/abueno/go-auth-jwt/internal/security"
	"github.com/abueno/go-auth-jwt/internal/token"
	"github.com/abueno/go-auth-jwt/internal/worker"
)

// mockEmailService for testing
type mockEmailService struct {
	sendFunc func(ctx context.Context, email email.Email) error
}

func (m *mockEmailService) Send(ctx context.Context, email email.Email) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, email)
	}
	return nil
}

// Create a wrapper struct that embeds EmailDispatcher
type testEmailDispatcher struct {
	*worker.EmailDispatcher
	enqueueFunc            func(email email.Email) error
	enqueueWithContextFunc func(ctx context.Context, email email.Email) error
}

// Override the methods we want to mock
func (t *testEmailDispatcher) Enqueue(email email.Email) error {
	if t.enqueueFunc != nil {
		return t.enqueueFunc(email)
	}
	return nil
}

func (t *testEmailDispatcher) EnqueueWithContext(ctx context.Context, email email.Email) error {
	if t.enqueueWithContextFunc != nil {
		return t.enqueueWithContextFunc(ctx, email)
	}
	return nil
}

// Mock user repository that implements the full interface
type mockUserRepositoryWithEmail struct {
	createFunc        func(ctx context.Context, user *domain.User) error
	getByEmailFunc    func(ctx context.Context, email string) (*domain.User, error)
	getByIDFunc       func(ctx context.Context, id string) (*domain.User, error)
	updateFunc        func(ctx context.Context, user *domain.User) error
	deleteFunc        func(ctx context.Context, id string) error
	existsByEmailFunc func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepositoryWithEmail) Create(ctx context.Context, user *domain.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	// Set a default ID if not already set
	if user.ID == "" {
		user.ID = "user-" + user.Email
	}
	return nil
}

func (m *mockUserRepositoryWithEmail) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return &domain.User{}, nil
}

func (m *mockUserRepositoryWithEmail) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.User{}, nil
}

func (m *mockUserRepositoryWithEmail) Update(ctx context.Context, user *domain.User) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, user)
	}
	return nil
}

func (m *mockUserRepositoryWithEmail) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockUserRepositoryWithEmail) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFunc != nil {
		return m.existsByEmailFunc(ctx, email)
	}
	return false, nil
}

// Mock refresh token repository
type mockRefreshTokenRepositoryWithEmail struct {
	tokens map[string]*domain.RefreshToken
	counter int
}

func newMockRefreshTokenRepositoryWithEmail() *mockRefreshTokenRepositoryWithEmail {
	return &mockRefreshTokenRepositoryWithEmail{
		tokens:  make(map[string]*domain.RefreshToken),
		counter: 0,
	}
}

func (m *mockRefreshTokenRepositoryWithEmail) Create(ctx context.Context, token *domain.RefreshToken) error {
	if m.tokens == nil {
		m.tokens = make(map[string]*domain.RefreshToken)
	}
	m.counter++
	token.Token = fmt.Sprintf("refresh-%s-%d", token.UserID, m.counter)
	m.tokens[token.Token] = token
	return nil
}

func (m *mockRefreshTokenRepositoryWithEmail) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if m.tokens == nil {
		return nil, domain.ErrInvalidToken
	}
	if t, ok := m.tokens[token]; ok {
		return t, nil
	}
	return nil, domain.ErrInvalidToken
}

func (m *mockRefreshTokenRepositoryWithEmail) GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	var tokens []*domain.RefreshToken
	for _, token := range m.tokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *mockRefreshTokenRepositoryWithEmail) Update(ctx context.Context, token *domain.RefreshToken) error {
	return nil
}

func (m *mockRefreshTokenRepositoryWithEmail) Revoke(ctx context.Context, token string) error {
	return nil
}

func (m *mockRefreshTokenRepositoryWithEmail) RevokeAllForUser(ctx context.Context, userID string) error {
	return nil
}

func (m *mockRefreshTokenRepositoryWithEmail) DeleteExpired(ctx context.Context) error {
	return nil
}

func (m *mockRefreshTokenRepositoryWithEmail) DeleteByToken(ctx context.Context, token string) error {
	return nil
}

// Helper to create test configuration
func createTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "Test App",
			BaseURL: "http://localhost:8080",
		},
		Email: config.EmailConfig{
			SupportEmail:           "support@test.com",
			SendLoginNotifications: true,
		},
	}
}

// Helper to create test auth service with email
func createTestAuthServiceWithEmail(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	emailSvc email.Service,
) *AuthServiceWithEmail {
	if userRepo == nil {
		userRepo = &mockUserRepositoryWithEmail{}
	}
	if refreshRepo == nil {
		refreshRepo = newMockRefreshTokenRepositoryWithEmail()
	}
	if emailSvc == nil {
		emailSvc = &mockEmailService{}
	}
	
	passwordHasher := security.NewPasswordHasher(10)
	tokenManager, _ := token.NewManager("HS256", "test-secret", "", "", "test-issuer", 3600*time.Second)
	
	authService := NewAuthService(userRepo, refreshRepo, passwordHasher, tokenManager, 24*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := createTestConfig()
	
	// Create a real EmailDispatcher with mock email service
	dispatcherConfig := worker.DefaultConfig()
	dispatcher := worker.NewEmailDispatcher(emailSvc, dispatcherConfig, logger)
	
	return NewAuthServiceWithEmail(authService, dispatcher, cfg, logger)
}

// Create a test helper that uses testEmailDispatcher
func createTestAuthServiceWithMockDispatcher(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	dispatcher *testEmailDispatcher,
) *AuthServiceWithEmail {
	if userRepo == nil {
		userRepo = &mockUserRepositoryWithEmail{}
	}
	if refreshRepo == nil {
		refreshRepo = newMockRefreshTokenRepositoryWithEmail()
	}
	
	passwordHasher := security.NewPasswordHasher(10)
	tokenManager, _ := token.NewManager("HS256", "test-secret", "", "", "test-issuer", 3600*time.Second)
	
	authService := NewAuthService(userRepo, refreshRepo, passwordHasher, tokenManager, 24*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := createTestConfig()
	
	return &AuthServiceWithEmail{
		AuthService:     authService,
		emailDispatcher: dispatcher.EmailDispatcher,
		config:          cfg,
		logger:          logger,
	}
}

func TestNewAuthServiceWithEmail(t *testing.T) {
	authService := &AuthService{}
	emailService := &mockEmailService{}
	cfg := createTestConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcherConfig := worker.DefaultConfig()
	dispatcher := worker.NewEmailDispatcher(emailService, dispatcherConfig, logger)
	
	service := NewAuthServiceWithEmail(authService, dispatcher, cfg, logger)
	
	if service.AuthService != authService {
		t.Error("Expected AuthService to be set")
	}
	if service.emailDispatcher == nil {
		t.Error("Expected emailDispatcher to be set")
	}
	if service.config != cfg {
		t.Error("Expected config to be set")
	}
	if service.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestAuthServiceWithEmail_SignupWithEmail(t *testing.T) {
	tests := []struct {
		name            string
		input           SignupInput
		userRepo        repository.UserRepository
		emailService    *mockEmailService
		expectError     bool
		expectEmailSent bool
	}{
		{
			name: "successful signup with email",
			input: SignupInput{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			emailService: &mockEmailService{
				sendFunc: func(ctx context.Context, email email.Email) error {
					return nil
				},
			},
			expectEmailSent: true,
		},
		{
			name: "signup succeeds even if email sending fails",
			input: SignupInput{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			emailService: &mockEmailService{
				sendFunc: func(ctx context.Context, email email.Email) error {
					return errors.New("email service down")
				},
			},
			expectEmailSent: false,
		},
		{
			name: "signup fails with duplicate email",
			input: SignupInput{
				Email:    "existing@example.com",
				Password: "Password123!",
			},
			userRepo: &mockUserRepositoryWithEmail{
				existsByEmailFunc: func(ctx context.Context, email string) (bool, error) {
					return true, nil
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestAuthServiceWithEmail(tt.userRepo, nil, tt.emailService)
			
			output, err := service.SignupWithEmail(context.Background(), tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if output == nil {
				t.Error("Expected output but got nil")
				return
			}
			
			if output.UserID == "" {
				t.Error("Expected UserID to be set")
			}
			
			if output.EmailVerificationToken == "" {
				t.Error("Expected EmailVerificationToken to be set")
			}
		})
	}
}

func TestAuthServiceWithEmail_ResendVerificationEmailWithNotification(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		userRepo        repository.UserRepository
		expectError     bool
	}{
		{
			name:  "successful resend",
			email: "test@example.com",
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					token := "old-token"
					expiry := time.Now().Add(-1 * time.Hour) // Expired
					return &domain.User{
						ID:                         "user-123",
						Email:                      email,
						EmailVerified:              false,
						EmailVerificationToken:     &token,
						EmailVerificationExpiresAt: &expiry,
						CreatedAt:                  time.Now(),
						UpdatedAt:                  time.Now(),
					}, nil
				},
			},
		},
		{
			name:  "user not found",
			email: "notfound@example.com",
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				},
			},
			expectError: true,
		},
		{
			name:  "user already verified",
			email: "verified@example.com",
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:            "user-123",
						Email:         email,
						EmailVerified: true,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					}, nil
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestAuthServiceWithEmail(tt.userRepo, nil, nil)
			
			output, err := service.ResendVerificationEmailWithNotification(context.Background(), tt.email)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if output == nil {
				t.Error("Expected output but got nil")
				return
			}
			
			if output.EmailVerificationToken == "" {
				t.Error("Expected EmailVerificationToken to be set")
			}
		})
	}
}

func TestAuthServiceWithEmail_LoginWithNotification(t *testing.T) {
	// Create a valid password hash for testing
	passwordHasher := security.NewPasswordHasher(10)
	validHash, _ := passwordHasher.Hash("Password123!")
	
	tests := []struct {
		name              string
		input             LoginInput
		userRepo          repository.UserRepository
		config            *config.Config
		expectError       bool
	}{
		{
			name: "successful login with notification",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:            "user-123",
						Email:         email,
						EmailVerified: true,
						PasswordHash:  validHash,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					}, nil
				},
			},
		},
		{
			name: "successful login without notification when disabled",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:            "user-123",
						Email:         email,
						EmailVerified: true,
						PasswordHash:  validHash,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					}, nil
				},
			},
			config: &config.Config{
				App: config.AppConfig{
					Name:    "Test App",
					BaseURL: "http://localhost:8080",
				},
				Email: config.EmailConfig{
					SupportEmail:           "support@test.com",
					SendLoginNotifications: false, // Disabled
				},
			},
		},
		{
			name: "invalid password",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "WrongPassword123!",
			},
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:            "user-123",
						Email:         email,
						EmailVerified: true,
						PasswordHash:  validHash,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					}, nil
				},
			},
			expectError: true,
		},
		{
			name: "user not found",
			input: LoginInput{
				Email:    "notfound@example.com",
				Password: "Password123!",
			},
			userRepo: &mockUserRepositoryWithEmail{
				getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestAuthServiceWithEmail(tt.userRepo, nil, nil)
			if tt.config != nil {
				service.config = tt.config
			}
			
			// Add timeout to avoid waiting indefinitely for goroutine
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			
			output, err := service.LoginWithNotification(ctx, tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if output == nil {
				t.Error("Expected output but got nil")
				return
			}
			
			if output.AccessToken == "" {
				t.Error("Expected AccessToken to be set")
			}
			
			if output.RefreshToken == "" {
				t.Error("Expected RefreshToken to be set")
			}
			
			// Give goroutine time to execute
			time.Sleep(50 * time.Millisecond)
		})
	}
}