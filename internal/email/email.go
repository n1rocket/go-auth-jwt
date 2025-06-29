package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"
)

// Email represents an email message
type Email struct {
	To          string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}

// Service defines the email service interface
type Service interface {
	Send(ctx context.Context, email Email) error
}

// Template represents an email template
type Template struct {
	Subject string
	Body    string
	HTML    string
}

// TemplateData represents data for email templates
type TemplateData struct {
	BaseURL               string
	AppName               string
	SupportEmail          string
	CurrentYear           int
	RecipientEmail        string
	RecipientName         string
	VerificationToken     string
	VerificationURL       string
	ResetToken           string
	ResetURL             string
	LoginURL             string
	ExpirationHours      int
}

// Templates for different email types
var (
	VerificationEmailTemplate = Template{
		Subject: "Verify your email address",
		Body: `Hello,

Welcome to {{.AppName}}! Please verify your email address by clicking the link below:

{{.VerificationURL}}

This link will expire in {{.ExpirationHours}} hours.

If you didn't create an account, please ignore this email.

Best regards,
The {{.AppName}} Team`,
		HTML: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify your email address</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #007bff; color: white; text-decoration: none; border-radius: 4px; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 14px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{.AppName}}!</h1>
        </div>
        <div class="content">
            <p>Hello,</p>
            <p>Thank you for signing up! Please verify your email address by clicking the button below:</p>
            <p style="text-align: center; margin: 30px 0;">
                <a href="{{.VerificationURL}}" class="button">Verify Email Address</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #007bff;">{{.VerificationURL}}</p>
            <p>This link will expire in {{.ExpirationHours}} hours.</p>
            <p>If you didn't create an account, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.CurrentYear}} {{.AppName}}. All rights reserved.</p>
            <p>If you have any questions, contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
        </div>
    </div>
</body>
</html>`,
	}

	PasswordResetEmailTemplate = Template{
		Subject: "Reset your password",
		Body: `Hello,

We received a request to reset your password for your {{.AppName}} account.

Click the link below to reset your password:

{{.ResetURL}}

This link will expire in {{.ExpirationHours}} hours.

If you didn't request a password reset, please ignore this email. Your password won't be changed.

Best regards,
The {{.AppName}} Team`,
		HTML: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset your password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #dc3545; color: white; text-decoration: none; border-radius: 4px; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 14px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hello,</p>
            <p>We received a request to reset your password for your {{.AppName}} account.</p>
            <p style="text-align: center; margin: 30px 0;">
                <a href="{{.ResetURL}}" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #dc3545;">{{.ResetURL}}</p>
            <p>This link will expire in {{.ExpirationHours}} hours.</p>
            <p>If you didn't request a password reset, please ignore this email. Your password won't be changed.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.CurrentYear}} {{.AppName}}. All rights reserved.</p>
            <p>If you have any questions, contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
        </div>
    </div>
</body>
</html>`,
	}

	LoginNotificationEmailTemplate = Template{
		Subject: "New login to your account",
		Body: `Hello,

We detected a new login to your {{.AppName}} account.

If this was you, you can safely ignore this email.

If you didn't log in, please secure your account immediately by changing your password.

Best regards,
The {{.AppName}} Team`,
		HTML: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New login detected</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .button { display: inline-block; padding: 12px 24px; background-color: #ffc107; color: #212529; text-decoration: none; border-radius: 4px; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 14px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Security Alert</h1>
        </div>
        <div class="content">
            <p>Hello,</p>
            <p>We detected a new login to your {{.AppName}} account.</p>
            <div class="warning">
                <p><strong>If this wasn't you:</strong></p>
                <p>Please secure your account immediately by changing your password.</p>
                <p style="text-align: center; margin: 20px 0;">
                    <a href="{{.LoginURL}}" class="button">Secure My Account</a>
                </p>
            </div>
            <p>If this was you, you can safely ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.CurrentYear}} {{.AppName}}. All rights reserved.</p>
            <p>If you have any questions, contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
        </div>
    </div>
</body>
</html>`,
	}
)

// RenderTemplate renders an email template with the provided data
func RenderTemplate(tmpl Template, data TemplateData) (Email, error) {
	// Set default values
	if data.CurrentYear == 0 {
		data.CurrentYear = time.Now().Year()
	}
	if data.ExpirationHours == 0 {
		data.ExpirationHours = 24
	}

	// Render subject
	subjectTmpl, err := template.New("subject").Parse(tmpl.Subject)
	if err != nil {
		return Email{}, fmt.Errorf("failed to parse subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return Email{}, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render plain text body
	bodyTmpl, err := template.New("body").Parse(tmpl.Body)
	if err != nil {
		return Email{}, fmt.Errorf("failed to parse body template: %w", err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return Email{}, fmt.Errorf("failed to render body: %w", err)
	}

	// Render HTML body if provided
	var htmlBuf bytes.Buffer
	if tmpl.HTML != "" {
		htmlTmpl, err := template.New("html").Parse(tmpl.HTML)
		if err != nil {
			return Email{}, fmt.Errorf("failed to parse HTML template: %w", err)
		}
		if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
			return Email{}, fmt.Errorf("failed to render HTML: %w", err)
		}
	}

	return Email{
		To:       data.RecipientEmail,
		Subject:  subjectBuf.String(),
		Body:     bodyBuf.String(),
		HTMLBody: htmlBuf.String(),
	}, nil
}