package email

import (
	"context"
	"io"
	"net"
	"net/smtp"
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

	service := NewSMTPService(config, nil)
	
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

	service := NewSMTPService(config, nil)
	
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
	
	service := NewSMTPService(config, nil)
	
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