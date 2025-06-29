package domain

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with uppercase",
			email:   "User@Example.COM",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
			errType: ErrInvalidEmail,
		},
		{
			name:    "invalid email format",
			email:   "invalid-email",
			wantErr: true,
			errType: ErrInvalidEmail,
		},
		{
			name:    "email with spaces",
			email:   " user@example.com ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != tt.errType {
				t.Errorf("NewUser() error = %v, want %v", err, tt.errType)
				return
			}

			if !tt.wantErr {
				if user == nil {
					t.Error("NewUser() returned nil user without error")
					return
				}

				// Check email is normalized
				expectedEmail := "user@example.com"
				if tt.email == "User@Example.COM" {
					expectedEmail = "user@example.com"
				}
				if user.Email != expectedEmail {
					t.Errorf("User.Email = %v, want %v", user.Email, expectedEmail)
				}

				if user.EmailVerified {
					t.Error("New user should not have verified email")
				}
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"valid email with plus", "user+tag@example.com", false},
		{"valid email with dots", "first.last@example.com", false},
		{"empty email", "", true},
		{"no @ symbol", "userexample.com", true},
		{"no domain", "user@", true},
		{"no username", "@example.com", true},
		{"no TLD", "user@example", true},
		{"multiple @", "user@@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "password123", false},
		{"exact 8 chars", "12345678", false},
		{"long password", "verylongpassword123456", false},
		{"too short", "1234567", true},
		{"empty password", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_MarkEmailVerified(t *testing.T) {
	user := &User{
		Email:         "user@example.com",
		EmailVerified: false,
	}

	token := "verification-token"
	expires := time.Now().Add(24 * time.Hour)
	user.SetEmailVerificationToken(token, expires)

	if user.EmailVerificationToken == nil || *user.EmailVerificationToken != token {
		t.Error("Email verification token not set correctly")
	}

	user.MarkEmailVerified()

	if !user.EmailVerified {
		t.Error("Email should be marked as verified")
	}

	if user.EmailVerificationToken != nil {
		t.Error("Email verification token should be cleared")
	}

	if user.EmailVerificationExpiresAt != nil {
		t.Error("Email verification expiration should be cleared")
	}
}

func TestUser_IsEmailVerificationTokenValid(t *testing.T) {
	user := &User{}

	// Test with no token set
	if user.IsEmailVerificationTokenValid("any-token") {
		t.Error("Should return false when no token is set")
	}

	// Set valid token
	validToken := "valid-token"
	futureTime := time.Now().Add(1 * time.Hour)
	user.SetEmailVerificationToken(validToken, futureTime)

	// Test with correct token
	if !user.IsEmailVerificationTokenValid(validToken) {
		t.Error("Should return true for valid token")
	}

	// Test with incorrect token
	if user.IsEmailVerificationTokenValid("wrong-token") {
		t.Error("Should return false for incorrect token")
	}

	// Test with expired token
	pastTime := time.Now().Add(-1 * time.Hour)
	user.EmailVerificationExpiresAt = &pastTime
	if user.IsEmailVerificationTokenValid(validToken) {
		t.Error("Should return false for expired token")
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	userID := "user-123"

	// Create valid token
	validToken := NewRefreshToken(userID, time.Now().Add(7*24*time.Hour))
	if !validToken.IsValid() {
		t.Error("New token should be valid")
	}

	// Test expired token
	expiredToken := NewRefreshToken(userID, time.Now().Add(-1*time.Hour))
	if expiredToken.IsValid() {
		t.Error("Expired token should not be valid")
	}

	// Test revoked token
	revokedToken := NewRefreshToken(userID, time.Now().Add(7*24*time.Hour))
	revokedToken.Revoke()
	if revokedToken.IsValid() {
		t.Error("Revoked token should not be valid")
	}

	if revokedToken.RevokedAt == nil {
		t.Error("RevokedAt should be set when token is revoked")
	}
}

func TestRefreshToken_UpdateLastUsed(t *testing.T) {
	token := NewRefreshToken("user-123", time.Now().Add(7*24*time.Hour))
	originalTime := token.LastUsedAt

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	token.UpdateLastUsed()

	if !token.LastUsedAt.After(originalTime) {
		t.Error("LastUsedAt should be updated to a later time")
	}
}
