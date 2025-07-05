package email

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template Template
		data     TemplateData
		wantErr  bool
		validate func(t *testing.T, email Email)
	}{
		{
			name:     "verification email",
			template: VerificationEmailTemplate,
			data: TemplateData{
				BaseURL:           "https://example.com",
				AppName:           "Test App",
				SupportEmail:      "support@example.com",
				RecipientEmail:    "user@example.com",
				VerificationToken: "abc123",
				VerificationURL:   "https://example.com/verify?token=abc123",
			},
			wantErr: false,
			validate: func(t *testing.T, email Email) {
				if email.To != "user@example.com" {
					t.Errorf("expected To = user@example.com, got %s", email.To)
				}
				if email.Subject != "Verify your email address" {
					t.Errorf("unexpected subject: %s", email.Subject)
				}
				if !strings.Contains(email.Body, "Welcome to Test App") {
					t.Error("body should contain app name")
				}
				if !strings.Contains(email.Body, "https://example.com/verify?token=abc123") {
					t.Error("body should contain verification URL")
				}
				if !strings.Contains(email.HTMLBody, "Welcome to Test App") {
					t.Error("HTML body should contain app name")
				}
			},
		},
		{
			name:     "password reset email",
			template: PasswordResetEmailTemplate,
			data: TemplateData{
				AppName:        "Test App",
				SupportEmail:   "support@example.com",
				RecipientEmail: "user@example.com",
				ResetToken:     "xyz789",
				ResetURL:       "https://example.com/reset?token=xyz789",
			},
			wantErr: false,
			validate: func(t *testing.T, email Email) {
				if email.Subject != "Reset your password" {
					t.Errorf("unexpected subject: %s", email.Subject)
				}
				if !strings.Contains(email.Body, "reset your password") {
					t.Error("body should contain reset message")
				}
				if !strings.Contains(email.Body, "https://example.com/reset?token=xyz789") {
					t.Error("body should contain reset URL")
				}
			},
		},
		{
			name:     "login notification email",
			template: LoginNotificationEmailTemplate,
			data: TemplateData{
				AppName:        "Test App",
				RecipientEmail: "user@example.com",
				LoginURL:       "https://example.com/login",
			},
			wantErr: false,
			validate: func(t *testing.T, email Email) {
				if email.Subject != "New login to your account" {
					t.Errorf("unexpected subject: %s", email.Subject)
				}
				if !strings.Contains(email.Body, "new login") {
					t.Error("body should contain login notification")
				}
			},
		},
		{
			name: "default values",
			template: Template{
				Subject: "Test {{.CurrentYear}}",
				Body:    "Expires in {{.ExpirationHours}} hours",
			},
			data: TemplateData{
				RecipientEmail: "user@example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, email Email) {
				currentYear := time.Now().Year()
				yearStr := strconv.Itoa(currentYear)
				if !strings.Contains(email.Subject, yearStr) {
					t.Errorf("subject should contain current year %s, got: %s", yearStr, email.Subject)
				}
				if !strings.Contains(email.Body, "24 hours") {
					t.Error("body should contain default expiration hours")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := RenderTemplate(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, email)
			}
		})
	}
}

func TestTemplateData_Defaults(t *testing.T) {
	data := TemplateData{
		RecipientEmail: "test@example.com",
	}

	email, err := RenderTemplate(Template{
		Subject: "Year: {{.CurrentYear}}, Hours: {{.ExpirationHours}}",
		Body:    "Test",
	}, data)

	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	if !strings.Contains(email.Subject, "Hours: 24") {
		t.Errorf("Expected default expiration hours in subject")
	}
}
