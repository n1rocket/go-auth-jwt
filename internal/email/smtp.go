package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	TLSEnabled  bool
	Timeout     time.Duration
}

// SMTPService implements the email service using SMTP
type SMTPService struct {
	config SMTPConfig
	logger *slog.Logger
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(config SMTPConfig, logger *slog.Logger) *SMTPService {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &SMTPService{
		config: config,
		logger: logger,
	}
}

// Send sends an email via SMTP
func (s *SMTPService) Send(ctx context.Context, email Email) error {
	// Create deadline from context
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(s.config.Timeout)
	}

	// Prepare email headers and body
	from := s.formatAddress(s.config.FromAddress, s.config.FromName)
	to := email.To
	
	// Build email message
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	
	// If we have HTML body, send multipart email
	if email.HTMLBody != "" {
		boundary := "boundary-" + fmt.Sprintf("%d", time.Now().UnixNano())
		message.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		message.WriteString("\r\n")
		
		// Plain text part
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.Body)
		message.WriteString("\r\n")
		
		// HTML part
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.HTMLBody)
		message.WriteString("\r\n")
		
		// End boundary
		message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Plain text only
		message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.Body)
	}

	// Connect to SMTP server with timeout
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout:  time.Until(deadline),
		Deadline: deadline,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Set deadline on connection
	conn.SetDeadline(deadline)

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Start TLS if enabled
	if s.config.TLSEnabled {
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
			MinVersion: tls.VersionTLS12,
		}
		
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send email data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(message.String()))
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write email data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	client.Quit()

	// Log successful send
	s.logger.Info("email sent successfully",
		"to", to,
		"subject", email.Subject,
	)

	return nil
}

// formatAddress formats an email address with optional name
func (s *SMTPService) formatAddress(email, name string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

// ValidateSMTPConfig validates SMTP configuration
func ValidateSMTPConfig(config SMTPConfig) error {
	if config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", config.Port)
	}
	
	if config.FromAddress == "" {
		return fmt.Errorf("from address is required")
	}
	
	// Validate email format
	if !strings.Contains(config.FromAddress, "@") {
		return fmt.Errorf("invalid from address format")
	}
	
	// Common SMTP ports
	validPorts := map[int]bool{
		25:  true, // SMTP
		465: true, // SMTPS
		587: true, // SMTP with STARTTLS
		2525: true, // Alternative SMTP
	}
	
	if !validPorts[config.Port] {
		slog.Warn("using non-standard SMTP port", "port", config.Port)
	}
	
	// TLS should be enabled for common secure ports
	if (config.Port == 465 || config.Port == 587) && !config.TLSEnabled {
		slog.Warn("TLS is disabled for secure SMTP port", "port", config.Port)
	}
	
	return nil
}