package email

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// MockService implements a mock email service for testing
type MockService struct {
	mu         sync.Mutex
	sentEmails []Email
	failNext   bool
	logger     *slog.Logger
}

// NewMockService creates a new mock email service
func NewMockService(logger *slog.Logger) *MockService {
	return &MockService{
		sentEmails: make([]Email, 0),
		logger:     logger,
	}
}

// Send mock implementation that stores emails in memory
func (m *MockService) Send(ctx context.Context, email Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we should fail
	if m.failNext {
		m.failNext = false
		return fmt.Errorf("mock email service: simulated failure")
	}

	// Store the email
	m.sentEmails = append(m.sentEmails, email)

	// Log the email
	m.logger.Info("mock email sent",
		"to", email.To,
		"subject", email.Subject,
		"body_length", len(email.Body),
		"html_length", len(email.HTMLBody),
	)

	return nil
}

// GetSentEmails returns all emails sent through this mock service
func (m *MockService) GetSentEmails() []Email {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid race conditions
	emails := make([]Email, len(m.sentEmails))
	copy(emails, m.sentEmails)
	return emails
}

// Clear removes all stored emails
func (m *MockService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = make([]Email, 0)
}

// FailNext causes the next Send call to fail
func (m *MockService) FailNext() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = true
}

// GetLastEmail returns the most recently sent email
func (m *MockService) GetLastEmail() (Email, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sentEmails) == 0 {
		return Email{}, false
	}

	return m.sentEmails[len(m.sentEmails)-1], true
}

// CountEmails returns the number of emails sent
func (m *MockService) CountEmails() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sentEmails)
}

// FindEmail finds an email by recipient
func (m *MockService) FindEmail(to string) (Email, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, email := range m.sentEmails {
		if email.To == to {
			return email, true
		}
	}

	return Email{}, false
}
