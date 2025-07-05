package request

import (
	"fmt"
	"strings"
)

// SignupRequest represents a user signup request
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TrimStrings trims whitespace from string fields
func (r *SignupRequest) TrimStrings() {
	r.Email = strings.TrimSpace(r.Email)
}

// Validate validates the signup request
func (r *SignupRequest) Validate() error {
	if err := ValidateEmail(r.Email); err != nil {
		return fmt.Errorf("email: %w", err)
	}

	if err := ValidatePassword(r.Password); err != nil {
		return fmt.Errorf("password: %w", err)
	}

	return nil
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TrimStrings trims whitespace from string fields
func (r *LoginRequest) TrimStrings() {
	r.Email = strings.TrimSpace(r.Email)
}

// Validate validates the login request
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}

	if r.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TrimStrings trims whitespace from string fields
func (r *RefreshTokenRequest) TrimStrings() {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
}

// Validate validates the refresh token request
func (r *RefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return fmt.Errorf("refresh_token is required")
	}

	if err := ValidateToken(r.RefreshToken); err != nil {
		return fmt.Errorf("refresh_token: %w", err)
	}

	return nil
}

// VerifyEmailRequest represents an email verification request
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// TrimStrings trims whitespace from string fields
func (r *VerifyEmailRequest) TrimStrings() {
	r.Token = strings.TrimSpace(r.Token)
}

// Validate validates the email verification request
func (r *VerifyEmailRequest) Validate() error {
	if r.Token == "" {
		return fmt.Errorf("token is required")
	}

	if len(r.Token) < 10 {
		return fmt.Errorf("invalid verification token")
	}

	return nil
}
