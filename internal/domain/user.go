package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")
	// ErrWeakPassword is returned when password doesn't meet requirements
	ErrWeakPassword = errors.New("password must be at least 8 characters long")
	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrDuplicateEmail is returned when email already exists
	ErrDuplicateEmail = errors.New("email already exists")
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrEmailNotVerified is returned when email is not verified
	ErrEmailNotVerified = errors.New("email not verified")
	// ErrTokenExpired is returned when a token has expired
	ErrTokenExpired = errors.New("token has expired")
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
)

// User represents a user in the system
type User struct {
	ID                         string
	Email                      string
	PasswordHash               string
	EmailVerified              bool
	EmailVerificationToken     *string
	EmailVerificationExpiresAt *time.Time
	PasswordResetToken         *string
	PasswordResetExpiresAt     *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// NewUser creates a new user with validation
func NewUser(email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	return &User{
		Email:         email,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}

	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	return nil
}

// MarkEmailVerified marks the user's email as verified
func (u *User) MarkEmailVerified() {
	u.EmailVerified = true
	u.EmailVerificationToken = nil
	u.EmailVerificationExpiresAt = nil
	u.UpdatedAt = time.Now()
}

// SetEmailVerificationToken sets the email verification token
func (u *User) SetEmailVerificationToken(token string, expiresAt time.Time) {
	u.EmailVerificationToken = &token
	u.EmailVerificationExpiresAt = &expiresAt
	u.UpdatedAt = time.Now()
}

// SetPasswordResetToken sets the password reset token
func (u *User) SetPasswordResetToken(token string, expiresAt time.Time) {
	u.PasswordResetToken = &token
	u.PasswordResetExpiresAt = &expiresAt
	u.UpdatedAt = time.Now()
}

// ClearPasswordResetToken clears the password reset token
func (u *User) ClearPasswordResetToken() {
	u.PasswordResetToken = nil
	u.PasswordResetExpiresAt = nil
	u.UpdatedAt = time.Now()
}

// IsEmailVerificationTokenValid checks if the email verification token is valid
func (u *User) IsEmailVerificationTokenValid(token string) bool {
	if u.EmailVerificationToken == nil || u.EmailVerificationExpiresAt == nil {
		return false
	}

	if *u.EmailVerificationToken != token {
		return false
	}

	return time.Now().Before(*u.EmailVerificationExpiresAt)
}

// IsPasswordResetTokenValid checks if the password reset token is valid
func (u *User) IsPasswordResetTokenValid(token string) bool {
	if u.PasswordResetToken == nil || u.PasswordResetExpiresAt == nil {
		return false
	}

	if *u.PasswordResetToken != token {
		return false
	}

	return time.Now().Before(*u.PasswordResetExpiresAt)
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	Token      string
	UserID     string
	ExpiresAt  time.Time
	Revoked    bool
	RevokedAt  *time.Time
	UserAgent  *string
	IPAddress  *string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

// NewRefreshToken creates a new refresh token
func NewRefreshToken(userID string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		UserID:     userID,
		ExpiresAt:  expiresAt,
		Revoked:    false,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	if rt.Revoked {
		return false
	}

	return time.Now().Before(rt.ExpiresAt)
}

// Revoke marks the token as revoked
func (rt *RefreshToken) Revoke() {
	rt.Revoked = true
	now := time.Now()
	rt.RevokedAt = &now
}

// UpdateLastUsed updates the last used timestamp
func (rt *RefreshToken) UpdateLastUsed() {
	rt.LastUsedAt = time.Now()
}
