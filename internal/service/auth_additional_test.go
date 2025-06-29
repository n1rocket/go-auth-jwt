package service

import (
	"context"
	"testing"
	"time"

	"github.com/abueno/go-auth-jwt/internal/domain"
)

func TestAuthService_Logout(t *testing.T) {
	service, _, refreshTokenRepo := createTestAuthService(t)
	ctx := context.Background()

	// Create a user and login to get a refresh token
	_, _ = service.Signup(ctx, SignupInput{
		Email:    "logout@example.com",
		Password: "password123",
	})

	loginOutput, _ := service.Login(ctx, LoginInput{
		Email:    "logout@example.com",
		Password: "password123",
	})

	tests := []struct {
		name         string
		refreshToken string
		wantErr      bool
	}{
		{
			name:         "valid logout",
			refreshToken: loginOutput.RefreshToken,
			wantErr:      false,
		},
		{
			name:         "already revoked token",
			refreshToken: loginOutput.RefreshToken,
			wantErr:      false, // Should not error on already revoked
		},
		{
			name:         "non-existent token",
			refreshToken: "non-existent-token",
			wantErr:      false, // Should not error on non-existent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Logout(ctx, tt.refreshToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("Logout() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify token is revoked
			if tt.refreshToken == loginOutput.RefreshToken {
				token, _ := refreshTokenRepo.GetByToken(ctx, tt.refreshToken)
				if token != nil && !token.Revoked {
					t.Errorf("Expected token to be revoked")
				}
			}
		})
	}
}

func TestAuthService_LogoutAll(t *testing.T) {
	service, _, refreshTokenRepo := createTestAuthService(t)
	ctx := context.Background()

	// Create a user
	signupOutput, _ := service.Signup(ctx, SignupInput{
		Email:    "logoutall@example.com",
		Password: "password123",
	})

	// Create multiple login sessions
	var refreshTokens []string
	for i := 0; i < 3; i++ {
		loginOutput, _ := service.Login(ctx, LoginInput{
			Email:    "logoutall@example.com",
			Password: "password123",
		})
		refreshTokens = append(refreshTokens, loginOutput.RefreshToken)
	}

	// Test logout all
	err := service.LogoutAll(ctx, signupOutput.UserID)
	if err != nil {
		t.Fatalf("LogoutAll() error = %v", err)
	}

	// Verify all tokens are revoked
	for _, tokenStr := range refreshTokens {
		token, _ := refreshTokenRepo.GetByToken(ctx, tokenStr)
		if token != nil && !token.Revoked {
			t.Errorf("Expected token %s to be revoked", tokenStr)
		}
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	service, _, _ := createTestAuthService(t)
	ctx := context.Background()

	// Create a user
	signupOutput, _ := service.Signup(ctx, SignupInput{
		Email:    "getuser@example.com",
		Password: "password123",
	})

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid user ID",
			userID:  signupOutput.UserID,
			wantErr: false,
		},
		{
			name:    "non-existent user ID",
			userID:  "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByID(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && user == nil {
				t.Errorf("Expected user, got nil")
			}
			if !tt.wantErr && user.ID != tt.userID {
				t.Errorf("Expected user ID %s, got %s", tt.userID, user.ID)
			}
		})
	}
}

func TestAuthService_ResendVerificationEmail(t *testing.T) {
	service, userRepo, _ := createTestAuthService(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		email       string
		wantErr     bool
		errContains string
	}{
		{
			name: "unverified user",
			setup: func() string {
				service.Signup(ctx, SignupInput{
					Email:    "unverified@example.com",
					Password: "password123",
				})
				return "unverified@example.com"
			},
			email:   "unverified@example.com",
			wantErr: false,
		},
		{
			name: "already verified user",
			setup: func() string {
				service.Signup(ctx, SignupInput{
					Email:    "verified@example.com",
					Password: "password123",
				})
				// Mark as verified
				user, _ := userRepo.GetByEmail(ctx, "verified@example.com")
				user.MarkEmailVerified()
				userRepo.Update(ctx, user)
				return "verified@example.com"
			},
			email:       "verified@example.com",
			wantErr:     true,
			errContains: "already verified",
		},
		{
			name:        "non-existent user",
			setup:       func() string { return "" },
			email:       "nonexistent@example.com",
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			output, err := service.ResendVerificationEmail(ctx, tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResendVerificationEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			}
			if !tt.wantErr && output.EmailVerificationToken == "" {
				t.Errorf("Expected verification token, got empty")
			}
		})
	}
}

func TestAuthService_EdgeCases(t *testing.T) {
	service, userRepo, refreshTokenRepo := createTestAuthService(t)
	ctx := context.Background()

	t.Run("Signup with existing email", func(t *testing.T) {
		// First signup
		_, err := service.Signup(ctx, SignupInput{
			Email:    "duplicate@example.com",
			Password: "password123",
		})
		if err != nil {
			t.Fatalf("First signup failed: %v", err)
		}

		// Duplicate signup
		_, err = service.Signup(ctx, SignupInput{
			Email:    "duplicate@example.com",
			Password: "password456",
		})
		if err != domain.ErrDuplicateEmail {
			t.Errorf("Expected ErrDuplicateEmail, got %v", err)
		}
	})

	t.Run("Login with non-existent user", func(t *testing.T) {
		_, err := service.Login(ctx, LoginInput{
			Email:    "nonexistent@example.com",
			Password: "password123",
		})
		if err != domain.ErrInvalidCredentials {
			t.Errorf("Expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("Refresh with expired token", func(t *testing.T) {
		// Create a user and login
		service.Signup(ctx, SignupInput{
			Email:    "expired@example.com",
			Password: "password123",
		})
		loginOutput, _ := service.Login(ctx, LoginInput{
			Email:    "expired@example.com",
			Password: "password123",
		})

		// Manually expire the token
		token, _ := refreshTokenRepo.GetByToken(ctx, loginOutput.RefreshToken)
		token.ExpiresAt = time.Now().Add(-1 * time.Hour)
		refreshTokenRepo.Update(ctx, token)

		// Try to refresh
		_, err := service.Refresh(ctx, RefreshInput{
			RefreshToken: loginOutput.RefreshToken,
		})
		if err != domain.ErrInvalidToken {
			t.Errorf("Expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("Verify email with invalid token", func(t *testing.T) {
		// Create a user
		service.Signup(ctx, SignupInput{
			Email:    "verify@example.com",
			Password: "password123",
		})

		// Try to verify with wrong token
		err := service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "verify@example.com",
			Token: "wrong-token",
		})
		if err != domain.ErrInvalidToken {
			t.Errorf("Expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("Verify already verified email", func(t *testing.T) {
		// Create and verify a user
		_, _ = service.Signup(ctx, SignupInput{
			Email:    "alreadyverified@example.com",
			Password: "password123",
		})
		
		user, _ := userRepo.GetByEmail(ctx, "alreadyverified@example.com")
		verifyToken := user.EmailVerificationToken

		// First verification
		err := service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "alreadyverified@example.com",
			Token: *verifyToken,
		})
		if err != nil {
			t.Fatalf("First verification failed: %v", err)
		}

		// Second verification (should succeed without error)
		err = service.VerifyEmail(ctx, VerifyEmailInput{
			Email: "alreadyverified@example.com",
			Token: *verifyToken,
		})
		if err != nil {
			t.Errorf("Expected no error for already verified email, got %v", err)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}