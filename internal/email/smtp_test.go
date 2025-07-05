package email

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

func TestNewSMTPService(t *testing.T) {
	config := SMTPConfig{
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  true,
		Timeout:     30 * time.Second,
	}

	service := NewSMTPService(config, slog.Default())

	if service == nil {
		t.Error("Expected non-nil service")
	}

	// Just check that service is not nil
	// We can't access internal fields directly
}

func TestValidateSMTPConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  SMTPConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
				FromName:    "Test App",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: SMTPConfig{
				Port:        587,
				Username:    "user@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        0,
				Username:    "user@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Password:    "password",
				FromAddress: "noreply@example.com",
			},
			wantErr: false, // Username is not validated
		},
		{
			name: "missing password",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user@example.com",
				FromAddress: "noreply@example.com",
			},
			wantErr: false, // Password is not validated
		},
		{
			name: "missing from address",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user@example.com",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        70000,
				Username:    "user@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSMTPConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSMTPConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test SMTP send with mock dialer
func TestSMTPService_Send(t *testing.T) {
	// This test requires a mock SMTP server or connection
	// For now, we'll test the error cases with invalid config

	config := SMTPConfig{
		Host:        "invalid.smtp.server",
		Port:        587,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  true,
		Timeout:     1 * time.Second, // Short timeout for test
	}

	service := NewSMTPService(config, slog.Default())

	ctx := context.Background()
	email := Email{
		To:      "recipient@example.com",
		Subject: "Test Email",
		Body:    "Test body",
	}

	// This should fail due to invalid server
	err := service.Send(ctx, email)
	if err == nil {
		t.Error("Expected error for invalid SMTP server")
	}

	// Test with canceled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = service.Send(cancelCtx, email)
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// Test SMTP dialer timeout
func TestSMTPService_DialerTimeout(t *testing.T) {
	// Create a listener that doesn't respond
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the port
	addr := listener.Addr().(*net.TCPAddr)

	config := SMTPConfig{
		Host:        "127.0.0.1",
		Port:        addr.Port,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  false,
		Timeout:     100 * time.Millisecond, // Very short timeout
	}

	service := NewSMTPService(config, slog.Default())

	// Accept connections but don't respond (simulate timeout)
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			time.Sleep(200 * time.Millisecond) // Wait longer than timeout
			conn.Close()
		}
	}()

	ctx := context.Background()
	email := Email{
		To:      "recipient@example.com",
		Subject: "Test Email",
		Body:    "Test body",
	}

	err = service.Send(ctx, email)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// Mock SMTP client for testing auth
type mockSMTPClient struct {
	authCalled bool
	authErr    error
	mailErr    error
	rcptErr    error
	dataErr    error
	quitErr    error
}

func (m *mockSMTPClient) Auth(a smtp.Auth) error {
	m.authCalled = true
	return m.authErr
}

func (m *mockSMTPClient) Mail(from string) error {
	return m.mailErr
}

func (m *mockSMTPClient) Rcpt(to string) error {
	return m.rcptErr
}

func (m *mockSMTPClient) Data() (io.WriteCloser, error) {
	if m.dataErr != nil {
		return nil, m.dataErr
	}
	return &nopWriteCloser{}, nil
}

func (m *mockSMTPClient) Quit() error {
	return m.quitErr
}

func (m *mockSMTPClient) Close() error {
	return nil
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (nopWriteCloser) Close() error {
	return nil
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		dispName string
		want     string
	}{
		{
			name:     "email only",
			email:    "test@example.com",
			dispName: "",
			want:     "test@example.com",
		},
		{
			name:     "email with name",
			email:    "test@example.com",
			dispName: "Test User",
			want:     "Test User <test@example.com>",
		},
		{
			name:     "email with empty name",
			email:    "test@example.com",
			dispName: "   ",
			want:     "    <test@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAddress(tt.email, tt.dispName)
			if got != tt.want {
				t.Errorf("FormatAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSMTPService_SendWithHTMLBody(t *testing.T) {
	config := SMTPConfig{
		Host:        "localhost",
		Port:        2525, // Use non-standard port to ensure failure
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  false,
		Timeout:     1 * time.Second,
	}

	service := NewSMTPService(config, slog.Default())

	ctx := context.Background()
	email := Email{
		To:       "recipient@example.com",
		Subject:  "Test Email",
		Body:     "Plain text body",
		HTMLBody: "<html><body>HTML body</body></html>",
	}

	// This should fail due to connection refused
	err := service.Send(ctx, email)
	if err == nil {
		t.Error("Expected error for connection refused")
	}
}

func TestSMTPService_SendWithDeadlineContext(t *testing.T) {
	config := SMTPConfig{
		Host:        "localhost",
		Port:        2525,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  false,
		Timeout:     30 * time.Second,
	}

	service := NewSMTPService(config, slog.Default())

	// Create context with deadline
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
	defer cancel()

	email := Email{
		To:      "recipient@example.com",
		Subject: "Test Email",
		Body:    "Test body",
	}

	// This should fail due to connection refused
	err := service.Send(ctx, email)
	if err == nil {
		t.Error("Expected error for connection refused")
	}
}

func TestNewSMTPService_DefaultTimeout(t *testing.T) {
	config := SMTPConfig{
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		// Timeout not set, should default to 30 seconds
	}

	service := NewSMTPService(config, slog.Default())

	if service.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", service.config.Timeout)
	}
}

// Test SMTP server mock that simulates various failure scenarios
func TestSMTPService_SendWithMockServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)

	// Start mock SMTP server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			// Send SMTP greeting
			conn.Write([]byte("220 mock.smtp.server ESMTP\r\n"))

			// Read and respond to commands
			buf := make([]byte, 1024)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					conn.Close()
					break
				}

				cmd := string(buf[:n])

				// Simple command parsing
				switch {
				case strings.HasPrefix(cmd, "EHLO"):
					conn.Write([]byte("250-mock.smtp.server\r\n250-AUTH PLAIN\r\n250 OK\r\n"))
				case strings.HasPrefix(cmd, "AUTH"):
					conn.Write([]byte("235 Authentication successful\r\n"))
				case strings.HasPrefix(cmd, "MAIL FROM"):
					conn.Write([]byte("250 OK\r\n"))
				case strings.HasPrefix(cmd, "RCPT TO"):
					conn.Write([]byte("250 OK\r\n"))
				case strings.HasPrefix(cmd, "DATA"):
					conn.Write([]byte("354 End data with <CR><LF>.<CR><LF>\r\n"))
				case strings.HasPrefix(cmd, "QUIT"):
					conn.Write([]byte("221 Bye\r\n"))
					conn.Close()
					break
				case strings.Contains(cmd, "\r\n.\r\n"):
					conn.Write([]byte("250 OK: queued\r\n"))
				}
			}
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	config := SMTPConfig{
		Host:        "127.0.0.1",
		Port:        addr.Port,
		Username:    "user@example.com",
		Password:    "password",
		FromAddress: "noreply@example.com",
		FromName:    "Test App",
		TLSEnabled:  false,
		Timeout:     5 * time.Second,
	}

	service := NewSMTPService(config, slog.Default())

	ctx := context.Background()
	email := Email{
		To:      "recipient@example.com",
		Subject: "Test Email",
		Body:    "Test body",
	}

	err = service.Send(ctx, email)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
