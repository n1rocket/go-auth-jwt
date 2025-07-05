package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/n1rocket/go-auth-jwt/internal/config"
	emailpkg "github.com/n1rocket/go-auth-jwt/internal/email"
	"github.com/n1rocket/go-auth-jwt/internal/worker"
)

// AuthServiceWithEmail extends AuthService with email functionality
type AuthServiceWithEmail struct {
	*AuthService
	emailDispatcher *worker.EmailDispatcher
	config          *config.Config
	logger          *slog.Logger
}

// NewAuthServiceWithEmail creates an auth service with email support
func NewAuthServiceWithEmail(
	authService *AuthService,
	emailDispatcher *worker.EmailDispatcher,
	config *config.Config,
	logger *slog.Logger,
) *AuthServiceWithEmail {
	return &AuthServiceWithEmail{
		AuthService:     authService,
		emailDispatcher: emailDispatcher,
		config:          config,
		logger:          logger,
	}
}

// SignupWithEmail creates a new user and sends verification email
func (s *AuthServiceWithEmail) SignupWithEmail(ctx context.Context, input SignupInput) (*SignupOutput, error) {
	// Call the base signup method
	output, err := s.AuthService.Signup(ctx, input)
	if err != nil {
		return nil, err
	}

	// Prepare email data
	emailData := emailpkg.TemplateData{
		BaseURL:           s.config.App.BaseURL,
		AppName:           s.config.App.Name,
		SupportEmail:      s.config.Email.SupportEmail,
		RecipientEmail:    input.Email,
		VerificationToken: output.EmailVerificationToken,
		VerificationURL: fmt.Sprintf("%s/verify-email?token=%s&email=%s",
			s.config.App.BaseURL,
			output.EmailVerificationToken,
			input.Email,
		),
		ExpirationHours: 24,
	}

	// Render verification email
	verificationEmail, err := emailpkg.RenderTemplate(emailpkg.VerificationEmailTemplate, emailData)
	if err != nil {
		s.logger.Error("failed to render verification email",
			"error", err,
			"user_id", output.UserID,
			"email", input.Email,
		)
		// Don't fail signup if email rendering fails
		return output, nil
	}

	// Queue email for sending
	if err := s.emailDispatcher.EnqueueWithContext(ctx, verificationEmail); err != nil {
		s.logger.Error("failed to queue verification email",
			"error", err,
			"user_id", output.UserID,
			"email", input.Email,
		)
		// Don't fail signup if email queueing fails
	} else {
		s.logger.Info("verification email queued",
			"user_id", output.UserID,
			"email", input.Email,
		)
	}

	return output, nil
}

// ResendVerificationEmailWithNotification resends verification email
func (s *AuthServiceWithEmail) ResendVerificationEmailWithNotification(ctx context.Context, emailAddress string) (*ResendVerificationEmailOutput, error) {
	// Call the base method
	output, err := s.AuthService.ResendVerificationEmail(ctx, emailAddress)
	if err != nil {
		return nil, err
	}

	// Prepare email data
	emailData := emailpkg.TemplateData{
		BaseURL:           s.config.App.BaseURL,
		AppName:           s.config.App.Name,
		SupportEmail:      s.config.Email.SupportEmail,
		RecipientEmail:    emailAddress,
		VerificationToken: output.EmailVerificationToken,
		VerificationURL: fmt.Sprintf("%s/verify-email?token=%s&email=%s",
			s.config.App.BaseURL,
			output.EmailVerificationToken,
			emailAddress,
		),
		ExpirationHours: 24,
	}

	// Render verification email
	verificationEmail, err := emailpkg.RenderTemplate(emailpkg.VerificationEmailTemplate, emailData)
	if err != nil {
		s.logger.Error("failed to render verification email",
			"error", err,
			"email", emailAddress,
		)
		return output, nil
	}

	// Queue email for sending
	if err := s.emailDispatcher.EnqueueWithContext(ctx, verificationEmail); err != nil {
		s.logger.Error("failed to queue verification email",
			"error", err,
			"email", emailAddress,
		)
	} else {
		s.logger.Info("verification email re-sent",
			"email", emailAddress,
		)
	}

	return output, nil
}

// LoginWithNotification sends login notification for security
func (s *AuthServiceWithEmail) LoginWithNotification(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Call the base login method
	output, err := s.AuthService.Login(ctx, input)
	if err != nil {
		return nil, err
	}

	// Check if login notifications are enabled
	if !s.config.Email.SendLoginNotifications {
		return output, nil
	}

	// Prepare email data
	emailData := emailpkg.TemplateData{
		BaseURL:        s.config.App.BaseURL,
		AppName:        s.config.App.Name,
		SupportEmail:   s.config.Email.SupportEmail,
		RecipientEmail: input.Email,
		LoginURL:       fmt.Sprintf("%s/account/security", s.config.App.BaseURL),
	}

	// Render login notification email
	loginEmail, err := emailpkg.RenderTemplate(emailpkg.LoginNotificationEmailTemplate, emailData)
	if err != nil {
		s.logger.Error("failed to render login notification email",
			"error", err,
			"email", input.Email,
		)
		return output, nil
	}

	// Queue email for sending (don't wait)
	go func() {
		if err := s.emailDispatcher.Enqueue(loginEmail); err != nil {
			s.logger.Error("failed to queue login notification email",
				"error", err,
				"email", input.Email,
			)
		}
	}()

	return output, nil
}
